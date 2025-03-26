package event

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/neo-vai/go-events/internal/model/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockEventService is a mock for EventService.
type MockEventService struct {
	mock.Mock
}

func (m *MockEventService) CreateEvent(ctx context.Context, ev *event.Event) error {
	args := m.Called(ctx, ev)
	return args.Error(0)
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

func setupEventRouter(service EventService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	h := NewHandler(service)
	r.POST("/api/v1/events", h.CreateEvent)
	r.GET("/api/v1/events", h.ListEvents)
	r.GET("/api/v1/events/:id", h.GetEvent)
	return r
}

// ---------- CreateEvent ----------
func TestCreateEvent_Success(t *testing.T) {
	svc := new(MockEventService)
	router := setupEventRouter(svc)

	reqBody := CreateEventRequest{
		AccountID: uuid.New().String(),
		Username:  "john",
		APIKeyID:  uuid.New().String(),
		Name:      "user.login",
		Payload:   `{"ip":"1.2.3.4"}`,
	}
	jsonBody, _ := json.Marshal(reqBody)

	svc.On("CreateEvent", mock.Anything, mock.MatchedBy(func(ev *event.Event) bool {
		return ev.AccountID == reqBody.AccountID && ev.Username == reqBody.Username && ev.Name == reqBody.Name
	})).Return(nil).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/events", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp EventResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, reqBody.AccountID, resp.AccountID)
	assert.Equal(t, reqBody.Username, resp.Username)
	assert.Equal(t, reqBody.Name, resp.Name)
	// ID не проверяем, т.к. мок его не выставляет
	svc.AssertExpectations(t)
}

func TestCreateEvent_InvalidJSON(t *testing.T) {
	svc := new(MockEventService)
	router := setupEventRouter(svc)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/events", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	svc.AssertNotCalled(t, "CreateEvent", mock.Anything, mock.Anything)
}

func TestCreateEvent_MissingRequiredFields(t *testing.T) {
	svc := new(MockEventService)
	router := setupEventRouter(svc)

	reqBody := CreateEventRequest{
		AccountID: uuid.New().String(),
		// missing Username
		Name: "event",
	}
	jsonBody, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/events", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	svc.AssertNotCalled(t, "CreateEvent", mock.Anything, mock.Anything)
}

func TestCreateEvent_ServiceError(t *testing.T) {
	svc := new(MockEventService)
	router := setupEventRouter(svc)

	reqBody := CreateEventRequest{
		AccountID: uuid.New().String(),
		Username:  "john",
		Name:      "event",
	}
	jsonBody, _ := json.Marshal(reqBody)

	svc.On("CreateEvent", mock.Anything, mock.Anything).Return(errors.New("db error")).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/events", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	svc.AssertExpectations(t)
}

// ---------- ListEvents ----------
func TestListEvents_Success(t *testing.T) {
	svc := new(MockEventService)
	router := setupEventRouter(svc)

	events := []*event.Event{
		{ID: "e1", AccountID: "acc1", Username: "john", Name: "login"},
		{ID: "e2", AccountID: "acc1", Username: "john", Name: "logout"},
	}

	svc.On("ListEvents", mock.Anything, "acc1", "john", "").Return(events, nil).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/events?account_id=acc1&user=john", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp []EventResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Len(t, resp, 2)
	assert.Equal(t, "e1", resp[0].ID)
	svc.AssertExpectations(t)
}

func TestListEvents_Empty(t *testing.T) {
	svc := new(MockEventService)
	router := setupEventRouter(svc)

	svc.On("ListEvents", mock.Anything, "", "", "").Return([]*event.Event{}, nil).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/events", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp []EventResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Empty(t, resp)
	svc.AssertExpectations(t)
}

func TestListEvents_ServiceError(t *testing.T) {
	svc := new(MockEventService)
	router := setupEventRouter(svc)

	svc.On("ListEvents", mock.Anything, "", "", "").Return(nil, errors.New("db error")).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/events", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	svc.AssertExpectations(t)
}

// ---------- GetEvent ----------
func TestGetEvent_Success(t *testing.T) {
	svc := new(MockEventService)
	router := setupEventRouter(svc)

	eventID := uuid.New().String()
	ev := &event.Event{
		ID:        eventID,
		AccountID: uuid.New().String(),
		Username:  "john",
		Name:      "login",
	}

	svc.On("GetByID", mock.Anything, eventID).Return(ev, nil).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/events/"+eventID, nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp EventResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, eventID, resp.ID)
	assert.Equal(t, ev.Name, resp.Name)
	svc.AssertExpectations(t)
}

func TestGetEvent_NotFound(t *testing.T) {
	svc := new(MockEventService)
	router := setupEventRouter(svc)

	eventID := uuid.New().String()
	svc.On("GetByID", mock.Anything, eventID).Return(nil, errors.New("not found")).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/events/"+eventID, nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	svc.AssertExpectations(t)
}
