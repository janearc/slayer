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
	height, err := codedHeight(request.Width, request.Height, request.Pad)
	if err != nil {
		return Settings{}, err
	}
	if err := checkTiming(request.Rate, request.Frames); err != nil {
		return Settings{}, err
	}
	coded := image.Pt(request.Width, height)
	asked := newDemand(coded, request.Rate)
	chosen, err := levelFor(asked)
	if err != nil {
		return Settings{}, err
	}
	picture := image.Pt(request.Width, request.Height)
	return Settings{path: path, picture: picture, coded: coded,
		rate: request.Rate, frames: request.Frames, sound: request.Sound,
		replace: request.Replace, level: *chosen,
		bitRate: asked.bitRate}, nil
}

// checkTiming refuses a rate that is not a whole number of frames a
// second within the most offered, and a count of frames that is none, or
// more than a duration can count.
func checkTiming(rate, frames int) error {
	if !positive(rate) || rate > mostRate {
		return fmt.Errorf("movie: a rate of %d frames a "+
			"second: give a whole number from 1 to %d, such as 24",
			rate, mostRate)
	}
	if !positive(frames) {
		return fmt.Errorf("movie: a movie of %d frames: "+
			"give how many frames it holds, one or more", frames)
	}
	if int64(frames) > mostFrames {
		return fmt.Errorf("movie: a movie of %d frames is longer than a "+
			"duration can count: give at most %d", frames, mostFrames)
	}
	return nil
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
	if err := vacant(absolute, replace); err != nil {
		return "", err
	}
	return absolute, nil
}

// vacant refuses a path with something at it, unless the caller asked to
// replace it and it is not a directory, which a movie cannot replace.
func vacant(absolute string, replace bool) error {
	existing, err := os.Lstat(absolute)
	if err != nil {
		return nil
	}
	if !replace {
		return fmt.Errorf("movie: %s already exists: set Replace "+
			"to write over it", absolute)
	}
	if existing.IsDir() {
		return fmt.Errorf("movie: %s is a directory: a movie cannot "+
			"replace one", absolute)
	}
	return nil
}

// codedHeight is the height a picture is coded at: the width even, no
// side beyond what any level holds, and the height a multiple of 16,
// padded at the foot only when asked.
func codedHeight(width, height int, pad bool) (int, error) {
	if !positive(width) || !positive(height) {
		return zero, fmt.Errorf("movie: a picture %dx%d: both "+
			"sides must be above nought", width, height)
	}
	if width > mostPictureSide || height > mostPictureSide {
		return zero, fmt.Errorf("movie: a picture %dx%d: a side is over "+
			"%d pixels, which no H.264 level holds", width, height,
			mostPictureSide)
	}
	if width%pair != zero {
		return zero, fmt.Errorf("movie: a width of %d is odd: "+
			"H.264 halves the colour across, so give an even width",
			width)
	}
	coded := int(macroblocks(height)) * macroblock
	if coded != height && !pad {
		return zero, fmt.Errorf("movie: a height of %d is not "+
			"a multiple of %d, and QuickTime will not play it: set Pad "+
			"to add %d black rows at the foot", height, macroblock,
			coded-height)
	}
	return coded, nil
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

// Duration is how long the movie plays; nought for settings that
// NewSettings did not make, which have no rate to divide by.
func (settings Settings) Duration() time.Duration {
	if !positive(settings.rate) {
		return zero
	}
	return time.Duration(settings.frames) * time.Second /
		time.Duration(settings.rate)
}

// Level is the H.264 level the picture is coded at, written as 4.1.
func (settings Settings) Level() string { return settings.level.name }

// Sound is the soundtrack; the zero Sound when there is none.
func (settings Settings) Sound() Sound { return settings.sound }
