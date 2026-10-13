package cache

import (
	"testing"
	"time"

	"github.com/matryer/is"
)

func present(c *Cache, key string) bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	_, ok := c.items[key]
	return ok
}

// REV-014: the sweeper must physically delete expired items from the
// map. Get already hides expired items without any cleanup running, so
// only a map-level assertion proves the worker did its job.
func TestCleanupPhysicallyDeletesExpiredItems(t *testing.T) {
	is := is.New(t)

	c := NewCache()
	stop := c.Cleanup(10 * time.Millisecond)
	defer stop()
	is.True(stop != nil)

	c.Set("expired", "x", -time.Minute)
	c.Set("fresh", "y", time.Hour)

	deadline := time.Now().Add(2 * time.Second)
	for present(c, "expired") {
		if time.Now().After(deadline) {
			t.Fatal("expired item was not physically deleted")
		}
		time.Sleep(10 * time.Millisecond)
	}

	is.True(present(c, "fresh"))

	v, found := c.Get("fresh")
	is.True(found)
	is.Equal(v, "y")
}

// REV-014: after stop, no further sweeps may run. An expired item added
// after the stop must stay physically present across several tick
// periods, proving worker exit rather than a silent Get filter.
func TestCleanupStopsSweepingAfterStop(t *testing.T) {
	is := is.New(t)

	c := NewCache()
	stop := c.Cleanup(10 * time.Millisecond)

	stop()
	stop()

	c.Set("post-stop-expired", "x", -time.Minute)
	time.Sleep(150 * time.Millisecond)
	is.True(present(c, "post-stop-expired"))

	c.Set("after-stop", "z", time.Hour)
	v, found := c.Get("after-stop")
	is.True(found)
	is.Equal(v, "z")
}
