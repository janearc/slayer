package host

import (
	"errors"
	"fmt"
	"image"
	"math"
	"os"
	"path/filepath"

	"github.com/janearc/slayer/interfaces/movie"
)

// clipEnding is how a clip's file is named: the clip's name and this.
const clipEnding = ".mp4"

// Loop is a short film on black, read whole, played round and round.
type Loop struct {
	frames []*image.RGBA
	rate   float64
}

// At is the loop's picture at seconds in, round and round.
func (loop Loop) At(seconds float64) *image.RGBA {
	count := len(loop.frames)
	at := int(math.Floor(seconds*loop.rate)) % count
	if at < none {
		at += count
	}
	return loop.frames[at]
}

// ReadLoop reads a film whole, each picture scaled to a size. It refuses
// a film it cannot open and one with no pictures.
func ReadLoop(path string, size image.Point) (Loop, error) {
	film, err := movie.OpenFilm(path, none, size)
	if err != nil {
		return Loop{}, fmt.Errorf("host: %s: %w", path, err)
	}
	defer film.Close()
	loop := Loop{rate: film.Rate()}
	for {
		picture, err := film.Next()
		if errors.Is(err, movie.ErrPastEnd) {
			break
		}
		if err != nil {
			return Loop{}, fmt.Errorf("host: %s: %w", path, err)
		}
		loop.frames = append(loop.frames, picture)
	}
	if len(loop.frames) == none || !(loop.rate > none) {
		return Loop{}, fmt.Errorf("host: %s has no pictures to loop",
			path)
	}
	return loop, nil
}

// Clips is the host's loops by name.
type Clips map[string]Loop

// ReadClips reads the host's five clips from a folder, each NAME.mp4,
// scaled to a size. A folder that does not exist is no clips, and the
// host is drawn as a placeholder; a folder missing one of the five is
// refused by its name.
func ReadClips(dir string, size image.Point) (Clips, error) {
	if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
		return Clips{}, nil
	}
	clips := Clips{}
	for _, name := range []string{Resting, Speaking, Angry, Flash,
		Quiet} {
		loop, err := ReadLoop(filepath.Join(dir, name+clipEnding), size)
		if err != nil {
			return nil, err
		}
		clips[name] = loop
	}
	return clips, nil
}
