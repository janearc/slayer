package door

import (
	"errors"
	"fmt"
	"image"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/janearc/slayer/interfaces/movie"
	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/render/finish"
)

// check checks the movie's settings before any frame is drawn: its rate
// and count always, and its size where the job or its plates say it. A
// refused movie leaves nothing behind, not even its folder.
func (going *filming) check() error {
	path := going.run.at.movie
	size, known := knownSize(going.run.wanted)
	if path == "" || !known {
		return nil
	}
	folder := filepath.Dir(path)
	_, err := os.Stat(folder)
	fresh := errors.Is(err, fs.ErrNotExist)
	if going.looking {
		if fresh {
			// the folder draw would make; its settings are the same
			path = filepath.Join(os.TempDir(), filepath.Base(path))
		}
		_, err := going.settingsAt(path, size)
		return err
	}
	if err := os.MkdirAll(folder, folderMode); err != nil {
		return err
	}
	settings, err := going.settingsFor(size)
	if err != nil && fresh {
		return errors.Join(err, os.Remove(folder))
	}
	going.settings = &settings
	return err
}

// knownSize is the size of a film's frames before any is drawn: the
// job's width and height where it says both, or its plates' size where
// it says neither and draws frames already rendered. A set or a smoke
// that leaves its size to the scene is known only from its first frame.
func knownSize(wanted job.Job) (image.Point, bool) {
	width, hasWidth := wanted.Film().Width()
	height, hasHeight := wanted.Film().Height()
	if hasWidth && hasHeight {
		return image.Pt(width, height), true
	}
	folder, isFrames := wanted.Scene().Frames()
	if !isFrames || hasWidth || hasHeight {
		return image.Point{}, false
	}
	plates, err := finish.OpenPlates(folder)
	if err != nil {
		return image.Point{}, false
	}
	width, height = plates.Size()
	return image.Pt(width, height), true
}

// settingsFor is the movie's settings for frames of a size, padded to
// the height the writer takes.
func (going *filming) settingsFor(size image.Point) (movie.Settings,
	error) {
	return going.settingsAt(going.run.at.movie, size)
}

// settingsAt is settingsFor with the movie at a path of the caller's.
func (going *filming) settingsAt(path string, size image.Point) (
	movie.Settings, error) {
	return movie.NewSettings(movie.Request{
		Path: path, Width: size.X, Height: size.Y,
		Rate: going.rate(), Frames: going.frames, Sound: going.sound,
		Pad: true, Replace: true})
}

// open opens the movie for its first picture: with the settings checked
// before drawing, which the picture must fit, or else with settings made
// for the picture's size. It says so once when it pads.
func (going *filming) open(size image.Point) error {
	run := going.run
	if going.settings == nil {
		if err := os.MkdirAll(filepath.Dir(run.at.movie),
			folderMode); err != nil {
			return err
		}
		settings, err := going.settingsFor(size)
		if err != nil {
			return err
		}
		going.settings = &settings
	}
	if planned := going.settings.Picture(); planned != size {
		return fmt.Errorf("the first frame is %dx%d and the movie was "+
			"set for %dx%d from the job; say the film's size the scene "+
			"draws", size.X, size.Y, planned.X, planned.Y)
	}
	opened, err := run.shop.Reel(run.ctx, *going.settings)
	if err != nil {
		return err
	}
	going.reel = opened
	if going.settings.Padding() == nothing {
		return nil
	}
	return going.said(fmt.Sprintf("padded %d rows to %d for the film",
		size.Y, going.settings.Coded().Y))
}
