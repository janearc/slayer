package movie

import (
	"fmt"
	"os"
)

// Probe reads a finished movie back and reports its Facts.
func Probe(path string) (Facts, error) {
	if err := readable(path); err != nil {
		return Facts{}, err
	}
	facts, err := probeFile(path)
	if err != nil {
		return Facts{}, err
	}
	facts.fastStart, err = fastStart(path)
	if err != nil {
		return Facts{}, err
	}
	return facts, nil
}

// ReadSound reads the first audio track of any file AVFoundation reads
// into a stereo Sound at 44100 samples a second. Mono is put on both
// channels.
func ReadSound(path string) (Sound, error) {
	if err := readable(path); err != nil {
		return Sound{}, err
	}
	decoded, count, err := decodeSound(path, readRate)
	if err != nil {
		return Sound{}, err
	}
	return fromChannels(readRate, count, decoded)
}

// readable refuses a path that is not a file that can be opened.
func readable(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("movie: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("movie: %s is not a file", path)
	}
	return nil
}
