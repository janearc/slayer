package branching

import (
	"cmp"
	"errors"
	"math"

	"github.com/janearc/slayer/maths"
)

// Arrangement locates the things a limb carries in rows.
type Arrangement interface {
	Rows() int
	Across(row int) int
	At(on Size, row, index int, from Draws) (Place, error)
}

// Seed supplies the stable draws and child seeds used by recursive growth.
type Seed interface {
	Draws
	OnPiece(piece int) Seed
	Sprout(row, index int) Seed
}

// Carried is a growth and the arrangement that sets it on a limb.
type Carried struct {
	Where Arrangement
	What  Growth
}

// Growth is a limb carrying any number of recursively grown limbs.
type Growth struct {
	Stem    Limb
	Carries []Carried
}

// ErrGrowth marks a missing limb or arrangement.
var ErrGrowth = errors.New("manufacture: a growth needs a limb, and " +
	"somewhere for each thing it carries to go")

// NewGrowth checks the limb and its carried arrangements.
func NewGrowth(stem Limb, carries ...Carried) (Growth, error) {
	if stem.Pieces < 1 {
		return Growth{}, ErrGrowth
	}
	for _, borne := range carries {
		if borne.Where == nil {
			return Growth{}, ErrGrowth
		}
	}
	return Growth{Stem: stem, Carries: carries}, nil
}

// Parts traverses the tree in depth-first order. The caller turns each limb's
// sleeves into its own parts, at the point where they occur in the traversal.
func Parts[T any](grown Growth, on Place, from Seed,
	limbParts func(Limb, Size, []Place) ([]T, error)) ([]T, error) {
	runs, err := grown.Stem.Runs(on)
	if err != nil {
		return nil, err
	}
	parts, err := limbParts(grown.Stem, on.Size, runs)
	if err != nil {
		return nil, err
	}
	for piece, run := range runs {
		for _, borne := range grown.Carries {
			more, err := carriedParts(borne, run, on.Size, from.OnPiece(piece), limbParts)
			if err != nil {
				return nil, err
			}
			parts = append(parts, more...)
		}
	}
	return parts, nil
}

func carriedParts[T any](borne Carried, run Place, whole Size, from Seed,
	limbParts func(Limb, Size, []Place) ([]T, error)) ([]T, error) {
	var parts []T
	for row := range borne.Where.Rows() {
		for index := range borne.Where.Across(row) {
			there, err := borne.Where.At(run.Size, row, index, from)
			if err != nil {
				return nil, err
			}
			there.Space = there.Space.Within(run.Space)
			if there.Size == (Size{}) {
				there.Size = whole
			}
			if borne.What.Stem.Flat() {
				there.Space = LyingAlong(there.Space, run.Space.Orientation.Out(maths.YAxis))
			}
			more, err := Parts(borne.What, there, from.Sprout(row, index), limbParts)
			if err != nil {
				return nil, err
			}
			parts = append(parts, more...)
		}
	}
	return parts, nil
}

// LyingAlong turns a blade into the plane of its carrier and its own run.
func LyingAlong(space maths.Subspace, toward maths.Vec) maths.Subspace {
	run := space.Orientation.Out(maths.YAxis)
	across := toward.Sub(run.Scale(toward.Dot(run)))
	if length := across.Len(); !(length > 0) || math.IsInf(length, 0) {
		return space
	}
	across = across.Unit()
	space.Orientation = maths.Orientation{X: across, Y: run, Z: across.Cross(run)}
	return space
}

// StemCarrying grows a stem whose pieces each carry a second limb.
func StemCarrying(stemPieces int, stemDroop float64, stemSkins []Sleeve,
	where Arrangement, pieces int, droop float64, skins []Sleeve) (Growth, error) {
	stem, stemError := NewLimb(stemPieces, 1, stemDroop, stemSkins...)
	bough, boughError := NewLimb(pieces, 1, droop, skins...)
	borne, growthError := NewGrowth(bough)
	if err := cmp.Or(stemError, boughError, growthError); err != nil {
		return Growth{}, err
	}
	return NewGrowth(stem, Carried{Where: where, What: borne})
}

// TrunkCarrying constructs a stem with whorled growth.
func TrunkCarrying(trunk Sleeve, count, lowest int, set Habit, what Growth) (Growth, error) {
	stem, err := NewLimb(1, 1, 0, trunk)
	if err != nil {
		return Growth{}, err
	}
	whorled, err := NewWhorls(count, lowest, set)
	if err != nil {
		return Growth{}, err
	}
	return NewGrowth(stem, Carried{Where: whorled, What: what})
}
