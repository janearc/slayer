package job

import (
	"testing"
	"time"
)

// TestHyperlapseMusicUsesPlaybackSeconds checks one hour in thirty
// seconds: a cue halfway through must hear fifteen seconds of music.
func TestHyperlapseMusicUsesPlaybackSeconds(t *testing.T) {
	clock := Clock{playing: 30, stretch: &quantity{Value: 3600}}
	if got := clock.Playback(30 * time.Minute); got != 15*time.Second {
		t.Fatalf("music followed scene time: %s", got)
	}
	if clock.Playback(0) != 0 || clock.Playback(time.Hour) != 30*time.Second {
		t.Fatal("playback mapping lost its endpoints")
	}
	// anti-test of TestHyperlapseMusicUsesPlaybackSeconds: no stretch
	// means ordinary real-time playback, without an accidental scale.
	if got := (Clock{}).Playback(17 * time.Second); got != 17*time.Second {
		t.Fatalf("ordinary playback was stretched: %s", got)
	}
}
