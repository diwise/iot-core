package measurements

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
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

type connTracker struct {
	mu    sync.Mutex
	conns map[net.Conn]http.ConnState
}

// REV-011: Close must release idle connections on the owned base
// transport. Calling CloseIdleConnections on the api client alone is
// not enough: its immediate transport is the oauth2 transport, which
// does not implement the method.
func TestMeasurementsClientCloseReleasesIdleConnections(t *testing.T) {
	is := is.New(t)

	tracker := &connTracker{conns: map[net.Conn]http.ConnState{}}

	var mu sync.Mutex
	tokenPosts := 0
	apiAuthHeader := ""
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/v0/measurements" {
			mu.Lock()
			apiAuthHeader = r.Header.Get("Authorization")
			mu.Unlock()
			fmt.Fprint(w, `{"data":{"max":2.5}}`)
			return
		}
		mu.Lock()
		tokenPosts++
		mu.Unlock()
		fmt.Fprint(w, `{"access_token":"test-token","token_type":"Bearer","expires_in":3600}`)
	}))
	srv.Config.ConnState = func(c net.Conn, s http.ConnState) {
		tracker.mu.Lock()
		defer tracker.mu.Unlock()
		if s == http.StateClosed || s == http.StateHijacked {
			delete(tracker.conns, c)
			return
		}
		tracker.conns[c] = s
	}
	srv.Start()
	defer srv.Close()

	idle := func() int {
		tracker.mu.Lock()
		defer tracker.mu.Unlock()
		n := 0
		for _, s := range tracker.conns {
			if s == http.StateIdle {
				n++
			}
		}
		return n
	}

	c, err := NewMeasurementsClient(context.Background(), srv.URL, srv.URL, "id", "secret", false)
	is.NoErr(err)

	v, err := c.GetMaxValue(context.Background(), "m1")
	is.NoErr(err)
	is.Equal(v, 2.5)

	// The oauth2 transport attaches the cached token itself; no
	// redundant token fetch may hit the wire per API call.
	mu.Lock()
	is.Equal(tokenPosts, 1)
	is.Equal(apiAuthHeader, "Bearer test-token")
	mu.Unlock()

	deadline := time.Now().Add(5 * time.Second)
	for idle() == 0 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	is.True(idle() > 0)

	c.Close()

	deadline = time.Now().Add(5 * time.Second)
	for idle() > 0 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	is.Equal(idle(), 0)
}

// REV-011: a token failure must also release the transport used for
// the failed exchange before returning the error.
func TestMeasurementsClientTokenFailureReleasesTransport(t *testing.T) {
	is := is.New(t)

	tracker := &connTracker{conns: map[net.Conn]http.ConnState{}}

	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	srv.Config.ConnState = func(c net.Conn, s http.ConnState) {
		tracker.mu.Lock()
		defer tracker.mu.Unlock()
		if s == http.StateClosed || s == http.StateHijacked {
			delete(tracker.conns, c)
			return
		}
		tracker.conns[c] = s
	}
	srv.Start()
	defer srv.Close()

	_, err := NewMeasurementsClient(context.Background(), "http://measurements", srv.URL, "id", "secret", false)
	is.True(err != nil)

	idle := func() int {
		tracker.mu.Lock()
		defer tracker.mu.Unlock()
		n := 0
		for _, s := range tracker.conns {
			if s == http.StateIdle {
				n++
			}
		}
		return n
	}

	deadline := time.Now().Add(5 * time.Second)
	for idle() > 0 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	is.Equal(idle(), 0)
}

// CORE-006: measurements-svar får aldrig loggas rutinmässigt. Klienten
// loggar svarstorlek som metadata; en unik markörsträng i bodyn får
// inte förekomma i loggarna.
func TestMeasurementsResponseBodyIsNotLogged(t *testing.T) {
	is := is.New(t)

	const marker = "LOGPROBE-RESP-4c1e"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/v0/measurements" {
			fmt.Fprintf(w, `{"data":{"max":2.5},"marker":%q}`, marker)
			return
		}
		fmt.Fprint(w, `{"access_token":"test-token","token_type":"Bearer","expires_in":3600}`)
	}))
	defer srv.Close()

	c, err := NewMeasurementsClient(context.Background(), srv.URL, srv.URL, "id", "secret", false)
	is.NoErr(err)
	defer c.Close()

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	ctx := logging.NewContextWithLogger(context.Background(), logger)

	v, err := c.GetMaxValue(ctx, "m1")
	is.NoErr(err)
	is.Equal(v, 2.5)

	out := buf.String()
	is.True(!strings.Contains(out, marker))
	is.True(strings.Contains(out, "size"))
}
