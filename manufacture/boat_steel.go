package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// The houses of the steel boats, which a fishing boat's wheelhouse and a
// ferry's upper storey share: a box under a roof, its windows each a pane in
// a frame, a door, a ladder to the roof, a railing round it, and on it a
// radar, a searchlight, a horn and the whip aerials. Every proportion here
// is chosen.
const (
	boatEaves      = 0.3  // chosen: metres a roof stands out past its house, in length and width
	boatRoofThick  = 0.08 // chosen: metres a roof is thick
	boatHouseReach = 0.9  // chosen: how much of the deck's room across a house may take
	boatGlassDepth = 0.05 // chosen: metres a window is thick, half of it proud of the wall
	boatWindowLow  = 0.45 // chosen: how high up a house a window's foot stands, as a share of the house
	boatWindowTall = 0.35 // chosen: how tall a window is, as a share of the house
	boatWindowFill = 0.62 // chosen: how much of its pitch a ferry's window takes
	boatWindowRun  = 0.92 // chosen: how much of a house's length its windows run along
	boatFrameWide  = 0.04 // chosen: metres a window's frame is wide
	boatFrameProud = 0.03 // chosen: metres a window's frame stands proud of the wall
	boatFunnelBack = 0.4  // chosen: metres aft of a wheelhouse its funnel stands
	boatFunnelWide = 0.25 // chosen: metres across a funnel's radius
	boatFunnelCap  = 0.12 // chosen: metres a funnel's rust cap is deep
	boatPanePitch  = 1.2  // chosen: metres between the middles of a house's side panes
	boatDoorWide   = 0.7  // chosen: metres a door is wide
	boatDoorTall   = 1.7  // chosen: metres a door is tall
)

// house is the deckhouse: a box on the deck, no wider than the room the
// deck has along its length, under a roof. It stands on the lower of the
// deck's heights at its two ends and is sunk a little into it. It gives
// its base and the top of its roof.
func (craft *boat) house(centre, long, wide, high float64) (base, roof float64) {
	hull := craft.hull
	front, back := centre-long*spaceHalf, centre+long*spaceHalf
	wide = math.Min(wide, 2*boatHouseReach*hull.room(front, back))
	base = math.Min(hull.deckAt(front), hull.deckAt(back)) - boatDeckLift
	return base, craft.storey(centre, long, wide, base, high)
}

// storey is a box of a house, standing on a base, with a roof: it gives
// the top of its roof, and puts its windows on its walls.
func (craft *boat) storey(centre, long, wide, base, high float64) float64 {
	craft.block(0, base, centre, wide, high, long, boatHouse)
	craft.block(0, base+high, centre, wide+boatEaves, boatRoofThick,
		long+boatEaves, boatSteel)
	craft.windows(centre, long, wide, base, high)
	return base + high + boatRoofThick
}

// window is one window of a wall: a thin box standing a little proud of
// it, lit or dark.
func (craft *boat) window(across, up, along, wide, tall, long float64,
	lit bool) {
	material := boatGlass
	if lit {
		material = boatWindow
	}
	craft.block(across, up, along, wide, tall, long, material)
}

// pane is a window in a frame on a wall that faces out to a side or to
// the ends: the glass, lit or dark, and four bars round it.
func (craft *boat) pane(across, up, along float64, onSide bool, wide,
	tall float64, lit bool) {
	thick, bar := boatGlassDepth, boatFrameWide
	proud := thick + boatFrameProud
	size := func(run, depth float64) (float64, float64) {
		if onSide {
			return depth, run
		}
		return run, depth
	}
	glassX, glassZ := size(wide, thick)
	craft.window(across, up, along, glassX, tall, glassZ, lit)
	if !craft.detailed(boatQuick) {
		return
	}
	for index := range 2 {
		turn := boatSide(index)
		barX, barZ := size(wide+bar, proud)
		craft.block(across, up+float64(index)*(tall-bar), along, barX, bar,
			barZ, boatSteel)
		stileX, stileZ := size(bar, proud)
		offX, offZ := size(turn*(wide+bar)*spaceHalf, 0)
		craft.block(across+offX, up, along+offZ, stileX, tall, stileZ,
			boatSteel)
	}
}

// glowing is whether a window, by its stream, is lit: by a draw of its
// own, below the design's share of windows lit.
func (craft *boat) glowing(stream int) bool {
	return craft.from.draw(housing, boatDrawWindows+stream, 0) <
		craft.design.pick("lit-windows", craft.from)
}

// windows are the windows of a storey: three panes in frames across its
// front, a row of them on each side at a pitch, a door in its back wall;
// at a low detail, one long window to a side and one at the front.
func (craft *boat) windows(centre, long, wide, base, high float64) {
	level, tall := base+boatWindowLow*high, boatWindowTall*high
	front, reach := centre-long*spaceHalf, wide*spaceHalf
	if !craft.detailed(boatQuick) {
		craft.window(0, level, front, wide*boatWindowRun, tall, boatGlassDepth,
			craft.glowing(0))
		craft.window(reach, level, centre, boatGlassDepth, tall,
			long*boatWindowRun, craft.glowing(1))
		craft.window(-reach, level, centre, boatGlassDepth, tall,
			long*boatWindowRun, craft.glowing(2))
		return
	}
	for index := range boatFrontPanes {
		across := (float64(index) - spaceOne) * wide * boatWindowRun /
			boatFrontPanes
		craft.pane(across, level, front, false, wide*boatWindowRun/
			boatFrontPanes-3*boatFrameWide, tall, craft.glowing(index))
	}
	craft.sidePanes(centre, long, reach, level, tall)
	craft.door(wide, centre+long*spaceHalf, base)
}

// boatFrontPanes is how many panes stand across a house's front: chosen.
const boatFrontPanes = 3

// sidePanes are the panes of the two sides of a house, at a pitch, as many
// as the length takes, each its own draw for its light.
func (craft *boat) sidePanes(centre, long, reach, level, tall float64) {
	count := max(int(long*boatWindowRun/boatPanePitch), 2)
	pitch := long * boatWindowRun / float64(count)
	for index := range 2 * count {
		along := centre + (float64(index/2)-float64(count-1)*spaceHalf)*pitch
		craft.pane(boatSide(index)*reach, level, along, true,
			pitch-boatFrameWide*3, tall, craft.glowing(boatFrontPanes+index))
	}
}

// door is a door in the back wall of a house, off the middle, with its
// head and a handle.
func (craft *boat) door(wide, back, foot float64) {
	across := wide * boatDoorAt
	craft.block(across, foot, back, boatDoorWide, boatDoorTall,
		boatGlassDepth, boatSteel)
	craft.block(across, foot+boatDoorTall, back, boatDoorWide+2*boatFrameWide,
		boatFrameWide, boatGlassDepth+boatFrameProud, boatHouse)
	grip := across + boatDoorWide*boatHandleAt*2
	craft.rod(maths.V(grip, foot+boatDoorTall*spaceHalf, back+boatGlassDepth),
		maths.V(grip, foot+boatDoorTall*spaceHalf+boatFrameWide*2,
			back+boatGlassDepth), boatRodThin, boatRust)
}

// boatDoorAt is how far off the middle a door stands, as a share of the
// house's width. Chosen.
const boatDoorAt = 0.2

// roofGear is what stands on a house's roof: a railing round its edge, a
// ladder up its side, the radar and the searchlight, a horn, and the whip
// aerials the design has.
func (craft *boat) roofGear(centre, long, wide, base, roof float64) {
	if !craft.detailed(boatClumps) {
		return
	}
	reach, half := wide*spaceHalf, long*spaceHalf
	craft.roofRailing(centre, long, wide, roof)
	craft.radar(maths.V(0, roof, centre-half*boatRadarAt))
	craft.aerials(centre, half, reach, roof)
	if craft.detailed(boatQuick) {
		craft.ladder(reach+boatGlassDepth, centre+half*boatLadderAt, base, roof,
			boatLadderWide)
	}
}

// roofRailing is a railing round the edge of a roof.
func (craft *boat) roofRailing(centre, long, wide, roof float64) {
	reach, half := wide*spaceHalf, long*spaceHalf
	corners := []maths.Vec{maths.V(-reach, roof, centre-half),
		maths.V(reach, roof, centre-half), maths.V(reach, roof, centre+half),
		maths.V(-reach, roof, centre+half)}
	for index, corner := range corners {
		craft.railing(corner, corners[(index+1)%len(corners)], boatRoofRail,
			boatRoofPitch)
	}
}

// The roof's fittings: the railing's height and its pitch, how far
// forward the radar stands as a share of half the house's length, how far
// aft the ladder stands as a share of the same, and how wide the ladder
// is. All chosen.
const (
	boatRoofRail   = 0.9
	boatRoofPitch  = 1.0
	boatRadarAt    = 0.3
	boatLadderAt   = 0.5
	boatLadderWide = 0.4
)

// radar is a radar's pedestal, its housing and its bar, standing at a
// place on a roof.
func (craft *boat) radar(at maths.Vec) {
	top := at.Add(maths.V(0, boatRadarHigh, 0))
	craft.rod(at.Sub(maths.V(0, boatDeckLift, 0)), top, boatRodThin*2, boatSteel)
	craft.block(top.X, top.Y, top.Z, boatRadarBox.X, boatRadarBox.Y,
		boatRadarBox.Z, boatHouse)
	craft.block(top.X, top.Y+boatRadarBox.Y, top.Z, boatRadarBar.X,
		boatRadarBar.Y, boatRadarBar.Z, boatSteel)
}

// A radar's housing and its bar, width, height and length, and its
// pedestal's height. Chosen.
var (
	boatRadarBox = maths.V(0.3, 0.12, 0.25)
	boatRadarBar = maths.V(1.2, 0.1, 0.06)
)

const boatRadarHigh = 0.5

// aerials are the whip aerials the design has, a thin rod rising a little
// out of true from the corners of a roof, a horn, and a searchlight on its
// post.
func (craft *boat) aerials(centre, half, reach, roof float64) {
	for index := range craft.design.count("aerials", craft.from) {
		turn := boatSide(index)
		along := centre + half*boatAerialAt*boatSide(index/2)
		foot := maths.V(turn*reach*boatAerialSet, roof, along)
		lean := craft.from.swing(housing, boatDrawGear+index, 1) *
			boatAerialLean
		tall := boatAerialHigh * (spaceOne +
			craft.from.draw(housing, boatDrawGear+index, 2))
		craft.rod(foot, foot.Add(maths.V(lean, tall, 0)), boatAerialRod,
			boatSteel)
	}
	horn := maths.V(-reach*boatAerialSet, roof+boatRodThin,
		centre-half*boatAerialAt)
	craft.rod(horn, horn.Add(maths.V(0, 0, boatHorn)), boatHornRadius,
		boatSteel)
	craft.searchlight(maths.V(reach*boatAerialSet, roof,
		centre-half*boatSearchAt))
}

// The aerials, the horn and the searchlight: how far along the roof the
// aerials stand as a share of half its length, how far out as a share of
// half its width, how far they lean and how tall they are, and how thick;
// the horn's length and radius; and how far forward the searchlight
// stands. All chosen.
const (
	boatAerialAt   = 0.7
	boatAerialSet  = 0.8
	boatAerialLean = 0.3
	boatAerialHigh = 1.5
	boatAerialRod  = 0.012
	boatHorn       = 0.35
	boatHornRadius = 0.07
	boatSearchAt   = 0.8
)

// searchlight is a searchlight: a post, a drum looking ahead, and its lens.
func (craft *boat) searchlight(at maths.Vec) {
	top := at.Add(maths.V(0, boatSearchPost, 0))
	craft.rod(at.Sub(maths.V(0, boatDeckLift, 0)), top, boatRodThin*2, boatSteel)
	craft.rod(top, top.Add(maths.V(0, 0, -boatSearchLong)), boatSearchRadius,
		boatSteel)
	craft.block(top.X, top.Y-boatSearchRadius, top.Z-boatSearchLong,
		boatSearchRadius*2, boatSearchRadius*2, boatGlassDepth, boatWhiteLens)
}

// A searchlight's post, its drum's length and radius. Chosen.
const (
	boatSearchPost   = 0.4
	boatSearchLong   = 0.35
	boatSearchRadius = 0.12
)

// funnel is a funnel standing on a roof, its height from the design's
// figures, with a cap of rust.
func (craft *boat) funnel(across, along, roof float64) {
	top := roof + craft.design.pick("funnel-height", craft.from)
	craft.rod(maths.V(across, roof-boatDeckLift, along),
		maths.V(across, top, along), boatFunnelWide, boatSteel)
	craft.rod(maths.V(across, top-boatFunnelCap, along),
		maths.V(across, top, along), boatFunnelWide*boatFunnelFlare,
		boatRust)
}

// boatFunnelFlare is how much wider than its funnel a cap is. Chosen.
const boatFunnelFlare = 1.1

// sternPost is a lamp on a post at the transom, on the centreline.
func (craft *boat) sternPost() {
	stern := craft.hull.length*spaceHalf - boatTransomGap
	deck := craft.hull.deckAt(stern)
	craft.mounts.stern = maths.V(0, deck+boatPostHigh, stern)
	craft.rod(maths.V(0, deck-boatDeckLift, stern), craft.mounts.stern,
		boatPost, boatSteel)
}
