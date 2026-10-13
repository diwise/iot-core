package decorators

import (
	"testing"
	"time"

	"github.com/matryer/is"
)

// REV-010: the cache sweeper stop handle must exist, be safe to call
// more than once, and leave the cache usable.
func TestStopCacheCleanup(t *testing.T) {
	is := is.New(t)

	is.True(stopCacheCleanup != nil)

	StopCacheCleanup()
	StopCacheCleanup()

	c.Set("k", 1.0, time.Hour)
	v, ok := c.Get("k")
	is.True(ok)
	is.Equal(v, 1.0)
}
