package main

import (
	"context"
	"os"
	"testing"

	"github.com/matryer/is"
)

// withUnsetEnv removes a variable for the test duration, restoring any
// ambient value afterwards. Ambient developer/CI environments must not
// leak into default assertions.
func withUnsetEnv(t *testing.T, key string) {
	t.Helper()

	v, ok := os.LookupEnv(key)
	os.Unsetenv(key)
	t.Cleanup(func() {
		if ok {
			os.Setenv(key, v)
		}
	})
}

// REV-015: tests target the production seam functions, not the env
// library, so a changed default or lookup breaks them.
func TestServicePortSeam(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	withUnsetEnv(t, "SERVICE_PORT")
	is.Equal(servicePort(ctx), "8080")

	t.Setenv("SERVICE_PORT", "9090")
	is.Equal(servicePort(ctx), "9090")

	// The env library maps both unset and empty to the default.
	t.Setenv("SERVICE_PORT", "")
	is.Equal(servicePort(ctx), "8080")
}

// REV-015: only the exact string "true" disables TLS verification.
// ParseBool-style spellings ("TRUE", "1", ...) intentionally do not.
func TestOAuthRealmInsecureSeam(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value *string
		want  bool
	}{
		{"unset uses secure default", nil, false},
		{"empty uses secure default", strptr(""), false},
		{"exact true disables verification", strptr("true"), true},
		{"uppercase TRUE keeps verification", strptr("TRUE"), false},
		{"numeric 1 keeps verification", strptr("1"), false},
		{"false keeps verification", strptr("false"), false},
		{"invalid keeps verification", strptr("bogus"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			is := is.New(t)

			if tc.value == nil {
				withUnsetEnv(t, "OAUTH2_REALM_INSECURE")
			} else {
				t.Setenv("OAUTH2_REALM_INSECURE", *tc.value)
			}

			is.Equal(oauthRealmInsecure(context.Background()), tc.want)
		})
	}
}

func strptr(s string) *string { return &s }

// REV-015: the functions file default lives in one constant used by
// flag registration; changing it breaks this test.
func TestDefaultFunctionsConfigPath(t *testing.T) {
	is := is.New(t)

	is.Equal(defaultFunctionsConfigPath, "/opt/diwise/config/functions.csv")
}
