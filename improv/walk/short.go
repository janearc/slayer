package walk

import (
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/janearc/slayer/set"
)

// Short is one film of the series: its name, its title, the bar the
// song starts at under it and whether it has a song, and its shots in
// order.
type Short struct {
	name, title string
	bar         int
	sounded     bool
	shots       []Shot
	host        []string
}

// Name is the short's name, in words, for its folder.
func (short Short) Name() string { return short.name }

// Bar is the bar of the song the sound starts at, as the set's song
// counts them, and whether the short has a song under it.
func (short Short) Bar() (int, bool) { return short.bar, short.sounded }

// Shots is the short's shots in order.
func (short Short) Shots() []Shot { return append([]Shot(nil), short.shots...) }

// Seconds is how long the short is.
func (short Short) Seconds() float64 {
	total := 0.0
	for _, each := range short.shots {
		total += each.seconds
	}
	return total
}

// Frames is how many frames the short is at filmRate.
func (short Short) Frames() int {
	return int(math.Round(short.Seconds() * filmRate))
}

// Shot is one picture of a short, and what is said over it.
type Shot struct {
	camera   string
	picture  still
	view     *Camera
	seconds  float64
	lit, tag string
	travel   [2]window
	name     string
	figures  []figure
	labels   []label
	bar      *Place
	outright *outrightBar
	sentence string
}

// Seconds is how long the shot stands.
func (shot Shot) Seconds() float64 { return shot.seconds }

// Sentence is what the shot says.
func (shot Shot) Sentence() string { return shot.sentence }

// figure is a number on the panel: as written, its unit, a few words,
// and where it comes from.
type figure struct {
	value, unit, says, from string
}

// label is words on the picture with a leader to the pixel of a place.
type label struct {
	words string
	at    Pixel
	side  image.Point
}

// still is a traced picture: its file and its size.
type still struct {
	path string
	size image.Point
}

// stillOf checks a shot's still is a PNG, and says its size.
func (where stage) stillOf(path string) (still, error) {
	if path == "" {
		return still{}, fmt.Errorf("%w: a shot with no picture; name "+
			"its still", errWalk)
	}
	full := filepath.Join(where.pictures, path)
	file, err := os.Open(full)
	if err != nil {
		return still{}, fmt.Errorf("%w: %w", errWalk, err)
	}
	defer file.Close()
	shape, err := png.DecodeConfig(file)
	if err != nil {
		return still{}, fmt.Errorf("%w: %s is not a png: %w", errWalk,
			path, err)
	}
	return still{path: full, size: image.Pt(shape.Width, shape.Height)},
		nil
}

// picture checks a shot's still against its camera: the set's camera
// through its rig, at the still's width, makes a picture its height.
func (where stage) picture(read set.Set, path, camera string) (still,
	Camera, error) {
	picture, err := where.stillOf(path)
	if err != nil {
		return still{}, Camera{}, err
	}
	view, err := CameraOf(read, camera, where.places, picture.size.X)
	if err != nil {
		return still{}, Camera{}, err
	}
	if _, height := view.size(); height != picture.size.Y {
		return still{}, Camera{}, fmt.Errorf("%w: %s is %d by %d; the "+
			"set's rig makes a picture %d across %d high; draw it again "+
			"from the set", errWalk, path, picture.size.X, picture.size.Y,
			picture.size.X, height)
	}
	return picture, view, nil
}

// window is a part of a still: its left and top in pixels, and its
// width; its height follows from the picture's shape.
type window struct {
	left, top, width float64
}

// tall is the window's height for the frame's picture.
func (part window) tall() float64 {
	return part.width * float64(pictureSize.Y) / float64(pictureSize.X)
}

// windowOf checks a shot's window travel against its still. No window
// is the widest part of the still the picture's shape fits, centred.
func windowOf(said *writtenWindow, picture still) ([2]window, error) {
	size := picture.size
	if said == nil {
		width := math.Min(float64(size.X), float64(size.Y)*
			float64(pictureSize.X)/float64(pictureSize.Y))
		whole := window{left: (float64(size.X) - width) * half,
			width: width}
		whole.top = (float64(size.Y) - whole.tall()) * half
		return [2]window{whole, whole}, nil
	}
	var ends [2]window
	for i, shares := range [][]float64{said.From, said.To} {
		if len(shares) != len(ends[0].shares()) {
			return ends, fmt.Errorf("%w: a window end of %v; give its "+
				"left, top and width as shares of the still", errWalk,
				shares)
		}
		ends[i] = window{left: shares[0] * float64(size.X),
			top: shares[1] * float64(size.Y), width: shares[2] *
				float64(size.X)}
		if !ends[i].inside(size) {
			return ends, fmt.Errorf("%w: the window %v is off its %d by "+
				"%d still; keep it inside", errWalk, shares, size.X, size.Y)
		}
	}
	return ends, nil
}

// shares is the window's three numbers, for counting them.
func (part window) shares() []float64 {
	return []float64{part.left, part.top, part.width}
}

// inside is whether the window lies on a still of a size.
func (part window) inside(size image.Point) bool {
	return part.left >= 0 && part.top >= 0 && part.width > 0 &&
		part.left+part.width <= float64(size.X)*(1+roundingSlack) &&
		part.top+part.tall() <= float64(size.Y)*(1+roundingSlack)
}

// sides is where a label's words may stand from its place, as a way on
// the picture. Up is the default.
var sides = map[string]image.Point{"": {0, -1}, "up": {0, -1},
	"down": {0, 1}, "left": {-1, 0}, "right": {1, 0}}

// label checks a label: one of a thing, a place or a way, which the
// camera sees on its still, or a pixel of the still given outright;
// and a side to stand on.
func (where stage) label(said writtenLabel, view *Camera,
	size image.Point) (label, error) {
	side, known := sides[said.Side]
	if !known || said.Words == "" {
		return label{}, fmt.Errorf("%w: the label %q on side %q; give it "+
			"words and a side of up, down, left or right", errWalk,
			said.Words, said.Side)
	}
	seen, err := where.target(said, view, size)
	if err != nil {
		return label{}, err
	}
	if !seen.ahead || !seen.on {
		return label{}, fmt.Errorf("%w: the label %q points at a place the "+
			"camera does not see; point it at one on the picture",
			errWalk, said.Words)
	}
	return label{words: said.Words, at: seen, side: side}, nil
}

// placeShares, wayShares and pixelShares are how many numbers a place, a
// way and a pixel are. Defined: three coordinates; azimuth and altitude;
// column and row.
const placeShares, wayShares, pixelShares = 3, 2, 2

// target is the pixel a label points at: its thing, place, way or
// pixel. Without a camera only a pixel can be pointed at.
func (where stage) target(said writtenLabel, view *Camera,
	size image.Point) (Pixel, error) {
	given := 0
	for _, says := range []bool{said.Thing != "", said.At != nil,
		said.Toward != nil, said.Pixel != nil} {
		if says {
			given++
		}
	}
	if given != 1 {
		return Pixel{}, fmt.Errorf("%w: the label %q points at %d things; "+
			"give one of thing, at, toward or pixel", errWalk, said.Words,
			given)
	}
	if said.Pixel != nil {
		return pixelOf(said, size)
	}
	if view == nil {
		return Pixel{}, fmt.Errorf("%w: the label %q points at a thing, a "+
			"place or a way, and the shot has no camera to find it "+
			"through; give it a pixel, or give the shot a camera or an "+
			"eye", errWalk, said.Words)
	}
	return where.projectedTarget(said, *view)
}

// pixelOf is a pixel given outright, which must lie on the still.
func pixelOf(said writtenLabel, size image.Point) (Pixel, error) {
	if len(said.Pixel) != pixelShares {
		return Pixel{}, fmt.Errorf("%w: the label %q: a pixel is a column "+
			"and a row", errWalk, said.Words)
	}
	x, y := said.Pixel[0], said.Pixel[1]
	on := x >= 0-half && y >= 0-half && x < float64(size.X)-half &&
		y < float64(size.Y)-half
	return Pixel{x: x, y: y, ahead: true, on: on}, nil
}

// projectedTarget is the pixel of a label's thing, place or way through
// a camera.
func (where stage) projectedTarget(said writtenLabel, view Camera) (Pixel,
	error) {
	switch {
	case said.Thing != "":
		at, found := where.places.of(said.Thing)
		if !found {
			return Pixel{}, fmt.Errorf("%w: the label %q points at %q, "+
				"which the places do not have", errWalk, said.Words,
				said.Thing)
		}
		return view.Project(at), nil
	case len(said.At) == placeShares:
		at, err := where.placeOf(said.At)
		return view.Project(at), err
	case len(said.Toward) == wayShares && !where.scene:
		return view.toward(said.Toward[0], said.Toward[1]), nil
	}
	return Pixel{}, fmt.Errorf("%w: the label %q: at is three numbers "+
		"(east, north and height on a set; x, y and z in a scene); toward "+
		"is azimuth and altitude, on a set only", errWalk, said.Words)
}

// placeOf is a place written in a script: east, north and height on a
// set's map; or x, y and z in a scene's own units, y up.
func (where stage) placeOf(numbers []float64) (Place, error) {
	if where.scene {
		return NewPlace(numbers[0], 0-numbers[2], numbers[1])
	}
	return NewPlace(numbers[0], numbers[1], numbers[2])
}

// outrightBar is a scale bar given outright: so many pixels of the
// still, and the length they stand for, in words.
type outrightBar struct {
	pixels float64
	words  string
}

// scaled checks a shot's scale bar: given outright, or at a thing or a
// place its camera sees, measured across the picture there.
func (where stage) scaled(shot *Shot, said writtenScale) error {
	if said.Pixels != 0 || said.Length != "" {
		if !(said.Pixels > 0) || said.Length == "" || said.Thing != "" ||
			said.At != nil || said.Toward != nil {
			return fmt.Errorf("%w: a scale bar given outright needs pixels "+
				"over nought and a length, and nothing else", errWalk)
		}
		shot.outright = &outrightBar{pixels: said.Pixels,
			words: strings.TrimSpace(said.Length + " " + said.Unit)}
		return nil
	}
	if said.Toward != nil || shot.view == nil {
		return fmt.Errorf("%w: a scale bar toward the sky, or measured "+
			"with no camera; put it at a thing or a place a camera sees, "+
			"or give its pixels and length outright", errWalk)
	}
	label := writtenLabel{Thing: said.Thing, At: said.At,
		Words: "the scale bar"}
	seen, err := where.target(label, shot.view, shot.picture.size)
	if err != nil {
		return err
	}
	if !seen.on {
		return fmt.Errorf("%w: the scale bar stands where the camera does "+
			"not see; put it on the picture", errWalk)
	}
	at, _ := where.places.of(said.Thing)
	if said.At != nil {
		at, _ = where.placeOf(said.At)
	}
	if _, err := shot.view.PerMetre(at); err != nil {
		return err
	}
	shot.bar = &at
	return nil
}
