package sky

import (
	"errors"
	"math"

	"github.com/janearc/slayer/maths"
)

// The catalogue's index: the sky cut into bands of equal height from
// pole to pole, each band into sectors of equal turn, so every cell has
// the same area. A star is kept in the cell it lies in.
const (
	catalogueBands   = 64  // picked: bands from the south pole to the north; a band is about 2.8 degrees at the equator
	catalogueSectors = 128 // picked: sectors round a band; a sector is about 2.8 degrees at the equator
	catalogueReach   = 1   // defined: a look searches its own cell and the ones beside it
	capBands         = 3   // picked: this near a pole a sector is narrow, and the whole of its band is searched
	widestStar       = 1.0 // picked: degrees; a star drawn wider than this would be missed in the cell beside, and is refused
)

// Star is one star of a catalogue: the way to it, a unit vector in the
// sky's own frame, and its light.
type Star struct {
	Toward maths.Vec
	Light  maths.Light
}

// Catalogue is a sky of real stars: each a small disc of its own light
// in its own place, and between them the dark. It is made by
// NewCatalogue and is not changed after.
type Catalogue struct {
	stars []Star
	cells [catalogueBands][catalogueSectors][]int32
	near  float64
	dark  maths.Light
}

// errCatalogue refuses a catalogue that cannot be drawn.
var errCatalogue = errors.New("sky: a catalogue wants stars with a way to " +
	"each, and a size above nought and no more than a degree")

// cell is the band and the sector a way lies in.
func cell(toward maths.Vec) (band, sector int) {
	high := (toward.Z + 1) / 2
	band = min(int(high*catalogueBands), catalogueBands-1)
	round := math.Atan2(toward.Y, toward.X)/(2*math.Pi) + 0.5
	sector = min(int(round*catalogueSectors), catalogueSectors-1)
	return max(band, 0), max(sector, 0)
}

// NewCatalogue is a sky of the stars given, each drawn as a disc so many
// radians in radius, over a dark. It refuses no stars, a star with no
// way to it or a light that is not a number, and a size that is not
// above nought or is more than a degree.
func NewCatalogue(stars []Star, size float64, dark maths.Light) (*Catalogue, error) {
	if len(stars) == 0 || !(size > 0) || size > widestStar*maths.Degree {
		return nil, errCatalogue
	}
	made := &Catalogue{near: math.Cos(size), dark: dark}
	for at, each := range stars {
		long := each.Toward.Len()
		sum := each.Light.R + each.Light.G + each.Light.B
		if !(long > 0) || math.IsInf(long, 0) || math.IsNaN(sum) || math.IsInf(sum, 0) {
			return nil, errCatalogue
		}
		each.Toward = each.Toward.Scale(1 / long)
		made.stars = append(made.stars, each)
		band, sector := cell(each.Toward)
		made.cells[band][sector] = append(made.cells[band][sector], int32(at))
	}
	return made, nil
}

// Light is the dark, and the light of every star whose disc the way
// falls in.
func (made *Catalogue) Light(toward maths.Vec) maths.Light {
	long := toward.Len()
	if !(long > 0) {
		return made.dark
	}
	toward = toward.Scale(1 / long)
	arrives := made.dark
	band, sector := cell(toward)
	for b := max(band-catalogueReach, 0); b <= min(band+catalogueReach, catalogueBands-1); b++ {
		from, to := sector-catalogueReach, sector+catalogueReach
		if b < capBands || b >= catalogueBands-capBands {
			from, to = 0, catalogueSectors-1
		}
		for s := from; s <= to; s++ {
			wrapped := (s + catalogueSectors) % catalogueSectors
			for _, at := range made.cells[b][wrapped] {
				if made.stars[at].Toward.Dot(toward) >= made.near {
					arrives = arrives.Add(made.stars[at].Light)
				}
			}
		}
	}
	return arrives
}

// Count is how many stars the catalogue holds.
func (made *Catalogue) Count() int { return len(made.stars) }
