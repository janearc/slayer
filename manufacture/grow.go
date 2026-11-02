package manufacture

import (
	"errors"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

// slot is which of a thing's matters a part is made of. body is what
// a thing is mainly made of: a stone's stone, a can's metal, a plant's
// wood. leaf is what a plant's leaves are.
type slot int

const (
	body slot = iota
	leaf
)

// form is what one part of a thing is shaped like: a rule for a
// surface, cut as finely as asked. a tube and a lump are forms.
type form interface {
	skin(fine cut) surface
}

// part is one piece of a made thing: a form, where it sits in the
// thing's space, how finely it is cut, and which matter it is of.
type part struct {
	shape form
	space maths.Subspace
	fine  cut
	of    slot
}

// skin is the part's surface in the thing's space.
func (one part) skin() surface {
	return one.shape.skin(one.fine).placed(one.space)
}

// joint is a limb's radius at one of its joints, as shares of the
// limb's own size: so much of its girth and so much of its length. a
// branch's bark narrows by its girth; its needles stand out by its
// length.
type joint struct{ ofGirth, ofLength float64 }

// radius is the joint's radius on a limb of a size, in the scene's unit
// (see inScene).
func (at joint) radius(of size) float64 {
	return at.ofGirth*inScene(of.girth) + at.ofLength*inScene(of.length)
}

// sleeve is one skin along a limb: a radius at every joint, foot to
// tip, how finely it is cut, and what it is of. flat, it is a blade.
//
// made only by newSleeve and newLeaf.
type sleeve struct {
	joints []joint
	fine   cut
	flat   bool
	of     slot
}

var errSleeve = errors.New("manufacture: a sleeve needs a cut, two " +
	"joints or more, shares of nought or above, and a radius above " +
	"nought at every joint but the last")

// newSleeve is a round skin of these joints, or why there can be none.
func newSleeve(joints []joint, fine cut, of slot) (sleeve, error) {
	if fine.around < leastAround || !jointed(joints) {
		return sleeve{}, errSleeve
	}
	return sleeve{joints: joints, fine: fine, of: of}, nil
}

// newLeaf is a flat skin of these joints, or why there can be none.
func newLeaf(joints []joint, of slot) (sleeve, error) {
	if !jointed(joints) {
		return sleeve{}, errSleeve
	}
	return sleeve{joints: joints, flat: true, of: of}, nil
}

// jointed is whether joints can be a sleeve's.
func jointed(joints []joint) bool {
	if len(joints) < 2 {
		return false
	}
	for index, at := range joints {
		last := index == len(joints)-1
		shares := at.ofGirth >= 0 && at.ofLength >= 0 &&
			finite(at.ofGirth) && finite(at.ofLength)
		none := at.ofGirth == 0 && at.ofLength == 0
		if !shares || (!last && none) {
			return false
		}
	}
	return true
}

// limb is a run of straight pieces, end to end, each drooping a
// little further toward the ground than the last, wearing one sleeve
// or more: ninja's branch, which is three pieces in bark and needles.
// a trunk is a limb of one piece that does not droop.
//
// made only by newLimb.
type limb struct {
	pieces      int
	span, droop float64
	skins       []sleeve
}

var errLimb = errors.New("manufacture: a limb needs a piece or more, " +
	"a span above nought and no more than one, a droop of nought or " +
	"above with an end, and a sleeve or more with a joint for each " +
	"end of each piece")

// newLimb is a limb of so many pieces, spanning a share of the length
// it is given, drooping so far a piece, in these sleeves, or why there
// can be none.
func newLimb(pieces int, span, droop float64,
	skins ...sleeve) (limb, error) {
	sound := pieces >= 1 && span > 0 && span <= 1 && droop >= 0 &&
		finite(droop) && len(skins) > 0
	for _, skin := range skins {
		sound = sound && len(skin.joints) == pieces+1
	}
	if !sound {
		return limb{}, errLimb
	}
	return limb{pieces: pieces, span: span, droop: droop,
		skins: skins}, nil
}

// flat is whether the limb is all blade.
func (made limb) flat() bool {
	for _, skin := range made.skins {
		if !skin.flat {
			return false
		}
	}
	return true
}

// scaleOf is a space's scale, where zero means one.
func scaleOf(space maths.Subspace) float64 {
	if space.Scale == 0 {
		return 1
	}
	return space.Scale
}

// runs is where each piece of the limb goes when the limb is put at a
// place: the first piece in the place's own space, each later one
// where the last ended, its way the last's with the droop added toward
// the ground, in the space orientedTube would stand a tube in.
func (made limb) runs(on place) ([]place, error) {
	step := inScene(on.size.length) * made.span / float64(made.pieces)
	each := size{length: measure.Metres(step), girth: on.size.girth}
	at, scale := on.space.Origin, scaleOf(on.space)
	way := on.space.Orientation.Out(maths.YAxis)
	first := on.space
	first.Scale = scale
	runs := []place{{space: first, size: each}}
	for len(runs) < made.pieces {
		at = at.Add(way.Scale(step * scale))
		way = way.Sub(maths.Up.Scale(made.droop)).Unit()
		space, err := spaceAlong(at, way)
		if err != nil {
			return nil, err
		}
		space.Scale = scale
		runs = append(runs, place{space: space, size: each})
	}
	return runs, nil
}

// along is the limb's parts on runs already worked out, for a limb of
// a size: every piece in every sleeve, piece by piece, and within a
// piece sleeve by sleeve, which is ninja's order.
func (made limb) along(of size, runs []place) ([]part, error) {
	var parts []part
	for piece, run := range runs {
		for _, skin := range made.skins {
			one, err := skin.piece(piece, of, run)
			if err != nil {
				return nil, err
			}
			parts = append(parts, one)
		}
	}
	return parts, nil
}

// piece is one piece of the sleeve on a limb of a size: a tube from
// one joint's radius to the next's, where the piece runs.
func (skin sleeve) piece(index int, of size, run place) (part, error) {
	door := newTube
	if skin.flat {
		door = newBlade
	}
	shape, err := door(run.size.length,
		measure.Metres(skin.joints[index].radius(of)),
		measure.Metres(skin.joints[index+1].radius(of)))
	if err != nil {
		return part{}, err
	}
	return part{shape: shape, space: run.space, fine: skin.fine,
		of: skin.of}, nil
}

// arrangement is where the parts a thing carries go, and how big each
// is: a closed form from a row, an index and the seed, on a carrier of
// a size. whorls and ring are arrangements.
type arrangement interface {
	rows() int
	across(row int) int
	at(on size, row, index int, from seed) (place, error)
}

// carried is what a growth carries, and where.
type carried struct {
	where arrangement
	what  growth
}

// growth is how a thing is grown: a limb, and what every piece of it carries,
// each of those a growth of its own. a spruce is a trunk carrying whorls of
// branches; a fern is a crown carrying a ring of fronds, each carrying
// leaflets. there is no tree and no fern beyond that.
//
// made only by newGrowth.
type growth struct {
	stem    limb
	carries []carried
}

var errGrowth = errors.New("manufacture: a growth needs a limb, and " +
	"somewhere for each thing it carries to go")

// newGrowth is a limb carrying these, or why there can be none.
func newGrowth(stem limb, carries ...carried) (growth, error) {
	if stem.pieces < 1 {
		return growth{}, errGrowth
	}
	for _, borne := range carries {
		if borne.where == nil {
			return growth{}, errGrowth
		}
	}
	return growth{stem: stem, carries: carries}, nil
}

// parts is the growth at a place, from a seed: its limb's parts, then
// what each piece of the limb carries, in the order of its rows.
func (grown growth) parts(on place, from seed) ([]part, error) {
	runs, err := grown.stem.runs(on)
	if err != nil {
		return nil, err
	}
	parts, err := grown.stem.along(on.size, runs)
	if err != nil {
		return nil, err
	}
	for piece, run := range runs {
		for _, borne := range grown.carries {
			more, err := borne.parts(run, on.size,
				from.onPiece(piece))
			if err != nil {
				return nil, err
			}
			parts = append(parts, more...)
		}
	}
	return parts, nil
}

// parts is everything carried on one piece of a limb. each thing grows from a
// sprout of the seed, at its place in the piece's space; where the arrangement
// gave it no size it takes the whole limb's. a blade is turned to lie along its
// carrier, so a leaflet lies in its frond's plane and is not edge on to it.
func (borne carried) parts(run place, whole size,
	from seed) ([]part, error) {
	var parts []part
	for row := range borne.where.rows() {
		for index := range borne.where.across(row) {
			there, err := borne.where.at(run.size, row, index, from)
			if err != nil {
				return nil, err
			}
			there.space = there.space.Within(run.space)
			if there.size == (size{}) {
				there.size = whole
			}
			if borne.what.stem.flat() {
				there.space = lyingAlong(there.space,
					run.space.Orientation.Out(maths.YAxis))
			}
			more, err := borne.what.parts(there,
				from.sprout(row, index))
			if err != nil {
				return nil, err
			}
			parts = append(parts, more...)
		}
	}
	return parts, nil
}

// lyingAlong is a space turned about its own y so that its x leans as
// far toward a way as it can: a blade in it lies in the plane of its
// own run and that way. a way along the space's y leaves it as it is.
func lyingAlong(space maths.Subspace, toward maths.Vec) maths.Subspace {
	run := space.Orientation.Out(maths.YAxis)
	across := toward.Sub(run.Scale(toward.Dot(run)))
	if !positive(across.Len()) {
		return space
	}
	across = across.Unit()
	space.Orientation = maths.Orientation{X: across, Y: run,
		Z: across.Cross(run)}
	return space
}

// sleeved is a round sleeve cut so many sides round.
func sleeved(joints []joint, around int, of slot) (sleeve, error) {
	fine, err := cutInto(around)
	if err != nil {
		return sleeve{}, err
	}
	return newSleeve(joints, fine, of)
}

// stemCarrying is a stem of so many pieces in its sleeves, each piece
// carrying limbs of so many pieces in theirs, set by an arrangement.
func stemCarrying(stemPieces int, stemDroop float64, stemSkins []sleeve,
	where arrangement, pieces int, droop float64,
	skins []sleeve) (growth, error) {
	stem, err := newLimb(stemPieces, 1, stemDroop, stemSkins...)
	if err != nil {
		return growth{}, err
	}
	bough, err := newLimb(pieces, 1, droop, skins...)
	if err != nil {
		return growth{}, err
	}
	borne, err := newGrowth(bough)
	if err != nil {
		return growth{}, err
	}
	return newGrowth(stem, carried{where: where, what: borne})
}

// trunkCarrying is a trunk carrying whorls of a growth up its height.
func trunkCarrying(trunk sleeve, count, lowest int, set habit,
	what growth) (growth, error) {
	stem, err := newLimb(1, 1, 0, trunk)
	if err != nil {
		return growth{}, err
	}
	whorled, err := newWhorls(count, lowest, set)
	if err != nil {
		return growth{}, err
	}
	return newGrowth(stem, carried{where: whorled, what: what})
}

// norwayShoots is how a norway spruce sets its branchlets along each
// piece of a branch: one at a time, each half a turn from the last, so
// they stand in two rows, leaning out from the branch, shorter toward
// its tip. set by eye, mine, 2026-10-14; none is measured.
func norwayShoots() habit {
	return habit{
		turn:  measure.Radians(fullTurn / 2),
		start: 0.15, top: 0.95,
		thinning: 0, fewest: 1,
		stray: 0.05,
		low:   measure.Degrees(65), lift: measure.Degrees(10),
		sway:  measure.Degrees(8),
		reach: 1.1, shortening: 0.4, uneven: 0.2,
		girth: 0.4, least: 0.01,
	}
}

// fernHabit is how a fern sets its leaflets along a frond: one at a time, each
// half a turn from the last, so they stand in two rows; longest low on the
// frond and shorter toward its tip.
//
// the half turn is what two rows means. the rest is set by eye, mine,
// 2026-10-06, and moves to the kinds table as the spruce's have.
func fernHabit() habit {
	return habit{
		turn:  measure.Radians(fullTurn / 2),
		start: 0.12, top: 0.98,
		thinning: 0, fewest: 1,
		stray: 0.02,
		low:   measure.Degrees(70), lift: measure.Degrees(15),
		sway:  measure.Degrees(4),
		reach: 0.22, shortening: 0.85, uneven: 0.06,
		girth: 3, least: 0.01,
	}
}

// fernGrowth is how a fern is grown: a short crown carrying a ring of fronds
// leaning out, each a tapering stalk carrying leaflets, each a blade.
//
// the size it is grown at is a frond's length and its stalk's radius at the
// foot. the numbers are set by eye, mine, 2026-10-06, and are not in the fern's
// row yet, as the spruce's now are (families step 4).
//
// only newWhorls and newRing can refuse here, for a count a row got
// wrong; the rest is made of this function's own numbers.
func fernGrowth(fronds, leaflets int) (growth, error) {
	const crownSides, stalkSides = 6, 4
	const crownSpan, lean, stray, uneven = 0.03, -35, 0.3, 0.2
	blade, err := newLeaf([]joint{{ofGirth: 1}, {ofGirth: 0.2}}, leaf)
	if err != nil {
		return growth{}, err
	}
	stalk, err := sleeved([]joint{{ofGirth: 1}, {ofGirth: 0.15}},
		stalkSides, body)
	if err != nil {
		return growth{}, err
	}
	pairs, err := newWhorls(leaflets, 1, fernHabit())
	if err != nil {
		return growth{}, err
	}
	frond, err := stemCarrying(1, 0, []sleeve{stalk}, pairs, 1, 0,
		[]sleeve{blade})
	if err != nil {
		return growth{}, err
	}
	stub, err := sleeved([]joint{{ofGirth: 3}, {ofGirth: 3}},
		crownSides, body)
	if err != nil {
		return growth{}, err
	}
	crown, err := newLimb(1, crownSpan, 0, stub)
	if err != nil {
		return growth{}, err
	}
	round, err := newRing(fronds, measure.Metres(0),
		measure.Degrees(lean), stray, uneven, 0)
	if err != nil {
		return growth{}, err
	}
	return newGrowth(crown, carried{where: round, what: frond})
}
