package kernel_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// lit is the fixture: matte ground of albedo a half under one lamp of
// strength ten, two up over the middle, seen from up and back.
var lit = struct {
	albedo, strength, height float64
	seen                     view
}{albedo: 0.5, strength: 10, height: 2,
	seen: looking(maths.V(0, 3, 4), maths.V(0, 0, 0), 24, 16)}

// litGround is the ground under its lamp.
func litGround(t *testing.T) *thing.World {
	t.Helper()
	floor := &thing.Thing{Shape: ground(t, 50),
		Material: &thing.Material{Colour: maths.Grey(lit.albedo)}}
	lamp := emit.Lit{At: maths.V(0, lit.height, 0),
		Strength: maths.Grey(lit.strength)}
	return &thing.World{Things: []*thing.Thing{floor},
		Lamps: []thing.Lamp{lamp}}
}

// radiance is what a camera sees of the ground at a point: albedo over
// pi of the irradiance, the lamp's strength times the cosine of its
// slant over the square of its distance.
func radiance(point maths.Vec) float64 {
	square := point.X*point.X + point.Z*point.Z + lit.height*lit.height
	slant := lit.height / math.Sqrt(square)
	return lit.albedo / math.Pi * lit.strength * slant / square
}

// onGround is where a ray meets the level ground, if it does.
func onGround(ray maths.Ray) (maths.Vec, bool) {
	if ray.Dir.Y >= 0 {
		return maths.Vec{}, false
	}
	return ray.Point(-ray.At.Y / ray.Dir.Y), true
}

// shares is, for every pixel that sees the ground, what each way draws it as a
// share of what physics says.
func shares(t *testing.T, way drawer) []float64 {
	t.Helper()
	seen := lit.seen
	picture := seen.drawn(t, way, litGround(t))
	var made []float64
	for y := range seen.high {
		for x := range seen.wide {
			point, met := onGround(seen.ray(x, y))
			if met {
				drawn := float64(picture.At(x, y).R)
				made = append(made, drawn/radiance(point))
			}
		}
	}
	return made
}

// The ground falls off as physics says, both ways: every pixel the same
// share of the radiance as every other.
func TestTheGroundFallsOffAsPhysicsSaysBothWays(t *testing.T) {
	for _, way := range drawers {
		made := shares(t, way)
		low, high := math.Inf(1), math.Inf(-1)
		for _, share := range made {
			low, high = min(low, share), max(high, share)
		}
		if high/low-1 > 1e-3 {
			t.Errorf("%s draws the ground from %.4f to %.4f of its "+
				"radiance", way.name, low, high)
		}
	}
}

// The ground is drawn as radiance, both ways: the share is one.
func TestTheGroundIsRadianceBothWays(t *testing.T) {
	for _, way := range drawers {
		made := shares(t, way)
		if len(made) == 0 {
			t.Fatalf("%s: no pixel sees the ground", way.name)
		}
		if share := made[0]; math.Abs(share-1) > 1e-3 {
			t.Errorf("%s draws the ground %.4f times its radiance",
				way.name, share)
		}
	}
}
