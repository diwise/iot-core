package measurements

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/matryer/is"
)

func tokenServer(t *testing.T) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"access_token":"test-token","token_type":"Bearer","expires_in":3600}`)
	}))
}

// BASE-012: the client owns a cache cleanup goroutine and HTTP idle
// connections; Close must release both and be safe to call twice.
func TestMeasurementsClientClose(t *testing.T) {
	is := is.New(t)

	ts := tokenServer(t)
	defer ts.Close()

	c, err := NewMeasurementsClient(context.Background(), "http://measurements", ts.URL, "id", "secret", false)
	is.NoErr(err)
	is.True(c != nil)

	c.Close()
	c.Close()
}
