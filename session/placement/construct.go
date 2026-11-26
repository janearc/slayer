// Package placement shares native manufactured construction and space transforms.
package placement

import (
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
)

// Specimen exposes the native manufactured object's immutable query methods.
type Specimen interface {
	Things(maths.Subspace) ([]thing.Thing, error)
	Descriptions() []manufacture.AttachedLight
	Bounds() maths.Box
}

// Construction identifies the same native manufacture call used by staging.
type Construction struct {
	Kind       manufacture.Kind
	SceneSeed  uint64
	ObjectSeed int
	Complexity string
}

// Make constructs native geometry from the canonical combined seed.
func Make(request Construction) (Specimen, error) {
	return manufacture.MakeWithComplexity(request.Kind, manufacture.Sow(Seed(request.SceneSeed, request.ObjectSeed)), request.Complexity)
}

// Seed combines the scene and object seeds using the staging finaliser.
func Seed(scene uint64, object int) uint64 {
	return noise.Mix(scene ^ (uint64(object+1) * noise.Golden))
}

// SpaceAt places a native thing in the right-handed metre frame.
func SpaceAt(each set.Thing) maths.Subspace {
	east, north := each.At()
	up := each.Height()
	if absolute, has := each.Position(); has {
		east, north, up = absolute.East(), absolute.North(), absolute.Up()
	}
	turn := float64(0)
	if angle, said := each.Heading(); said {
		turn = -angle * maths.Degree
	}
	at := maths.Subspace{Origin: maths.V(east, up, -north), Orientation: maths.About(maths.YAxis, turn)}
	if scale, has := each.Scale(); has {
		at.Scale = scale
	}
	return at
}
