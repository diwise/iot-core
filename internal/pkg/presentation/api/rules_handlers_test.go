package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/diwise/iot-core/internal/pkg/infrastructure/database/rules"
	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/go-chi/chi/v5"
)

func TestCreateRuleHandler_CreatesRule(t *testing.T) {
	var received *rules.Rule
	app := &appMock{
		createRuleFunc: func(ctx context.Context, rule *rules.Rule) error {
			rule.ID = "rule-1"
			received = rule
			return nil
		},
	}

	ctx := testLoggerContext()
	handler := NewCreateRuleHandler(ctx, app)

	body := `{"measurement_id":"m1","device_id":"dev-123","rule_values":{"vs":{"value":"foo"}}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v0/rules", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rec.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["id"] != "rule-1" {
		t.Fatalf("expected id rule-1, got %s", resp["id"])
	}
	if received == nil || received.DeviceID != "dev-123" {
		t.Fatalf("expected rule to be passed to app, got %+v", received)
	}
}

func TestGetRulesByDeviceHandler_ReturnsRules(t *testing.T) {
	app := &appMock{
		getRulesByDeviceFunc: func(ctx context.Context, deviceID string) ([]*rules.Rule, error) {
			return []*rules.Rule{
				{ID: "rule-2", DeviceID: deviceID},
			}, nil
		},
	}

	ctx := testLoggerContext()
	handler := NewGetRulesByDeviceHandler(ctx, app)

	req := httptest.NewRequest(http.MethodGet, "/api/v0/rules/device/dev-123", nil)
	req = withChiURLParam(req, "deviceId", "dev-123")
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp []rules.Rule
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(resp) != 1 || resp[0].ID != "rule-2" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestGetRuleHandler_ReturnsRule(t *testing.T) {
	app := &appMock{
		getRuleFunc: func(ctx context.Context, ruleID string) (*rules.Rule, error) {
			return &rules.Rule{ID: ruleID, DeviceID: "dev-123"}, nil
		},
	}

	ctx := testLoggerContext()
	handler := NewGetRuleHandler(ctx, app)

	req := httptest.NewRequest(http.MethodGet, "/api/v0/rules/rule-3", nil)
	req = withChiURLParam(req, "ruleId", "rule-3")
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp rules.Rule
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.ID != "rule-3" {
		t.Fatalf("unexpected rule: %+v", resp)
	}
}

func TestUpdateRuleHandler_ForwardsRule(t *testing.T) {
	var received *rules.Rule
	app := &appMock{
		updateRuleFunc: func(ctx context.Context, rule *rules.Rule) error {
			received = rule
			return nil
		},
	}

	ctx := testLoggerContext()
	handler := NewUpdateRuleHandler(ctx, app)

	body := `{"measurement_id":"m1","device_id":"dev-123","rule_values":{"vs":{"value":"bar"}}}`
	req := httptest.NewRequest(http.MethodPut, "/api/v0/rules/rule-4", bytes.NewBufferString(body))
	req = withChiURLParam(req, "ruleId", "rule-4")
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	if received == nil || received.ID != "rule-4" {
		t.Fatalf("expected rule id to be set from path, got %+v", received)
	}
}

func TestDeleteRuleHandler_DeletesRule(t *testing.T) {
	var deletedID string
	app := &appMock{
		deleteRuleFunc: func(ctx context.Context, ruleID string) error {
			deletedID = ruleID
			return nil
		},
	}

	ctx := testLoggerContext()
	handler := NewDeleteRuleHandler(ctx, app)

	req := httptest.NewRequest(http.MethodDelete, "/api/v0/rules/rule-5", nil)
	req = withChiURLParam(req, "ruleId", "rule-5")
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", rec.Code)
	}
	if deletedID != "rule-5" {
		t.Fatalf("expected delete to receive rule-5, got %s", deletedID)
	}
}

func withChiURLParam(req *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func testLoggerContext() context.Context {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return logging.NewContextWithLogger(context.Background(), logger)
}

type appMock struct {
	createRuleFunc       func(ctx context.Context, rule *rules.Rule) error
	getRulesByDeviceFunc func(ctx context.Context, deviceID string) ([]*rules.Rule, error)
	getRuleFunc          func(ctx context.Context, ruleID string) (*rules.Rule, error)
	updateRuleFunc       func(ctx context.Context, rule *rules.Rule) error
	deleteRuleFunc       func(ctx context.Context, ruleID string) error
}

func (m *appMock) MessageAccepted(ctx context.Context, evt events.MessageAccepted) error {
	return nil
}

func (m *appMock) MessageReceived(ctx context.Context, msg events.MessageReceived) (*events.MessageAccepted, error) {
	return nil, nil
}

func (m *appMock) FunctionUpdated(ctx context.Context, body []byte) error {
	return nil
}

func (m *appMock) CreateRule(ctx context.Context, rule *rules.Rule) error {
	if m.createRuleFunc != nil {
		return m.createRuleFunc(ctx, rule)
	}
	return nil
}

func (m *appMock) GetRulesByDevice(ctx context.Context, deviceID string) ([]*rules.Rule, error) {
	if m.getRulesByDeviceFunc != nil {
		return m.getRulesByDeviceFunc(ctx, deviceID)
	}
	return nil, nil
}

func (m *appMock) GetRule(ctx context.Context, ruleID string) (*rules.Rule, error) {
	if m.getRuleFunc != nil {
		return m.getRuleFunc(ctx, ruleID)
	}
	return nil, nil
}

func (m *appMock) UpdateRule(ctx context.Context, rule *rules.Rule) error {
	if m.updateRuleFunc != nil {
		return m.updateRuleFunc(ctx, rule)
	}
	return nil
}

func (m *appMock) DeleteRule(ctx context.Context, ruleID string) error {
	if m.deleteRuleFunc != nil {
		return m.deleteRuleFunc(ctx, ruleID)
	}
	return nil
}
