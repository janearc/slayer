//go:build draw

package session

import (
	"bytes"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/moonlight"
)

func TestTheSkyFromTheCacheIsTheSkyWorkedOut(t *testing.T) {
	cachedHome(t)
	cold, coldNote, err := keptSky(measure.Metres(cachedHeight),
		cachedToward, moonlight.Full(), cachedExposure, "the sky")
	if err != nil || coldNote.hit || coldNote.keptErr != nil {
		t.Fatalf("cold: %v, %v", coldNote, err)
	}
	warm, warmNote, err := keptSky(measure.Metres(cachedHeight),
		cachedToward, moonlight.Full(), cachedExposure, "the sky")
	if err != nil || !warmNote.hit {
		t.Fatalf("warm: %v, %v", warmNote, err)
	}
	if !bytes.Equal(warm.Bytes(), cold.Bytes()) {
		t.Fatal("the sky from the cache is not the sky worked out")
	}
	t.Logf("worked out in %v, read in %v", coldNote.spent, warmNote.spent)
}
