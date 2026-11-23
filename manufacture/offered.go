package manufacture

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/janearc/celestial/bands"
	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/strategies"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// The seeded construction API exposes kinds, seeds, generated objects, and
// attached light descriptions. Older conifer exports remain as reference
// implementations for kernel checks.
//
// Kind and AttachedLight name opaque recipe and light values. Generated
// specimen types remain private.
//
// Callers can infer them with :=, call public methods, and pass values through
// interfaces or generics.
//
// To store generated specimens in fields or collections, define an interface
// containing the methods needed. Read a light's private return values through
// their methods, as emit.Described does. Use go doc -u to inspect the methods
// on generated specimens.
//
// the session uses object construction; emit reads light descriptions. Keep
// additional implementation details private until a client needs them.

// Sow constructs a deterministic random seed, not an object age or state.
func Sow(number uint64) seed { return sow(number) }

// Kind names a checked construction recipe. Its fields remain private;
// use KindNamed or a checked instrument constructor to obtain one.
// The alias lets scene adapters hold recipes from several constructors.
type Kind = kind

// AttachedLight is a local light description with private fields. A scene
// can retain these descriptions after releasing construction geometry.
type AttachedLight = lit

// KindNames returns available kind names in alphabetical order.
func KindNames() []string { return kindNames() }

// KindNamed looks up a kind and reports available names on failure.
func KindNamed(name string) (kind, error) { return kindNamed(name) }

// Make generates a kind from a seed, selecting values within the kind's
// declared ranges. It returns an error when construction is invalid.
func Make(row kind, from seed) (good, error) { return makeOf(row, from) }

// ExtentOf is the box that holds what a kind makes from a seed, in the
// thing's own frame, said without making the thing where the kind can. It
// may be too large and is never too small.
func ExtentOf(row kind, from seed) (maths.Box, error) {
	return extentOf(row, from)
}

// SeenOf is how an eye sees what a kind makes from a seed standing at a
// place: whether any pixel's ray can meet it, and how many pixels across it
// spans.
func SeenOf(row kind, from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenOf(row, from, at, by)
}

var errTaller = errors.New("manufacture: only a whorled tree grows " +
	"taller, and by a number of times above nought")

var errBranched = errors.New("manufacture: only a whorled tree's " +
	"branches carry branches")

// Branched is a whorled tree whose limbs carry side branches of their
// own, spaced along them in two rows, each in its own foliage: a bough
// is a fan of sprays, not a line of them (2026-11-10: "the
// branches themselves should also have branches"). For a tree seen
// close; the trees Make and MakeWithComplexity grow are unchanged.
func Branched(row kind) (kind, error) {
	switch how := row.how.(type) {
	case physicalSpruce:
		how.legacy = how.legacy.branched()
		return kind{name: row.name, how: how}, nil
	case conifer:
		return kind{name: row.name, how: how.branched()}, nil
	}
	return kind{}, fmt.Errorf("%w: %s is not one", errBranched, row.name)
}

// Taller is a whorled tree grown so many times taller from the same
// seed: its trunk as much taller and thicker, as many more whorls as fit
// up it, branches longer by less and as thick as their own length asks,
// and needles at their own size, as many more as its crown has room
// for. The trees Make and MakeWithComplexity grow are unchanged.
//
// A tree magnified instead thickens every limb with its trunk,
// lengthens every needle, and keeps as few needles over far more crown.
func Taller(row kind, times float64) (kind, error) {
	if !positive(times) {
		return kind{}, fmt.Errorf("%w: %v times", errTaller, times)
	}
	switch how := row.how.(type) {
	case physicalSpruce:
		how.legacy = how.legacy.taller(times)
		return kind{name: row.name, how: how}, nil
	case conifer:
		return kind{name: row.name, how: how.taller(times)}, nil
	}
	return kind{}, fmt.Errorf("%w: %s is not one", errTaller, row.name)
}

// Things places the generated object in a scene, returning one thing.Thing per
// material.
func (made good) Things(at maths.Subspace) ([]thing.Thing, error) {
	return made.things(at)
}

// Lights returns attached lights transformed to the object's scene placement.
func (made good) Lights(at maths.Subspace) ([]lit, error) {
	return PlaceLights(made.lights, at)
}

// Descriptions returns an owned copy of the local light descriptions. It
// allows placement caches to release the specimen's construction geometry.
func (made good) Descriptions() []AttachedLight {
	return slices.Clone(made.lights)
}

// PlaceLights transforms retained descriptions without their source specimen.
// Even an empty list checks placement, as Lights does.
func PlaceLights(local []AttachedLight,
	at maths.Subspace,
) ([]AttachedLight, error) {
	if !shape.ValidPlacement(at) {
		return nil, errors.New("manufacture: invalid light placement")
	}
	var lights []lit
	for _, one := range local {
		one.at = at.Out(one.at)
		one.toward = at.Orientation.Out(one.toward)
		one.beyond = one.beyond.Times(scaleOf(at))
		lights = append(lights, one)
	}
	return lights, nil
}

// At returns the light's world-space position.
func (one lit) At() maths.Vec { return one.at }

// Toward returns the light's unit direction.
func (one lit) Toward() maths.Vec { return one.toward }

// Beyond returns the source exclusion distance from At, preventing the light's
// own housing from shadowing its illumination.
func (one lit) Beyond() measure.Length { return one.beyond }

// Light returns the attached light's emission description.
func (one lit) Light() light { return one.made }

// Bands returns the Celestial spectrum's shape; Strength sets its intensity.
func (made light) Bands() bands.Light { return made.bands() }

// Strength returns beam intensity in watts per steradian.
func (made light) Strength() float64 { return made.strength() }

// Angles returns the full-strength beam half-angle and the additional soft-edge
// angle. A half-turn with no soft edge denotes isotropic emission.
func (made light) Angles() (half, soft measure.Angle) {
	return made.angles()
}

// Share returns relative strength at a time, reducing unity during flicker.
func (made light) Share(at time.Duration) float64 {
	return made.share(at)
}
