package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/diwise/iot-core/internal/pkg/infrastructure/database/rules"
	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/jackc/pgx/v5/pgconn"
)

type appMock struct {
	createRuleFunc       func(ctx context.Context, rule *rules.Rule) error
	getRulesByDeviceFunc func(ctx context.Context, deviceID string) ([]*rules.Rule, error)
	getRuleFunc          func(ctx context.Context, ruleID string) (*rules.Rule, error)
	updateRuleFunc       func(ctx context.Context, rule *rules.Rule) error
	deleteRuleFunc       func(ctx context.Context, ruleID string) error
}

func (m *appMock) MessageAccepted(ctx context.Context, evt events.MessageAccepted) error {
	panic("unexpected call to MessageAccepted")
}

func (m *appMock) MessageReceived(ctx context.Context, msg events.MessageReceived) (*events.MessageAccepted, error) {
	panic("unexpected call to MessageReceived")
}

func (m *appMock) FunctionUpdated(ctx context.Context, body []byte) error {
	panic("unexpected call to FunctionUpdated")
}

func (m *appMock) CreateRule(ctx context.Context, rule *rules.Rule) error {
	if m.createRuleFunc == nil {
		return nil
	}
	return m.createRuleFunc(ctx, rule)
}

func (m *appMock) GetRulesByDevice(ctx context.Context, deviceID string) ([]*rules.Rule, error) {
	if m.getRulesByDeviceFunc == nil {
		return nil, nil
	}
	return m.getRulesByDeviceFunc(ctx, deviceID)
}

func (m *appMock) GetRule(ctx context.Context, ruleID string) (*rules.Rule, error) {
	if m.getRuleFunc == nil {
		return nil, nil
	}
	return m.getRuleFunc(ctx, ruleID)
}

func (m *appMock) UpdateRule(ctx context.Context, rule *rules.Rule) error {
	if m.updateRuleFunc == nil {
		return nil
	}
	return m.updateRuleFunc(ctx, rule)
}

func (m *appMock) DeleteRule(ctx context.Context, ruleID string) error {
	if m.deleteRuleFunc == nil {
		return nil
	}
	return m.deleteRuleFunc(ctx, ruleID)
}

func withURLParam(r *http.Request, key, value string) *http.Request {
	r.SetPathValue(key, value)
	return r
}

func TestCreateRuleHandler_ReturnsJSONCreated(t *testing.T) {
	handler := NewCreateRuleHandler(context.Background(), &appMock{
		createRuleFunc: func(ctx context.Context, rule *rules.Rule) error {
			rule.ID = "generated-id"
			return nil
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v0/rules", strings.NewReader(`{"measurement_id":"m1","device_id":"d1","measurement_type":1,"should_abort":false,"rule_values":{"vs":{"value":"test"}}}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
		t.Fatalf("expected application/json content-type, got %q", got)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if body["id"] != "generated-id" {
		t.Fatalf("expected generated id in response, got %q", body["id"])
	}
}

func TestCreateRuleHandler_ReturnsBadRequest_ForValidationError(t *testing.T) {
	handler := NewCreateRuleHandler(context.Background(), &appMock{
		createRuleFunc: func(ctx context.Context, rule *rules.Rule) error {
			return rules.ErrorNoKindSet
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v0/rules", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestCreateRuleHandler_ReturnsConflict_ForDuplicateID(t *testing.T) {
	handler := NewCreateRuleHandler(context.Background(), &appMock{
		createRuleFunc: func(ctx context.Context, rule *rules.Rule) error {
			return &pgconn.PgError{Code: "23505"}
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v0/rules", strings.NewReader(`{"id":"existing-id"}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d", http.StatusConflict, rec.Code)
	}
}

func TestGetRuleHandler_ReturnsNotFound_ForMissingRule(t *testing.T) {
	handler := NewGetRuleHandler(context.Background(), &appMock{
		getRuleFunc: func(ctx context.Context, ruleID string) (*rules.Rule, error) {
			return nil, rules.ErrNotFound
		},
	})

	req := withURLParam(httptest.NewRequest(http.MethodGet, "/api/v0/rules/missing", nil), "ruleId", "missing")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestGetRuleHandler_ReturnsInternalServerError_ForUnexpectedError(t *testing.T) {
	handler := NewGetRuleHandler(context.Background(), &appMock{
		getRuleFunc: func(ctx context.Context, ruleID string) (*rules.Rule, error) {
			return nil, errors.New("db down")
		},
	})

	req := withURLParam(httptest.NewRequest(http.MethodGet, "/api/v0/rules/r1", nil), "ruleId", "r1")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestUpdateRuleHandler_ReturnsNotFound_ForMissingRule(t *testing.T) {
	handler := NewUpdateRuleHandler(context.Background(), &appMock{
		updateRuleFunc: func(ctx context.Context, rule *rules.Rule) error {
			return rules.ErrNotFound
		},
	})

	req := withURLParam(httptest.NewRequest(http.MethodPut, "/api/v0/rules/missing", strings.NewReader(`{"measurement_id":"m1"}`)), "ruleId", "missing")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestDeleteRuleHandler_ReturnsNotFound_ForMissingRule(t *testing.T) {
	handler := NewDeleteRuleHandler(context.Background(), &appMock{
		deleteRuleFunc: func(ctx context.Context, ruleID string) error {
			return rules.ErrNotFound
		},
	})

	req := withURLParam(httptest.NewRequest(http.MethodDelete, "/api/v0/rules/missing", nil), "ruleId", "missing")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}
