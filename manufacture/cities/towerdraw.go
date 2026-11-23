package cities

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// towerLooks are the materials of a tower, drawn once: the glass of its
// body, the grid on it, the dark plant of its mechanical floors, its
// podium, its beacons, and the panels of an unlit crown. The body is what
// the faces stand on, the glass unless the skin is a wall, and skin the
// materials the skin dresses it in.
type towerLooks struct {
	glass, grid, plant, podium, beacon, panel *thing.Material
	body                                      *thing.Material
	skin                                      skinLooks
	// fine is what the demo level draws its small parts in, and lobby
	// the stone of its ground floor's plinth.
	fine  fineLooks
	lobby *thing.Material
}

// looks draws the tower's materials from the culture.
func (tower *Tower) looks() (towerLooks, error) {
	drawing, at := tower.drawing, place{}
	glass, err := material(roleGlazing)
	if err != nil {
		return towerLooks{}, err
	}
	pick := func(name string) option {
		return drawing.pick(name, drawing.options(name), at)
	}
	lamp := pick("beacon-colour")
	glow := colourOf(lamp).Scale(drawing.fixed("beacon-glow"))
	made := towerLooks{glass: glass, grid: paint("grid", pick("grid-colour")),
		plant:  paint("plant", pick("plant-colour")),
		podium: paint("podium", pick("podium-colour")),
		panel:  paint("crown panel", pick("crown-panel-colour")),
		beacon: &thing.Material{Name: "cities beacon",
			Colour: maths.Grey(0.05), Glow: glow}}
	if err := tower.skinned(&made); err != nil {
		return towerLooks{}, err
	}
	if drawing.level == demoLevel {
		made.fine = tower.fined(made)
		made.body = made.fine.wall
		tower.surfaced(&made)
		made.lobby = tower.lobbyOf(made)
	}
	if tower.Cyber {
		made = tower.wetted(made)
	}
	return made, drawing.trouble
}

// coreSide is the side of the square core of a tower of this many
// storeys, in metres. A taller tower needs more lifts, because a lift
// serves only so many storeys before its journeys are too long to wait
// for, and each lift needs its shaft, so the core grows with the storeys:
// the lifts, and a fixed area for stairs, as a square.
func coreSide(storeys int) (float64, error) {
	drawing, err := newDrawer(towerCulture, 0)
	if err != nil {
		return 0, err
	}
	lifts := math.Ceil(float64(storeys) / drawing.fixed("lift-storeys"))
	area := lifts*drawing.fixed("lift-area") + drawing.fixed("stair-area")
	return math.Sqrt(area), drawing.trouble
}

// Design is the tower at a time, in seconds, its segments nodes that are
// drawn at the level they are drawn at. At the hull level a segment is
// its massing only, with no windows, grid or plant bands, but for the
// lights it keeps when the level it is made for is finer than the hull.
// The level given is the finest: the materials and the rooms' depth are
// drawn for it. The time moves only the lights of a lit crown.
func (tower *Tower) Design(level Detail, seconds float64) (Design,
	error) {
	tower.drawing.level = level
	looks, err := tower.looks()
	if err != nil {
		return Design{}, err
	}
	base := tower.baseHeight()
	parts := tower.podium(looks, base)
	tower.grown, tower.finest = nil, level
	for index := range tower.Shafts {
		shaft, err := tower.shaftParts(index, looks, base, seconds)
		if err != nil {
			return Design{}, err
		}
		parts = append(parts, shaft...)
		parts = append(parts, tower.core(index, looks, base)...)
		parts = append(parts, tower.top(index, looks, base, seconds)...)
		parts = append(parts, tower.roof(index, looks)...)
		if tower.Cyber {
			parts = append(parts, tower.searchlights(index, looks)...)
		}
	}
	parts = append(parts, tower.joins(looks, base)...)
	return Design{Name: "tower", Elements: parts}, tower.drawing.trouble
}

// core is the shaft's core, a square box in the middle of it, sized from
// the shaft's storeys by CoreSide, from the base to a lift overrun above
// the roof. It is inside the glass but for the overrun.
func (tower *Tower) core(index int, looks towerLooks, base float64) []placedElement {
	shaft := tower.Shafts[index]
	side, err := coreSide(shaft.Storeys)
	tower.drawing.fail(err)
	storey, rise, whole := tower.storeyHeight(), 0.0, true
	for number, segment := range shaft.Segments {
		_, top, err := shaft.layer(number)
		tower.drawing.fail(err)
		if inradius(top) < side/2+tower.drawing.fixed("recess") {
			whole = false
			break
		}
		rise += float64(segment.Storeys) * storey
	}
	if whole {
		rise += tower.drawing.fixed("core-overrun")
	}
	if !(rise > 0) {
		return nil
	}
	return []placedElement{put("core", Box{side, rise, side}, looks.plant,
		maths.V(shaft.Centre.X-side/2, base, shaft.Centre.Z-side/2))}
}

// baseHeight is the height of the podium the shafts stand on: for a single
// shaft, the open storey at its foot if it has one, and otherwise nothing.
func (tower *Tower) baseHeight() float64 {
	if len(tower.Shafts) < 2 {
		return tower.foot
	}
	return tower.drawing.fixed("podium-height")
}

// storeyHeight is the height of every storey of the tower.
func (tower *Tower) storeyHeight() float64 {
	return tower.drawing.number("storey-height", place{})
}

// height is the height of a shaft's roof above the ground, in metres.
func (tower *Tower) height(shaft int) float64 {
	return tower.baseHeight() + float64(tower.Shafts[shaft].Storeys)*
		tower.storeyHeight()
}

// podium is the base the shafts stand on, a box a margin wider than all
// of them together, when there is more than one shaft.
func (tower *Tower) podium(looks towerLooks, height float64) []placedElement {
	if len(tower.Shafts) < 2 || !(height > 0) {
		return nil
	}
	margin := tower.drawing.fixed("podium-margin")
	first, last := tower.Shafts[0], tower.Shafts[len(tower.Shafts)-1]
	radius := first.radius()
	deep := 0.0
	for _, shaft := range tower.Shafts {
		deep = math.Max(deep, shaft.radius())
	}
	left := first.Centre.X - radius - margin
	right := last.Centre.X + last.radius() + margin
	return []placedElement{put("podium", Box{right - left, height,
		2 * (deep + margin)}, looks.podium,
		maths.V(left, 0, -deep-margin))}
}

// shaftParts is a shaft's spine, if it is nested, and a node for each of
// its segments: the segment's body and, above the hull level, its faces
// dressed. The body is a prism for each part of each segment, drawn in
// from the face by the recess, in glass or in the wall of its skin.
func (tower *Tower) shaftParts(index int, looks towerLooks,
	base, seconds float64) ([]placedElement, error) {
	shaft := tower.Shafts[index]
	var parts []placedElement
	if shaft.Nested {
		spine, err := tower.spine(index, looks, base)
		if err != nil {
			return nil, err
		}
		parts = spine
	}
	run := segmentRun{shaft: index, seconds: seconds, looks: looks,
		base: base}
	for number, segment := range shaft.Segments {
		bottom, top, err := shaft.layer(number)
		if err != nil {
			return nil, err
		}
		run.number, run.bottom, run.top = number, bottom, top
		run.height = float64(segment.Storeys) * tower.storeyHeight()
		run.cavity = tower.cavityOf(bottom, top)
		run.body, err = tower.body(bottom, top, run.base, run.height, looks,
			run.cavity)
		if err != nil {
			return nil, err
		}
		run.grown = len(tower.grown)
		tower.grown = append(tower.grown, map[int][]Tree{})
		parts = append(parts, deferred(tower.segmentNode(run)))
		run.face += len(frames(bottom, top, run.base, run.height))
		run.first += segment.Storeys
		run.base += run.height
	}
	return parts, tower.drawing.trouble
}

// faceParts is one face dressed in the tower's skin, with the cyberpunk
// layer over it and the planting on it, if the tower has them, and the
// trees of the planting.
func (tower *Tower) faceParts(job faceJob) ([]placedElement, []Tree) {
	parts := job.dress()
	if tower.Cyber {
		parts = append(parts, job.cyberFace()...)
	}
	if !tower.Planted {
		return parts, nil
	}
	grown, trees := job.planted()
	return append(parts, grown...), trees
}

// ledge is how far the plan of a segment steps in from the top of the
// segment below, in metres, which is the width of the ledge the setback
// leaves along its faces: nought for the first segment and for a segment
// that is as wide as the one below.
func (tower *Tower) ledge(shaft shaft, number int) float64 {
	if number == 0 {
		return 0
	}
	drop := shaft.Segments[number-1].Top.radius() -
		shaft.Segments[number].Bottom.radius()
	return math.Max(0, drop)
}

// spine is the core a nested shaft's body is wrapped round: the core's
// plan, run up from the foot through all of the body to where the core
// itself rises out of it.
func (tower *Tower) spine(index int, looks towerLooks,
	base float64) ([]placedElement, error) {
	shaft := tower.Shafts[index]
	last := len(shaft.Segments) - 1
	plan, _, err := shaft.layer(last)
	if err != nil {
		return nil, err
	}
	var storeys int
	for _, segment := range shaft.Segments[:last] {
		storeys += segment.Storeys
	}
	var parts []placedElement
	for _, corners := range plan {
		parts = append(parts, put("spine", prism{Bottom: polygon{corners},
			Top: polygon{corners}, Height: float64(storeys) *
				tower.storeyHeight()}, looks.podium, maths.V(0, base, 0)))
	}
	return parts, nil
}

// cavityOf is how deep the rooms behind the glass of a segment are at the
// demo level, which is the culture's depth but never more than a share of
// the segment's inradius, so that a narrow top is not hollowed out. Below
// the demo level a segment has no rooms.
func (tower *Tower) cavityOf(bottom, top [][]outlineCorner) float64 {
	drawing := tower.drawing
	if drawing.level != demoLevel {
		return 0
	}
	return min(drawing.fixed("room-cavity"),
		drawing.fixed("room-cavity-share")*min(inradius(bottom), inradius(top)))
}

// body is the prisms of a segment's parts, each drawn in from its faces
// by the recess and the cavity of its rooms.
func (tower *Tower) body(bottom, top [][]outlineCorner, base, height float64,
	looks towerLooks, cavity float64) ([]placedElement, error) {
	recess := tower.drawing.fixed("recess") + cavity
	name := "glass"
	if looks.body != looks.glass {
		name = "body"
	}
	var parts []placedElement
	for index := range bottom {
		low, err := inset(bottom[index], recess)
		if err != nil {
			return nil, err
		}
		high, err := inset(top[index], recess)
		if err != nil {
			return nil, err
		}
		parts = append(parts, put(name, prism{Bottom: polygon{low},
			Top: polygon{high}, Height: height}, looks.body,
			maths.V(0, base, 0)))
	}
	return parts, nil
}

// job is the dressing of one face, with its measures drawn.
func (tower *Tower) job(frame faceFrame, shaft, face, first, storeys,
	total int, looks towerLooks) faceJob {
	drawing := tower.drawing
	module := drawing.fixed("window-module")
	if tower.Skin == skinGothic {
		module = drawing.fixed("gothic-bay")
	}
	return faceJob{drawing: drawing, looks: looks, frame: frame,
		shaft: shaft, face: face, storeys: storeys, firstStorey: first,
		shaftStoreys: total, skin: tower.Skin,
		demo:      drawing.level == demoLevel,
		mechEvery: drawing.count("mechanical-every", place{}),
		module:    module,
		mullion:   drawing.fixed("mullion-width"),
		spandrel:  drawing.fixed("spandrel-height"),
		recess:    drawing.fixed("recess"),
		margin:    drawing.fixed("edge-margin"),
		pane:      drawing.fixed("glass-thickness")}
}

// extent is how far along x the plan of the shaft reaches at a height
// above the ground, from the lowest corner to the highest, and whether
// the shaft stands at that height. The plan is the plan at that share of
// the segment's height between its bottom and its top.
func (shaft shaft) extent(at, base, storey float64) (low, high float64,
	ok bool) {
	low, high = math.Inf(1), math.Inf(-1)
	for index, segment := range shaft.Segments {
		height := float64(segment.Storeys) * storey
		if at >= base && at <= base+height {
			bottom, top, err := shaft.layer(index)
			if err != nil {
				return 0, 0, false
			}
			share := (at - base) / height
			for part := range bottom {
				for corner := range bottom[part] {
					x := bottom[part][corner].X +
						(top[part][corner].X-bottom[part][corner].X)*share
					low, high = math.Min(low, x), math.Max(high, x)
				}
			}
			return low, high, true
		}
		base += height
	}
	return 0, 0, false
}

// joins are what joins the shafts: a skybridge between each neighbouring
// pair at one storey, or a deck across all their tops.
func (tower *Tower) joins(looks towerLooks, base float64) []placedElement {
	switch tower.Span {
	case SpanSkybridge:
		return tower.bridges(looks, base)
	case SpanDeck:
		return tower.deck(looks, base)
	}
	return nil
}

// bridges are the skybridges: from the face of one shaft at the storey
// to the face of the next, lapping into each so that it touches both.
func (tower *Tower) bridges(looks towerLooks, base float64) []placedElement {
	drawing, storey := tower.drawing, tower.storeyHeight()
	at := base + float64(tower.SpanStorey)*storey
	lap, deep := drawing.fixed("bridge-lap"), drawing.fixed("bridge-depth")
	height := drawing.fixed("bridge-storeys") * storey
	plate := drawing.fixed("bridge-plate")
	var parts []placedElement
	for index := 0; index+1 < len(tower.Shafts); index++ {
		_, from, _ := tower.Shafts[index].extent(at, base, storey)
		to, _, _ := tower.Shafts[index+1].extent(at, base, storey)
		left, long := from-lap, to-from+2*lap
		parts = append(parts,
			put("bridge", Box{long, height, deep}, looks.glass,
				maths.V(left, at, -deep/2)),
			put("bridge plate", Box{long, plate, deep}, looks.grid,
				maths.V(left, at+height, -deep/2)),
			put("bridge plate", Box{long, plate, deep}, looks.grid,
				maths.V(left, at-plate, -deep/2)))
	}
	return parts
}

// deck is the deck laid across the tops of the shafts, which stand level
// under it: from past the outer face of the first to past that of the
// last, by the overhang the culture draws, with the beacons on its ends.
func (tower *Tower) deck(looks towerLooks, base float64) []placedElement {
	drawing, storey := tower.drawing, tower.storeyHeight()
	top := base + float64(tower.Shafts[0].Storeys)*storey
	left, right, deep := math.Inf(1), math.Inf(-1), 0.0
	for _, shaft := range tower.Shafts {
		low, high, _ := shaft.extent(top, base, storey)
		left, right = math.Min(left, low), math.Max(right, high)
		deep = math.Max(deep, shaft.radius())
	}
	overhang := drawing.fixed("deck-overhang")
	thick := drawing.fixed("deck-thickness")
	left, right = left-overhang, right+overhang
	parts := []placedElement{put("deck", Box{right - left, thick, 2 * deep},
		looks.grid, maths.V(left, top, -deep))}
	return append(parts, tower.beacons(looks, top+thick, []maths.Vec{
		maths.V(left, 0, -deep), maths.V(right, 0, deep)})...)
}

// beacons are the red lights at the given places on a roof at a height,
// if that height is over the least the culture says a tower must be to
// carry them. A beacon is a small box that glows.
func (tower *Tower) beacons(looks towerLooks, height float64,
	at []maths.Vec) []placedElement {
	drawing := tower.drawing
	if height < drawing.fixed("beacon-height") {
		return nil
	}
	size := drawing.fixed("beacon-size")
	var lamps []placedElement
	for _, place := range at {
		lamps = append(lamps, put("beacon", Box{size, size, size},
			looks.beacon, maths.V(place.X-size/2, height, place.Z-size/2)))
	}
	return lamps
}

// check is a refusal of what cannot be drawn: here, shafts that overlap.
func (tower *Tower) check() error {
	for index := 0; index+1 < len(tower.Shafts); index++ {
		left, right := tower.Shafts[index], tower.Shafts[index+1]
		reach := left.Centre.X + left.radius()
		if reach > right.Centre.X-right.radius() {
			return fmt.Errorf("cities: shafts %d and %d overlap", index,
				index+1)
		}
	}
	return nil
}
