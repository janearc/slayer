package hud

import (
	"fmt"
	"math"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

//go:generate sh mocks/generate.sh

// Blocker is what may stand between the eye and a point: the scene,
// asked whether a ray meets anything between near and far, and how far
// along it does.
type Blocker interface {
	Hit(ray maths.Ray, near, far float64) (float64, bool)
}

// View is how the scene under a hud was seen: the eye and the sensor
// that traced it, and the picture's size in the pixels it was traced
// at, which say its shape. Blocker, if there is one, is the scene, for
// a label that asks to be hidden behind it.
type View struct {
	Eye           maths.Subspace
	Sensor        film.Sensor
	Width, Height int
	Blocker       Blocker
}

// pinhole is the view's sensor as the pinhole it is: a pinhole, or a
// lens, which is a pinhole of the lens's width. a sensor of any other
// kind cannot be projected through, and is an error.
func (seen View) pinhole() (film.Pinhole, error) {
	switch made := seen.Sensor.(type) {
	case nil:
		return film.Pinhole{}, nil
	case film.Pinhole:
		return made, nil
	case film.Lens:
		return film.Pinhole{Wide: made.Wide(), Pixel: made.Pixel}, nil
	}
	return film.Pinhole{}, fmt.Errorf("%w: a pinned label cannot be "+
		"projected through a sensor of kind %T", ErrHud, seen.Sensor)
}

// Project is where a point of the world falls on the picture, as
// shares of its width from the left and its height from the top, and
// whether it is in front of the eye: the inverse of film.Look, so that
// the ray film.Look makes through that place passes through the point.
func (seen View) Project(p maths.Vec) (across, down float64, ahead bool,
	err error) {
	lens, err := seen.pinhole()
	if err != nil || seen.Width < 1 || seen.Height < 1 {
		if err == nil {
			err = fmt.Errorf("%w: a view of %dx%d pixels", ErrHud,
				seen.Width, seen.Height)
		}
		return 0, 0, false, err
	}
	local := seen.Eye.In(p)
	if !(local.Z < 0) {
		return 0, 0, false, nil
	}
	half := math.Tan(wide(lens) / 2)
	aspect := float64(seen.Width) * pixel(lens) / float64(seen.Height)
	u, v := local.X/-local.Z, local.Y/-local.Z
	return (u/half + 1) / 2, (1 - v*aspect/half) / 2, true, nil
}

// wide is how wide a pinhole sees, zero read as sixty degrees, as film's own
// pinhole reads it.
func wide(lens film.Pinhole) float64 {
	if lens.Wide == 0 {
		return 60 * maths.Degree
	}
	return lens.Wide
}

// pixel is a pinhole's pixel shape, zero read as square.
func pixel(lens film.Pinhole) float64 {
	if lens.Pixel == 0 {
		return 1
	}
	return lens.Pixel
}

// hidden is whether something of the scene stands between the eye and
// p: a ray from the eye toward p meets it short of p, by more than a
// millionth of the way, so a point on a surface is not hidden by it.
func (seen View) hidden(p maths.Vec) bool {
	if seen.Blocker == nil {
		return false
	}
	way := p.Sub(seen.Eye.Origin)
	far := way.Len()
	ray := maths.Ray{At: seen.Eye.Origin, Dir: way.Unit()}
	_, hit := seen.Blocker.Hit(ray, 0, far*(1-1e-6))
	return hit
}

// Seen is a world at a moment as a blocker: every thing in it that is
// drawn and can answer a ray, placed where its motion has it then.
func Seen(world *thing.World, at time.Duration) Blocker {
	return scene{world: world, at: at}
}

// scene is a world at a moment, asked as a blocker.
type scene struct {
	world *thing.World
	at    time.Duration
}

// Hit is the nearest of the scene's things a ray meets between near and
// far, each asked in its own space, where distances are its scale times
// smaller.
func (made scene) Hit(ray maths.Ray, near, far float64) (float64, bool) {
	best, met := far, false
	for _, one := range made.world.Things {
		hitter, ok := one.Shape.(shape.Hitter)
		if !ok || one.Hidden() {
			continue
		}
		space := one.At(made.at)
		scale := space.Scale
		if scale == 0 {
			scale = 1
		}
		t, hit := hitter.Hit(space.RayIn(ray), near/scale, best/scale)
		if hit && t*scale < best {
			best, met = t*scale, true
		}
	}
	return best, met
}
