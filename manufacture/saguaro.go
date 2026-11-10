package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// saguaro is authored desert scenery, not an age or water-storage model.
type saguaro struct{}

// cactusColumn retains the physical path independently of mesh detail.
type cactusColumn struct {
	stem cactusStem
	pose maths.Subspace
}

// cactusValue keeps all picked cactus controls in the shared Pzm number book.
func cactusValue(name string) float64 { return plantValue("cactus-" + name) }

// made supplies the ordinary quick specimen.
func (plant saguaro) made(from seed) (good, error) { return plant.madeAt(from, "quick") }

// madeAt keeps arms, caps and reserved spine extent stable across details.
func (plant saguaro) madeAt(from seed, detail string) (good, error) {
	if !from.made() {
		//ninja:vow feint maker needs a value; Things refuses this one
		return good{}, errGood
	}
	if err := validCactusChoices(); err != nil {
		//ninja:vow feint maker needs a value; Things refuses this one
		return good{}, err
	}
	columns := plant.columns(from)
	made := plant.body(columns, detail)
	return plant.spines(made, columns, detail)
}

// body builds the ordinary skin and reserves the finest physical envelope.
func (plant saguaro) body(columns []cactusColumn, detail string) good {
	made := good{dressed: looks{
		body: plain("cactus skin", cactusValue("green-r"), cactusValue("green-g"), cactusValue("green-b")),
		leaf: plain("cactus spines", cactusValue("spine-r"), cactusValue("spine-g"), cactusValue("spine-b")),
	}}
	first := columns[spaceZero]
	extent := transformed(first.stem.bounds(), first.pose)
	for _, column := range columns {
		stem, fine := column.surface(detail)
		made.parts = append(made.parts, part{shape: stem, fine: fine, space: column.pose, of: body})
		box := transformed(column.stem.bounds(), column.pose)
		extent = enclosing(extent, box)
	}
	made.extent = &extent
	return made
}

// columns selects zero or more arms around a single upright stem.
func (plant saguaro) columns(from seed) []cactusColumn {
	height := cactusPick(from, "height", spaceZero)
	radius := cactusPick(from, "radius", spaceOne)
	columns := []cactusColumn{{stem: cactusStem{run: height, radius: radius,
		ribs: int(cactusValue("ribs")), depth: cactusValue("pleat-depth")}, pose: maths.Local()}}
	counts := span{least: cactusValue("arms-minimum"), most: cactusValue("arms-maximum")}
	count := counts.count(from.draw(picking, spaceZero, intPairCount))
	for index := range count {
		columns = append(columns, plant.arm(from, index, count, height, radius))
	}
	return columns
}

// cactusPick samples a named, picked physical range without coupling channels.
func cactusPick(from seed, name string, index int) float64 {
	rangeOf := span{least: cactusValue(name + "-minimum"), most: cactusValue(name + "-maximum")}
	return rangeOf.pick(from.draw(picking, index, spaceZero))
}

// arm begins inside the trunk and curves out into a separate upright column.
func (plant saguaro) arm(from seed, index, count int, height, radius float64) cactusColumn {
	branch := from.sprout(index, spaceOne)
	turn := fullTurn * (float64(index)/float64(count) +
		cactusValue("arm-turn-variation")*branch.swing(growing, index, spaceZero))
	sine, cosine := math.Sincos(turn)
	attachment := cactusPick(branch, "attachment", spaceZero) * height
	inset := radius * cactusValue("root-radius-share")
	at := maths.V(inset*cosine, attachment, -inset*sine)
	pose := maths.NewSubspace(at, maths.Rotation(maths.YAxis, turn), spaceOne)
	return cactusColumn{stem: cactusStem{run: cactusPick(branch, "arm-run", spaceOne),
		radius: radius * cactusValue("arm-radius-share"),
		bend:   cactusPick(branch, "bend", intPairCount),
		ribs:   int(cactusValue("ribs")), depth: cactusValue("pleat-depth")}, pose: pose}
}

// surface changes path sampling and pleat resolution, never arm dimensions.
func (column cactusColumn) surface(detail string) (cactusStem, cut) {
	stem := column.stem
	stem.curve = int(cactusValue("curve-" + detail))
	stem.cap = int(cactusValue("cap-" + detail))
	sides := int(cactusValue("sides-" + detail))
	if detail == "hull" || detail == "clumps" {
		stem.depth = spaceZero
	}
	return stem, cut{around: sides}
}
