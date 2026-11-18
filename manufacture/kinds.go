package manufacture

import (
	"cmp"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/thing"
)

// the kinds: what the manufactory makes, each a row of numbers. a
// caller names a kind and gives a seed, and the seed picks every
// number the row leaves open, from the range the row allows.
//
// the numbers in these rows are fiction, decided 2026-10-06 when the
// question was which published tables to use: they are made up to sound
// right, by the author unless a row says whose, and none is measured.
//
// three are from memory and not looked up: the can's size, the sizes at which a
// stone changes its name, and (with the lights) hydrogen's line.
//
// a row's ranges are the one place a bare number is a length, an
// angle, a heat or a power: a span says in its row's own words which,
// and what a seed picks from it is a measure quantity from then on.

// span is a range a seed picks from: bare numbers in the unit the
// row that holds it says.
type span struct{ least, most float64 }

// pick is the number a share of the way from the least to the most.
func (of span) pick(share float64) float64 {
	return of.least + (of.most-of.least)*share
}

// metres, degrees, kelvin and watts are a pick as the quantity its
// row means.
func (of span) metres(share float64) measure.Length {
	return measure.Metres(of.pick(share))
}

// degrees is a pick as an angle.
func (of span) degrees(share float64) measure.Angle {
	return measure.Degrees(of.pick(share))
}

// kelvin is a pick as a temperature.
func (of span) kelvin(share float64) measure.Temperature {
	return measure.Kelvin(of.pick(share))
}

// watts is a pick as a power.
func (of span) watts(share float64) measure.Power {
	return measure.Watts(of.pick(share))
}

// count is pick for whole numbers: each from the least to the most is
// as likely as the next.
func (of span) count(share float64) int {
	wide := of.most - of.least + 1
	return int(of.least + math.Min(math.Floor(share*wide), wide-1))
}

// the places a row's picks are drawn from in its seed. the thing
// itself grows from a sprout of the seed, so its own draws are none of
// these.
const (
	picking = iota
	growing
	stoning
	sticking
	flaring
	housing
)

// maker is what a row knows: how to make one of its kind from a seed.
type maker interface {
	made(from seed) (good, error)
}

// kind is a row of the table: a name, and how one is made.
//
// found only by kindNamed.
type kind struct {
	name string
	how  maker
}

var errKind = errors.New("manufacture: no such kind")

// table is every kind there is, by name.
func table() map[string]maker {
	rows := map[string]maker{"spruce": physicalSpruce{legacy: spruceRow()}, "fern": fernRow(),
		"norway": physicalSpruce{legacy: norwayRow()},
		"log":    logRow(), "can": canRow(), "campfire": fireRow(),
		"bonfire": pyreRow(), "spotlight": spotRow(), "maser": maserRow(),
		"home": homeRow()}
	rows["guide-star-launcher"] = guideLauncher{}
	rows["saguaro"] = saguaro{}
	rows["human-scale-2m"] = scaleHuman{}
	rows["optical-observatory"] = optical{Optical{
		Diameter: measure.Metres(24), Aperture: measure.Metres(8),
		Opening: measure.Degrees(35), Azimuth: measure.Degrees(180),
		Altitude: measure.Degrees(55), Segmented: true,
	}}
	rows["coast-redwood"], rows["giant-sequoia"] = redwoodRow(), sequoiaRow()
	rows["coast-redwood-shoot"] = shootKind{style: 1}
	rows["giant-sequoia-shoot"] = shootKind{style: 2}
	for name, row := range stoneRows() {
		rows[name] = row
	}
	for name, row := range plateRows() {
		rows[name] = row
	}
	for name, row := range spacecraftRows() {
		rows[name] = row
	}
	for name, row := range personalRows() {
		rows[name] = row
	}
	for name, row := range yardRows() {
		rows[name] = row
	}
	for name, row := range vegetationRows() {
		rows[name] = row
	}
	for name, row := range owlRows() {
		rows[name] = row
	}
	for name, row := range summitRows() {
		rows[name] = row
	}
	for name, row := range stageRows() {
		rows[name] = row
	}
	for name, row := range truckRows() {
		rows[name] = row
	}
	for name, row := range saucerRows() {
		rows[name] = row
	}
	for name, row := range stackRows() {
		rows[name] = row
	}
	for name, row := range lunarRows() {
		rows[name] = row
	}
	for name, row := range veneraRows() {
		rows[name] = row
	}
	for name, row := range j58Rows() {
		rows[name] = row
	}
	for name, row := range alfaRows() {
		rows[name] = row
	}
	for name, row := range macroRows() {
		rows[name] = row
	}
	for name, row := range cityRows() {
		rows[name] = row
	}
	rows["city-clustered-10000"] = clusteredCityMaker{DefaultClusteredCity()}
	rows["city-clustered-green-10000"] = clusteredCityMaker{greenClusteredCity(false)}
	rows["city-clustered-night-10000"] = clusteredCityMaker{greenClusteredCity(true)}
	rows["griffith"] = civicObservatory{}
	rows["space-observatory"] = rows["spitzer"]
	for name, row := range groundRows() {
		rows[name] = row
	}
	return rows
}

// kindNames is the name of every kind, in alphabetical order.
func kindNames() []string {
	var names []string
	for name := range table() {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// kindNamed is the kind of a name, or a refusal that lists the kinds
// there are.
func kindNamed(name string) (kind, error) {
	how, known := table()[name]
	if !known {
		return kind{}, fmt.Errorf("%w as %q; there are %s", errKind,
			name, strings.Join(kindNames(), ", "))
	}
	return kind{name: name, how: how}, nil
}

// makeOf is one of a kind, from a seed, or why there can be none. it
// is the one door: there is no door a kind.
func makeOf(row kind, from seed) (good, error) {
	if row.how == nil {
		return good{}, errKind
	}
	if !from.made() {
		return good{}, errGood
	}
	return row.how.made(from)
}

// plain is a look of one plain colour.
func plain(name string, red, green, blue float64) *thing.Material {
	return &thing.Material{Name: name,
		Colour: maths.RGB(red, green, blue)}
}

// stone is the row of a stone: how long its longest axis may be, in
// metres, its middle axis as a share of that and its shortest as a
// share of the middle, how worn it may be, how far it may lean from
// lying flat, in degrees, how much of its thickness is sunk in the ground,
// as a share, how much of its face dust covers, as a share, how finely it
// is cut, and whether it breaks as a plate of a layered rock does.
type stone struct {
	long, middle, short, worn, lean, sunk, dust span
	around                                      int
	dressed                                     looks
	platy                                       bool
}

// the draws a stone's lean, its sinking and its dust are picked from, in
// the picking part of its seed
const (
	leaning = 4 // defined: the draw after its three axes and its wear
	sinking = 5 // defined: the draw after its lean
	dusting = 6 // defined: the draw after its sinking
)

// the dust that lies on a plate
const (
	// dustShare is how much of the plate's own light the dust gives back.
	// picked: the soil's, which the set picks at the same share of the
	// plates' read reflectance, darker, as the lander's panoramas show it.
	dustShare = 0.45 // picked: see above
	// dustPatch is about how wide a patch of dust is, in metres. picked:
	// a few centimetres, as the dust lies in the panoramas.
	dustPatch = 0.04 // picked: see above
)

// dark basalt's share of the light it gives back, red, green and blue: the
// ground the venus landers stood on, dark and nearly grey. read from the
// venus surface reflectance abstract, ntrs 19870013979: page 166, figure 1,
// albedo by band from the photometers of two landers; page 165, the text.
const (
	// darkRed: read, page 166, figure 1, about 6 per cent at 0.65 to 0.7 um.
	darkRed = 0.06 // read: page 166
	// darkGreen: read, page 166, figure 1, about 5 per cent at 0.55 to 0.6 um.
	darkGreen = 0.05 // read: page 166
	// darkBlue: picked. no band reaches below 0.5 um; page 165 says the
	// later panoramas show rock and soil grey to within a tenth, so the blue
	// is held within a tenth of the green.
	darkBlue = 0.046 // picked: see above
)

// plateRows is the plates of a layered rock broken where it lies: a slab
// and a chip, each far wider than it is thick, freshly broken, some
// leaning, in dark basalt. their shapes are picked by looking at the
// photographs a venus lander took of the ground it stood on.
func plateRows() map[string]stone {
	dark := looks{body: plain("dark basalt", darkRed, darkGreen, darkBlue)}
	slab := stone{
		long:    span{0.15, 0.9},  // picked: a hand to an arm long
		middle:  span{0.5, 0.9},   // picked: plates, not rods
		short:   span{0.04, 0.14}, // picked: centimetres to a decimetre thick
		worn:    span{0, 0.25},    // picked: broken where it lies
		lean:    span{0, 10},      // picked: most lie flat, some tilted
		sunk:    span{0.4, 0.85},  // picked: bedded in the soil, near flush
		dust:    span{0, 0},       // picked: none for now; the patches drew as even spots, not drifted soil (2026-11-15)
		around:  24,               // picked: fine enough to be seen near
		dressed: dark,
		platy:   true,
	}
	chip := slab
	chip.long = span{0.01, 0.12} // picked: grit to a palm across
	chip.short = span{0.2, 0.5}  // picked: thicker for its size
	chip.worn = span{0.1, 0.5}   // picked: a little rounded
	chip.sunk = span{0, 0.4}     // picked: lying on the soil or in it
	chip.dust = span{0, 0}       // picked: none, as on the slabs
	chip.around = 6              // picked: as a pebble is cut
	return map[string]stone{"slab": slab, "chip": chip}
}

// stoneRows is the three stones, named by size. the sizes at which the
// name changes are the geologists', from memory: 4, 64 and 256 mm. the
// smaller a stone the further it has usually travelled, so the more
// worn it is allowed to be.
func stoneRows() map[string]stone {
	granite := looks{body: plain("granite", 0.42, 0.41, 0.4)}
	shape := stone{middle: span{0.6, 0.95}, short: span{0.5, 0.9},
		dressed: granite}
	pebble, cobble, boulder := shape, shape, shape
	pebble.long, pebble.worn, pebble.around = span{0.004, 0.064},
		span{0.6, 1}, 6
	cobble.long, cobble.worn, cobble.around = span{0.064, 0.256},
		span{0.3, 0.9}, 12
	boulder.long, boulder.worn, boulder.around = span{0.256, 2},
		span{0, 0.5}, 12
	return map[string]stone{"pebble": pebble, "cobble": cobble,
		"boulder": boulder}
}

// made is a stone of this row from a seed, leaned about its longest axis
// where its row lets it lean, so one long edge sinks into the ground and
// the other lifts, and sunk by a share of its thickness where its row
// sinks it.
func (row stone) made(from seed) (good, error) {
	long := row.long.metres(from.draw(picking, 0, 0))
	middle := long.Times(row.middle.pick(from.draw(picking, 1, 0)))
	short := middle.Times(row.short.pick(from.draw(picking, 2, 0)))
	shape, err := newLump(long, middle, short,
		row.worn.pick(from.draw(picking, 3, 0)),
		from.sprout(growing, 0))
	if err != nil {
		return good{}, err
	}
	if row.platy {
		shape = shape.asPlate()
	}
	made, err := shaped(shape, row.around, row.dressing(from))
	if err != nil || row.lean.most == 0 && row.sunk.most == 0 {
		return made, err
	}
	lean := row.lean.degrees(from.draw(picking, leaning, 0))
	depth := inScene(short) * row.sunk.pick(from.draw(picking, sinking, 0))
	return made.placed(maths.Subspace{Origin: maths.V(0, -depth, 0),
		Orientation: maths.About(maths.XAxis, lean.InRadians())})
}

// dressing is a stone's looks: its row's, or where its row dusts it, its
// row's rock with dust lying on it in patches, as much as the seed draws,
// the patches the seed's own.
func (row stone) dressing(from seed) looks {
	if row.dust.most == 0 {
		return row.dressed
	}
	rock := *row.dressed[body]
	rock.Texture = texture.Dusted{Rock: rock.Colour,
		Dust:  rock.Colour.Scale(dustShare),
		Share: row.dust.pick(from.draw(picking, dusting, 0)),
		Patch: dustPatch, Seed: from.number}
	return looks{body: &rock}
}

// shaped is a thing of one form, cut so many round, in its looks.
func shaped(shape form, around int, dressed looks) (good, error) {
	fine, err := cutInto(around)
	if err != nil {
		return good{}, err
	}
	// a cut that was made and a form a door made is a thing newGood
	// takes: its error is for a form or a cut that is nothing
	made, err := newGood(shape, fine)
	if err != nil {
		return good{}, err
	}
	return made.in(dressed), nil
}

// conifer is the row of a whorled tree: how tall, in metres, its
// trunk's radius as a share of its height, how far apart its whorls
// are, in metres, how many branches its lowest whorl has, how its
// branches hang, and the whorled architecture it grows by.
type conifer struct {
	tall, slender, between, lowest span
	hangs                          habit
	grows                          whorled
	dressed                        looks
	foliageStyle                   int
	flare                          float64
}

// spruceRow is the spruce, a pinaceae. how its branches hang is ninja's
// habit, and how it grows is spruceWhorled.
func spruceRow() conifer {
	return conifer{tall: span{4, 11}, slender: span{0.02, 0.03},
		between: span{0.4, 0.6}, lowest: span{5, 8},
		hangs: spruceHabit(), grows: spruceWhorled(),
		dressed: looks{body: plain("bark", 0.26, 0.17, 0.1),
			leaf: plain("needles", 0.07, 0.24, 0.1)}}
}

// spruceWhorled is how a spruce grows, ninja's, from ConiferProper and
// branchGeometry: a trunk of one piece carrying whorls of branches,
// each three drooping pieces in bark with a sleeve of needles. set by
// eye, none measured.
func spruceWhorled() whorled {
	return whorled{
		trunk:  round{joints: tapering(), sides: 8},
		bark:   round{joints: girths(1, 0.75, 0.5, 0.25), sides: 5},
		branch: limbing{pieces: 3, droop: 0.22},
		clad: sleeveFoliage{joints: lengths(0.10, 0.22, 0.15, 0),
			sides: 6},
	}
}

// norwayRow is the norway spruce, picea abies, a pinaceae: taller and
// more slender than the spruce, its whorls further apart. its numbers
// are set by eye, mine, 2026-10-14; its branches hang by the spruce's
// habit, and how it grows is norwayWhorled.
func norwayRow() conifer {
	return conifer{tall: span{6, 13}, slender: span{0.015, 0.025},
		between: span{0.5, 0.75}, lowest: span{5, 7},
		hangs: spruceHabit(), grows: norwayWhorled(),
		dressed: looks{body: plain("bark", 0.24, 0.15, 0.09),
			leaf: plain("needles", 0.06, 0.2, 0.08)}}
}

// norwayWhorled is how a norway spruce grows: a spruce's trunk and
// whorls, every piece of a branch carrying two rows of branchlets that
// droop toward the ground. most of its needles are on the branchlets,
// the curtains an old one hangs in. set by eye, mine, 2026-10-14.
func norwayWhorled() whorled {
	return whorled{
		trunk:  round{joints: tapering(), sides: 8},
		bark:   round{joints: girths(1, 0.7, 0.45, 0.2), sides: 5},
		branch: limbing{pieces: 3, droop: 0.12},
		clad: sleeveFoliage{joints: lengths(0.05, 0.07, 0.06, 0.03),
			sides: 5},
		shoots: &branchlets{
			per:   span{4, 6},
			shoot: limbing{pieces: 2, droop: 0.6},
			set:   norwayShoots(),
			wear: sleeveFoliage{joints: lengths(0.12, 0.2, 0.08),
				sides: 4},
		},
	}
}

const (
	// branchGrowth is how a grown tree's branches lengthen with its
	// height: so many times taller, its branches that many times to
	// this power as long. 2026-11-10: "they need to be much
	// longer. probably scale branch length as much as you scale the
	// tree height"; one.
	branchGrowth = 1
	// whorlFill is how a grown tree's whorls fill in: so many times
	// taller, so many times to this power the branches in each, so the
	// longer branches' tips stand no further apart than the root of it.
	// Its whorls already come as many times more often up its height.
	// picked, mine, 2026-11-10, with "increase frequency as well?
	// probably not directly?".
	whorlFill = 0.5
	// grownGirth is a grown tree's branch's radius as a share of its own
	// length: its foot 2.2 hundredths of its length across.
	// 2026-11-10: "the branches should be a little bit thicker". picked
	// by eye, not measured.
	grownGirth = 0.011
	// grownPieces is how many pieces a grown tree's limb is in, and
	// grownWander the most it turns at a joint, in radians: limbs that
	// flow and twist slightly, where three pieces kinked twice
	// (2026-11-10: "they should be non-straight, but they should flow and
	// twist just slightly"). picked by eye, mine.
	grownPieces = 8
	grownWander = 0.06
)

// resampled is joints, foot to tip, taken again at so many places along
// the limb, each share eased between the two joints either side.
func resampled(joints []joint, count int) []joint {
	out := make([]joint, count)
	last := float64(len(joints) - 1)
	for at := range out {
		place := float64(at) / float64(count-1) * last
		low := min(int(place), len(joints)-2)
		f := place - float64(low)
		a, b := joints[low], joints[low+1]
		out[at] = joint{ofGirth: a.ofGirth + f*(b.ofGirth-a.ofGirth),
			ofLength: a.ofLength + f*(b.ofLength-a.ofLength)}
	}
	return out
}

// flowing is the architecture with its limbs in so many pieces that
// wander, the same droop spread over their joints, and their bark and
// foliage taken again at each joint. A foliage it cannot take again
// leaves the limbs as they were.
func (tree whorled) flowing(pieces int, wander float64) whorled {
	clad, ok := tree.clad.(sleeveFoliage)
	if !ok || tree.branch.pieces < 2 {
		return tree
	}
	joints := tree.branch.pieces - 1
	tree.branch = limbing{pieces: pieces,
		droop:  tree.branch.droop * float64(joints) / float64(pieces-1),
		wander: wander}
	tree.bark.joints = resampled(tree.bark.joints, pieces+1)
	clad.joints = resampled(clad.joints, pieces+1)
	tree.clad = clad
	return tree
}

// taller is the row with its height so many times over: the same seed
// picks the same share of the taller span, so the same tree, taller,
// with as many more whorls as fit, on a trunk as much thicker.
//
// its branches are not as much thicker (2026-11-10: "their
// branches must not increase in proportion to their trunks"): each is
// as thick as its own length asks, grownGirth of it. They are longer by
// the times to branchGrowth's power, more to a whorl by whorlFill's, and
// flow in grownPieces pieces rather than kink in three.
func (row conifer) taller(times float64) conifer {
	row.tall = span{row.tall.least * times, row.tall.most * times}
	row.hangs.reach *= math.Pow(times, branchGrowth-1)
	row.hangs.own = grownGirth
	fill := math.Pow(times, whorlFill)
	row.lowest = span{row.lowest.least * fill, row.lowest.most * fill}
	row.grows = row.grows.flowing(grownPieces, grownWander)
	return row
}

const (
	// sideSpacing is how far apart a branched tree's side branches stand
	// along a limb, in metres, and sideReach how long the longest is, as
	// a share of the whole limb. picked by eye, mine, 2026-11-10.
	sideSpacing = 0.35
	sideReach   = 1.0 / 3
)

// branched is the row with side branches on its limbs: each piece of its
// longest limb carries as many as fit at sideSpacing, and every limb as
// many; each side branch in foliage of its own, so a bough is a fan of
// sprays, not a line of them. A row that has branchlets already keeps
// their shape and takes the spacing.
func (row conifer) branched() conifer {
	pieces := float64(row.grows.branch.pieces)
	longest := row.hangs.reach * (row.tall.least + row.tall.most) / 2
	each := math.Max(2, longest/pieces/sideSpacing)
	sides := branchlets{
		per:   span{0.8 * each, 1.2 * each},
		shoot: limbing{pieces: 2, droop: 0.15},
		set:   spruceSides(),
		wear:  sleeveFoliage{joints: lengths(0.10, 0.18, 0.08), sides: 5},
	}
	// a side branch is sized against the piece it stands on, so a share
	// of the whole limb is that share times the pieces, as the redwood's
	// branchlets are
	sides.set.reach = sideReach * pieces
	if row.grows.shoots != nil {
		sides.shoot = row.grows.shoots.shoot
		sides.set = row.grows.shoots.set
		sides.wear = row.grows.shoots.wear
	}
	row.grows.shoots = &sides
	return row
}

// spruceSides is how a branched spruce sets its side branches along a
// limb: each half a turn from the last, so they stand in two rows,
// leaning 55 degrees out from the limb, shorter toward each piece's
// tip; branched sets their reach. set by eye, mine, 2026-11-10; none is
// measured.
func spruceSides() habit {
	return habit{
		turn:  measure.Radians(fullTurn / 2),
		start: 0.05, top: 0.95,
		thinning: 0, fewest: 1,
		stray: 0.1,
		low:   measure.Degrees(55), lift: measure.Degrees(10),
		sway:  measure.Degrees(10),
		reach: 1, shortening: 0.3, uneven: 0.25,
		girth: 0.4, least: 0.002,
	}
}

// made is a whorled tree from a seed: as many whorls as fit up its
// height, at quick.
func (row conifer) made(from seed) (good, error) {
	made, err := row.scaffold(from)
	if err != nil || row.foliageStyle == 0 {
		return made, err
	}
	return recursiveFoliage(made, from, row.foliageStyle, "quick")
}

// scaffold retains the seeded major limbs while foliage is constructed
// separately. Legacy spruce recipes still use its original sleeves.
func (row conifer) scaffold(from seed) (good, error) {
	tall := row.tall.metres(from.draw(picking, 0, 0))
	girth := tall.Times(row.slender.pick(from.draw(picking, 1, 0)))
	between := row.between.metres(from.draw(picking, 2, 0))
	whorls := int(math.Round(tall.Over(between)))
	sprays := 0
	if row.grows.shoots != nil {
		sprays = row.grows.shoots.per.count(from.draw(picking, 4, 0))
	}
	grown, err := row.grows.grown(whorls,
		row.lowest.count(from.draw(picking, 3, 0)), sprays, row.hangs,
		atQuick)
	if err != nil {
		return good{}, err
	}
	made, err := grownIn(grown, tall, girth, maths.Subspace{}, from, row.dressed)
	if err == nil && row.flare > 0 {
		h, r := tall.InMetres(), girth.InMetres()
		made.parts[0].shape = turnedProfile{[][2]float64{
			{0, 0}, {r * row.flare, 0}, {r, h * 0.08}, {r * 0.65, h * 0.4}, {0, h}}}
	}
	return made, err
}

// grownIn is a growth grown to a size, standing or lying in a space,
// in its looks.
func grownIn(grown growth, length, girth measure.Length,
	in maths.Subspace, from seed, dressed looks) (good, error) {
	made, err := grow(grown, size{length: length, girth: girth},
		from.sprout(growing, 0))
	if err != nil {
		return good{}, err
	}
	placed, err := made.placed(in)
	return placed.in(dressed), err
}

// fern is the row of a fern: how long a frond is and how thick its
// stalk at the foot, in metres, and how many fronds and how many
// leaflets a frond.
type fern struct {
	long, stalk, fronds, leaflets span
	dressed                       looks
}

// fernRow is the fern.
func fernRow() fern {
	green := plain("frond", 0.2, 0.5, 0.12)
	return fern{long: span{0.4, 0.9}, stalk: span{0.005, 0.008},
		fronds: span{6, 11}, leaflets: span{14, 24},
		dressed: looks{body: green, leaf: green}}
}

// made is a fern from a seed.
func (row fern) made(from seed) (good, error) {
	grown, err := fernGrowth(row.fronds.count(from.draw(picking, 2, 0)),
		row.leaflets.count(from.draw(picking, 3, 0)))
	if err != nil {
		return good{}, err
	}
	return grownIn(grown, row.long.metres(from.draw(picking, 0, 0)),
		row.stalk.metres(from.draw(picking, 1, 0)), maths.Subspace{},
		from, row.dressed)
}

// lying is the space of a thing laid on its side on the ground: its
// length along x, its middle over the space's start, lifted by its
// radius so it rests on the ground and is not half sunk in it, and
// rolled about its length by a share of a turn.
func lying(length, girth measure.Length, roll float64) maths.Subspace {
	quarter := fullTurn / 4
	down := maths.About(maths.ZAxis, -quarter)
	foot := maths.V(-inScene(length)/2, inScene(girth), 0)
	return maths.Subspace{Origin: foot,
		Orientation: down.Then(maths.About(maths.YAxis, fullTurn*roll))}
}

// can is the row of a soda can: how tall, how far across, and the
// share of them left standing.
type can struct {
	tall, across measure.Length
	upright      float64
	around       int
	dressed      looks
}

// canRow is the can: the can makers' 211 by 413, which is 2 11/16
// inches across and 4 13/16 tall, from memory.
func canRow() can {
	const inch = 0.0254
	tin := &thing.Material{Name: "aluminium", Colour: maths.Grey(0.8),
		Gloss: 0.6}
	return can{tall: measure.Metres((4 + 13.0/16) * inch),
		across:  measure.Metres((2 + 11.0/16) * inch),
		upright: 0.3, around: 16, dressed: looks{body: tin}}
}

// made is a can from a seed. every can is the same can; the seed says
// whether somebody stood it up or it rolled, and how far.
func (row can) made(from seed) (good, error) {
	radius := row.across.Times(midway)
	shape, err := newTube(row.tall, radius, radius)
	if err != nil {
		return good{}, err
	}
	made, err := shaped(shape, row.around, row.dressed)
	if err != nil || from.draw(picking, 0, 0) < row.upright {
		return made, err
	}
	return made.placed(lying(row.tall, radius,
		from.draw(picking, 1, 0)))
}

// log is the row of a fallen log: how long, in metres, its radius as
// a share of its length, how many broken stubs of branches it still
// has, how much of its radius it keeps to its top, and how finely the
// trunk and the stubs are cut.
type log struct {
	long, thick, stubs span
	taper              float64
	around, stubAround int
	dressed            looks
}

// logRow is the log.
func logRow() log {
	return log{long: span{1.5, 4}, thick: span{0.04, 0.07},
		stubs: span{2, 5}, taper: 0.8, around: 10, stubAround: 5,
		dressed: looks{body: plain("bark", 0.24, 0.16, 0.1)}}
}

// stubHabit is how a log's broken branches stand along it: one at a
// time, turned by the golden angle as they grew, short, and near
// square to the trunk.
func stubHabit() habit {
	return habit{
		turn:  spruceHabit().turn,
		start: 0.15, top: 0.85,
		thinning: 0, fewest: 1,
		stray: 0.1,
		low:   measure.Degrees(80), lift: measure.Degrees(0),
		sway:  measure.Degrees(15),
		reach: 0.05, shortening: 0, uneven: 0.4,
		girth: 0.5, least: 0.008,
	}
}

// made is a log from a seed: a trunk narrowing a little toward its
// top, carrying its stubs, laid on the ground.
func (row log) made(from seed) (good, error) {
	long := row.long.metres(from.draw(picking, 0, 0))
	girth := long.Times(row.thick.pick(from.draw(picking, 1, 0)))
	trunk, err := sleeved([]joint{{ofGirth: 1}, {ofGirth: row.taper}},
		row.around, body)
	if err != nil {
		return good{}, err
	}
	stub, err := sleeved([]joint{{ofGirth: 1}, {}}, row.stubAround,
		body)
	if err != nil {
		return good{}, err
	}
	along, whorlsError := newWhorls(row.stubs.count(from.draw(picking, 2, 0)), 1,
		stubHabit())
	grown, growthError := stemCarrying(1, 0, []sleeve{trunk}, along, 1, 0,
		[]sleeve{stub})
	if err := cmp.Or(whorlsError, growthError); err != nil {
		return good{}, err
	}
	return grownIn(grown, long, girth,
		lying(long, girth, from.draw(picking, 3, 0)), from, row.dressed)
}
