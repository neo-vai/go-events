package event

import (
	"context"
	"errors"
	"testing"

	"github.com/neo-vai/go-events/internal/model/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockEventRepository is a mock for EventRepository.
type MockEventRepository struct {
	mock.Mock
}

func (m *MockEventRepository) ListAll(ctx context.Context, page, limit int, sort, order string, filters map[string]interface{}) ([]*event.Event, int64, error) {
	args := m.Called(ctx, page, limit, sort, order, filters)
	return args.Get(0).([]*event.Event), args.Get(1).(int64), args.Error(2)
}

func (m *MockEventRepository) Create(ctx context.Context, ev *event.Event) error {
	args := m.Called(ctx, ev)
	return args.Error(0)
}

func (m *MockEventRepository) GetByID(ctx context.Context, id string) (*event.Event, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*event.Event), args.Error(1)
}

func (m *MockEventRepository) GetByAccountID(ctx context.Context, accountID string) ([]*event.Event, error) {
	args := m.Called(ctx, accountID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*event.Event), args.Error(1)
}

func (m *MockEventRepository) GetByAccountAndUser(ctx context.Context, accountID, username string) ([]*event.Event, error) {
	args := m.Called(ctx, accountID, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*event.Event), args.Error(1)
}

func (m *MockEventRepository) GetByAPIKeyID(ctx context.Context, apiKeyID string) ([]*event.Event, error) {
	args := m.Called(ctx, apiKeyID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*event.Event), args.Error(1)
}

func TestEventService_CreateEvent(t *testing.T) {
	repo := new(MockEventRepository)
	svc := NewEventService(repo)
	ctx := context.Background()

	ev := &event.Event{
		AccountID: "acc-123",
		Username:  "john",
		Name:      "login",
		Payload:   `{"ip":"1.2.3.4"}`,
	}

	repo.On("Create", ctx, mock.MatchedBy(func(e *event.Event) bool {
		return e.ID != "" && e.CreatedAt.Unix() > 0
	})).Return(nil).Once()

	err := svc.CreateEvent(ctx, ev)
	assert.NoError(t, err)
	assert.NotEmpty(t, ev.ID)
	assert.NotZero(t, ev.CreatedAt)
	repo.AssertExpectations(t)
}

func TestEventService_CreateEvent_RepoError(t *testing.T) {
	repo := new(MockEventRepository)
	svc := NewEventService(repo)
	ctx := context.Background()

	ev := &event.Event{}
	repo.On("Create", ctx, mock.Anything).Return(errors.New("db error")).Once()

	err := svc.CreateEvent(ctx, ev)
	assert.Error(t, err)
	repo.AssertExpectations(t)
}

func TestEventService_GetByID(t *testing.T) {
	repo := new(MockEventRepository)
	svc := NewEventService(repo)
	ctx := context.Background()

	expected := &event.Event{ID: "ev-1", Name: "test"}
	repo.On("GetByID", ctx, "ev-1").Return(expected, nil).Once()

	ev, err := svc.GetByID(ctx, "ev-1")
	assert.NoError(t, err)
	assert.Equal(t, expected, ev)
	repo.AssertExpectations(t)
}

func TestEventService_GetByID_NotFound(t *testing.T) {
	repo := new(MockEventRepository)
	svc := NewEventService(repo)
	ctx := context.Background()

	repo.On("GetByID", ctx, "missing").Return(nil, errors.New("not found")).Once()

	_, err := svc.GetByID(ctx, "missing")
	assert.Error(t, err)
	repo.AssertExpectations(t)
}

func TestEventService_ListEvents(t *testing.T) {
	repo := new(MockEventRepository)
	svc := NewEventService(repo)
	ctx := context.Background()

	t.Run("only accountID", func(t *testing.T) {
		expected := []*event.Event{{ID: "e1"}, {ID: "e2"}}
		repo.On("GetByAccountID", ctx, "acc-1").Return(expected, nil).Once()

		events, err := svc.ListEvents(ctx, "acc-1", "", "")
		assert.NoError(t, err)
		assert.Equal(t, expected, events)
	})

	t.Run("only username", func(t *testing.T) {
		// ListEvents returns empty slice when only username is provided
		events, err := svc.ListEvents(ctx, "", "bob", "")
		assert.NoError(t, err)
		assert.Empty(t, events)
	})

	t.Run("only apiKeyID", func(t *testing.T) {
		expected := []*event.Event{{ID: "e3"}}
		repo.On("GetByAPIKeyID", ctx, "key-1").Return(expected, nil).Once()

		events, err := svc.ListEvents(ctx, "", "", "key-1")
		assert.NoError(t, err)
		assert.Equal(t, expected, events)
	})

	t.Run("accountID and username", func(t *testing.T) {
		expected := []*event.Event{{ID: "e4"}}
		repo.On("GetByAccountAndUser", ctx, "acc-1", "bob").Return(expected, nil).Once()

		events, err := svc.ListEvents(ctx, "acc-1", "bob", "")
		assert.NoError(t, err)
		assert.Equal(t, expected, events)
	})

	t.Run("accountID and apiKeyID", func(t *testing.T) {
		byAccount := []*event.Event{
			{ID: "e1", APIKeyID: "key-1"},
			{ID: "e2", APIKeyID: "key-2"},
		}
		repo.On("GetByAccountID", ctx, "acc-1").Return(byAccount, nil).Once()

		events, err := svc.ListEvents(ctx, "acc-1", "", "key-1")
		assert.NoError(t, err)
		assert.Len(t, events, 1)
		assert.Equal(t, "e1", events[0].ID)
	})

	t.Run("username and apiKeyID", func(t *testing.T) {
		byKey := []*event.Event{
			{ID: "e1", Username: "bob", APIKeyID: "key-1"},
			{ID: "e2", Username: "alice", APIKeyID: "key-1"},
		}
		repo.On("GetByAPIKeyID", ctx, "key-1").Return(byKey, nil).Once()

		events, err := svc.ListEvents(ctx, "", "bob", "key-1")
		assert.NoError(t, err)
		assert.Len(t, events, 1)
		assert.Equal(t, "e1", events[0].ID)
	})

	t.Run("all three filters", func(t *testing.T) {
		byUser := []*event.Event{
			{ID: "e1", Username: "bob", APIKeyID: "key-1"},
			{ID: "e2", Username: "bob", APIKeyID: "key-2"},
		}
		repo.On("GetByAccountAndUser", ctx, "acc-1", "bob").Return(byUser, nil).Once()

		events, err := svc.ListEvents(ctx, "acc-1", "bob", "key-1")
		assert.NoError(t, err)
		assert.Len(t, events, 1)
		assert.Equal(t, "e1", events[0].ID)
	})

	t.Run("no filters", func(t *testing.T) {
		events, err := svc.ListEvents(ctx, "", "", "")
		assert.NoError(t, err)
		assert.Empty(t, events)
	})

	t.Run("repo error", func(t *testing.T) {
		repo.On("GetByAccountID", ctx, "acc-err").Return(nil, errors.New("db error")).Once()
		_, err := svc.ListEvents(ctx, "acc-err", "", "")
		assert.Error(t, err)
	})
}
