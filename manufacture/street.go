package manufacture

import (
	"fmt"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/strategies"
)

// streetKind is the name of the kind street: a carriageway with a kerb and
// a paved footway down each side, a painted centre line and a crossing,
// gully gratings in the gutters, and, as the street grows wet, a film of
// water on every surface and puddles in the low places of the road.
//
// The street runs along the thing's own z axis, from north to south, and
// is centred on its origin. Its walls are the set's, not the street's: the
// footways end at x of plus and minus six, which is where a twelve metre
// street's buildings stand.
const streetKind = "street"

// the street's size, in metres. every number is picked, as the kinds'
// rows are, to be a street a car and a pavement fit in.
const (
	// streetDefaultLength is the street's length when a set says none.
	streetDefaultLength = 36.0
	// streetLeastLength is the shortest street, a crossing and a dash.
	streetLeastLength = 8.0
	// streetMostLength keeps a street under the face limit of one thing.
	streetMostLength = 100.0
	// streetRoadHalf is half the width of the carriageway.
	streetRoadHalf = 3.5
	// streetKerbWide is how wide the kerb stone is.
	streetKerbWide = 0.2
	// streetFootway is how wide each footway is.
	streetFootway = 2.3
	// streetCamber is how far the carriageway falls from its crown to
	// its gutters, so that rain runs off it.
	streetCamber = 0.06
	// streetKerbUp is how high the kerb and the footway stand above the
	// crown of the road.
	streetKerbUp = 0.05
	// streetFooting is how deep the kerb and footway stones go below
	// their tops.
	streetFooting = 0.3
	// streetCellAcross and streetCellAlong are the sides of one cell of
	// the road's grid. picked: a cell is smaller than the least puddle.
	streetCellAcross = 0.14
	streetCellAlong  = 0.2
)

// StreetSpec is what a set says of a street recipe: how wet it is, from
// nought, dry, to one, flooded, and how long it is in metres, or nought
// for the default.
type StreetSpec struct {
	Wetness float64
	Length  float64
}

// streetRow is the kind street. Without a recipe it is a dry street of
// the default length.
type streetRow struct{ spec StreetSpec }

// Street is the kind street with a recipe's wetness and length. It refuses
// a wetness outside nought to one, and a length outside the street's
// least and most.
func Street(spec StreetSpec) (Kind, error) {
	if !(spec.Wetness >= 0 && spec.Wetness <= 1) {
		return Kind{}, fmt.Errorf("manufacture: a street's wetness is "+
			"from nought to one, not %g", spec.Wetness)
	}
	if spec.Length != 0 && !(spec.Length >= streetLeastLength &&
		spec.Length <= streetMostLength) {
		return Kind{}, fmt.Errorf("manufacture: a street is from %g to %g "+
			"metres long, not %g", streetLeastLength, streetMostLength,
			spec.Length)
	}
	return Kind{name: streetKind, how: streetRow{spec: spec}}, nil
}

// length is the street's length, in metres.
func (row streetRow) length() float64 {
	if row.spec.Length == 0 {
		return streetDefaultLength
	}
	return row.spec.Length
}

// made is the street at the quick detail level.
func (row streetRow) made(from seed) (good, error) {
	return row.madeAt(from, "quick")
}

// madeAt is the street at the named detail level. Every level is the
// same street: its cells are already smaller than anything laid on it. It
// returns errGood when the seed is not made.
func (row streetRow) madeAt(from seed, _ string) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	ground := newStreetGround(row.length(), row.spec.Wetness, from)
	assembly := &instrumentAssembly{fine: cut{around: leastAround},
		object: good{dressed: ground.looks(from)}}
	ground.build(assembly)
	made, err := assembly.finish(ground.bounds())
	if err != nil {
		return good{}, err
	}
	return *made, nil
}

// bounds is the box the street lies in: its footways, the footing below
// them and the flood above.
func (ground streetGround) bounds() maths.Box {
	return streetBox(ground.length)
}

// streetBox is the box a street of a length lies in, from its recipe's
// numbers alone.
func streetBox(length float64) maths.Box {
	across := streetRoadHalf + streetKerbWide + streetFootway
	return maths.Box{
		Min: maths.V(-across, streetKerbUp-streetFooting, -length/2),
		Max: maths.V(across, streetKerbUp, length/2),
	}
}

// extent is the street's box: the same whatever the seed.
func (row streetRow) extent(from seed) (maths.Box, error) {
	return streetBox(row.length()), nil
}

// seen is how an eye sees the street, from its extent.
func (row streetRow) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}
