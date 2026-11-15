package bodies

import (
	"fmt"
	"math"

	"github.com/janearc/celestial/radiation"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// Span is a length in kilometres, held exactly as it was said: the
// earth is Kilometres(6371), and a worked figure of halves and quarters
// stays the float it was written as. measure's lengths live in natural
// units, so a trip through them can cost a last digit: 64 km comes back
// as 64.000000000000014. A body's radii and orbits key integer-valued
// geometry, where that digit changes a name, so they are held exactly.
//
// Span is for lengths only. Mass, temperature and angle stay measure's
// types: they feed periods and colours that this package compares to a
// tolerance, never keys on, so the digit does not matter to them.
type Span struct {
	kilometres float64
}

// Kilometres is a span said in kilometres, exactly.
func Kilometres(amount float64) Span { return Span{amount} }

// InKilometres is the span in kilometres, exactly as it was said.
func (span Span) InKilometres() float64 { return span.kilometres }

// Loss is how much light a medium takes, said with PerKilometre: an
// extinction at a part's densest, falling off by the medium's own
// profile, one number for every band. Made inline, it carries any
// complaint privately; the constructor that receives it surfaces the
// complaint through Extinction.
type Loss struct {
	perKilometre float64
	complaint    error
}

// PerKilometre is the share of the light lost in a kilometre of path
// where the part is densest. Below zero, or no number, is no loss.
func PerKilometre(amount float64) Loss {
	if math.IsNaN(amount) || math.IsInf(amount, 0) || amount < 0 {
		return Loss{complaint: fmt.Errorf(
			"PerKilometre(%v) is no extinction", amount)}
	}
	return Loss{perKilometre: amount}
}

// Extinction is the loss per kilometre, or the complaint its making
// carried: a constructor taking a Loss surfaces this.
func (loss Loss) Extinction() (float64, error) {
	if loss.complaint != nil {
		return loss.perKilometre, fmt.Errorf("bodies: %w",
			loss.complaint)
	}
	return loss.perKilometre, nil
}

// problem is the loss per kilometre and the complaint its making
// carried, without the package prefix, for a constructor that wraps it
// under its own.
func (loss Loss) problem() (float64, error) {
	return loss.perKilometre, loss.complaint
}

// finite is whether x is a number a length, angle or brightness can
// be: not NaN, not infinite.
func finite(x float64) bool { return !math.IsNaN(x) && !math.IsInf(x, 0) }

// positiveFinite is whether x is above zero and finite.
func positiveFinite(x float64) bool { return x > 0 && !math.IsInf(x, 1) }

// nonNegativeFinite is whether x is zero or above and finite.
func nonNegativeFinite(x float64) bool { return x >= 0 && !math.IsInf(x, 1) }

// finiteLight is whether every component of the light is finite.
func finiteLight(light maths.Light) bool {
	return finite(light.R) && finite(light.G) && finite(light.B)
}

// numberComplaint is the complaint, without the package prefix, for an
// argument that is not finite, or nil when it is.
func numberComplaint(what string, x float64) error {
	if finite(x) {
		return nil
	}
	return fmt.Errorf("%s is %v: a finite number is expected", what, x)
}

// lightComplaint is the complaint, without the package prefix, for a
// light with a component that is not finite, or nil when none is.
func lightComplaint(what string, light maths.Light) error {
	if finiteLight(light) {
		return nil
	}
	return fmt.Errorf("%s is %v: finite red, green and blue are expected",
		what, light)
}

// firstComplaint is the first complaint that is not nil, or nil.
func firstComplaint(complaints ...error) error {
	for _, complaint := range complaints {
		if complaint != nil {
			return complaint
		}
	}
	return nil
}

// vecComplaint is the complaint, without the package prefix, for a
// vector with a component that is not finite, or nil when none is.
func vecComplaint(what string, vec maths.Vec) error {
	if finite(vec.X) && finite(vec.Y) && finite(vec.Z) {
		return nil
	}
	return fmt.Errorf("%s is %v: finite components are expected", what,
		vec)
}

// radiantComplaint is the complaint, without the package prefix, for a
// radiant that is nil or, when it is a thing.Spectrum, holds a share
// that is not finite. Any other radiant is answered by its own methods
// and cannot be listed, so it is taken as it is.
func radiantComplaint(what string, radiant thing.Radiant) error {
	if radiant == nil {
		return fmt.Errorf("%s is nil: a radiant is expected", what)
	}
	var spectrum thing.Spectrum
	switch held := radiant.(type) {
	case thing.Spectrum:
		spectrum = held
	case *thing.Spectrum:
		if held == nil {
			return fmt.Errorf("%s is a nil *thing.Spectrum: a radiant "+
				"is expected", what)
		}
		spectrum = *held
	default:
		return nil
	}
	for _, shares := range []struct {
		name string
		by   map[radiation.Range]float64
	}{{"passes", spectrum.Passes}, {"gives", spectrum.Gives},
		{"returns", spectrum.Returns}} {
		for each, share := range shares.by {
			if !finite(share) {
				return fmt.Errorf("%s %s %v of %v: a finite number is "+
					"expected", what, shares.name, share, each)
			}
		}
	}
	return nil
}

// hold records a complaint as the first one a part or body holds,
// leaving an earlier one in place.
func hold(held *error, complaint error) {
	if *held == nil && complaint != nil {
		*held = complaint
	}
}
