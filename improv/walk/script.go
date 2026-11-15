// Package walk makes short vertical films that walk a stranger through a
// set: a traced picture with labels pointing at real places, and a panel
// naming what is looked at, its numbers, and a sentence. The README
// beside this file is the design.
package walk

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"unicode/utf8"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/set"
	yaml "gopkg.in/yaml.v3"
)

// written is a script as it is on disk.
type written struct {
	Set    string         `yaml:"set"`
	Host   string         `yaml:"host"`
	Places yaml.Node      `yaml:"places"`
	Shorts []writtenShort `yaml:"shorts"`
}

// writtenScene is a scene's places as they are on disk: their unit,
// and each a name and its x, y and z, y up.
type writtenScene struct {
	Unit string               `yaml:"unit"`
	At   map[string][]float64 `yaml:"at"`
}

// writtenEye is a scene's camera as it is on disk: where it stands,
// what it looks at, which way is up, in the scene's units, and the
// angle a pinhole sees across the picture, in degrees.
type writtenEye struct {
	From   []float64 `yaml:"from"`
	LookAt []float64 `yaml:"look-at"`
	Up     []float64 `yaml:"up"`
	Wide   float64   `yaml:"wide"`
}

// writtenShort is a short as it is on disk.
type writtenShort struct {
	Name  string        `yaml:"name"`
	Title string        `yaml:"title"`
	Sound *writtenSound `yaml:"sound"`
	Shots []writtenShot `yaml:"shots"`
}

// writtenSound is where under a short the song starts.
type writtenSound struct {
	Bar *int `yaml:"bar"`
}

// writtenShot is a shot as it is on disk.
type writtenShot struct {
	Set      string          `yaml:"set"`
	Camera   string          `yaml:"camera"`
	Eye      *writtenEye     `yaml:"eye"`
	Picture  string          `yaml:"picture"`
	Seconds  float64         `yaml:"seconds"`
	Lit      string          `yaml:"lit"`
	Tag      string          `yaml:"tag"`
	Window   *writtenWindow  `yaml:"window"`
	Name     string          `yaml:"name"`
	Figures  []writtenFigure `yaml:"figures"`
	Labels   []writtenLabel  `yaml:"labels"`
	Scale    *writtenScale   `yaml:"scale"`
	Sentence string          `yaml:"sentence"`
}

// writtenWindow is a window's travel over its still, each end its left,
// top and width as shares of the still.
type writtenWindow struct {
	From []float64 `yaml:"from"`
	To   []float64 `yaml:"to"`
}

// writtenFigure is a number on the panel as it is on disk.
type writtenFigure struct {
	Value string `yaml:"value"`
	Unit  string `yaml:"unit"`
	Says  string `yaml:"says"`
	From  string `yaml:"from"`
}

// writtenLabel is a label as it is on disk: one of a thing, a place or
// a way in the sky, the words, and the side the words stand.
type writtenLabel struct {
	Thing  string    `yaml:"thing"`
	At     []float64 `yaml:"at"`
	Toward []float64 `yaml:"toward"`
	Pixel  []float64 `yaml:"pixel"`
	Words  string    `yaml:"words"`
	Side   string    `yaml:"side"`
}

// writtenScale is a scale bar as it is on disk: at a thing or a place
// of a set, measured through its camera; or given outright, so many
// pixels of the still standing for a length in a unit.
type writtenScale struct {
	Thing  string    `yaml:"thing"`
	At     []float64 `yaml:"at"`
	Toward []float64 `yaml:"toward"`
	Pixels float64   `yaml:"pixels"`
	Length string    `yaml:"length"`
	Unit   string    `yaml:"unit"`
}

// Script is a series of shorts read and checked against its set.
type Script struct {
	shorts []Short
}

// Shorts is the script's shorts, in the order it gives them.
func (script Script) Shorts() []Short {
	return append([]Short(nil), script.shorts...)
}

// Short is the short of a name, and whether the script has one.
func (script Script) Short(name string) (Short, bool) {
	for _, each := range script.shorts {
		if each.name == name {
			return each, true
		}
	}
	return Short{}, false
}

// stage is what a script is checked against and drawn from: its set,
// where the set's things stand, and the script's own directory.
type stage struct {
	read     set.Set
	places   Places
	pictures string
	others   map[string]set.Set
	// scene is a script with no set: its places, if any, are given in
	// it, in its unit, x, y and z with y up, and its shots' cameras are
	// eyes or nothing
	scene bool
	unit  string
	// host is the frames of the host's looping clip, in order, or none
	host []string
}

// ReadScript reads a script and checks it against the set and places it
// names, each relative to it, and the shots' pictures and own sets,
// relative to a folder of pictures (the script's own folder when none
// is given). Whatever is wrong is refused by name: the short, the shot,
// and what to change.
func ReadScript(path, pictures string) (Script, error) {
	source, err := os.ReadFile(path)
	if err != nil {
		return Script{}, fmt.Errorf("%w: %w", ErrWalk, err)
	}
	var said written
	if err := yaml.Unmarshal(source, &said); err != nil {
		return Script{}, fmt.Errorf("%w: %s: %w", ErrWalk, path, err)
	}
	where, err := stageOf(said, filepath.Dir(path))
	if err != nil {
		return Script{}, fmt.Errorf("%s: %w", path, err)
	}
	if pictures != "" {
		where.pictures = pictures
	}
	if said.Host != "" {
		frames, err := hostFrames(filepath.Join(where.pictures, said.Host))
		if err != nil {
			return Script{}, fmt.Errorf("%s: %w", path, err)
		}
		where.host = frames
	}
	if len(said.Shorts) == 0 {
		return Script{}, fmt.Errorf("%w: %s has no shorts; give it one",
			ErrWalk, path)
	}
	var script Script
	for _, each := range said.Shorts {
		short, err := where.short(each)
		if err != nil {
			return Script{}, fmt.Errorf("%s: %w", path, err)
		}
		if _, twice := script.Short(short.name); twice {
			return Script{}, fmt.Errorf("%w: %s: the short %q is given "+
				"twice", ErrWalk, path, short.name)
		}
		script.shorts = append(script.shorts, short)
	}
	return script, nil
}

// stageOf reads the set and places a script names.
func stageOf(said written, dir string) (stage, error) {
	if said.Set == "" {
		return sceneOf(said.Places, dir)
	}
	var tablePath string
	if said.Places.Kind != yaml.ScalarNode ||
		said.Places.Decode(&tablePath) != nil || tablePath == "" {
		return stage{}, fmt.Errorf("%w: a script with a set names its "+
			"places table, relative to it", ErrWalk)
	}
	read, err := readSet(filepath.Join(dir, said.Set))
	if err != nil {
		return stage{}, err
	}
	table, err := os.ReadFile(filepath.Join(dir, tablePath))
	if err != nil {
		return stage{}, fmt.Errorf("%w: %w", ErrWalk, err)
	}
	places, err := ReadPlaces(table)
	if err != nil {
		return stage{}, err
	}
	return stage{read: read, places: places, pictures: dir,
		others: map[string]set.Set{}, unit: metres}, nil
}

// sceneOf is the stage of a script with no set: its places given in it,
// in a unit, or none.
func sceneOf(given yaml.Node, dir string) (stage, error) {
	made := stage{pictures: dir, scene: true,
		places: Places{named: map[string]Place{}}}
	if given.Kind == 0 {
		return made, nil
	}
	var said writtenScene
	if given.Kind != yaml.MappingNode || given.Decode(&said) != nil ||
		said.Unit == "" {
		return stage{}, fmt.Errorf("%w: a script with no set gives its "+
			"places as {unit, at: {name: [x, y, z]}}", ErrWalk)
	}
	made.unit = said.Unit
	for name, numbers := range said.At {
		if len(numbers) != placeShares {
			return stage{}, fmt.Errorf("%w: the place %q is %v; give x, y "+
				"and z", ErrWalk, name, numbers)
		}
		at, err := made.placeOf(numbers)
		if err != nil {
			return stage{}, err
		}
		made.places.named[name] = at
	}
	return made, nil
}

// readSet reads a set file.
func readSet(path string) (set.Set, error) {
	source, err := os.ReadFile(path)
	if err != nil {
		return set.Set{}, fmt.Errorf("%w: %w", ErrWalk, err)
	}
	read, err := set.Read(path, source)
	if err != nil {
		return set.Set{}, fmt.Errorf("%w: %w", ErrWalk, err)
	}
	return read, nil
}

// setFor is the set a shot was drawn from: its own, relative to the
// script, or else the script's. A shot's own set is read once.
func (where stage) setFor(path string) (set.Set, error) {
	if path == "" {
		return where.read, nil
	}
	if read, found := where.others[path]; found {
		return read, nil
	}
	read, err := readSet(filepath.Join(where.pictures, path))
	if err == nil {
		where.others[path] = read
	}
	return read, err
}

// short checks one short.
func (where stage) short(said writtenShort) (Short, error) {
	if said.Name == "" || said.Title == "" || len(said.Shots) == 0 {
		return Short{}, fmt.Errorf("%w: a short needs a name, a title and "+
			"a shot; %q has not", ErrWalk, said.Name)
	}
	made := Short{name: said.Name, title: said.Title, host: where.host}
	if said.Sound != nil {
		if said.Sound.Bar == nil {
			return Short{}, fmt.Errorf("%w: short %s: the sound names no "+
				"bar; give the bar it starts at", ErrWalk, said.Name)
		}
		made.bar, made.sounded = *said.Sound.Bar, true
	}
	for i, each := range said.Shots {
		shot, err := where.shot(each)
		if err != nil {
			return Short{}, fmt.Errorf("short %s, shot %d: %w", said.Name,
				i+1, err)
		}
		made.shots = append(made.shots, shot)
	}
	return made, nil
}

// lighting is how a shot may be lit, and the tag the picture carries
// for it: as in the show, none; the set's own light with the show's
// lights off; a plain light for inspection; or a computed picture of
// something no camera has taken.
var lighting = map[string]string{"show": "",
	"plate":      "The set's own light only, the show's lights off",
	"inspection": "Lit for inspection, not as in the show",
	"drawing":    "A drawing, not a photograph"}

// tagMost is the most characters a tag in the author's own words may
// have. Picked here: it fits across the picture on one row at tagSize.
const tagMost = 48

// tagOf is the tag a shot's picture carries: from how it was lit, or in
// the author's own words; one of the two, never both.
func tagOf(lit, own string) (string, error) {
	if own != "" {
		if lit != "" || utf8.RuneCountInString(own) > tagMost {
			return "", fmt.Errorf("%w: the tag %q beside lit %q; give "+
				"one of the two, a tag of at most %d characters", ErrWalk,
				own, lit, tagMost)
		}
		return own, nil
	}
	tag, known := lighting[lit]
	if !known {
		return "", fmt.Errorf("%w: lit %q; say show, plate, inspection "+
			"or drawing, or give a tag in your own words", ErrWalk, lit)
	}
	return tag, nil
}

// shot checks one shot: its words first, then its picture and camera,
// then its labels against them.
func (where stage) shot(said writtenShot) (Shot, error) {
	tag, err := tagOf(said.Lit, said.Tag)
	if err != nil {
		return Shot{}, err
	}
	if err := sentenceFits(said.Sentence, said.Seconds); err != nil {
		return Shot{}, err
	}
	if err := panelFits(said.Sentence, said.Figures,
		len(where.host) > 0); err != nil {
		return Shot{}, err
	}
	figures, err := figuresOf(said.Figures)
	if err != nil {
		return Shot{}, err
	}
	still, view, err := where.pictured(said)
	if err != nil {
		return Shot{}, err
	}
	travel, err := windowOf(said.Window, still)
	if err != nil {
		return Shot{}, err
	}
	made := Shot{camera: said.Camera, picture: still, view: view,
		seconds: said.Seconds, lit: said.Lit, tag: tag, travel: travel,
		name: said.Name, figures: figures, sentence: said.Sentence}
	for _, each := range said.Labels {
		label, err := where.label(each, view, still.size)
		if err != nil {
			return Shot{}, err
		}
		made.labels = append(made.labels, label)
	}
	if said.Scale != nil {
		if err := where.scaled(&made, *said.Scale); err != nil {
			return Shot{}, err
		}
	}
	return made, nil
}

// pictured is a shot's still and the camera it was drawn through: a
// set's camera, by name; in a script with no set, an eye given in the
// shot, or no camera at all.
func (where stage) pictured(said writtenShot) (still, *Camera, error) {
	if !where.scene {
		if said.Eye != nil {
			return still{}, nil, fmt.Errorf("%w: a shot of a set with an "+
				"eye; name one of the set's cameras", ErrWalk)
		}
		read, err := where.setFor(said.Set)
		if err != nil {
			return still{}, nil, err
		}
		picture, view, err := where.picture(read, said.Picture, said.Camera)
		return picture, &view, err
	}
	if said.Camera != "" || said.Set != "" {
		return still{}, nil, fmt.Errorf("%w: the shot names camera %q and "+
			"set %q, and the script names no set; give the shot an eye, "+
			"or the script a set and places", ErrWalk, said.Camera,
			said.Set)
	}
	picture, err := where.stillOf(said.Picture)
	if err != nil || said.Eye == nil {
		return picture, nil, err
	}
	view, err := where.eyed(*said.Eye, picture.size)
	return picture, &view, err
}

// eyed is a scene's camera from the eye a shot gives.
func (where stage) eyed(said writtenEye, size image.Point) (Camera, error) {
	var points [3]maths.Vec
	for i, numbers := range [][]float64{said.From, said.LookAt, said.Up} {
		if len(numbers) != placeShares {
			return Camera{}, fmt.Errorf("%w: an eye's from, look-at and up "+
				"are each x, y and z; one is %v", ErrWalk, numbers)
		}
		points[i] = maths.V(numbers[0], numbers[1], numbers[2])
	}
	eye, err := NewEye(points[0], points[1], points[2])
	if err != nil {
		return Camera{}, err
	}
	if where.unit == "" {
		return Camera{}, fmt.Errorf("%w: a shot with an eye in a script "+
			"whose places say no unit; give the script places with a "+
			"unit", ErrWalk)
	}
	return PinholeCamera(eye, said.Wide, size, where.unit)
}

// sentenceFits refuses a shot with no time, or a sentence a stranger
// cannot read in it at readingRate.
func sentenceFits(sentence string, seconds float64) error {
	if !(seconds > 0) {
		return fmt.Errorf("%w: a shot of %v seconds; give it time",
			ErrWalk, seconds)
	}
	needs := float64(utf8.RuneCountInString(sentence)) / readingRate
	if needs > seconds {
		return fmt.Errorf("%w: %q takes %.1f s to read at %g characters "+
			"a second, and the shot is %g s; shorten it or lengthen the "+
			"shot", ErrWalk, sentence, needs, readingRate, seconds)
	}
	return nil
}

// mostFigures is how many figures a panel has room for. Picked here.
const mostFigures = 3

// figuresOf checks a shot's figures: each has a value, a unit or words,
// and where it comes from.
func figuresOf(said []writtenFigure) ([]Figure, error) {
	if len(said) > mostFigures {
		return nil, fmt.Errorf("%w: %d figures; the panel has room for %d",
			ErrWalk, len(said), mostFigures)
	}
	var made []Figure
	for _, each := range said {
		if each.Value == "" || each.From == "" {
			return nil, fmt.Errorf("%w: the figure %q %s says no value or "+
				"no from; every number says where it comes from", ErrWalk,
				each.Value, each.Unit)
		}
		made = append(made, Figure{value: each.Value, unit: each.Unit,
			says: each.Says, from: each.From})
	}
	return made, nil
}

// panelFits refuses a sentence longer than the panel's rows, or a
// figure wider than its column, at the panel's sizes.
func panelFits(sentence string, figures []writtenFigure, hosted bool) error {
	rows, err := wrapped(sentence, sentenceSize, sentenceWidth(hosted))
	if err != nil {
		return err
	}
	if len(rows) > sentenceRows {
		return fmt.Errorf("%w: %q takes %d rows; the panel has room for "+
			"%d: shorten it", ErrWalk, sentence, len(rows), sentenceRows)
	}
	column := (frameSize.X - 2*margin) / max(len(figures), 1)
	for _, each := range figures {
		number, err := measure(each.Value, figureSize)
		if err != nil {
			return err
		}
		unit, err := measure(each.Unit, unitSize)
		if err != nil {
			return err
		}
		says, err := measure(each.Says, saysSize)
		if err != nil {
			return err
		}
		if max(number+unitGap+unit, says) > column-unitGap {
			return fmt.Errorf("%w: the figure %q %s, %q, is wider than "+
				"its column of %d pixels; shorten it or give fewer",
				ErrWalk, each.Value, each.Unit, each.Says, column)
		}
	}
	return nil
}

// sentenceWidth is how wide the sentence may run: the panel less its
// margins, and less the host's corner and a margin when there is a host.
func sentenceWidth(hosted bool) int {
	if hosted {
		return frameSize.X - 3*margin - hostBox.Dx()
	}
	return frameSize.X - 2*margin
}

// hostFrames is the PNG frames of a host's clip in a folder, in name
// order. A folder with none is refused.
func hostFrames(dir string) ([]string, error) {
	frames, err := filepath.Glob(filepath.Join(dir, "*.png"))
	if err != nil || len(frames) == 0 {
		return nil, fmt.Errorf("%w: the host %s has no png frames; give a "+
			"folder of its clip's frames", ErrWalk, dir)
	}
	return frames, nil
}
