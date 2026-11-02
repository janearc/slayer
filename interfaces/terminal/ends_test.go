package terminal

import (
	"testing"
	"time"
)

// ends runs call and fails the test if it has not come back in a few
// seconds. a loop that counts the wrong way never ends, and would hold
// up the whole run instead of failing in it. call must not touch t.
func ends(t *testing.T, what string, call func()) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		call()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatalf("%s did not finish in three seconds", what)
	}
}
