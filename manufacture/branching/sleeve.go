package branching

import (
	"errors"
	"math"
)

// Joint describes a radius as shares of a limb's girth and length.
type Joint struct{ OfGirth, OfLength float64 }

// Radius resolves the joint's shares against a physical limb size.
func (at Joint) Radius(length, girth float64) float64 {
	return at.OfGirth*girth + at.OfLength*length
}

// Sleeve is a round or flat skin along a limb, with a material role.
type Sleeve struct {
	Joints   []Joint
	Sides    int
	Flat     bool
	Role     int
	Terminal bool
}

// ErrSleeve marks a sleeve with an invalid cut or joint profile.
var ErrSleeve = errors.New("manufacture: a sleeve needs a cut, two " +
	"joints or more, shares of nought or above, and a radius above " +
	"nought at every joint but the last")

// NewSleeve checks a round skin's joints and its cut.
func NewSleeve(joints []Joint, sides, role int) (Sleeve, error) {
	if sides < 3 || !Jointed(joints) {
		return Sleeve{}, ErrSleeve
	}
	return Sleeve{Joints: joints, Sides: sides, Role: role}, nil
}

// NewLeaf checks a flat skin's joints.
func NewLeaf(joints []Joint, role int) (Sleeve, error) {
	if !Jointed(joints) {
		return Sleeve{}, ErrSleeve
	}
	return Sleeve{Joints: joints, Flat: true, Role: role}, nil
}

// Jointed checks that each joint has finite nonnegative shares and that all
// but the last have a positive radius.
func Jointed(joints []Joint) bool {
	if len(joints) < 2 {
		return false
	}
	for index, at := range joints {
		last := index == len(joints)-1
		shares := at.OfGirth >= 0 && at.OfLength >= 0 &&
			!math.IsNaN(at.OfGirth) && !math.IsInf(at.OfGirth, 0) &&
			!math.IsNaN(at.OfLength) && !math.IsInf(at.OfLength, 0)
		none := at.OfGirth == 0 && at.OfLength == 0
		if !shares || (!last && none) {
			return false
		}
	}
	return true
}
