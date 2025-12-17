package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/matryer/is"
)

func TestRuleClient_CRUD_WithHTTPtestServer(t *testing.T) {
	t.Run("GetRulesByDevice returns list", func(t *testing.T) {
		is := is.New(t)

		expected := []Rule{
			{
				ID:              "r1",
				MeasurementID:   "m1",
				DeviceID:        "dev-1",
				MeasurementType: 1,
				RuleValues: RuleValues{
					V: &RuleV{MinValue: float64Ptr(1), MaxValue: float64Ptr(2)},
				},
			},
			{
				ID:              "r2",
				MeasurementID:   "m2",
				DeviceID:        "dev-1",
				MeasurementType: 2,
				RuleValues: RuleValues{
					Vs: &RuleVs{Value: stringPtr("x")},
				},
			},
		}

		body, err := json.Marshal(expected)
		is.NoErr(err)

		client := newRuleClientForTest(t, func(w http.ResponseWriter, r *http.Request) {
			is.Equal(r.Method, http.MethodGet)
			is.Equal(r.URL.Path, "/api/v0/rules/device/dev-1")
			is.Equal(r.Header.Get("Authorization"), "Bearer test-token")

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(body)
		})

		rules, err := client.GetRulesByDevice(context.Background(), "dev-1")
		is.NoErr(err)
		is.Equal(len(rules), 2)
		is.Equal(rules[0].ID, "r1")
		is.Equal(rules[1].ID, "r2")
	})

	t.Run("GetRule returns a rule", func(t *testing.T) {
		is := is.New(t)

		expected := Rule{
			ID:              "r-123",
			MeasurementID:   "m-123",
			DeviceID:        "dev-123",
			MeasurementType: 42,
			RuleValues: RuleValues{
				Vb: &RuleVb{Value: boolPtr(true)},
			},
		}

		body, err := json.Marshal(expected)
		is.NoErr(err)

		client := newRuleClientForTest(t, func(w http.ResponseWriter, r *http.Request) {
			is.Equal(r.Method, http.MethodGet)
			is.Equal(r.URL.Path, "/api/v0/rules/r-123")
			is.Equal(r.Header.Get("Authorization"), "Bearer test-token")

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(body)
		})

		rule, err := client.GetRule(context.Background(), "r-123")
		is.NoErr(err)
		is.Equal(rule.ID, "r-123")
		is.Equal(rule.RuleValues.Vb.Value != nil && *rule.RuleValues.Vb.Value, true)
	})

	t.Run("GetRule returns ErrNotFound on 404", func(t *testing.T) {
		is := is.New(t)

		client := newRuleClientForTest(t, func(w http.ResponseWriter, r *http.Request) {
			is.Equal(r.Method, http.MethodGet)
			is.Equal(r.URL.Path, "/api/v0/rules/missing")
			w.WriteHeader(http.StatusNotFound)
		})

		_, err := client.GetRule(context.Background(), "missing")
		is.True(err != nil)
		is.True(strings.Contains(err.Error(), "failed to get rule"))
		is.True(strings.Contains(err.Error(), ErrNotFound.Error()))
	})

	t.Run("UpdateRule requires ID", func(t *testing.T) {
		is := is.New(t)

		client := newRuleClientForTest(t, func(w http.ResponseWriter, r *http.Request) {
			t.Fatalf("handler should not be called")
		})

		_, err := client.UpdateRule(context.Background(), Rule{ID: ""})
		is.True(err != nil)
		is.True(strings.Contains(err.Error(), "rule ID must be set"))
	})

	t.Run("UpdateRule sends PUT and returns message", func(t *testing.T) {
		is := is.New(t)

		client := newRuleClientForTest(t, func(w http.ResponseWriter, r *http.Request) {
			is.Equal(r.Method, http.MethodPut)
			is.Equal(r.URL.Path, "/api/v0/rules/r-1")
			is.Equal(r.Header.Get("Content-Type"), "application/json")
			is.Equal(r.Header.Get("Authorization"), "Bearer test-token")

			reqBody, err := io.ReadAll(r.Body)
			is.NoErr(err)
			is.True(strings.Contains(string(reqBody), "\"id\":\"r-1\""))

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"message":"updated"}`))
		})

		resp, err := client.UpdateRule(context.Background(), Rule{ID: "r-1", DeviceID: "dev-1"})
		is.NoErr(err)
		is.Equal(resp.Message, "updated")
	})

	t.Run("DeleteRule sends DELETE and succeeds on 204", func(t *testing.T) {
		is := is.New(t)

		client := newRuleClientForTest(t, func(w http.ResponseWriter, r *http.Request) {
			is.Equal(r.Method, http.MethodDelete)
			is.Equal(r.URL.Path, "/api/v0/rules/r-del")
			is.Equal(r.Header.Get("Authorization"), "Bearer test-token")
			w.WriteHeader(http.StatusNoContent)
		})

		err := client.DeleteRule(context.Background(), "r-del")
		is.NoErr(err)
	})

	t.Run("DeleteRule returns ErrNotFound on 404", func(t *testing.T) {
		is := is.New(t)

		client := newRuleClientForTest(t, func(w http.ResponseWriter, r *http.Request) {
			is.Equal(r.Method, http.MethodDelete)
			is.Equal(r.URL.Path, "/api/v0/rules/r-missing")
			w.WriteHeader(http.StatusNotFound)
		})

		err := client.DeleteRule(context.Background(), "r-missing")
		is.True(err != nil)
		is.True(strings.Contains(err.Error(), "failed to delete rule"))
		is.True(strings.Contains(err.Error(), ErrNotFound.Error()))
	})
}

func float64Ptr(v float64) *float64 { return &v }
func stringPtr(v string) *string    { return &v }
func boolPtr(v bool) *bool          { return &v }
