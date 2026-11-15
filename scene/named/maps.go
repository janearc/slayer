package named

// The second kind of named scene: one drawn as maps.
//
// Design. A traced scene builds a world and an eye, and a renderer makes
// its one picture a frame. A scene of maps has neither: it is carried to
// a moment and makes its own pictures there, several a moment, each with
// the file name its command gives it. The weather is one: its vorticity,
// its bands and its air's age, flat in latitude and longitude.
//
// Both kinds say their name, what they are and what they take, and check
// a with block the same way; that is Listed, and All lists both. Get
// finds a traced scene and GetMaps a scene of maps; each refuses the
// other kind's name by saying which kind it is.

import (
	"errors"
	"fmt"
	"image"
	"io"
	"slices"
	"time"
)

// Listed is what every named scene says of itself, of either kind: its
// name, what it is, what it takes, and whether a with block would build.
type Listed interface {
	Name() string
	About() string
	Parameters() []Parameter
	Check(with map[string]any) error
}

// Maps is a named scene drawn as maps: carried to a moment, it makes
// several pictures there, each named, with no world and no eye.
type Maps struct {
	name, about string
	renderers   []Renderer
	pictures    []string
	parameters  []Parameter
	build       mapBuilder
	cache       func() (string, error)
}

// mapBuilder makes a scene of maps from checked values and what the
// caller asks of it.
type mapBuilder func(held values, ask MapsAsk) (Mapper, error)

// Mapper is a scene of maps built: its pictures at a moment of scene
// time, in the order the scene lists them; which of them a film shows;
// and the renderer that carries it.
type Mapper interface {
	At(moment time.Duration) ([]Map, error)
	Shows() string
	Renderer() Renderer
}

// Map is one picture a scene of maps makes at a moment: which of its
// pictures it is, its file name with no extension, and the picture.
type Map struct {
	kind, name string
	picture    *image.RGBA
}

// Kind is which of the scene's pictures this is, as Pictures names it.
func (made Map) Kind() string { return made.kind }

// Name is the picture's file name, with no extension, as the scene's
// command names it.
func (made Map) Name() string { return made.name }

// Picture is the map.
func (made Map) Picture() *image.RGBA { return made.picture }

// MapsAsk is what a caller asks of a scene of maps besides its values:
// the renderer, and whether it was said; the cache it keeps its moments
// in, or none to carry it straight; how far apart, in scene time, it
// keeps them; and where it says what it did.
type MapsAsk struct {
	renderer Renderer
	said     bool
	cache    string
	every    time.Duration
	out      io.Writer
}

// ErrEvery is a cache asked for with no spacing between its moments.
var ErrEvery = errors.New("named: a cache needs its moments spaced " +
	"by more than nothing")

// AskMaps is what a caller asks of a scene of maps. With no renderer
// said, the scene takes its own default. A cache's moments must be
// spaced by more than nothing; with no cache, every is not read. A nil
// out says nothing.
func AskMaps(renderer Renderer, said bool, cache string,
	every time.Duration, out io.Writer) (MapsAsk, error) {
	if cache != "" && every <= nothing {
		return MapsAsk{}, fmt.Errorf("%w: every is %s", ErrEvery, every)
	}
	if out == nil {
		out = io.Discard
	}
	return MapsAsk{renderer: renderer, said: said, cache: cache,
		every: every, out: out}, nil
}

// Name is the scene's name, as a job gives it.
func (scene Maps) Name() string { return scene.name }

// About is one line saying what the scene is.
func (scene Maps) About() string { return scene.about }

// Renderers are the renderers that can carry the scene, its default
// first.
func (scene Maps) Renderers() []Renderer {
	return slices.Clone(scene.renderers)
}

// Draws is whether the renderer can carry the scene.
func (scene Maps) Draws(renderer Renderer) bool {
	return slices.Contains(scene.renderers, renderer)
}

// Cache is where the scene keeps its moments between runs, shared with
// whatever else draws it: the weather's is hillside's.
func (scene Maps) Cache() (string, error) { return scene.cache() }

// Pictures are the kinds of picture the scene makes at each moment, in
// the order it makes them.
func (scene Maps) Pictures() []string { return slices.Clone(scene.pictures) }

// Parameters are what the scene takes, in the order it lists them.
func (scene Maps) Parameters() []Parameter { return owned(scene.parameters) }

// Check is whether the values would build.
func (scene Maps) Check(with map[string]any) error {
	_, err := readValues(scene.name, scene.parameters, with)
	return err
}

// Build is the scene made from the values and what the caller asks. The
// values are checked first, and a renderer said that the scene is not
// carried by is refused by name.
func (scene Maps) Build(with map[string]any, ask MapsAsk) (Mapper, error) {
	held, err := readValues(scene.name, scene.parameters, with)
	if err != nil {
		return nil, err
	}
	if ask.said && !scene.Draws(ask.renderer) {
		return nil, fmt.Errorf("%w: the %s scene is not carried by %q; "+
			"it is carried by %v", ErrRenderer, scene.name, ask.renderer,
			scene.renderers)
	}
	if ask.out == nil {
		ask.out = io.Discard
	}
	return scene.build(held, ask)
}

// ErrRenderer is a renderer a scene is not drawn by.
var ErrRenderer = errors.New("named: not drawn by that renderer")

// GetMaps is the scene of maps of that name, or a refusal that lists the
// names there are, or says the name is a traced scene's.
func GetMaps(name string) (*Maps, error) {
	for _, scene := range maps {
		if scene.name == name {
			found := scene
			return &found, nil
		}
	}
	if _, err := Get(name); err == nil {
		return nil, fmt.Errorf("%w: %q is traced, not drawn as maps",
			ErrNoScene, name)
	}
	return nil, fmt.Errorf("%w: %q; the scenes drawn as maps are %v",
		ErrNoScene, name, mapNames())
}

// mapNames are the names of the scenes drawn as maps, in order.
func mapNames() []string {
	all := make([]string, len(maps))
	for i, scene := range maps {
		all[i] = scene.name
	}
	return all
}

// maps is every named scene drawn as maps. It is made once and never
// changed; All and GetMaps hand out copies.
var maps = []Maps{weatherScene()}
