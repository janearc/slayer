package branching

import (
	"errors"
	"math"

	"github.com/janearc/celestial/measure"

	"github.com/janearc/slayer/maths"
)

// ErrWhorls marks an invalid row or branch count.
var ErrWhorls = errors.New("manufacture: whorls need one whorl or " +
	"more, and one part or more in the lowest")

// ErrStem marks a stem whose dimensions cannot carry a whorl.
var ErrStem = errors.New("manufacture: whorls stand on a stem with a " +
	"length and a girth above nought, each with an end")

// ErrNoRun marks an absent or nonfinite direction.
var ErrNoRun = errors.New("manufacture: a run needs a way to go, with " +
	"an end")

// Whorls arranges rows of branches around a carrier.
// NewWhorls checks the counts and habit before construction.
type Whorls struct {
	Count, Lowest int
	Habit         Habit
}

// NewWhorls checks branch counts and their arrangement habit.
func NewWhorls(count, lowest int, habit Habit) (Whorls, error) {
	if count < 1 || lowest < 1 {
		return Whorls{}, ErrWhorls
	}
	if err := habit.Sound(); err != nil {
		return Whorls{}, err
	}
	return Whorls{Count: count, Lowest: lowest, Habit: habit}, nil
}

func (up Whorls) climb(row int) float64 {
	return (float64(row) + .5) / float64(up.Count)
}

// Rows is the number of rows on the carrier.
func (up Whorls) Rows() int { return up.Count }

// Across counts the branches on one row.
func (up Whorls) Across(row int) int {
	kept := 1 - up.Habit.Thinning*up.climb(row)
	return max(up.Habit.Fewest, int(math.Round(float64(up.Lowest)*kept)))
}

// At picks a branch's physical dimensions and local pose from stable draws.
func (up Whorls) At(stem Size, row, index int, from Draws) (Place, error) {
	length, girth := stem.Length.InMetres(), stem.Girth.InMetres()
	set, climb := up.Habit, up.climb(row)
	if !(length > 0) || math.IsInf(length, 0) || !(girth > 0) || math.IsInf(girth, 0) {
		return Place{}, ErrStem
	}
	height := length * (set.Start + (set.Top-set.Start)*climb)
	gap := (2 * math.Pi) / float64(up.Across(row))
	turn := float64(row)*set.Turn.InRadians() + gap*float64(index) +
		set.Stray*from.Swing(row, index, 0)*gap
	down := set.Low.InRadians() - set.Lift.InRadians()*climb +
		set.Sway.InRadians()*from.Swing(row, index, 1)
	reach := set.Reach * length * (1 - set.Shortening*climb) *
		(1 + set.Uneven*from.Swing(row, index, 2))
	stemHere := girth * (1 - height/length)
	way := maths.V(math.Sin(down)*math.Cos(turn), math.Cos(down),
		-math.Sin(down)*math.Sin(turn))
	space, err := SpaceAlong(maths.V(0, height, 0), way)
	if err != nil {
		return Place{}, err
	}
	thick := math.Max(set.Girth*stemHere, set.Least*length)
	if set.Own > 0 {
		thick = math.Min(set.Own*reach, set.Girth*stemHere)
	}
	return Place{Space: space, Size: Size{Length: measure.Metres(reach), Girth: measure.Metres(thick)}}, nil
}

// SpaceAlong places a right-handed local frame along a finite run.
func SpaceAlong(at, run maths.Vec) (maths.Subspace, error) {
	length := run.Len()
	if !(length > 0) || math.IsInf(length, 0) {
		return maths.Subspace{}, ErrNoRun
	}
	axis := run.Unit()
	side := maths.XAxis
	if math.Abs(axis.X) >= .9 {
		side = maths.ZAxis.Neg()
	}
	side = side.Sub(axis.Scale(side.Dot(axis))).Unit()
	return maths.Subspace{Origin: at, Orientation: maths.Orientation{
		X: side, Y: axis, Z: side.Cross(axis)}}, nil
}
