package main

import (
	"github.com/janearc/slayer/embers"
	"github.com/janearc/slayer/flame"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
	"math"
	"time"
)

// emberShape shares one small closed mesh across every firebrand instance.
func emberShape() (*mesh.Mesh, error) {
	corners := []maths.Vec{maths.YAxis, maths.YAxis.Neg(), maths.XAxis, maths.ZAxis, maths.XAxis.Neg(), maths.ZAxis.Neg()}
	faces := [][3]int{{0, 3, 2}, {0, 4, 3}, {0, 5, 4}, {0, 2, 5},
		{1, 2, 3}, {1, 3, 4}, {1, 4, 5}, {1, 5, 2}}
	return mesh.New(corners, faces, corners, true)
}

// retainedSparks keeps the source and its thermal table across movie frames.
type retainedSparks struct {
	source  embers.Source
	colours flame.Palette
	shape   *mesh.Mesh
	place   maths.Subspace
	life    time.Duration
	maximum int
}

// prepareSparks validates the immutable particle source once per scene.
func (stage *staging) prepareSparks(spec manufacture.BurnSpec, at maths.Subspace) (*retainedSparks, error) {
	if spec.Sparks == nil {
		return nil, nil
	}
	sparks := *spec.Sparks
	sparks.Seed = spec.Flame.Seed
	from, speed := stage.recipes.Wind()
	sparks.Air = sparks.Air.Add(skyward(from, 0).Neg().Scale(speed))
	source, err := embers.New(sparks)
	if err != nil {
		return nil, err
	}
	colours, err := flame.Colours(flame.ColourSpec{Coolest: sparks.Cool, Hottest: sparks.Hot,
		Exposure: stage.given.exposure})
	if err != nil {
		return nil, err
	}
	shape := stage.firebrandShape
	if shape == nil {
		shape, err = emberShape()
		if err != nil {
			return nil, err
		}
		stage.firebrandShape = shape
	}

	return &retainedSparks{source: source, colours: colours, shape: shape, place: at, life: sparks.Life, maximum: int(math.Ceil(sparks.Rate*sparks.Life.Seconds())) + 2}, nil
}

// thingsAt updates particle poses and glow without rebuilding shared geometry.
func (sparks *retainedSparks) thingsAt(at time.Duration) []thing.Thing {
	return sparks.thingsIn(at, 0)
}

// thingsIn stops new births at the burn duration while retaining airborne
// particles. Zero duration is the original indefinitely burning source.
func (sparks *retainedSparks) thingsIn(at, duration time.Duration) []thing.Thing {

	if sparks == nil {
		return nil
	}
	var particles []thing.Thing
	for _, particle := range sparks.source.At(at) {
		if duration > 0 && at-particle.Age() >= duration {
			continue
		}
		scale := particle.Radius().InMetres()
		if sparks.place.Scale != 0 {
			scale *= sparks.place.Scale
		}
		place := maths.Subspace{Origin: sparks.place.Out(particle.Position()), Scale: scale}
		particles = append(particles, thing.Thing{Shape: sparks.shape,
			Material: &thing.Material{Name: "firebrand", Colour: maths.Grey(.03),
				Glow: sparks.colours.At(particle.Heat())}, Motion: thing.Still(place)})
	}
	return particles
}

// budget bounds a source's live particles before rendering begins.
func (sparks *retainedSparks) budget() int {
	if sparks == nil {
		return 0
	}
	return sparks.maximum
}
