// Package named holds scenes built in Go, by name: the lantern's swarm, the
// orbital, the ring and moonrise, which are traced, and the weather, which
// is drawn as maps (maps.go says how the two kinds sit side by side). Each
// was built inside its command from flags; here a job can ask for one by
// name with its parameters, and the command asks the same way, so both
// draw the same picture.
//
// This package is a bridge, agreed as one: each named scene is owed a set
// file, and named goes when the last of them has one.
package named

import (
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

	"github.com/janearc/slayer/lantern"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/scene/thing"
)

// Renderer is a way of drawing a scene, by the name a job gives it.
type Renderer string

// the renderers a named scene may be drawn by
const (
	// CPU is slayer's tracer, trace.Sampled.
	CPU Renderer = "cpu"
	// GPU is the Metal kernel, gpu.Sampled.
	GPU Renderer = "gpu"
)

// ErrNoScene is a name no scene has.
var ErrNoScene = errors.New("named: no such scene")

// Scene is one named scene: what it is, what it takes, what it is drawn
// at by default, and how it is built.
type Scene struct {
	authored          *Authored
	fastLighting      bool
	name, about, unit string
	renderers         []Renderer
	parameters        []Parameter
	defaults          Defaults
	build             builder
}

// builder makes a scene's world from checked values, saying what it built
// to said.
type builder func(held values, said io.Writer) (*Built, error)

// Name is the scene's name, as a job gives it.
func (scene Scene) Name() string { return scene.name }

// About is one line saying what the scene is.
func (scene Scene) About() string { return scene.about }

// Unit is what one unit of the scene's lengths is.
func (scene Scene) Unit() string { return scene.unit }

// Renderers are the renderers that can draw the scene.
func (scene Scene) Renderers() []Renderer {
	return slices.Clone(scene.renderers)
}

// Draws is whether the renderer can draw the scene.
func (scene Scene) Draws(renderer Renderer) bool {
	return slices.Contains(scene.renderers, renderer)
}

// Parameters are what the scene takes, in the order it lists them.
func (scene Scene) Parameters() []Parameter { return owned(scene.parameters) }

// Defaults are the film, lens, time and tone the scene is drawn at when
// nothing else is asked.
func (scene Scene) Defaults() Defaults { return scene.defaults }

// Check is whether the values would build: every key one of the scene's
// parameters, every value of its kind and in its bounds.
func (scene Scene) Check(with map[string]any) error {
	if scene.authored != nil {
		return scene.authored.Check(with)
	}
	_, err := scene.read(with)
	return err
}

// Build is the scene made from the values, saying what it built to said.
// The values are checked first, and the first that is wrong is refused by
// name; a parameter left out takes its default.
func (scene Scene) Build(with map[string]any, said io.Writer) (*Built,
	error) {
	if scene.authored != nil {
		if err := scene.authored.Check(with); err != nil {
			return nil, err
		}
		if said == nil {
			said = io.Discard
		}
		world, eye, advance, err := scene.authored.Build(with, said)
		if err != nil {
			return nil, err
		}
		if world == nil {
			return nil, fmt.Errorf("authored scene %q returned no world", scene.name)
		}
		var lens func(time.Duration) float64
		if scene.authored.Lens != nil {
			lens, err = scene.authored.Lens(with)
			if err != nil {
				return nil, err
			}
		}
		return &Built{world: world, eye: eye, advance: advance, wide: lens}, nil
	}
	held, err := scene.read(with)
	if err != nil {

		return nil, err
	}
	if said == nil {
		said = io.Discard
	}
	return scene.build(held, said)
}

// Defaults are what a scene is drawn at when nothing else is asked, as its
// command's flags default to them.
type Defaults struct {
	width, height, rays int
	wide                float64
	at                  time.Duration
	tone                colour.Tone
}

// Width is the picture's width in pixels.
func (made Defaults) Width() int { return made.width }

// Height is the picture's height in pixels.
func (made Defaults) Height() int { return made.height }

// Rays is how many rays go through each pixel along each side.
func (made Defaults) Rays() int { return made.rays }

// Wide is how much the lens sees from its left edge to its right, in
// radians.
func (made Defaults) Wide() float64 { return made.wide }

// At is the scene's time for a still, or for a film's first frame.
func (made Defaults) At() time.Duration { return made.at }

// Tone is how the scene's light becomes a picture.
func (made Defaults) Tone() colour.Tone { return made.tone }

// Built is a scene made: its world, the eye its command looks from, and,
// where the scene has lamps that keep their own time, their clock, or
// where its light changes with time, what to tell it.
type Built struct {
	advance func(time.Duration) (maths.Subspace, error)
	wide    func(time.Duration) float64
	world   *thing.World
	eye     maths.Subspace
	clock   *time.Duration
	played  *lantern.Played
	tick    func(at time.Duration)
}

// World is what is in the scene.
func (made *Built) World() *thing.World { return made.world }

// Eye is where the scene is seen from, as its command sees it.
func (made *Built) Eye() maths.Subspace { return made.eye }

// Timed is whether the scene has lamps that keep their own time, or
// light that changes with it, so that Set must be told the time of each
// frame before it is drawn.
func (made *Built) Timed() bool {
	return made.clock != nil || made.tick != nil || made.advance != nil
}

// Set tells the scene's lamps the time of the frame about to be drawn. A
// scene with no lamps that keep their own time has nothing to tell.
func (made *Built) Set(at time.Duration) {
	if made.clock != nil {
		*made.clock = at
	}
	if made.tick != nil {
		made.tick(at)
	}
}

// ErrUnplayed is a value for a channel of a scene that has none.
var ErrUnplayed = errors.New("named: this scene has no channels")

// Channels are the channels music can play the scene by, each with its
// range and rest; none for a scene that has none. Only zappy has them.
func (made *Built) Channels() []lantern.Channel {
	if made.played == nil {
		return nil
	}
	return made.played.Channels()
}

// Play holds the channels' values for the moment about to be drawn: those
// given, and the rest at their rests. A channel the scene has not and a
// value out of range are refused by name; a scene with no channels
// refuses any value. Set still tells the scene the time; Play moves what
// the channels move from there.
func (made *Built) Play(values map[string]float64) error {
	if made.played == nil {
		if len(values) == 0 {
			return nil
		}
		return ErrUnplayed
	}
	return made.played.Hold(values)
}

// All is every named scene of both kinds, the traced first, each once, a
// list a caller may change.
func All() []Listed {
	all := make([]Listed, nothing, len(registry)+len(maps))
	for _, scene := range registry {
		all = append(all, scene)
	}
	for _, scene := range maps {
		all = append(all, scene)
	}
	return all
}

// Traced is every traced scene, each once, a copy a caller may change.
func Traced() []Scene {
	return slices.Clone(registry)
}

// Get is the traced scene of that name, or a refusal that lists the
// names there are, or says the name is drawn as maps.
func Get(name string) (*Scene, error) {
	for _, scene := range registry {
		if scene.name == name {
			found := scene
			return &found, nil
		}
	}
	if slices.Contains(mapNames(), name) {
		return nil, fmt.Errorf("%w: %q is drawn as maps, not traced; "+
			"slayer draw draws it", ErrNoScene, name)
	}
	return nil, fmt.Errorf("%w: %q; the scenes are %s", ErrNoScene, name,
		strings.Join(append(names(), mapNames()...), ", "))
}

// names are the scenes' names, in the registry's order.
func names() []string {
	all := make([]string, len(registry))
	for i, scene := range registry {
		all[i] = scene.name
	}
	return all
}

// registry is every named scene. It is made once and never changed; All
// and Get hand out copies.
var registry = []Scene{swarmScene(), zappyScene(), orbitalScene(),
	ringScene(), moonriseScene(), magnetarScene()}

// the counts a scene's parts are made of, each defined
const (
	nothing = 0 // defined: no value, no time
	single  = 1 // defined: one, the same again
)
