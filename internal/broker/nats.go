package broker

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/neo-vai/go-events/internal/model/event"
)

const (
	streamName     = "EVENTS"
	streamSubjects = "events"
)

// NATSClient wraps a NATS connection.
type NATSClient struct {
	Conn *nats.Conn
	JS   nats.JetStreamContext // optional, nil if JetStream disabled
}

// NewNATSClient connects to NATS with the given URL and optionally initializes JetStream.
func NewNATSClient(url string, jetStreamEnabled bool) (*NATSClient, error) {
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, err
	}
	slog.Info("connected to NATS", "url", url)

	client := &NATSClient{Conn: nc}
	if jetStreamEnabled {
		js, err := nc.JetStream()
		if err != nil {
			nc.Close()
			return nil, err
		}
		client.JS = js
		if err := client.ensureStream(); err != nil {
			nc.Close()
			return nil, err
		}
		slog.Info("JetStream enabled and stream ensured", "stream", streamName)
	}
	return client, nil
}

// ensureStream creates the stream if it does not exist.
func (c *NATSClient) ensureStream() error {
	if c.JS == nil {
		return nil
	}
	_, err := c.JS.StreamInfo(streamName)
	if err == nil {
		return nil
	}
	if !errors.Is(err, nats.ErrStreamNotFound) {
		return err
	}
	_, err = c.JS.AddStream(&nats.StreamConfig{
		Name:      streamName,
		Subjects:  []string{streamSubjects},
		Retention: nats.WorkQueuePolicy,
		Storage:   nats.FileStorage,
	})
	return err
}

// Close closes the NATS connection.
func (c *NATSClient) Close() {
	c.Conn.Close()
}

// NATSPublisher implements Publisher using NATS (core or JetStream).
type NATSPublisher struct {
	conn             *nats.Conn
	js               nats.JetStreamContext
	subject          string
	jetStreamEnabled bool
}

// NewNATSPublisher creates a new NATS publisher.
func NewNATSPublisher(client *NATSClient, subject string) *NATSPublisher {
	return &NATSPublisher{
		conn:             client.Conn,
		js:               client.JS,
		subject:          subject,
		jetStreamEnabled: client.JS != nil,
	}
}

// PublishEvent marshals and publishes an event to NATS.
func (p *NATSPublisher) PublishEvent(ctx context.Context, ev *event.Event) error {
	data, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	if p.jetStreamEnabled {
		_, err = p.js.Publish(p.subject, data)
	} else {
		err = p.conn.Publish(p.subject, data)
	}
	if err != nil {
		slog.Error("failed to publish event to NATS", "error", err)
		return err
	}
	slog.Debug("event published to NATS", "event_id", ev.ID)
	return nil
}

// NATSSubscriber implements Subscriber using NATS (core or JetStream).
type NATSSubscriber struct {
	conn             *nats.Conn
	js               nats.JetStreamContext
	subject          string
	queueGroup       string
	jetStreamEnabled bool
	sub              interface {
		Unsubscribe() error
	}
}

// NewNATSSubscriber creates a new NATS subscriber with a queue group.
func NewNATSSubscriber(client *NATSClient, subject, queueGroup string) *NATSSubscriber {
	return &NATSSubscriber{
		conn:             client.Conn,
		js:               client.JS,
		subject:          subject,
		queueGroup:       queueGroup,
		jetStreamEnabled: client.JS != nil,
	}
}

// Subscribe starts listening for messages and calls the handler for each.
func (s *NATSSubscriber) Subscribe(handler func(ev *event.Event) error) error {
	if s.jetStreamEnabled {
		// JetStream with durable consumer and manual ack.
		sub, err := s.js.QueueSubscribe(
			s.subject,
			s.queueGroup,
			func(msg *nats.Msg) {
				var ev event.Event
				if err := json.Unmarshal(msg.Data, &ev); err != nil {
					slog.Error("failed to unmarshal event", "error", err)
					_ = msg.Nak()
					return
				}
				if err := handler(&ev); err != nil {
					slog.Error("event handler failed", "error", err, "event_id", ev.ID)
					_ = msg.Nak()
					return
				}
				_ = msg.Ack()
				slog.Info("event processed", "event_id", ev.ID)
			},
			nats.Durable(s.queueGroup),
			nats.ManualAck(),
			nats.AckWait(30*time.Second),
			nats.MaxAckPending(100),
		)
		if err != nil {
			return err
		}
		s.sub = sub
		return nil
	}

	// Core NATS subscription.
	sub, err := s.conn.QueueSubscribe(s.subject, s.queueGroup, func(msg *nats.Msg) {
		var ev event.Event
		if err := json.Unmarshal(msg.Data, &ev); err != nil {
			slog.Error("failed to unmarshal event", "error", err)
			return
		}
		if err := handler(&ev); err != nil {
			slog.Error("event handler failed", "error", err, "event_id", ev.ID)
			return
		}
		slog.Info("event processed", "event_id", ev.ID)
	})
	if err != nil {
		return err
	}
	s.sub = sub
	return nil
}

// Close unsubscribes from the subject.
func (s *NATSSubscriber) Close() error {
	if s.sub != nil {
		return s.sub.Unsubscribe()
	}
	return nil
}
