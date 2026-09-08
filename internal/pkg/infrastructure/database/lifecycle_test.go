package database

import (
	"testing"

	"github.com/matryer/is"
)

// BASE-012: Close must be safe on a zero Storage so partial startup
// cleanup and deferred closes never panic.
func TestStorageCloseIsNilSafe(t *testing.T) {
	is := is.New(t)

	impl := &impl{}
	impl.Close()

	var s Storage = impl
	s.Close()

	mock := &StorageMock{
		CloseFunc: func() {},
	}
	mock.Close()
	is.Equal(len(mock.CloseCalls()), 1)
}
