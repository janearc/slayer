package movie

import (
	"errors"
	"fmt"
	"image"
	"time"
)

// ErrPastEnd is a film read past its last picture, or opened at a moment
// after it.
var ErrPastEnd = errors.New("movie: past the film's end")

// pictures is a film's pictures as the platform decodes them, in order.
type pictures interface {
	// next decodes the next picture into a frame and says its time in
	// seconds; false when there are no more.
	next(into *image.RGBA) (float64, bool, error)
	// release stops reading.
	release()
}

// filmFacts is what opening a film says of it: its length in seconds,
// its nominal rate, and its own size.
type filmFacts struct {
	seconds, rate float64
	size          image.Point
}

// Film is a film being read for its pictures, in order, each scaled by
// the decoder to one size. Shape is the caller's to keep: a size of
// another shape is stretched to it.
type Film struct {
	source pictures
	facts  filmFacts
	size   image.Point
	from   float64
	shown  float64
}

// OpenFilm opens a film to read its pictures from a moment, each scaled
// to a size. A moment at or after the film's end is ErrPastEnd; a size
// of nothing, or over mostSide on a side, is refused by name.
func OpenFilm(path string, from time.Duration, size image.Point) (*Film,
	error) {
	if err := readable(path); err != nil {
		return nil, err
	}
	if size.X < 1 || size.Y < 1 || size.X > mostSide || size.Y > mostSide {
		return nil, fmt.Errorf("movie: pictures of %dx%d; each side is 1 "+
			"to %d", size.X, size.Y, mostSide)
	}
	if from < 0 {
		return nil, fmt.Errorf("movie: a film read from %v; start at "+
			"nought or after", from)
	}
	source, facts, err := openFilm(path, from.Seconds(), size)
	if err != nil {
		return nil, err
	}
	if from.Seconds() >= facts.seconds {
		source.release()
		return nil, fmt.Errorf("%w: %s ends at %.3f s; there is no "+
			"picture at %.3f s", ErrPastEnd, path, facts.seconds,
			from.Seconds())
	}
	return &Film{source: source, facts: facts, size: size,
		from: from.Seconds(), shown: from.Seconds()}, nil
}

// Next is the film's next picture: the first is the one showing at the
// moment it was opened at, and each after follows in order. Past the
// last it is ErrPastEnd.
func (film *Film) Next() (*image.RGBA, error) {
	if film.source == nil {
		return nil, fmt.Errorf("movie: the film is closed; open it again")
	}
	picture := image.NewRGBA(image.Rectangle{Max: film.size})
	for {
		at, more, err := film.source.next(picture)
		if err != nil {
			return nil, err
		}
		if !more {
			return nil, fmt.Errorf("%w: no picture after %.3f s; the "+
				"film ends at %.3f s", ErrPastEnd, film.shown,
				film.facts.seconds)
		}
		// a reader may start a little early, at the picture before
		// the one showing at the moment asked: pass it over. A moment
		// asked as whole nanoseconds may fall a hair before the picture
		// it names; seekSlack keeps that from showing the one before.
		if film.facts.rate > 0 && at+1/film.facts.rate <= film.from+seekSlack {
			continue
		}
		film.shown = at
		return picture, nil
	}
}

// Shown is the time of the picture Next last gave, from the film's
// start; before any, the moment the film was opened at.
func (film *Film) Shown() time.Duration { return seconds(film.shown) }

// Length is the film's length.
func (film *Film) Length() time.Duration { return seconds(film.facts.seconds) }

// Rate is the film's frames a second, as it says of itself.
func (film *Film) Rate() float64 { return film.facts.rate }

// Native is the film's own size, before scaling.
func (film *Film) Native() image.Point { return film.facts.size }

// Close stops reading. It is safe to call more than once.
func (film *Film) Close() {
	if film.source != nil {
		film.source.release()
		film.source = nil
	}
}

// FrameAt is the picture showing at a moment of a film, scaled to a
// size.
func FrameAt(path string, at time.Duration, size image.Point) (
	*image.RGBA, error) {
	film, err := OpenFilm(path, at, size)
	if err != nil {
		return nil, err
	}
	defer film.Close()
	return film.Next()
}

// seconds is a count of seconds as a duration.
func seconds(count float64) time.Duration {
	return time.Duration(count * float64(time.Second))
}
