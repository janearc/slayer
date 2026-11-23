package medium

import (
	"errors"
	"math"

	"github.com/janearc/slayer/maths"
)

// riseThree, riseTwo and entrained are Briggs's rise of a plume bent over
// by a wind, as Koushafar, Sohn and Gordon give it (arXiv 2302.07416, eq.
// 1, citing G. Briggs, "Plume rise predictions", AMS lectures on air
// pollution, 1975): the buoyant part, the cube root of 3 F x^2 over
// 2 x 0.6^2 u^3, 0.6 being "an entrainment rate". Read.
const riseThree, riseTwo, entrained = 3, 2, 0.6

// acrossFactor and acrossPower are Martin's fit to the Pasquill-Gifford
// spread across the wind in neutral air, class D, a x to the b with x in
// kilometres and the spread in metres (D. O. Martin 1976, as the
// University of Washington's CEE 490 page on dispersion coefficients
// gives it). Read.
const acrossFactor, acrossPower = 68, 0.894

// nearFactor, nearPower and nearLess are Martin's fit to the spread up and
// down, class D, within a kilometre: c x to the d plus f, f being -1.7.
// farFactor, farPower and farLess are the same beyond a kilometre, f
// being -13.0. The same page. Read.
const (
	nearFactor, nearPower, nearLess = 33.2, 0.725, 1.7
	farFactor, farPower, farLess    = 44.5, 0.516, 13.0
)

// metresPerKilometre is the unit Martin's fits take their distance in.
// Defined.
const metresPerKilometre = 1000.0

// gaussianReach is how many spreads from its middle a plume's box reaches,
// where its gaussian is under a hundredth of its middle's. Picked.
const gaussianReach = 3.0

// halfSquare and oneThird are a gaussian's half square and a cube root's
// power; a gaussian's two pi is fullTurn. Defined.
const (
	halfSquare = 0.5
	oneThird   = 1.0 / 3.0
)

// Stack is what a plume leaves its mouth as and into: its mouth's radius,
// in metres; Briggs's buoyancy flux, metres to the fourth a second cubed;
// the wind, metres a second; the rate the particles leave, grams a second;
// their mass extinction, square metres a gram; how far over the mouth the
// lid is, in metres, nought for none; and the particles' asymmetry and
// Angstrom exponent.
type Stack struct {
	radius, buoyancy, wind, rate, extinction, lid float64
	asymmetry, exponent                           float64
}

// NewStack is a stack's plume as it leaves. Every number MUST be finite,
// the radius, wind and rate above nought, the flux, extinction and lid
// from nought, and the asymmetry strictly between minus one and one.
func NewStack(radius, buoyancy, wind, rate, extinction,
	lid float64) (Stack, error) {
	for _, part := range []float64{radius, buoyancy, wind, rate,
		extinction, lid} {
		if !weatherFinite(part) || part < nought {
			return Stack{}, errStack
		}
	}
	if radius <= nought || wind <= nought || rate <= nought {
		return Stack{}, errStack
	}
	return Stack{radius: radius, buoyancy: buoyancy, wind: wind,
		rate: rate, extinction: extinction, lid: lid}, nil
}

var errStack = errors.New("medium: a stack has a mouth, a wind and a " +
	"rate above nought, and a flux, extinction and lid from nought")

// Scattering is the stack whose particles scatter by a Henyey-Greenstein
// asymmetry, strictly between minus one and one, and thin toward the red
// by an Angstrom exponent, from nought to the most an aerosol takes.
func (stack Stack) Scattering(asymmetry, exponent float64) (Stack, error) {
	if !(math.Abs(asymmetry) < unity) ||
		!(exponent >= nought && exponent <= chosen.exponentMost) {
		return Stack{}, errors.New("medium: a stack's particles have an " +
			"asymmetry inside one and an Angstrom exponent from nought " +
			"to two and a half")
	}
	stack.asymmetry, stack.exponent = asymmetry, exponent
	return stack, nil
}

// Tint is how many times its density each of the eye's channels meets, by
// the particles' Angstrom exponent from the green: green is one.
func (bent *Bent) Tint() maths.Light {
	return maths.RGB(
		math.Pow(chosen.wavelengthRed/chosen.wavelengthGreen,
			-bent.stack.exponent), unity,
		math.Pow(chosen.wavelengthBlue/chosen.wavelengthGreen,
			-bent.stack.exponent))
}

// Bent is a stretch of a stack's plume, bent over by the wind, in the
// stack's own space: x down the wind from the mouth, y up, z across. Its
// middle rises by Briggs's law until it meets the lid, and its section is
// a gaussian whose spread joins the rising plume's, its radius (the
// mouth's and 0.6 of the rise) over root two, with the air's by Martin's
// fits.
//
// What leaves is carried through each section: on the middle the
// density is the rate over two pi, the wind and both spreads, times the
// mass extinction.
//
// Over the lid it holds nothing: the lid is the top of the air that mixes,
// and the share of the plume over it is laid back under it, an image of
// the plume in the lid.
type Bent struct {
	stack      Stack
	from, upto float64
}

// NewBent is the stretch of a stack's plume from one distance down the
// wind to another, from nought and running forward.
func NewBent(stack Stack, from, upto float64) (*Bent, error) {
	if !weatherFinite(from) || !weatherFinite(upto) || from < nought ||
		upto <= from || stack.wind <= nought {
		return nil, errors.New("medium: a stretch of a plume runs " +
			"forward from nought, from a stack")
	}
	return &Bent{stack: stack, from: from, upto: upto}, nil
}

// Asymmetry is the Henyey-Greenstein asymmetry the particles scatter with.
func (bent *Bent) Asymmetry() float64 { return bent.stack.asymmetry }

// rise is how far the plume's middle stands over the mouth at a distance
// down the wind: Briggs's buoyant rise, held at the lid.
func (stack Stack) rise(along float64) float64 {
	cubed := stack.wind * stack.wind * stack.wind
	climb := math.Pow(riseThree*stack.buoyancy*along*along/
		(riseTwo*entrained*entrained*cubed), oneThird)
	if stack.lid > nought {
		return math.Min(climb, stack.lid)
	}
	return climb
}

// spreads are the plume's spreads across the wind and up and down at a
// distance down it: the rising plume's, its radius over root two, joined
// in quadrature with the air's.
func (stack Stack) spreads(along float64) (across, upward float64) {
	growing := (stack.radius + entrained*stack.rise(along)) / math.Sqrt2
	kilometres := along / metresPerKilometre
	air := acrossFactor * math.Pow(kilometres, acrossPower)
	vertical := nearFactor*math.Pow(kilometres, nearPower) - nearLess
	if kilometres > unity {
		vertical = farFactor*math.Pow(kilometres, farPower) - farLess
	}
	vertical = math.Max(vertical, nought)
	return math.Hypot(growing, air), math.Hypot(growing, vertical)
}

// Bounds is the stretch's box: its length, and its widest section a few
// spreads either side of where its middle stands, held under the lid.
func (bent *Bent) Bounds() maths.Box {
	across, upward := bent.stack.spreads(bent.upto)
	top := bent.stack.rise(bent.upto) + gaussianReach*upward
	if bent.stack.lid > nought {
		top = math.Min(top, bent.stack.lid)
	}
	low := bent.stack.rise(bent.from) - gaussianReach*upward
	return corners(maths.V(bent.from, low, -gaussianReach*across),
		maths.V(bent.upto, top, gaussianReach*across))
}

// Density is the particles' extinction at a point of the stretch, and
// nought out of it and over the lid.
func (bent *Bent) Density(at maths.Vec) float64 {
	stack := bent.stack
	if !weatherFinite(at.Len()) || at.X < bent.from || at.X > bent.upto ||
		(stack.lid > nought && at.Y > stack.lid) {
		return nought
	}
	across, upward := stack.spreads(at.X)
	middle := stack.rise(at.X)
	height := bell(at.Y-middle, upward)
	if stack.lid > nought {
		height += bell(double*stack.lid-at.Y-middle, upward)
	}
	carried := stack.rate / (fullTurn * stack.wind * across * upward)
	return stack.extinction * carried * bell(at.Z, across) * height
}

// bell is a gaussian's height, one at its middle, at a distance from it.
func bell(distance, spread float64) float64 {
	return math.Exp(-halfSquare * distance * distance / (spread * spread))
}
