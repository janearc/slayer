package movie

import (
	"errors"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// extension is the only name ending a movie is written under. Defined:
// the container is MPEG-4 part 14.
const extension = ".mp4"

// Request is what a caller asks for. NewSettings checks it.
//
//ninja:vow practice NewSettings takes one
type Request struct {
	Path    string //ninja:vow gate a request, whose fields are what is asked
	Width   int    // pixels across each frame
	Height  int    // pixels down each frame, before any padding
	Rate    int    // frames a second
	Frames  int    // how many frames the movie holds
	Sound   Sound  // the soundtrack; the zero Sound is none
	Pad     bool   // add black rows at the foot to a multiple of 16
	Replace bool   // replace a file already at Path
}

// Settings is a checked Request: everything New needs, and the H.264
// level the picture will be coded at.
//
//ninja:vow practice NewSettings makes one and New takes it
type Settings struct {
	path    string
	picture image.Point
	coded   image.Point
	rate    int
	frames  int
	sound   Sound
	replace bool
	level   level
	bitRate int64
}

// NewSettings checks a request, and refuses by name anything the file
// could not hold or QuickTime would not play.
func NewSettings(request Request) (Settings, error) {
	path, err := checkPath(request.Path, request.Replace)
	if err != nil {
		return Settings{}, err
	}
	coded, err := checkSize(request.Width, request.Height, request.Pad)
	if err != nil {
		return Settings{}, err
	}
	if request.Rate <= 0 || request.Rate > mostRate {
		return Settings{}, fmt.Errorf("movie: a rate of %d frames a "+
			"second: give a whole number from 1 to %d, such as 24",
			request.Rate, mostRate)
	}
	if request.Frames <= 0 {
		return Settings{}, fmt.Errorf("movie: a movie of %d frames: "+
			"give how many frames it holds, one or more", request.Frames)
	}
	asked := newDemand(coded, request.Rate)
	chosen, err := levelFor(asked)
	if err != nil {
		return Settings{}, err
	}
	picture := image.Pt(request.Width, request.Height)
	return Settings{path: path, picture: picture, coded: coded,
		rate: request.Rate, frames: request.Frames, sound: request.Sound,
		replace: request.Replace, level: chosen,
		bitRate: asked.bitRate}, nil
}

// checkPath is the absolute path a movie is written to: one ending
// .mp4, in a directory that exists, with nothing there unless the
// caller asked to replace it.
func checkPath(path string, replace bool) (string, error) {
	if path == "" {
		return "", errors.New("movie: no path: say where the movie goes")
	}
	if !strings.EqualFold(filepath.Ext(path), extension) {
		return "", fmt.Errorf("movie: %s does not end %s, and an mp4 is "+
			"what is written", path, extension)
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("movie: %s: %w", path, err)
	}
	folder := filepath.Dir(absolute)
	info, err := os.Stat(folder)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("movie: the output directory %s does not "+
			"exist: make it first", folder)
	}
	if _, err := os.Lstat(absolute); err == nil && !replace {
		return "", fmt.Errorf("movie: %s already exists: set Replace "+
			"to write over it", absolute)
	}
	return absolute, nil
}

// checkSize is the coded size of a picture: the width even, and the
// height a multiple of 16, padded at the foot only when asked.
func checkSize(width, height int, pad bool) (image.Point, error) {
	if width <= 0 || height <= 0 {
		return image.Point{}, fmt.Errorf("movie: a picture %dx%d: both "+
			"sides must be above nought", width, height)
	}
	if width%pair != 0 {
		return image.Point{}, fmt.Errorf("movie: a width of %d is odd: "+
			"H.264 halves the colour across, so give an even width",
			width)
	}
	coded := int(macroblocks(height)) * macroblock
	if coded != height && !pad {
		return image.Point{}, fmt.Errorf("movie: a height of %d is not "+
			"a multiple of %d, and QuickTime will not play it: set Pad "+
			"to add %d black rows at the foot", height, macroblock,
			coded-height)
	}
	return image.Pt(width, coded), nil
}

// Path is the absolute path the movie is published at.
func (settings Settings) Path() string { return settings.path }

// Picture is the size of each frame handed to Frame.
func (settings Settings) Picture() image.Point { return settings.picture }

// Coded is the size written to the file: the picture and any padding.
func (settings Settings) Coded() image.Point { return settings.coded }

// Padding is how many black rows are added at the foot.
func (settings Settings) Padding() int {
	return settings.coded.Y - settings.picture.Y
}

// Rate is the frames a second.
func (settings Settings) Rate() int { return settings.rate }

// Frames is how many frames the movie holds.
func (settings Settings) Frames() int { return settings.frames }

// Duration is how long the movie plays.
func (settings Settings) Duration() time.Duration {
	return time.Duration(settings.frames) * time.Second /
		time.Duration(settings.rate)
}

// Level is the H.264 level the picture is coded at, written as 4.1.
func (settings Settings) Level() string { return settings.level.name }

// Sound is the soundtrack; the zero Sound when there is none.
func (settings Settings) Sound() Sound { return settings.sound }
