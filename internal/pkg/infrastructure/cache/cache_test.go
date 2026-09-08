package cache

import (
	"testing"
	"time"

	"github.com/matryer/is"
)

// BASE-012: expired items are swept while cleanup runs, and the
// returned stop func terminates the background goroutine. Stopping
// twice must be safe and the cache must stay usable afterwards.
func TestCleanupSweepsExpiredItemsAndStops(t *testing.T) {
	is := is.New(t)

	c := NewCache()
	stop := c.Cleanup(10 * time.Millisecond)
	is.True(stop != nil)

	c.Set("expired", "x", -time.Minute)
	c.Set("fresh", "y", time.Hour)

	deadline := time.Now().Add(2 * time.Second)
	for {
		_, found := c.Get("expired")
		if !found {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("expired item was not swept")
		}
		time.Sleep(10 * time.Millisecond)
	}

	v, found := c.Get("fresh")
	is.True(found)
	is.Equal(v, "y")

	stop()
	stop()

	c.Set("after-stop", "z", time.Hour)
	v, found = c.Get("after-stop")
	is.True(found)
	is.Equal(v, "z")
}
