package film_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/ninjafilm/data"
	"github.com/janearc/slayer/ninjafilm/film"
	"pgregory.net/rapid"
)

// the tables the laser tests are run over
const (
	laserAlongFile  = "testdata/laser-along.csv"
	laserAcrossFile = "testdata/laser-across.csv"
	laserBoxFile    = "testdata/laser-box.csv"
)

// what the laser tests choose, each picked in picked-numbers.yml
var (
	laserReach  = data.Picked("laser-reach")
	laserCutoff = data.Picked("laser-cutoff-widths")
)

// Lasers light their axis from just past the source out to the end of
// their reach. They light the end too. They light nothing behind the
// source or past the reach. Lasers of no aim point down minus z. They
// reach a thousand units.
func TestALasersBeamLiesAlongItsAim(t *testing.T) {
	for _, row := range table(t, laserAlongFile) {
		glass := film.NewLaser(maths.Vec{}, vector(t, row, "aim"), whole,
			maths.Grey(whole), number(t, row, "length"))
		aim := vector(t, row, "aim")
		if aim == (maths.Vec{}) {
			aim = maths.V(none, none, -whole)
		}
		along := number(t, row, "along")
		_, _, light := glass.Light(aim.Unit().Scale(along), maths.Up)
		if lit := number(t, row, "lit"); (light.R > none) != (lit > none) {
			t.Errorf("%s: %v", label(row), light)
		}
	}
}

// Lasers fall off across their width by a gaussian. About a seventh is
// left a width out, which is e to the minus two. Three widths out is the
// last glimmer. Nothing is past that, and nothing is lit by a beam of no
// width.
func TestALasersBeamFallsOffAcrossItsWidth(t *testing.T) {
	for _, row := range table(t, laserAcrossFile) {
		glass := film.NewLaser(maths.Vec{}, maths.V(whole, none, none),
			number(t, row, "width"), maths.Grey(whole), laserReach)
		across := maths.V(laserReach/pair, number(t, row, "across"), none)
		_, _, light := glass.Light(across, maths.Up)
		want := number(t, row, "share")
		if math.Abs(light.R-want) > loose*want {
			t.Errorf("%s: %v, want %v", label(row), light.R, want)
		}
	}
}

// aligned draws a laser along a drawn unit direction from a drawn place.
// It has a width and a reach.
func aligned(t *rapid.T) (film.Laser, maths.Vec, maths.Vec, float64, float64) {
	at, aim := place(t, "at"), apart(t, "aim", maths.Vec{}).Unit()
	width := rapid.Float64Range(wideLeast, whole).Draw(t, "width")
	length := rapid.Float64Range(whole, eyeMost).Draw(t, "length")
	made := film.NewLaser(at, aim, width, maths.Grey(whole), length)
	return made, at, aim, width, length
}

// crossing is the plane through the axis at right angles to an aim: the
// way across it that a drawn offset goes.
func crossing(aim maths.Vec) maths.Vec {
	side := aim.Cross(maths.Up)
	if side.Len() < loose {
		side = aim.Cross(maths.XAxis)
	}
	return side.Unit()
}

// The light of a beam at a distance along and an offset across is the
// gaussian of the offset over the width. The way back is down the beam.
func TestALasersLightIsAGaussianOfTheOffsetOverTheWidth(t *testing.T) {
	rapid.Check(t, laserLightIsAGaussian)
}

func laserLightIsAGaussian(t *rapid.T) {
	glass, at, aim, width, length := aligned(t)
	along := rapid.Float64Range(half, length*insideMost).Draw(t, "along")
	share := rapid.Float64Range(none, laserCutoff*insideMost).Draw(t, "across")
	across := crossing(aim).Scale(share * width)
	point := at.Add(aim.Scale(along)).Add(across)
	toward, far, light := glass.Light(point, maths.Up)
	want := math.Exp(-pair * share * share)
	if math.Abs(light.R-want) > loose || !toward.Near(aim.Neg(), loose) ||
		math.Abs(far-along) > loose*length {
		t.Fatalf("%v along and %v widths across: %v, %v, %v, want %v", along,
			share, toward, far, light, want)
	}
}

// Every point a beam lights is inside the box of its bounds.
func TestEveryPointALaserLightsIsInsideItsBounds(t *testing.T) {
	rapid.Check(t, litPointsAreInsideBounds)
}

func litPointsAreInsideBounds(t *rapid.T) {
	glass, at, aim, width, length := aligned(t)
	along := rapid.Float64Range(none, length).Draw(t, "along")
	share := rapid.Float64Range(none, laserCutoff).Draw(t, "across")
	point := at.Add(aim.Scale(along)).Add(crossing(aim).Scale(share * width))
	if _, _, light := glass.Light(point, maths.Up); light.R <= none {
		return
	}
	box := glass.Bounds()
	if point.X < box.Min.X-loose || point.X > box.Max.X+loose ||
		point.Y < box.Min.Y-loose || point.Y > box.Max.Y+loose ||
		point.Z < box.Min.Z-loose || point.Z > box.Max.Z+loose {
		t.Fatalf("%v is lit and outside %v", point, box)
	}
}

// The box of a beam is its length and three widths round it.
func TestALasersBoxIsItsLengthAndThreeWidthsRound(t *testing.T) {
	for _, row := range table(t, laserBoxFile) {
		glass := film.NewLaser(vector(t, row, "at"), vector(t, row, "aim"),
			number(t, row, "width"), maths.Grey(whole),
			number(t, row, "length"))
		box := glass.Bounds()
		if !box.Min.Near(vector(t, row, "min"), loose) ||
			!box.Max.Near(vector(t, row, "max"), loose) {
			t.Errorf("%s: the box is %v", label(row), box)
		}
	}
}
