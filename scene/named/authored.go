package named

import (
	"fmt"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/scene/thing"
	"io"
	"time"
)

// Authored makes a procedural world available to the normal job/session
// pipeline. It supplies scene content and an absolute clock, never a renderer,
// image writer or developer. Register these only during program startup.
type Authored struct {
	Name, About, Unit   string
	FastLighting        bool
	Width, Height, Rays int
	Wide                float64
	Tone                colour.Tone
	Check               func(map[string]any) error
	Lens                func(map[string]any) (func(time.Duration) float64, error)
	Build               func(map[string]any, io.Writer) (*thing.World, maths.Subspace, func(time.Duration) (maths.Subspace, error), error)
}

// RegisterAuthored registers an application-owned procedural scene before
// jobs are read. Duplicate names and incomplete definitions are refused.
func RegisterAuthored(a Authored) error {
	if a.Name == "" || a.Unit == "" || a.Width < 1 || a.Height < 1 || a.Rays < 1 || a.Wide <= 0 || a.Check == nil || a.Build == nil {
		return fmt.Errorf("incomplete authored scene %q", a.Name)
	}
	for _, s := range registry {
		if s.name == a.Name {
			return fmt.Errorf("scene %q already registered", a.Name)
		}
	}
	registry = append(registry, Scene{name: a.Name, about: a.About, unit: a.Unit, renderers: []Renderer{CPU, GPU},
		defaults: Defaults{width: a.Width, height: a.Height, rays: a.Rays, wide: a.Wide, tone: a.Tone}, authored: &a})
	return nil
}

// Advance also updates an authored scene's camera and propagates build-time
// motion errors. Set retains the older, infallible named-scene interface.
func (made *Built) Advance(at time.Duration) error {
	made.Set(at)
	if made.advance != nil {
		eye, err := made.advance(at)
		if err != nil {
			return err
		}
		made.eye = eye
	}
	return nil
}

// AllowsFastLighting says this scene explicitly opts into GPU lighting previews.
func (scene *Scene) AllowsFastLighting() bool {
	return scene.fastLighting || (scene.authored != nil && scene.authored.FastLighting)
}

// WideAt returns an authored shot's field of view in radians when it varies.
func (made *Built) WideAt(at time.Duration) (float64, bool) {
	if made.wide == nil {
		return 0, false
	}
	return made.wide(at), true
}
