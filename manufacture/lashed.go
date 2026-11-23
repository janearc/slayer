package manufacture

import (
	"fmt"
	"math"
	"math/rand/v2"

	"github.com/janearc/slayer/manufacture/cities"
	"github.com/janearc/slayer/maths"
)

// LashedStreetKind is the name of the kind lashed-street: a street of
// plain blocks with lashings across it, cables hung with lanterns and
// signs. The lashings are a decoration over any group of buildings
// (package cities, NewLashings); this kind gives them a group of its own,
// two rows of plain blocks, so that a set can draw them without one.
const LashedStreetKind = "lashed-street"

// lashedSignStream is the stream of draws a lashed street takes the parts
// that stand in its sign slots from, which no other use of its seed shares.
const lashedSignStream = 17

// LashingsSpec is what a set says of a lashed street: how thickly it is
// hung, from nought to one, if DensitySaid; the width of the street and its
// length in metres, nought for the culture's own; and the name of the kind
// that stands in the slot of each sign, or nothing for plain glowing
// panels.
//
// Any kind of the manufactory may be a sign, but a lashed street,
// a cluster and a superblock. Style is how it is hung, one of
// cities.Styles, or nothing for the festival.
type LashingsSpec struct {
	Density     float64
	DensitySaid bool
	Street      float64
	Length      float64
	Sign        string
	Style       string
}

// LashedStreet is the kind lashed-street with a recipe's density, street
// and sign. It draws a street once to check them, and refuses what it
// cannot keep: a density that is not from nought to one, a street below
// nought, and a sign that is no kind of part.
func LashedStreet(spec LashingsSpec) (Kind, error) {
	known := table()
	if _, made := known[spec.Sign]; spec.Sign != "" && (!made ||
		spec.Sign == ClusterKind || spec.Sign == SuperblockKind ||
		spec.Sign == LashedStreetKind) {
		return Kind{}, fmt.Errorf("manufacture: a lashed street hangs "+
			"%q as its signs, which is no kind of part", spec.Sign)
	}
	row := lashedStreetRow{spec: spec}
	if _, _, err := row.drawn(sow(0)); err != nil {
		return Kind{}, err
	}
	return Kind{name: LashedStreetKind, how: row}, nil
}

// lashedStreetRow is the kind lashed-street: its blocks and the lashings
// over them, drawn by package cities from the seed and made a good here,
// with a part of the sign kind in each slot if there is one.
type lashedStreetRow struct{ spec LashingsSpec }

// drawn is the blocks and the lashings of a seed, before they are made a
// good.
func (row lashedStreetRow) drawn(from seed) (*cities.PlainRows,
	*cities.Lashings, error) {
	rows, err := cities.NewPlainRows(cities.RowsAsk{Width: row.spec.Street,
		Length: row.spec.Length}, cities.Seed(from.number))
	if err != nil {
		return nil, nil, err
	}
	density := row.spec.Density
	if !row.spec.DensitySaid {
		if density, err = cities.DefaultDensity(); err != nil {
			return nil, nil, err
		}
	}
	lashings, err := cities.NewLashings(cities.LashingsAsk{Density: density,
		Sign: row.spec.Sign, Style: row.spec.Style}, rows.Volumes,
		cities.Seed(from.number))
	return rows, lashings, err
}

// made is the lashed street at the quick detail level.
func (row lashedStreetRow) made(from seed) (good, error) {
	return row.madeAt(from, "quick")
}

// madeAt is the lashed street at the named detail level. It returns
// errGood when the seed is not made.
func (row lashedStreetRow) madeAt(from seed, detail string) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	rows, lashings, err := row.drawn(from)
	if err != nil {
		return good{}, err
	}
	design := rows.Design()
	design.Elements = append(design.Elements, lashings.Design().Elements...)
	drawn, err := design.Rendered(cities.Detail(detail))
	if err != nil {
		return good{}, err
	}
	whole, err := goodOfRendered(drawn)
	if err != nil || row.spec.Sign == "" || len(lashings.Slots) == 0 {
		return whole, err
	}
	return row.signed(whole, lashings.Slots, from, detail)
}

// signed is the street with a part of the sign kind in each slot, each
// drawn from the seed in the order the slots are in.
func (row lashedStreetRow) signed(whole good, slots []cities.Slot, from seed,
	detail string) (good, error) {
	rng := rand.New(rand.NewPCG(from.number, lashedSignStream))
	goods := []good{whole}
	for _, slot := range slots {
		part, err := partOf(row.spec.Sign, rng.Uint64(), maths.Identity,
			detail)
		if err != nil {
			return good{}, err
		}
		placed, err := part.turned.placed(inSlot(part, slot))
		if err != nil {
			return good{}, err
		}
		goods = append(goods, placed)
	}
	return gather(goods...)
}

// inSlot is where a part stands to fill a slot: turned so that its wider
// horizontal side runs along the slot's width, scaled as much as fits
// the slot's width and height both, and its middle at the slot's middle.
func inSlot(part scavengedPart, slot cities.Slot) maths.Subspace {
	size := part.piece.Box.Max.Sub(part.piece.Box.Min)
	wide, turn := size.X, maths.Orientation{X: slot.Along, Y: maths.YAxis,
		Z: slot.Along.Cross(maths.YAxis)}
	if size.Z > size.X {
		wide = size.Z
		turn = maths.Orientation{X: maths.YAxis.Cross(slot.Along),
			Y: maths.YAxis, Z: slot.Along}
	}
	scale := math.Min(slot.Width/wide, slot.Height/size.Y)
	middle := part.piece.Box.Min.Add(part.piece.Box.Max).Scale(half)
	return maths.NewSubspace(slot.Middle.Sub(turn.Out(middle).Scale(scale)),
		turn, scale)
}

// half is the middle of a share.
const half = 0.5
