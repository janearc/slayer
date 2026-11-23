package session

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

// retainedSparks keeps the source and its thermal table across movie frames,
// and how long it throws sparks: nought for as long as it burns.
type retainedSparks struct {
	source  embers.Source
	colours flame.Palette
	shape   *mesh.Mesh
	place   maths.Subspace
	life    time.Duration
	stop    time.Duration
	maximum int
}

// fired is a mortar fired at a time after the film's start: its At is
// the mortar's that long after.
type fired struct {
	mortar embers.Mortar
	at     time.Duration
}

// At is the shell or its stars at a time in the film.
func (shot fired) At(when time.Duration) []embers.Particle {
	return shot.mortar.At(when - shot.at)
}

// prepareShell is a display shell fired from the fire's place, with the
// wind on its stars, in place of the fire's sparks.
func (stage *staging) prepareShell(spec manufacture.BurnSpec, at maths.Subspace) (*retainedSparks, error) {
	shell, err := embers.DisplayShell(stage.given.shellInches, spec.Flame.Seed)
	if err != nil {
		return nil, err
	}
	from, speed := stage.recipes.Wind()
	shell.Air = shell.Air.Add(skyward(from, 0).Neg().Scale(speed))
	mortar, err := embers.NewMortar(shell)
	if err != nil {
		return nil, err
	}
	colours, err := flame.Colours(flame.ColourSpec{Coolest: shell.Cool,
		Hottest: shell.Hot, Exposure: stage.given.exposure})
	if err != nil {
		return nil, err
	}
	shape, err := stage.sparkShape()
	if err != nil {
		return nil, err
	}
	return &retainedSparks{source: fired{mortar, stage.given.shellAt},
		colours: colours, shape: shape, place: at,
		life: shell.Fuse + shell.Burn, maximum: shell.Stars + 1}, nil
}

// sparkShape is the ember mesh every spark and star shares, made once.
func (stage *staging) sparkShape() (*mesh.Mesh, error) {
	if stage.firebrandShape == nil {
		shape, err := emberShape()
		if err != nil {
			return nil, err
		}
		stage.firebrandShape = shape
	}
	return stage.firebrandShape, nil
}

// prepareSparks validates the immutable particle source once per scene,
// or, with -shell, fires a display shell in its place.
func (stage *staging) prepareSparks(spec manufacture.BurnSpec, at maths.Subspace) (*retainedSparks, error) {
	if stage.given.shellInches != 0 {
		return stage.prepareShell(spec, at)
	}
	if spec.Sparks == nil {
		return nil, nil
	}
	sparks := *spec.Sparks
	sparks.Seed = spec.Flame.Seed
	sparks.Sphere = stage.given.sparksSphere
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
	shape, err := stage.sparkShape()
	if err != nil {
		return nil, err
	}

	// as many as can be living at once: a lifetime's births, or, when it
	// stops throwing sooner, the births before it stops
	throwing := sparks.Life
	if stop := stage.given.sparksFor; stop > 0 && stop < throwing {
		throwing = stop
	}
	return &retainedSparks{source: source, colours: colours, shape: shape, place: at, life: sparks.Life,
		stop: stage.given.sparksFor, maximum: int(math.Ceil(sparks.Rate*throwing.Seconds())) + 2}, nil
}

// thingsAt updates particle poses and glow without rebuilding shared geometry,
// throwing no new sparks past the source's stop, when it has one.
func (sparks *retainedSparks) thingsAt(at time.Duration) []thing.Thing {
	if sparks == nil {
		return nil
	}
	return sparks.thingsIn(at, sparks.stop)
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
