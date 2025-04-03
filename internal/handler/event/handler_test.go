package event

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/neo-vai/go-events/internal/middleware"
	"github.com/neo-vai/go-events/internal/model/event"
	eventService "github.com/neo-vai/go-events/internal/service/event"
	"github.com/neo-vai/go-events/internal/validator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockEventService struct {
	mock.Mock
}

func (m *MockEventService) GetByID(ctx context.Context, id string) (*event.Event, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*event.Event), args.Error(1)
}

func (m *MockEventService) ListEvents(ctx context.Context, accountID, username, apiKeyID string) ([]*event.Event, error) {
	args := m.Called(ctx, accountID, username, apiKeyID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*event.Event), args.Error(1)
}

func (m *MockEventService) ListEventsPaginated(ctx context.Context, accountID string, offset, limit int, sort, order, username, apiKeyID, searchQuery string) ([]*event.Event, int64, error) {
	args := m.Called(ctx, accountID, offset, limit, sort, order, username, apiKeyID, searchQuery)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*event.Event), args.Get(1).(int64), args.Error(2)
}

type MockPublisher struct {
	mock.Mock
}

func (m *MockPublisher) PublishEvent(ctx context.Context, ev *event.Event) error {
	args := m.Called(ctx, ev)
	return args.Error(0)
}

func setupEventHandlerTest() (*gin.Engine, *MockEventService, *MockPublisher, string) {
	gin.SetMode(gin.TestMode)
	validator.RegisterCustomValidators()
	svc := new(MockEventService)
	publisher := new(MockPublisher)
	h := NewHandler(svc, publisher)
	accountID := uuid.New().String()

	r := gin.New()
	r.Use(middleware.ValidationErrorHandler())
	protected := r.Group("/api/v1")
	protected.Use(func(c *gin.Context) {
		c.Set(string(middleware.AccountIDKey), accountID)
		c.Next()
	})
	{
		protected.POST("/events", h.CreateEvent)
		protected.GET("/events", h.ListEvents)
		protected.GET("/events/:id", h.GetEvent)
	}
	return r, svc, publisher, accountID
}

func TestHandler_CreateEvent_Success(t *testing.T) {
	r, svc, publisher, accountID := setupEventHandlerTest()

	reqBody := CreateEventRequest{
		Username: "testuser",
		Name:     "user_login",
		Payload:  `{"ip":"192.168.1.1"}`,
	}
	jsonBody, _ := json.Marshal(reqBody)

	publisher.On("PublishEvent", mock.Anything, mock.MatchedBy(func(ev *event.Event) bool {
		return ev.AccountID == accountID &&
			ev.Username == "testuser" &&
			ev.Name == "user_login" &&
			ev.Payload == `{"ip":"192.168.1.1"}`
	})).Return(nil).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/events", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)

	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.NotEmpty(t, resp["id"])
	assert.Equal(t, "queued", resp["status"])
	assert.NotEmpty(t, w.Header().Get("Location"))

	svc.AssertNotCalled(t, "CreateEvent")
	publisher.AssertExpectations(t)
}

func TestHandler_CreateEvent_WithAPIKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	validator.RegisterCustomValidators()
	svc := new(MockEventService)
	publisher := new(MockPublisher)
	h := NewHandler(svc, publisher)
	accountID := uuid.New().String()
	apiKeyID := uuid.New().String()

	r := gin.New()
	r.Use(middleware.ValidationErrorHandler())
	r.Use(func(c *gin.Context) {
		c.Set(string(middleware.AccountIDKey), accountID)
		c.Set(string(middleware.APIKeyIDKey), apiKeyID)
		c.Next()
	})
	r.POST("/api/v1/events", h.CreateEvent)

	reqBody := CreateEventRequest{
		Username: "testuser",
		Name:     "api_call",
		Payload:  `{}`,
	}
	jsonBody, _ := json.Marshal(reqBody)

	publisher.On("PublishEvent", mock.Anything, mock.MatchedBy(func(ev *event.Event) bool {
		return ev.APIKeyID == apiKeyID
	})).Return(nil).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/events", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	publisher.AssertExpectations(t)
}

func TestHandler_CreateEvent_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	validator.RegisterCustomValidators()
	svc := new(MockEventService)
	publisher := new(MockPublisher)
	h := NewHandler(svc, publisher)

	r := gin.New()
	r.Use(middleware.ValidationErrorHandler())
	r.POST("/api/v1/events", h.CreateEvent)

	reqBody := CreateEventRequest{Username: "test", Name: "test", Payload: `{}`}
	jsonBody, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/events", bytes.NewReader(jsonBody))
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	publisher.AssertNotCalled(t, "PublishEvent")
}

func TestHandler_CreateEvent_InvalidJSON(t *testing.T) {
	r, _, publisher, _ := setupEventHandlerTest()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/events", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	publisher.AssertNotCalled(t, "PublishEvent")
}

func TestHandler_CreateEvent_MissingRequiredFields(t *testing.T) {
	r, _, publisher, _ := setupEventHandlerTest()

	reqBody := map[string]string{"name": "test"}
	jsonBody, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/events", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	publisher.AssertNotCalled(t, "PublishEvent")
}

func TestHandler_CreateEvent_PublisherError(t *testing.T) {
	r, _, publisher, _ := setupEventHandlerTest()

	reqBody := CreateEventRequest{
		Username: "testuser",
		Name:     "test_event",
		Payload:  `{}`,
	}
	jsonBody, _ := json.Marshal(reqBody)

	publisher.On("PublishEvent", mock.Anything, mock.Anything).Return(errors.New("broker unavailable")).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/events", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	publisher.AssertExpectations(t)
}

func TestHandler_ListEvents_Success(t *testing.T) {
	r, svc, _, accountID := setupEventHandlerTest()

	events := []*event.Event{
		{ID: "ev1", AccountID: accountID, Username: "user1", Name: "login", Payload: `{}`, CreatedAt: time.Now()},
		{ID: "ev2", AccountID: accountID, Username: "user2", Name: "logout", Payload: `{}`, CreatedAt: time.Now()},
	}

	svc.On("ListEventsPaginated", mock.Anything, accountID, 0, 20, "id", "ASC", "", "", "").
		Return(events, int64(2), nil).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/events", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "2", w.Header().Get("X-Total-Count"))

	var resp []EventResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Len(t, resp, 2)

	svc.AssertExpectations(t)
}

func TestHandler_ListEvents_WithPagination(t *testing.T) {
	r, svc, _, accountID := setupEventHandlerTest()

	events := []*event.Event{
		{ID: "ev3", AccountID: accountID, Username: "user3", Name: "event3", Payload: `{}`, CreatedAt: time.Now()},
	}

	svc.On("ListEventsPaginated", mock.Anything, accountID, 10, 10, "created_at", "DESC", "bob", "key123", "search").
		Return(events, int64(1), nil).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/events?_start=10&_end=20&_sort=created_at&_order=DESC&filter={\"username\":\"bob\",\"api_key_id\":\"key123\",\"q\":\"search\"}", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	svc.AssertExpectations(t)
}

func TestHandler_ListEvents_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	validator.RegisterCustomValidators()
	svc := new(MockEventService)
	publisher := new(MockPublisher)
	h := NewHandler(svc, publisher)

	r := gin.New()
	r.Use(middleware.ValidationErrorHandler())
	r.GET("/api/v1/events", h.ListEvents)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/events", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	svc.AssertNotCalled(t, "ListEventsPaginated")
}

func TestHandler_ListEvents_ServiceError(t *testing.T) {
	r, svc, _, accountID := setupEventHandlerTest()

	svc.On("ListEventsPaginated", mock.Anything, accountID, 0, 20, "id", "ASC", "", "", "").
		Return(nil, int64(0), errors.New("database error")).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/events", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	svc.AssertExpectations(t)
}

func TestHandler_GetEvent_Success(t *testing.T) {
	r, svc, _, accountID := setupEventHandlerTest()
	eventID := uuid.New().String()

	ev := &event.Event{
		ID:        eventID,
		AccountID: accountID,
		Username:  "testuser",
		Name:      "user_login",
		Payload:   `{"ip":"192.168.1.1"}`,
		CreatedAt: time.Now(),
	}

	svc.On("GetByID", mock.Anything, eventID).Return(ev, nil).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/events/"+eventID, nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp EventResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, eventID, resp.ID)
	assert.Equal(t, "testuser", resp.Username)

	svc.AssertExpectations(t)
}

func TestHandler_GetEvent_NotFound(t *testing.T) {
	r, svc, _, _ := setupEventHandlerTest()
	eventID := uuid.New().String()

	svc.On("GetByID", mock.Anything, eventID).Return(nil, eventService.ErrEventNotFound).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/events/"+eventID, nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	svc.AssertExpectations(t)
}

func TestHandler_GetEvent_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	validator.RegisterCustomValidators()
	svc := new(MockEventService)
	publisher := new(MockPublisher)
	h := NewHandler(svc, publisher)

	r := gin.New()
	r.Use(middleware.ValidationErrorHandler())
	r.GET("/api/v1/events/:id", h.GetEvent)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/events/"+uuid.New().String(), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	svc.AssertNotCalled(t, "GetByID")
}

func TestHandler_GetEvent_ForbiddenNotOwner(t *testing.T) {
	r, svc, _, accountID := setupEventHandlerTest()
	eventID := uuid.New().String()
	otherAccountID := uuid.New().String()

	ev := &event.Event{
		ID:        eventID,
		AccountID: otherAccountID,
		Username:  "testuser",
		Name:      "user_login",
		Payload:   `{}`,
		CreatedAt: time.Now(),
	}

	svc.On("GetByID", mock.Anything, eventID).Return(ev, nil).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/events/"+eventID, nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Equal(t, accountID, accountID)
	svc.AssertExpectations(t)
}
