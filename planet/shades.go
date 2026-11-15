package planet

import (
	"math"
	"sort"
	"sync"

	"github.com/janearc/celestial/bands"
	theme "github.com/janearc/libtheme-css/primitives/bands"
	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/slayer/observer"
)

// a shade is what a face works out once for a band of its light, as none of
// it depends on the point asked: the band's middle in nanometres,
// the star's power in it, how much of sodium's d lines it holds, how much
// of the observer's X, Y and Z a unit of its power counts for, and its
// warmth.
type shade struct {
	centre     float64
	power      float64
	held       float64
	wx, wy, wz float64
	warm       warmth
}

// shadesOf is a shade for each band of a light, shortest first.
func shadesOf(light bands.Light) []shade {
	first, last := light.Span()
	var made []shade
	for band := first; band <= last; band++ {
		lo, hi := band.Edges()
		wx, wy, wz := weighed(theme.Band(band))
		made = append(made, shade{centre: math.Sqrt(lo * hi),
			power: light.Power(band), held: heldIn(lo, hi),
			wx: wx, wy: wy, wz: wz, warm: warmthOf(lo, hi)})
	}
	return made
}

// weighed is the observer's mean response across the whole nanometres in a
// band, as libtheme works it out: none for a band with no nanometre the
// observer lists.
func weighed(band theme.Band) (x, y, z float64) {
	lo, hi := band.Edges()
	count := 0
	for nm := int(math.Ceil(lo)); float64(nm) < hi+1; nm++ {
		if theme.Of(float64(nm)) != band {
			continue
		}
		mx, my, mz := swatch.Monochrome(nm).XYZ()
		x, y, z = x+mx, y+my, z+mz
		count++
	}
	if count == 0 {
		return 0, 0, 0
	}
	return x / float64(count), y / float64(count), z / float64(count)
}

// whiteUnit is one over D65's luminance, summed over every wavelength of
// its table in order, as libtheme works it out: the scale that gives
// D65 a Y of one.
var whiteUnit = sync.OnceValue(unitOfWhite)

// unitOfWhite is whiteUnit's sum.
func unitOfWhite() float64 {
	white := swatch.D65()
	order := make([]int, 0, len(white))
	for nm := range white {
		order = append(order, nm)
	}
	sort.Ints(order)
	var sum float64
	for _, nm := range order {
		_, my, _ := swatch.Monochrome(nm).XYZ()
		sum += white[nm] * my
	}
	return 1 / sum
}

// colourOf is the colour of a light given as its power in each of the
// shades, by the same sums libtheme makes: each band that has power weighed
// and added in order, then the sums scaled.
func colourOf(shades []shade, unit float64, powers []float64) swatch.Swatch {
	var x, y, z float64
	for i, p := range powers {
		if p == 0 {
			continue
		}
		x += p * shades[i].wx
		y += p * shades[i].wy
		z += p * shades[i].wz
	}
	return swatch.FromXYZ(x*unit, y*unit, z*unit)
}

// leaveOut says which bands the lights agreeing tries leave out: none, every
// second one, every third; powerStep is how much more each band gets than
// the last, so no two are alike. all are picked.
const (
	leaveOutNone  = 0
	leaveOutTwo   = 2
	leaveOutThree = 3
	powerStep     = 0.37
)

// agreeing is whether the shades give the same colour, in every bit, as
// the observer does, for lights that put a different power in every band, and
// that leave some out, and whether each band is where the observer puts it.
// a face that does not agree asks the observer for each point.
func agreeing(light bands.Light, shades []shade, unit float64) bool {
	first, _ := light.Span()
	for i, each := range shades {
		band := first + bands.Band(i)
		short, long := theme.Band(band).Edges()
		if bands.Of(each.centre) != band ||
			theme.Of(math.Sqrt(short*long)) != theme.Band(band) {
			return false
		}
	}
	for _, skip := range []int{leaveOutNone, leaveOutTwo, leaveOutThree} {
		powers := make([]float64, len(shades))
		lines := map[float64]float64{}
		for i, each := range shades {
			if skip == leaveOutNone || i%skip != 0 {
				powers[i] = 1 + float64(i)*powerStep
			}
			lines[each.centre] = powers[i]
		}
		got := colourOf(shades, unit, powers)
		want := observer.Swatch(bands.Lines(lines))
		if !sameSwatch(got, want) {
			return false
		}
	}
	return true
}

// sameSwatch is whether two swatches are equal in every bit.
func sameSwatch(a, b swatch.Swatch) bool {
	ax, ay, az := a.XYZ()
	bx, by, bz := b.XYZ()
	return math.Float64bits(ax) == math.Float64bits(bx) &&
		math.Float64bits(ay) == math.Float64bits(by) &&
		math.Float64bits(az) == math.Float64bits(bz)
}
