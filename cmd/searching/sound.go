package main

import (
	"fmt"
	"github.com/janearc/slayer/interfaces/movie"
)

// excerptSound releases the full decoded track before scene preparation.
// Only samples that can be heard in this take cross the phase boundary.
func excerptSound(path string, from, count, fps int) (movie.Sound, error) {
	sound, err := movie.ReadSound(path)
	if err != nil {
		return movie.Sound{}, err
	}
	first := int(int64(from) * int64(sound.Rate()) / int64(fps))
	last := min(sound.Length(), int(int64(from+count)*int64(sound.Rate())/int64(fps)))
	if first >= last {
		return movie.Sound{}, fmt.Errorf("audio trim past end")
	}
	return sound.Slice(first, last)
}
