// Package handlers provides tests for HTTP handlers.
// These tests use mock interfaces for clean dependency injection.
package handlers

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"rule-service/internal/database"
)

// TestHandlers_CreateClient tests the CreateClient handler.
func TestHandlers_CreateClient(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		body           string
		setupMock      func(*mockRepository)
		expectedStatus int
	}{
		{
			name:   "successful create",
			method: http.MethodPost,
			body:   `{"client_id":"client-1","name":"Test Client"}`,
			setupMock: func(m *mockRepository) {
				m.CreateClientFn = func(ctx context.Context, clientID, name string) error {
					return nil
				}
				m.GetClientFn = func(ctx context.Context, clientID string) (*database.Client, error) {
					return &database.Client{ClientID: clientID, Name: "Test Client", CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
				}
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "wrong method",
			method:         http.MethodGet,
			body:           `{"client_id":"client-1","name":"Test Client"}`,
			setupMock:      func(m *mockRepository) {},
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "invalid JSON",
			method:         http.MethodPost,
			body:           `invalid json`,
			setupMock:      func(m *mockRepository) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing client_id",
			method:         http.MethodPost,
			body:           `{"name":"Test Client"}`,
			setupMock:      func(m *mockRepository) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing name",
			method:         http.MethodPost,
			body:           `{"client_id":"client-1"}`,
			setupMock:      func(m *mockRepository) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "duplicate client",
			method: http.MethodPost,
			body:   `{"client_id":"client-1","name":"Test Client"}`,
			setupMock: func(m *mockRepository) {
				m.CreateClientFn = func(ctx context.Context, clientID, name string) error {
					return fmt.Errorf("client already exists: %s", clientID)
				}
			},
			expectedStatus: http.StatusConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &mockRepository{}
			tt.setupMock(mockDB)

			h := NewHandlersWithDeps(mockDB, &mockPublisher{}, nil)
			req := httptest.NewRequest(tt.method, "/api/v1/clients", bytes.NewBufferString(tt.body))
			w := httptest.NewRecorder()

			h.CreateClient(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("CreateClient() status = %v, want %v, body = %s", w.Code, tt.expectedStatus, w.Body.String())
			}
		})
	}
}

// TestHandlers_GetClient tests the GetClient handler.
func TestHandlers_GetClient(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		query          string
		setupMock      func(*mockRepository)
		expectedStatus int
	}{
		{
			name:   "successful get",
			method: http.MethodGet,
			query:  "?client_id=client-1",
			setupMock: func(m *mockRepository) {
				m.GetClientFn = func(ctx context.Context, clientID string) (*database.Client, error) {
					return &database.Client{ClientID: clientID, Name: "Test Client", CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "wrong method",
			method:         http.MethodPost,
			query:          "?client_id=client-1",
			setupMock:      func(m *mockRepository) {},
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "missing client_id",
			method:         http.MethodGet,
			query:          "",
			setupMock:      func(m *mockRepository) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "client not found",
			method: http.MethodGet,
			query:  "?client_id=client-999",
			setupMock: func(m *mockRepository) {
				m.GetClientFn = func(ctx context.Context, clientID string) (*database.Client, error) {
					return nil, fmt.Errorf("client not found: %s", clientID)
				}
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &mockRepository{}
			tt.setupMock(mockDB)

			h := NewHandlersWithDeps(mockDB, &mockPublisher{}, nil)
			req := httptest.NewRequest(tt.method, "/api/v1/clients"+tt.query, nil)
			w := httptest.NewRecorder()

			h.GetClient(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("GetClient() status = %v, want %v", w.Code, tt.expectedStatus)
			}
		})
	}
}

// TestHandlers_ListClients tests the ListClients handler.
func TestHandlers_ListClients(t *testing.T) {
	t.Run("successful list", func(t *testing.T) {
		mockDB := &mockRepository{}
		mockDB.ListClientsFn = func(ctx context.Context, limit, offset int) (*database.ClientListResult, error) {
			return &database.ClientListResult{
				Clients: []*database.Client{
					{ClientID: "client-1", Name: "Client 1"},
					{ClientID: "client-2", Name: "Client 2"},
				},
				Total:  2,
				Limit:  limit,
				Offset: offset,
			}, nil
		}

		h := NewHandlersWithDeps(mockDB, &mockPublisher{}, nil)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/clients", nil)
		w := httptest.NewRecorder()

		h.ListClients(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("ListClients() status = %v, want %v", w.Code, http.StatusOK)
		}
	})

	t.Run("wrong method", func(t *testing.T) {
		h := NewHandlersWithDeps(&mockRepository{}, &mockPublisher{}, nil)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/clients", nil)
		w := httptest.NewRecorder()

		h.ListClients(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("ListClients() status = %v, want %v", w.Code, http.StatusMethodNotAllowed)
		}
	})
}

// TestHandlers_CreateRule tests the CreateRule handler.
func TestHandlers_CreateRule(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		body           string
		setupMock      func(*mockRepository)
		expectedStatus int
	}{
		{
			name:   "successful create",
			method: http.MethodPost,
			body:   `{"client_id":"client-1","severity":"HIGH","source":"source-1","name":"alert-1"}`,
			setupMock: func(m *mockRepository) {
				m.CreateRuleFn = func(ctx context.Context, clientID, severity, source, name string) (*database.Rule, error) {
					return &database.Rule{
						RuleID: "rule-1", ClientID: clientID, Severity: severity, Source: source, Name: name,
						Enabled: true, Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
					}, nil
				}
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "wrong method",
			method:         http.MethodGet,
			body:           `{"client_id":"client-1","severity":"HIGH","source":"source-1","name":"alert-1"}`,
			setupMock:      func(m *mockRepository) {},
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "invalid JSON",
			method:         http.MethodPost,
			body:           `invalid json`,
			setupMock:      func(m *mockRepository) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing client_id",
			method:         http.MethodPost,
			body:           `{"severity":"HIGH","source":"source-1","name":"alert-1"}`,
			setupMock:      func(m *mockRepository) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing severity",
			method:         http.MethodPost,
			body:           `{"client_id":"client-1","source":"source-1","name":"alert-1"}`,
			setupMock:      func(m *mockRepository) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid severity",
			method:         http.MethodPost,
			body:           `{"client_id":"client-1","severity":"INVALID","source":"source-1","name":"alert-1"}`,
			setupMock:      func(m *mockRepository) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "all wildcards",
			method:         http.MethodPost,
			body:           `{"client_id":"client-1","severity":"*","source":"*","name":"*"}`,
			setupMock:      func(m *mockRepository) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "client not found",
			method: http.MethodPost,
			body:   `{"client_id":"client-999","severity":"HIGH","source":"source-1","name":"alert-1"}`,
			setupMock: func(m *mockRepository) {
				m.CreateRuleFn = func(ctx context.Context, clientID, severity, source, name string) (*database.Rule, error) {
					return nil, fmt.Errorf("client not found: %s", clientID)
				}
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &mockRepository{}
			tt.setupMock(mockDB)

			h := NewHandlersWithDeps(mockDB, &mockPublisher{}, nil)
			req := httptest.NewRequest(tt.method, "/api/v1/rules", bytes.NewBufferString(tt.body))
			w := httptest.NewRecorder()

			h.CreateRule(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("CreateRule() status = %v, want %v, body = %s", w.Code, tt.expectedStatus, w.Body.String())
			}
		})
	}
}

// TestHandlers_GetRule tests the GetRule handler.
func TestHandlers_GetRule(t *testing.T) {
	t.Run("successful get", func(t *testing.T) {
		mockDB := &mockRepository{}
		mockDB.GetRuleFn = func(ctx context.Context, ruleID string) (*database.Rule, error) {
			return &database.Rule{RuleID: ruleID, ClientID: "client-1", Severity: "HIGH", Enabled: true, Version: 1}, nil
		}

		h := NewHandlersWithDeps(mockDB, &mockPublisher{}, nil)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/rules?rule_id=rule-1", nil)
		w := httptest.NewRecorder()

		h.GetRule(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("GetRule() status = %v, want %v", w.Code, http.StatusOK)
		}
	})
}

// TestHandlers_ListRules tests the ListRules handler.
func TestHandlers_ListRules(t *testing.T) {
	t.Run("list all", func(t *testing.T) {
		mockDB := &mockRepository{}
		mockDB.ListRulesFn = func(ctx context.Context, clientID *string, limit, offset int) (*database.RuleListResult, error) {
			return &database.RuleListResult{Rules: []*database.Rule{{RuleID: "rule-1"}}, Total: 1, Limit: limit, Offset: offset}, nil
		}

		h := NewHandlersWithDeps(mockDB, &mockPublisher{}, nil)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/rules", nil)
		w := httptest.NewRecorder()

		h.ListRules(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("ListRules() status = %v, want %v", w.Code, http.StatusOK)
		}
	})

	t.Run("list by client", func(t *testing.T) {
		mockDB := &mockRepository{}
		mockDB.ListRulesFn = func(ctx context.Context, clientID *string, limit, offset int) (*database.RuleListResult, error) {
			if clientID == nil || *clientID != "client-1" {
				t.Error("Expected client_id filter")
			}
			return &database.RuleListResult{Rules: []*database.Rule{{RuleID: "rule-1", ClientID: *clientID}}, Total: 1, Limit: limit, Offset: offset}, nil
		}

		h := NewHandlersWithDeps(mockDB, &mockPublisher{}, nil)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/rules?client_id=client-1", nil)
		w := httptest.NewRecorder()

		h.ListRules(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("ListRules() status = %v, want %v", w.Code, http.StatusOK)
		}
	})
}

// TestHandlers_UpdateRule tests the UpdateRule handler.
func TestHandlers_UpdateRule(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		query          string
		body           string
		setupMock      func(*mockRepository)
		expectedStatus int
	}{
		{
			name:   "successful update",
			method: http.MethodPut,
			query:  "?rule_id=rule-1",
			body:   `{"severity":"CRITICAL","source":"source-2","name":"alert-2","version":1}`,
			setupMock: func(m *mockRepository) {
				m.UpdateRuleFn = func(ctx context.Context, ruleID string, severity, source, name string, expectedVersion int) (*database.Rule, error) {
					return &database.Rule{RuleID: ruleID, Severity: severity, Source: source, Name: name, Version: 2, UpdatedAt: time.Now()}, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "wrong method",
			method:         http.MethodPost,
			query:          "?rule_id=rule-1",
			body:           `{"severity":"CRITICAL","source":"source-2","name":"alert-2","version":1}`,
			setupMock:      func(m *mockRepository) {},
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "missing rule_id",
			method:         http.MethodPut,
			query:          "",
			body:           `{"severity":"CRITICAL","source":"source-2","name":"alert-2","version":1}`,
			setupMock:      func(m *mockRepository) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "version mismatch",
			method: http.MethodPut,
			query:  "?rule_id=rule-1",
			body:   `{"severity":"CRITICAL","source":"source-2","name":"alert-2","version":1}`,
			setupMock: func(m *mockRepository) {
				m.UpdateRuleFn = func(ctx context.Context, ruleID string, severity, source, name string, expectedVersion int) (*database.Rule, error) {
					return nil, fmt.Errorf("rule version mismatch: expected version %d", expectedVersion)
				}
			},
			expectedStatus: http.StatusConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &mockRepository{}
			tt.setupMock(mockDB)

			h := NewHandlersWithDeps(mockDB, &mockPublisher{}, nil)
			req := httptest.NewRequest(tt.method, "/api/v1/rules/update"+tt.query, bytes.NewBufferString(tt.body))
			w := httptest.NewRecorder()

			h.UpdateRule(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("UpdateRule() status = %v, want %v, body = %s", w.Code, tt.expectedStatus, w.Body.String())
			}
		})
	}
}

// TestHandlers_ToggleRuleEnabled tests the ToggleRuleEnabled handler.
func TestHandlers_ToggleRuleEnabled(t *testing.T) {
	t.Run("successful toggle", func(t *testing.T) {
		mockDB := &mockRepository{}
		mockDB.ToggleRuleEnabledFn = func(ctx context.Context, ruleID string, enabled bool, expectedVersion int) (*database.Rule, error) {
			return &database.Rule{RuleID: ruleID, Enabled: enabled, Version: 2, UpdatedAt: time.Now()}, nil
		}

		h := NewHandlersWithDeps(mockDB, &mockPublisher{}, nil)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/rules/toggle?rule_id=rule-1", bytes.NewBufferString(`{"enabled":false,"version":1}`))
		w := httptest.NewRecorder()

		h.ToggleRuleEnabled(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("ToggleRuleEnabled() status = %v, want %v", w.Code, http.StatusOK)
		}
	})
}

// TestHandlers_DeleteRule tests the DeleteRule handler.
func TestHandlers_DeleteRule(t *testing.T) {
	t.Run("successful delete", func(t *testing.T) {
		mockDB := &mockRepository{}
		mockDB.GetRuleFn = func(ctx context.Context, ruleID string) (*database.Rule, error) {
			return &database.Rule{RuleID: ruleID, ClientID: "client-1", Version: 1}, nil
		}
		mockDB.DeleteRuleFn = func(ctx context.Context, ruleID string) error {
			return nil
		}

		h := NewHandlersWithDeps(mockDB, &mockPublisher{}, nil)
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/rules/delete?rule_id=rule-1", nil)
		w := httptest.NewRecorder()

		h.DeleteRule(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("DeleteRule() status = %v, want %v", w.Code, http.StatusNoContent)
		}
	})
}

// TestHandlers_CreateEndpoint tests the CreateEndpoint handler.
func TestHandlers_CreateEndpoint(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		body           string
		setupMock      func(*mockRepository)
		expectedStatus int
	}{
		{
			name:   "successful create",
			method: http.MethodPost,
			body:   `{"rule_id":"rule-1","type":"email","value":"test@example.com"}`,
			setupMock: func(m *mockRepository) {
				m.CreateEndpointFn = func(ctx context.Context, ruleID, endpointType, value string) (*database.Endpoint, error) {
					return &database.Endpoint{EndpointID: "endpoint-1", RuleID: ruleID, Type: endpointType, Value: value, Enabled: true}, nil
				}
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "invalid type",
			method:         http.MethodPost,
			body:           `{"rule_id":"rule-1","type":"invalid","value":"test@example.com"}`,
			setupMock:      func(m *mockRepository) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "rule not found",
			method: http.MethodPost,
			body:   `{"rule_id":"rule-999","type":"email","value":"test@example.com"}`,
			setupMock: func(m *mockRepository) {
				m.CreateEndpointFn = func(ctx context.Context, ruleID, endpointType, value string) (*database.Endpoint, error) {
					return nil, fmt.Errorf("rule not found: %s", ruleID)
				}
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &mockRepository{}
			tt.setupMock(mockDB)

			h := NewHandlersWithDeps(mockDB, &mockPublisher{}, nil)
			req := httptest.NewRequest(tt.method, "/api/v1/endpoints", bytes.NewBufferString(tt.body))
			w := httptest.NewRecorder()

			h.CreateEndpoint(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("CreateEndpoint() status = %v, want %v", w.Code, tt.expectedStatus)
			}
		})
	}
}

// TestHandlers_GetEndpoint tests the GetEndpoint handler.
func TestHandlers_GetEndpoint(t *testing.T) {
	t.Run("successful get", func(t *testing.T) {
		mockDB := &mockRepository{}
		mockDB.GetEndpointFn = func(ctx context.Context, endpointID string) (*database.Endpoint, error) {
			return &database.Endpoint{EndpointID: endpointID, RuleID: "rule-1", Type: "email", Value: "test@example.com"}, nil
		}

		h := NewHandlersWithDeps(mockDB, &mockPublisher{}, nil)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/endpoints?endpoint_id=endpoint-1", nil)
		w := httptest.NewRecorder()

		h.GetEndpoint(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("GetEndpoint() status = %v, want %v", w.Code, http.StatusOK)
		}
	})
}

// TestHandlers_ListEndpoints tests the ListEndpoints handler.
func TestHandlers_ListEndpoints(t *testing.T) {
	t.Run("successful list", func(t *testing.T) {
		mockDB := &mockRepository{}
		mockDB.ListEndpointsFn = func(ctx context.Context, ruleID *string, limit, offset int) (*database.EndpointListResult, error) {
			return &database.EndpointListResult{Endpoints: []*database.Endpoint{{EndpointID: "endpoint-1"}}, Total: 1, Limit: limit, Offset: offset}, nil
		}

		h := NewHandlersWithDeps(mockDB, &mockPublisher{}, nil)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/endpoints?rule_id=rule-1", nil)
		w := httptest.NewRecorder()

		h.ListEndpoints(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("ListEndpoints() status = %v, want %v", w.Code, http.StatusOK)
		}
	})
}

// TestHandlers_UpdateEndpoint tests the UpdateEndpoint handler.
func TestHandlers_UpdateEndpoint(t *testing.T) {
	t.Run("successful update", func(t *testing.T) {
		mockDB := &mockRepository{}
		mockDB.UpdateEndpointFn = func(ctx context.Context, endpointID, endpointType, value string) (*database.Endpoint, error) {
			return &database.Endpoint{EndpointID: endpointID, Type: endpointType, Value: value}, nil
		}

		h := NewHandlersWithDeps(mockDB, &mockPublisher{}, nil)
		req := httptest.NewRequest(http.MethodPut, "/api/v1/endpoints/update?endpoint_id=endpoint-1", bytes.NewBufferString(`{"type":"webhook","value":"https://example.com"}`))
		w := httptest.NewRecorder()

		h.UpdateEndpoint(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("UpdateEndpoint() status = %v, want %v", w.Code, http.StatusOK)
		}
	})
}

// TestHandlers_ToggleEndpointEnabled tests the ToggleEndpointEnabled handler.
func TestHandlers_ToggleEndpointEnabled(t *testing.T) {
	t.Run("successful toggle", func(t *testing.T) {
		mockDB := &mockRepository{}
		mockDB.ToggleEndpointEnabledFn = func(ctx context.Context, endpointID string, enabled bool) (*database.Endpoint, error) {
			return &database.Endpoint{EndpointID: endpointID, Enabled: enabled}, nil
		}

		h := NewHandlersWithDeps(mockDB, &mockPublisher{}, nil)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/endpoints/toggle?endpoint_id=endpoint-1", bytes.NewBufferString(`{"enabled":false}`))
		w := httptest.NewRecorder()

		h.ToggleEndpointEnabled(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("ToggleEndpointEnabled() status = %v, want %v", w.Code, http.StatusOK)
		}
	})
}

// TestHandlers_DeleteEndpoint tests the DeleteEndpoint handler.
func TestHandlers_DeleteEndpoint(t *testing.T) {
	t.Run("successful delete", func(t *testing.T) {
		mockDB := &mockRepository{}
		mockDB.DeleteEndpointFn = func(ctx context.Context, endpointID string) error {
			return nil
		}

		h := NewHandlersWithDeps(mockDB, &mockPublisher{}, nil)
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/endpoints/delete?endpoint_id=endpoint-1", nil)
		w := httptest.NewRecorder()

		h.DeleteEndpoint(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("DeleteEndpoint() status = %v, want %v", w.Code, http.StatusNoContent)
		}
	})
}

// TestHandlers_GetNotification tests the GetNotification handler.
func TestHandlers_GetNotification(t *testing.T) {
	t.Run("successful get", func(t *testing.T) {
		mockDB := &mockRepository{}
		mockDB.GetNotificationFn = func(ctx context.Context, notificationID string) (*database.Notification, error) {
			return &database.Notification{NotificationID: notificationID, ClientID: "client-1", Status: "RECEIVED", RuleIDs: []string{"rule-1"}}, nil
		}

		h := NewHandlersWithDeps(mockDB, &mockPublisher{}, nil)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications?notification_id=notif-1", nil)
		w := httptest.NewRecorder()

		h.GetNotification(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("GetNotification() status = %v, want %v", w.Code, http.StatusOK)
		}
	})
}

// TestHandlers_ListNotifications tests the ListNotifications handler.
func TestHandlers_ListNotifications(t *testing.T) {
	t.Run("list all with pagination", func(t *testing.T) {
		mockDB := &mockRepository{}
		mockDB.ListNotificationsFn = func(ctx context.Context, clientID *string, status *string, limit, offset int) (*database.NotificationListResult, error) {
			return &database.NotificationListResult{
				Notifications: []*database.Notification{{NotificationID: "notif-1", Status: "RECEIVED"}},
				Total:         1,
				Limit:         limit,
				Offset:        offset,
			}, nil
		}

		h := NewHandlersWithDeps(mockDB, &mockPublisher{}, nil)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications?limit=50&offset=0", nil)
		w := httptest.NewRecorder()

		h.ListNotifications(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("ListNotifications() status = %v, want %v", w.Code, http.StatusOK)
		}
	})
}

// TestRuleEventPublishing verifies that rule CRUD operations publish events correctly.
func TestRuleEventPublishing(t *testing.T) {
	t.Run("create publishes CREATED event", func(t *testing.T) {
		mockDB := &mockRepository{}
		mockDB.CreateRuleFn = func(ctx context.Context, clientID, severity, source, name string) (*database.Rule, error) {
			return &database.Rule{RuleID: "rule-1", ClientID: clientID, Severity: severity, Version: 1, UpdatedAt: time.Now()}, nil
		}
		mockPub := &mockPublisher{}
		mockMetrics := &mockMetrics{}

		h := NewHandlersWithDeps(mockDB, mockPub, mockMetrics)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/rules", bytes.NewBufferString(`{"client_id":"client-1","severity":"HIGH","source":"src","name":"alert"}`))
		w := httptest.NewRecorder()

		h.CreateRule(w, req)

		if len(mockPub.Published) != 1 {
			t.Fatalf("Expected 1 published event, got %d", len(mockPub.Published))
		}
		if mockPub.Published[0].Action != "CREATED" {
			t.Errorf("Expected CREATED action, got %s", mockPub.Published[0].Action)
		}
		if mockMetrics.PublishedCount != 1 {
			t.Errorf("Expected 1 published metric, got %d", mockMetrics.PublishedCount)
		}
	})

	t.Run("update publishes UPDATED event", func(t *testing.T) {
		mockDB := &mockRepository{}
		mockDB.UpdateRuleFn = func(ctx context.Context, ruleID string, severity, source, name string, expectedVersion int) (*database.Rule, error) {
			return &database.Rule{RuleID: ruleID, Severity: severity, Version: 2, UpdatedAt: time.Now()}, nil
		}
		mockPub := &mockPublisher{}

		h := NewHandlersWithDeps(mockDB, mockPub, nil)
		req := httptest.NewRequest(http.MethodPut, "/api/v1/rules/update?rule_id=rule-1", bytes.NewBufferString(`{"severity":"CRITICAL","source":"src","name":"alert","version":1}`))
		w := httptest.NewRecorder()

		h.UpdateRule(w, req)

		if len(mockPub.Published) != 1 {
			t.Fatalf("Expected 1 published event, got %d", len(mockPub.Published))
		}
		if mockPub.Published[0].Action != "UPDATED" {
			t.Errorf("Expected UPDATED action, got %s", mockPub.Published[0].Action)
		}
	})

	t.Run("delete publishes DELETED event", func(t *testing.T) {
		mockDB := &mockRepository{}
		mockDB.GetRuleFn = func(ctx context.Context, ruleID string) (*database.Rule, error) {
			return &database.Rule{RuleID: ruleID, ClientID: "client-1", Version: 1}, nil
		}
		mockDB.DeleteRuleFn = func(ctx context.Context, ruleID string) error {
			return nil
		}
		mockPub := &mockPublisher{}

		h := NewHandlersWithDeps(mockDB, mockPub, nil)
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/rules/delete?rule_id=rule-1", nil)
		w := httptest.NewRecorder()

		h.DeleteRule(w, req)

		if len(mockPub.Published) != 1 {
			t.Fatalf("Expected 1 published event, got %d", len(mockPub.Published))
		}
		if mockPub.Published[0].Action != "DELETED" {
			t.Errorf("Expected DELETED action, got %s", mockPub.Published[0].Action)
		}
	})

	t.Run("toggle disabled publishes DISABLED event", func(t *testing.T) {
		mockDB := &mockRepository{}
		mockDB.ToggleRuleEnabledFn = func(ctx context.Context, ruleID string, enabled bool, expectedVersion int) (*database.Rule, error) {
			return &database.Rule{RuleID: ruleID, Enabled: false, Version: 2, UpdatedAt: time.Now()}, nil
		}
		mockPub := &mockPublisher{}

		h := NewHandlersWithDeps(mockDB, mockPub, nil)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/rules/toggle?rule_id=rule-1", bytes.NewBufferString(`{"enabled":false,"version":1}`))
		w := httptest.NewRecorder()

		h.ToggleRuleEnabled(w, req)

		if len(mockPub.Published) != 1 {
			t.Fatalf("Expected 1 published event, got %d", len(mockPub.Published))
		}
		if mockPub.Published[0].Action != "DISABLED" {
			t.Errorf("Expected DISABLED action, got %s", mockPub.Published[0].Action)
		}
	})
}
