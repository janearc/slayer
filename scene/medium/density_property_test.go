package medium

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
	"pgregory.net/rapid"
)

// densityPropertyRanges is the ranges the properties draw fields and points
// from: a size from least to most, a point from minus to plus reach, a count
// of at most counts, a time of at most seconds, and a share up to ratio.
type densityPropertyRanges struct {
	least, most, reach, ratio, seconds float64
	counts                             int
}

// densityPropertyRanged is the ranges the properties draw from, from
// numbers.yaml.
func densityPropertyRanged(t *testing.T) densityPropertyRanges {
	t.Helper()
	return densityPropertyRanges{
		least:   picked(t, "density-property-least"),
		most:    picked(t, "density-property-most"),
		reach:   picked(t, "density-property-reach"),
		ratio:   picked(t, "density-property-ratio"),
		seconds: picked(t, "density-property-seconds"),
		counts:  pickedCount(t, "density-property-counts")}
}

// densityPropertyOrigin is the middle of space.
func densityPropertyOrigin() maths.Vec {
	return maths.V(nought, nought, nought)
}

// size is a size within the property's range.
func (ranges densityPropertyRanges) size(draw *rapid.T, label string) float64 {
	return rapid.Float64Range(ranges.least, ranges.most).Draw(draw, label)
}

// share is a number from nought to the ratio.
func (ranges densityPropertyRanges) share(draw *rapid.T, label string) float64 {
	return rapid.Float64Range(nought, ranges.ratio).Draw(draw, label)
}

// coordinate is a number within the property's reach of nought.
func (ranges densityPropertyRanges) coordinate(draw *rapid.T,
	label string) float64 {
	return rapid.Float64Range(-ranges.reach, ranges.reach).Draw(draw, label)
}

// count is a count of nought up to the property's most.
func (ranges densityPropertyRanges) count(draw *rapid.T) int {
	return rapid.IntRange(nought, ranges.counts).Draw(draw, "count")
}

// point is a point within the property's reach of the origin.
func (ranges densityPropertyRanges) point(draw *rapid.T) maths.Vec {
	return maths.V(ranges.coordinate(draw, "x"), ranges.coordinate(draw, "y"),
		ranges.coordinate(draw, "z"))
}

// shells is the body-bound media: shells about a body, with a thin one
// between low and high.
func (ranges densityPropertyRanges) shells(draw *rapid.T) []Medium {
	radius, low := ranges.size(draw, "radius"), ranges.size(draw, "low")
	high := low + ranges.size(draw, "depth")
	return []Medium{
		Air{Radius: radius, Height: ranges.size(draw, "height"),
			Thick: ranges.size(draw, "thick")},
		Aurora{Radius: radius, Low: low, High: high,
			Lat:   rapid.Float64Range(nought, math.Pi/double).Draw(draw, "lat"),
			Width: ranges.share(draw, "width") + ranges.least,
			Axis:  ranges.point(draw), Thick: ranges.size(draw, "thick"),
			Curtains: ranges.share(draw, "curtains"),
			Seed:     rapid.Uint64().Draw(draw, "seed")},
		Clouds{Radius: radius, Low: low, High: high,
			Cover: ranges.share(draw, "cover"),
			Thick: ranges.size(draw, "thick"),
			Seed:  rapid.Uint64().Draw(draw, "seed")},
		Giant{Radius: radius, Depth: ranges.size(draw, "giant depth"),
			Thick: ranges.size(draw, "thick")},
		Puff{Radius: radius, Thick: ranges.size(draw, "thick")},
	}
}

// streams is the media that run along y: jets, tails, plumes, fizz and
// pillars, with their dressings in their ranges.
func (ranges densityPropertyRanges) streams(draw *rapid.T) []Medium {
	length := ranges.size(draw, "length")
	return []Medium{
		Jet{Length: length, Base: ranges.size(draw, "base"),
			Spread: ranges.share(draw, "spread"),
			Thick:  ranges.size(draw, "thick"),
			From:   length * ranges.share(draw, "from"),
			Knots:  ranges.share(draw, "knots"),
			Seed:   rapid.Uint64().Draw(draw, "seed"),
			Flow:   ranges.size(draw, "flow"), Time: ranges.size(draw, "time")},
		Tail{Length: length, Base: ranges.size(draw, "base"),
			Spread: ranges.share(draw, "spread"),
			Curve:  double*ranges.share(draw, "curve") - ranges.ratio,
			Thick:  ranges.size(draw, "thick")},
		Plume{Height: length, Width: ranges.size(draw, "width"),
			Thick: ranges.size(draw, "thick")},
		Fizz{Radius: ranges.size(draw, "radius"), Height: length,
			Thick: ranges.size(draw, "thick"), Rise: ranges.size(draw, "rise"),
			Churn: ranges.share(draw, "churn"),
			Seed:  rapid.Uint64().Draw(draw, "seed")},
		Pillars{Count: ranges.count(draw), Height: length,
			Width:     ranges.size(draw, "width"),
			Thick:     ranges.size(draw, "thick"),
			Seed:      rapid.Uint64().Draw(draw, "seed"),
			Filaments: ranges.share(draw, "filaments")},
	}
}

// disc is a disc of a profile of up to a few entries, each from nought to
// a few, with the grain and ringlets in their ranges.
func (ranges densityPropertyRanges) disc(draw *rapid.T) Medium {
	inner := ranges.size(draw, "inner")
	profile := rapid.SliceOfN(rapid.Float64Range(nought, ranges.most), nought,
		ranges.counts)
	return Disc{Inner: inner, Outer: inner + ranges.size(draw, "ring"),
		Thick: ranges.size(draw, "thick"), Dense: ranges.size(draw, "dense"),
		Profile:  profile.Draw(draw, "profile"),
		Grain:    ranges.share(draw, "grain"),
		Ringlets: ranges.size(draw, "ringlets"),
		Seed:     rapid.Uint64().Draw(draw, "seed")}
}

// wisps is a puff made ragged, its amount and scale in range.
func (ranges densityPropertyRanges) wisps(draw *rapid.T) Medium {
	return Wispy{Of: Puff{Radius: ranges.size(draw, "radius"),
		Thick: ranges.size(draw, "thick")},
		Seed:   rapid.Uint64().Draw(draw, "seed"),
		Scale:  ranges.size(draw, "scale"),
		Amount: ranges.share(draw, "amount")}
}

// media is one of every plain-struct medium, fields drawn within the
// ranges the documentation gives them. These media have no constructor and
// check nothing, so the ranges are the ones a caller is trusted to keep.
func (ranges densityPropertyRanges) media(draw *rapid.T) []Medium {
	media := append(ranges.shells(draw), ranges.streams(draw)...)
	return append(media, ranges.disc(draw), ranges.wisps(draw))
}

// check draws a time, a point and the plain media, and stops the property at
// the first density that is no number, infinite or below nought.
func (ranges densityPropertyRanges) check(draw *rapid.T) {
	seconds := rapid.Float64Range(nought, ranges.seconds).Draw(draw, "seconds")
	at := time.Duration(seconds * float64(time.Second))
	point := ranges.point(draw)
	for _, each := range ranges.media(draw) {
		if moving, ok := each.(Timed); ok {
			each = moving.Tick(at)
		}
		density := each.Density(point)
		if math.IsNaN(density) || math.IsInf(density, nought) ||
			density < nought {
			draw.Fatalf("%T %+v is %v at %v", each, each, density, point)
		}
	}
}

// every plain-struct medium's density is a finite number of at least zero at
// any point, over fields in their documented ranges, at the time it is
// ticked to.
func TestPlainMediaAreFiniteAndNeverNegative(t *testing.T) {
	rapid.Check(t, densityPropertyRanged(t).check)
}

// the property is bounded by its ranges, and the check sees past them: a
// medium made of a field outside its range is not refused, so a negative
// Thick gives a negative density. This is why the property is drawn from the
// documented ranges, and why a caller must keep to them.
func TestAFieldOutsideItsRangeIsNotRefused(t *testing.T) {
	negative := Puff{Radius: unity, Thick: -unity}
	if got := negative.Density(densityPropertyOrigin()); got >= nought {
		t.Errorf("a puff of negative Thick is %v at its middle", got)
	}
	unreadable := Puff{Radius: unity, Thick: math.NaN()}
	if got := unreadable.Density(densityPropertyOrigin()); !math.IsNaN(got) {
		t.Errorf("a puff of Thick no number is %v at its middle", got)
	}
}
