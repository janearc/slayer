package cities

import (
	"math"
	"math/rand/v2"
	"slices"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/thing"
)

// The roles of a bivouac's parts. What each is made of is in data.
const (
	// roleNylon is the sack's coated nylon.
	roleNylon partRole = "nylon"
	// roleAlloy is the hoop, the stakes and the stove.
	roleAlloy partRole = "alloy"
	// roleCord is a guy line, and the tape over a seam.
	roleCord partRole = "cord"
	// rolePack is the pack.
	rolePack partRole = "pack"
	// roleBoot is a boot.
	roleBoot partRole = "boot"
)

// stoveStream is the stream of numbers a bivouac's stove is drawn from,
// which nothing else in a bivouac is drawn from.
const stoveStream = 3

// archMiddle is the angle halfway up a section's arch, where a side's guy
// line is tied on, in radians.
const archMiddle = math.Pi / 4

// sackSizes are the sack's measures, in metres: its length and width, how
// many facets its arch is cut into, and the height and length of each of
// its three runs, the foot, the shoulder and the hood.
type sackSizes struct {
	length, width                        float64
	facets                               int
	foot, shoulderLength, shoulderHeight float64
	hoodLength, hoodHeight               float64
}

// rigSizes are the hoop, the guy lines and the stakes, in metres.
type rigSizes struct {
	hoop, proud, guy, reach, side, stake, stakeHeight, lean float64
}

// kitSizes are the pack and the boots, in metres.
type kitSizes struct {
	packWidth, packHeight, packDepth, lid, packOffset, packAt float64
	bootWidth, bootLength, sole, upper, heel                  float64
	gap, bootOffset, bootAt                                   float64
}

// stoveSizes are the small stove that some bivouacs have: how likely one
// is, its burner and pot, where it stands, and its flame's heat and
// strength.
type stoveSizes struct {
	share, burnerRadius, burnerHeight, potRadius, potHeight float64
	offset, at, heat, strength                              float64
}

// finishSizes are what the demo level adds to the nylon: the taped
// seams and the ripstop's weave.
type finishSizes struct{ thread, pitch, depth float64 }

// bivouacSizes is everything a bivouac takes from data.
type bivouacSizes struct {
	sack   sackSizes
	rig    rigSizes
	kit    kitSizes
	stove  stoveSizes
	finish finishSizes
}

// BivouacFacts is how many a bivouac sleeps, its floor and its pitch, or
// why the data does not say.
func BivouacFacts() (Facts, error) {
	reader := &choiceReader{}
	facts := Facts{Sleeps: reader.whole("bivouac-sleeps"),
		FloorArea: reader.of("bivouac-floor-area"),
		PitchArea: reader.of("bivouac-pitch-area")}
	if reader.trouble != nil {
		return Facts{}, reader.trouble
	}
	return facts, nil
}

// readBivouac is a bivouac's sizes from data, or why one is not there.
func readBivouac() (bivouacSizes, error) {
	reader := &choiceReader{}
	sizes := bivouacSizes{
		sack: sackSizes{length: reader.of("bivy-length"),
			width: reader.of("bivy-width"), facets: reader.whole("bivy-facets"),
			foot:           reader.of("bivy-foot-height"),
			shoulderLength: reader.of("bivy-shoulder-length"),
			shoulderHeight: reader.of("bivy-shoulder-height"),
			hoodLength:     reader.of("bivy-hood-length"),
			hoodHeight:     reader.of("bivy-hood-height")},
		rig: rigSizes{reader.of("hoop-thickness"), reader.of("hoop-proud"),
			reader.of("guy-thickness"), reader.of("guy-reach"),
			reader.of("guy-side"), reader.of("stake-thickness"),
			reader.of("stake-height"), reader.of("stake-lean")},
		kit: readKit(reader), stove: readStove(reader),
		finish: finishSizes{reader.of("seam-thread"),
			reader.of("nylon-pitch"), reader.of("nylon-depth")},
	}
	return sizes, reader.trouble
}

// readKit is the pack's and the boots' sizes, read from data.
func readKit(reader *choiceReader) kitSizes {
	return kitSizes{packWidth: reader.of("pack-width"),
		packHeight: reader.of("pack-height"), packDepth: reader.of("pack-depth"),
		lid: reader.of("pack-lid-height"), packOffset: reader.of("pack-offset"),
		packAt: reader.of("pack-at"), bootWidth: reader.of("boot-width"),
		bootLength: reader.of("boot-length"), sole: reader.of("boot-sole"),
		upper: reader.of("boot-upper"), heel: reader.of("boot-heel"),
		gap: reader.of("boot-gap"), bootOffset: reader.of("boot-offset"),
		bootAt: reader.of("boot-at")}
}

// readStove is the stove's sizes, read from data.
func readStove(reader *choiceReader) stoveSizes {
	return stoveSizes{share: reader.of("stove-share"),
		burnerRadius: reader.of("burner-radius"),
		burnerHeight: reader.of("burner-height"),
		potRadius:    reader.of("pot-radius"), potHeight: reader.of("pot-height"),
		offset: reader.of("stove-offset"), at: reader.of("stove-at"),
		heat: reader.of("flame-heat"), strength: reader.of("flame-strength")}
}

// bivouacDress is a bivouac's sizes and the materials it reads, for
// drawing it.
type bivouacDress struct {
	sizes   bivouacSizes
	clothes *wardrobe
	nylon   *thing.Material
}

// BivouacAt is a bivouac drawn for a detail level, from a seed that
// decides whether it has a stove out. At every level it is a sack with a
// low hoop at its head, guy lines and stakes, a pack and a pair of boots,
// and sometimes a stove. At the demo level the nylon is ripstop, the seams
// are taped, and the pack has a lid with straps.
func BivouacAt(level Detail, seed Seed) (Design, error) {
	sizes, err := readBivouac()
	if err != nil {
		return Design{}, err
	}
	dress := bivouacDress{sizes: sizes, clothes: &wardrobe{}}
	dress.nylon = dress.clothes.of(roleNylon)
	if level == demoLevel {
		dress.nylon = dress.clothes.dressed(roleNylon, "ripstop",
			texture.Canvas{Cloth: dress.clothes.colour(roleNylon),
				Pitch: sizes.finish.pitch, Depth: sizes.finish.depth,
				Radius: sizes.sack.width, SoilHeight: 1})
	}
	parts := slices.Concat(dress.sack(), dress.hoop(), dress.guys(),
		dress.kit(), dress.stove(seed))
	if level == demoLevel {
		parts = slices.Concat(parts, dress.seams(), dress.packFittings())
	}
	if dress.clothes.trouble != nil {
		return Design{}, dress.clothes.trouble
	}
	return Design{Name: "bivouac", Elements: parts}, nil
}

// archOf is the corners of an arched section, a width across and a height
// up, cut into facets, from one side of its floor over to the other.
func archOf(width, height float64, facets int) []outlineCorner {
	corners := make([]outlineCorner, 0, facets+1)
	for index := range facets + 1 {
		angle := math.Pi * float64(index) / float64(facets)
		corners = append(corners, outlineCorner{width / half * math.Cos(angle),
			height * math.Sin(angle)})
	}
	return corners
}

// run is one length of the sack: a name, how high its arch is, and how
// long it is.
type run struct {
	name           string
	height, length float64
}

// runs are the sack's three lengths from foot to head.
func (sack sackSizes) runs() []run {
	return []run{
		{"foot", sack.foot,
			sack.length - sack.shoulderLength - sack.hoodLength},
		{"shoulder", sack.shoulderHeight, sack.shoulderLength},
		{"hood", sack.hoodHeight, sack.hoodLength}}
}

// sack is the bivy sack: three arched extrusions end to end along its
// length, the foot, the shoulder and the hood, each higher than the last.
func (dress bivouacDress) sack() []placedElement {
	sack := dress.sizes.sack
	along := -sack.length / half
	var parts []placedElement
	for _, one := range sack.runs() {
		form := extrusion{Section: archOf(sack.width, one.height, sack.facets),
			Length: one.length, Ridge: ridgeAlongZ}
		parts = append(parts, part(one.name, form, dress.nylon,
			standingAt(maths.V(0, 0, along))))
		along += one.length
	}
	return parts
}

// outline is the points of a run's arch at a place along the sack, each
// moved out of the cloth by a distance, for a pole or a seam to follow.
func (sack sackSizes) outline(height, along, proud float64) []maths.Vec {
	across, up := (sack.width/half+proud)/(sack.width/half),
		(height+proud)/height
	var points []maths.Vec
	for _, corner := range archOf(sack.width, height, sack.facets) {
		points = append(points, maths.V(corner.X*across, corner.Z*up, along))
	}
	return points
}

// follow is a bar between each pair of points in turn, of a square
// section, in a material.
func follow(name string, points []maths.Vec, thickness float64,
	material *thing.Material) []placedElement {
	var bars []placedElement
	for index := 1; index < len(points); index++ {
		bars = append(bars, rod(name, points[index-1], points[index],
			thickness, material))
	}
	return bars
}

// hoop is the low hoop over the head: a pole bent round the hood, as
// straight bars from point to point, a little out of the cloth.
func (dress bivouacDress) hoop() []placedElement {
	sack, rig := dress.sizes.sack, dress.sizes.rig
	hood := sack.runs()[2]
	middle := sack.length/half - hood.length/half
	return follow("hoop", sack.outline(hood.height, middle, rig.proud),
		rig.hoop, dress.clothes.of(roleAlloy))
}

// guy is a guy line from a point on the sack to the top of a stake driven
// at a point on the ground, with the stake, which leans away from the sack.
func (dress bivouacDress) guy(from, stake maths.Vec) []placedElement {
	rig := dress.sizes.rig
	away := maths.V(stake.X-from.X, 0, stake.Z-from.Z).Unit()
	top := stake.Add(away.Scale(rig.lean)).Add(maths.V(0, rig.stakeHeight, 0))
	return []placedElement{
		rod("guy", from, top, rig.guy, dress.clothes.of(roleCord)),
		rod("stake", stake, top, rig.stake, dress.clothes.of(roleAlloy))}
}

// guys are the guy lines: one from the top of the hood ahead, one from the
// top of the foot behind, and one from each side of the shoulder.
func (dress bivouacDress) guys() []placedElement {
	sack, rig := dress.sizes.sack, dress.sizes.rig
	end := sack.length / half
	shoulder := -end + sack.runs()[0].length + sack.shoulderLength/half
	side := func(sign float64) []placedElement {
		return dress.guy(maths.V(sign*sack.width/half*math.Cos(archMiddle),
			sack.shoulderHeight*math.Sin(archMiddle), shoulder),
			maths.V(sign*(sack.width/half+rig.side), 0, shoulder))
	}
	return slices.Concat(
		dress.guy(maths.V(0, sack.hoodHeight, end),
			maths.V(0, 0, end+rig.reach)),
		dress.guy(maths.V(0, sack.foot, -end), maths.V(0, 0, -end-rig.reach)),
		side(kerbSide), side(offSide))
}

// kit is the pack standing beside the sack and the pair of boots by its
// foot, on the other side.
func (dress bivouacDress) kit() []placedElement {
	kit, clothes := dress.sizes.kit, dress.clothes
	pack := Box{kit.packWidth, kit.packHeight, kit.packDepth}
	sole := Box{kit.bootWidth, kit.sole, kit.bootLength}
	upper := Box{kit.bootWidth, kit.upper, kit.heel}
	parts := []placedElement{part("pack", pack, clothes.of(rolePack),
		seated(pack, standingAt(maths.V(kit.packOffset, 0, kit.packAt))))}
	for _, side := range []float64{kerbSide, offSide} {
		across := -kit.bootOffset + side*(kit.gap/half+kit.bootWidth/half)
		foot := maths.V(across-kit.bootWidth/half, 0, kit.bootAt)
		parts = append(parts, part("sole", sole, clothes.of(roleBoot),
			standingAt(foot)),
			part("upper", upper, clothes.of(roleBoot),
				standingAt(foot.Add(maths.V(0, kit.sole, 0)))))
	}
	return parts
}

// stove is the small stove, if the seed gives the bivouac one: a burner
// that is a flame's light, and the pot on it.
func (dress bivouacDress) stove(seed Seed) []placedElement {
	stove, alloy := dress.sizes.stove, dress.clothes.of(roleAlloy)
	draw := rand.New(rand.NewPCG(uint64(seed), stoveStream)).Float64()
	if draw >= stove.share {
		return nil
	}
	burner := outlineCircle{Radius: stove.burnerRadius}
	pot := outlineCircle{Radius: stove.potRadius}
	at := maths.V(-stove.offset, 0, stove.at)
	flame := solid{Name: "burner", Material: alloy,
		Form: prism{Bottom: burner, Top: burner, Height: stove.burnerHeight},
		Lamp: &Lamp{Heat: stove.heat, Luminance: stove.strength}}
	return []placedElement{{Element: flame, Place: standingAt(at)},
		part("pot", prism{Bottom: pot, Top: pot, Height: stove.potHeight},
			alloy, standingAt(at.Add(maths.V(0, stove.burnerHeight, 0))))}
}

// seams are the taped seams at the demo level: a tape along the top of
// each run, and a tape round the arch where a run meets the next.
func (dress bivouacDress) seams() []placedElement {
	sack, thread := dress.sizes.sack, dress.sizes.finish.thread
	tape := dress.clothes.of(roleCord)
	along := -sack.length / half
	var seams []placedElement
	for index, one := range sack.runs() {
		top := maths.V(0, one.height+thread/half, along)
		seams = append(seams, rod("seam", top,
			top.Add(maths.V(0, 0, one.length)), thread, tape))
		along += one.length
		if index < len(sack.runs())-1 {
			seams = append(seams, follow("seam",
				sack.outline(one.height, along, thread/half), thread, tape)...)
		}
	}
	return seams
}

// packFittings are the pack's lid and the two straps over it, at the demo
// level.
func (dress bivouacDress) packFittings() []placedElement {
	kit, clothes := dress.sizes.kit, dress.clothes
	lid := Box{kit.packWidth, kit.lid, kit.packDepth}
	base := maths.V(kit.packOffset, kit.packHeight, kit.packAt)
	front := kit.packDepth / half
	parts := []placedElement{part("lid", lid, clothes.of(rolePack),
		seated(lid, standingAt(base)))}
	for _, side := range []float64{kerbSide, offSide} {
		across := side * kit.packWidth / half / half
		parts = append(parts, rod("strap",
			base.Add(maths.V(across, kit.lid, -front)),
			base.Add(maths.V(across, kit.lid, front)),
			bothSides*dress.sizes.finish.thread, clothes.of(roleBoot)))
	}
	return parts
}
