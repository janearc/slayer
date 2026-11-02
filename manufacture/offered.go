package manufacture

import (
	"time"

	"github.com/janearc/celestial/bands"
	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// The seeded construction API exposes kinds, seeds, generated objects, and
// attached light descriptions. Older conifer exports remain as reference
// implementations for kernel checks.
//
// Returned implementation types are private.
//
// Callers can infer them with :=, call public methods, and pass values through
// interfaces or generics.
//
// To store them in named fields or collections, define an interface containing
// the methods needed. A light’s private return types require reading its
// methods individually, as emit.Described does. Use go doc -u to inspect
// methods on these private types.
//
// cmd/hillside uses object construction; emit reads light descriptions. Keep
// additional implementation details private until a client needs them.

// Sow constructs a deterministic random seed, not an object age or state.
func Sow(number uint64) seed { return sow(number) }

// KindNames returns available kind names in alphabetical order.
func KindNames() []string { return kindNames() }

// KindNamed looks up a kind and reports available names on failure.
func KindNamed(name string) (kind, error) { return kindNamed(name) }

// Make generates a kind from a seed, selecting values within the kind’s
// declared ranges. It returns an error when construction is invalid.
func Make(row kind, from seed) (good, error) { return makeOf(row, from) }

// Things places the generated object in a scene, returning one thing.Thing per
// material.
func (made good) Things(at maths.Subspace) ([]thing.Thing, error) {
	return made.things(at)
}

// Lights returns attached lights transformed to the object’s scene placement.
func (made good) Lights(at maths.Subspace) []lit {
	return made.placed(at).lights
}

// At returns the light’s world-space position.
func (one lit) At() maths.Vec { return one.at }

// Toward returns the light’s unit direction.
func (one lit) Toward() maths.Vec { return one.toward }

// Beyond returns the source exclusion distance from At, preventing the light’s
// own housing from shadowing its illumination.
func (one lit) Beyond() measure.Length { return one.beyond }

// Light returns the attached light’s emission description.
func (one lit) Light() light { return one.made }

// Bands returns the libtheme spectrum’s shape; Strength sets its intensity.
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
