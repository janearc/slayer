package finish

import (
	"bufio"
	"fmt"
	"image"
	"image/color"
	_ "image/png" // a plate may be a png
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/janearc/slayer/raw"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/render/frame"
)

// the kinds of plate a directory may hold, by extension.
const (
	pfmKind = ".pfm"
	pngKind = ".png"
)

// pfmColour is the mark a colour pfm's first line holds, as package raw
// writes and reads it.
const pfmColour = "PF"

// Plates are frames already rendered and kept in a directory, in name
// order: the frames of a job that traces nothing. A .pfm plate is linear
// light. A .png plate is decoded and each code value turned back to
// linear light by render/colour's Decode; its light was clipped at white
// when it was written, so no exposure brings back what it lost.
//
// The plates are all one kind and all one size. They are not read until
// asked for, one at a time, so a long film is not held in memory.
type Plates struct {
	paths         []string
	kind          string
	width, height int
}

// OpenPlates is the plates of a directory. A directory with none, with
// both kinds, or with plates of two sizes is refused, the odd file by
// name; so is a file whose header cannot be read. Files of other kinds
// are left alone.
func OpenPlates(directory string) (Plates, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return Plates{}, fmt.Errorf("finish: the frames directory "+
			"cannot be read: %w", err)
	}
	var found Plates
	for _, entry := range entries {
		kind := strings.ToLower(filepath.Ext(entry.Name()))
		if entry.IsDir() || kind != pfmKind && kind != pngKind {
			continue
		}
		if err := found.add(filepath.Join(directory, entry.Name()),
			kind); err != nil {
			return Plates{}, err
		}
	}
	if len(found.paths) == none {
		return Plates{}, fmt.Errorf("finish: %s holds no .pfm or .png "+
			"frames; point the job at the directory that does",
			directory)
	}
	return found, nil
}

// add takes a file as the next plate, if it is of the kind and size of
// the ones before it.
func (made *Plates) add(path, kind string) error {
	width, height, err := sizeOf(path, kind)
	if err != nil {
		return err
	}
	if len(made.paths) == none {
		made.kind, made.width, made.height = kind, width, height
	}
	if kind != made.kind {
		return fmt.Errorf("finish: %s is a %s among %s frames; keep one "+
			"kind of frame in a directory", path, kind, made.kind)
	}
	if width != made.width || height != made.height {
		return fmt.Errorf("finish: %s is %dx%d among frames %dx%d; "+
			"render every frame at one size", path, width, height,
			made.width, made.height)
	}
	made.paths = append(made.paths, path)
	return nil
}

// sizeOf is the width and height of a plate, from its header alone.
func sizeOf(path, kind string) (int, int, error) {
	file, err := os.Open(path)
	if err != nil {
		return none, none, fmt.Errorf("finish: %w", err)
	}
	defer file.Close()
	reading := bufio.NewReader(file)
	if kind == pngKind {
		config, _, err := image.DecodeConfig(reading)
		if err != nil {
			return none, none, fmt.Errorf("finish: %s is not a png: %w",
				path, err)
		}
		return config.Width, config.Height, nil
	}
	var mark string
	var width, height int
	if _, err := fmt.Fscanln(reading, &mark); err != nil ||
		mark != pfmColour {
		return none, none, fmt.Errorf("finish: %s is not a colour pfm",
			path)
	}
	if _, err := fmt.Fscanln(reading, &width, &height); err != nil ||
		width < one || height < one {
		return none, none, fmt.Errorf("finish: %s has no size in its "+
			"pfm header", path)
	}
	return width, height, nil
}

// Len is how many plates there are.
func (made Plates) Len() int { return len(made.paths) }

// Size is the width and height of every plate, in pixels.
func (made Plates) Size() (width, height int) {
	return made.width, made.height
}

// Frame is the light of the plate at a place in name order, counted from
// nought. A place there is no plate at is refused.
func (made Plates) Frame(place int) (*frame.Frame, error) {
	if place < none || place >= len(made.paths) {
		return nil, fmt.Errorf("finish: there is no frame %d among %d; "+
			"ask for 0 to %d", place, len(made.paths), len(made.paths)-one)
	}
	path := made.paths[place]
	if made.kind == pngKind {
		return decoded(path)
	}
	return raw.Read(path)
}

// decoded is a png plate as linear light, each channel's code value
// through render/colour's Decode. Alpha is not light and is dropped.
func decoded(path string) (*frame.Frame, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("finish: %w", err)
	}
	defer file.Close()
	picture, _, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("finish: %s is not a png: %w", path, err)
	}
	size := picture.Bounds()
	made := frame.New(size.Dx(), size.Dy())
	for y := range size.Dy() {
		for x := range size.Dx() {
			code := color.NRGBAModel.Convert(picture.At(size.Min.X+x,
				size.Min.Y+y)).(color.NRGBA)
			pixel := made.At(x, y)
			pixel.R = float32(colour.Decode(code.R))
			pixel.G = float32(colour.Decode(code.G))
			pixel.B = float32(colour.Decode(code.B))
			made.Set(x, y, pixel)
		}
	}
	return made, nil
}

// Names are the plates' paths, in the order Frame takes them.
func (made Plates) Names() []string { return slices.Clone(made.paths) }
