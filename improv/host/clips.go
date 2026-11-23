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

// loop is a short film on black, read whole, played round and round.
type loop struct {
	frames []*image.RGBA
	rate   float64
}

// at is the loop's picture at seconds in, round and round.
func (playing loop) at(seconds float64) *image.RGBA {
	count := len(playing.frames)
	turns := int(math.Floor(seconds*playing.rate)) % count
	if turns < none {
		turns += count
	}
	return playing.frames[turns]
}

// readLoop reads a film whole, each picture scaled to a size. It refuses
// a film it cannot open and one with no pictures.
func readLoop(path string, size image.Point) (loop, error) {
	film, err := movie.OpenFilm(path, none, size)
	if err != nil {
		return loop{}, fmt.Errorf("host: %s: %w", path, err)
	}
	defer film.Close()
	read := loop{rate: film.Rate()}
	for {
		picture, err := film.Next()
		if errors.Is(err, movie.ErrPastEnd) {
			break
		}
		if err != nil {
			return loop{}, fmt.Errorf("host: %s: %w", path, err)
		}
		read.frames = append(read.frames, picture)
	}
	if len(read.frames) == none || !(read.rate > none) {
		return loop{}, fmt.Errorf("host: %s has no pictures to loop",
			path)
	}
	return read, nil
}

// Clips is the host's loops by name.
type Clips map[string]loop

// ReadClips reads the host's five clips from a folder, each NAME.mp4,
// scaled to a size. A folder that does not exist is no clips, and the
// host is drawn as a placeholder; a folder missing one of the five is
// refused by its name.
func ReadClips(dir string, size image.Point) (Clips, error) {
	if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
		return Clips{}, nil
	}
	clips := Clips{}
	for _, name := range []string{resting, speaking, angry, flash,
		quiet} {
		read, err := readLoop(filepath.Join(dir, name+clipEnding), size)
		if err != nil {
			return nil, err
		}
		clips[name] = read
	}
	return clips, nil
}
