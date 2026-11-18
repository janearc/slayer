package terminal

import (
	"os"
	"syscall"
	"testing"
	"time"
)

// TestFollowStopsWhenTheLiveViewIsClosedWhileItSends checks that a resize nobody reads is dropped when the view closes, and the screen is still brought up to date first.
func TestFollowStopsWhenTheLiveViewIsClosedWhileItSends(t *testing.T) {
	master, slave := ptyPair(t)
	resize(t, master, laterCols, laterRows, 0, 0)
	live := &Live{out: slave, events: make(chan Event), done: make(chan struct{}),
		winch:  make(chan os.Signal, 1),
		screen: Screen{Cols: startCols, Rows: startRows}}
	live.workers.Add(1)
	go live.follow()
	live.winch <- syscall.SIGWINCH
	// The follower measures, finds a new size and blocks sending the event,
	// since nothing reads live.events. Closing done must free it.
	deadline := time.Now().Add(eventLimit)
	for live.Screen().Cols != laterCols {
		if time.Now().After(deadline) {
			t.Fatal("the screen was not remeasured")
		}
		time.Sleep(time.Millisecond)
	}
	close(live.done)
	finished := make(chan struct{})
	go func() { live.workers.Wait(); close(finished) }()
	select {
	case <-finished:
	case <-time.After(eventLimit):
		t.Fatal("the follower did not stop when the view closed")
	}
	if got := live.Screen(); got.Cols != laterCols || got.Rows != laterRows {
		t.Errorf("screen is %+v, want %d by %d", got, laterCols, laterRows)
	}
	select {
	case event := <-live.events:
		t.Errorf("an event was delivered after close: %+v", event)
	default:
	}
}
