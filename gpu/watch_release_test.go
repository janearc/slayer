package gpu

import (
	"context"
	"testing"
)

// A watch keeps the release of the device's waits while it is kept, and
// lets it go, once, when it is let go: a watch left holding one would
// stop the next frame's waits on a context that was done.
func TestAWatchLetsGoOfTheDeviceWhenItIsLetGo(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	Watched(NewWatch(ctx, nil))
	if watched.release == nil {
		t.Error("a kept watch holds no release of the device's waits")
	}
	Unwatched()
	if watched.release != nil {
		t.Error("a watch let go still holds a release")
	}
}

// never is a watch's context when it has none.
var never context.Context

// A watch with no context has nothing to release and is let go all the
// same, as is one that was never kept.
func TestAWatchOfNoContextIsLetGoQuietly(t *testing.T) {
	Unwatched()
	Watched(NewWatch(never, nil))
	Unwatched()
	if watched.release != nil {
		t.Error("a watch of no context holds a release after it is let go")
	}
}
