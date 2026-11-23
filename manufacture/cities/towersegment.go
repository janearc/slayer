package cities

import (
	"math"
	"slices"

	"github.com/janearc/slayer/maths"
)

// A segment of a tower levelled is a node: whole, its body and, above
// the hull, its faces dressed. When the storeys of one segment would be
// drawn at different levels, its lower storeys near and its crown far,
// it unfurls into bands of storeys, each dressed or bare on its own.
//
// The bands of a segment are a node themselves, and a run of them that
// would all be drawn at one level is drawn as one band. What stands the
// height of a face, as a mullion does, then stands once for the run and
// not once for each band of it. A run opens into its two halves only
// where one of its bands would be drawn at another level than the run,
// so a segment seen from its foot is a few runs and not every band.

// noBand is the key of a segment's trees when it is drawn whole.
const noBand = -1

// leastRun is the fewest bands of a run that can be drawn at different
// levels.
const leastRun = 2 // defined: one band has no other to differ from

// halves is how many runs a run of bands opens into.
const halves = 2 // defined: a run opens into its lower part and its upper

// segmentRun is one segment of a shaft as its node draws it: which shaft
// and segment, its first storey and first face counted up the shaft, the
// slot of its trees, where it stands and how high it is, the time, how
// deep the rooms behind its glass are, its plans at its foot and its
// head, its body, and the tower's looks.
type segmentRun struct {
	shaft, number, first, face, grown int
	base, height, seconds, cavity     float64
	bottom, top                       [][]outlineCorner
	body                              []placedElement
	looks                             towerLooks
}

// storeyBand is a run of a segment's storeys: its index among the bands,
// or noBand for the whole, the storey it starts at counted from the
// segment's foot, and how many storeys it holds.
type storeyBand struct{ index, from, storeys int }

// segmentNode is a segment as a node, in the tower's frame.
func (tower *Tower) segmentNode(run segmentRun) *node {
	whole := storeyBand{index: noBand, storeys: tower.segment(run).Storeys}
	return &node{Bounds: tower.bandBox(run, whole),
		Draw: func(level Detail) ([]placedElement, error) {
			tower.grown[run.grown] = map[int][]Tree{}
			if level == hullLevel {
				return append(slices.Clone(run.body),
					tower.lights(run, whole)...), tower.drawing.trouble
			}
			faces := tower.bandFaces(run, whole, level)
			return append(slices.Clone(run.body), faces...),
				tower.drawing.trouble
		},
		Opens: func(level Detail, levelOf func(maths.Box) Detail) bool {
			cut, err := tower.bands(run)
			return err == nil && level != hullLevel &&
				tower.bandsDiffer(run, cut, level, levelOf)
		},
		Unfurl: func(Detail) ([]placedElement, error) { return tower.banded(run) }}
}

// segment is the segment a run draws.
func (tower *Tower) segment(run segmentRun) segment {
	return tower.Shafts[run.shaft].Segments[run.number]
}

// bands are a segment's storeys cut into the bands the levels file says,
// the last band taking what is left, or why they cannot be.
func (tower *Tower) bands(run segmentRun) ([]storeyBand, error) {
	per, err := band("tower-storeys")
	if err != nil {
		return nil, err
	}
	storeys := tower.segment(run).Storeys
	var cut []storeyBand
	for from := 0; from < storeys; from += per {
		cut = append(cut, storeyBand{index: len(cut), from: from,
			storeys: min(per, storeys-from)})
	}
	return cut, nil
}

// bandsDiffer is whether any of a run of bands is drawn at another level
// than the run whole. A run of one band fills the box the run does, so
// it is at the run's level and does not differ.
func (tower *Tower) bandsDiffer(run segmentRun, cut []storeyBand,
	level Detail, levelOf func(maths.Box) Detail) bool {
	for _, each := range cut {
		if levelOf(tower.bandBox(run, each)) != level {
			return true
		}
	}
	return false
}

// banded is a segment's body and the node of all its bands.
func (tower *Tower) banded(run segmentRun) ([]placedElement, error) {
	cut, err := tower.bands(run)
	if err != nil {
		return nil, err
	}
	tower.grown[run.grown] = map[int][]Tree{}
	return append(slices.Clone(run.body),
		deferred(tower.bandsNode(run, cut))), nil
}

// joined is a run of bands as one band: the index and the first storey
// of the first of them, and the storeys of them all.
func joined(cut []storeyBand) storeyBand {
	whole := storeyBand{index: cut[0].index, from: cut[0].from}
	for _, each := range cut {
		whole.storeys += each.storeys
	}
	return whole
}

// bandsNode is a run of a segment's bands as a node. It is drawn as one
// band, bare at the hull and its faces dressed at any other level, and
// its trees are kept under its first band. It opens into its two halves
// where any of its bands would be drawn at another level than the run;
// a run of one band has no halves and never opens.
func (tower *Tower) bandsNode(run segmentRun, cut []storeyBand) *node {
	whole := joined(cut)
	made := &node{Bounds: tower.bandBox(run, whole),
		Draw: func(level Detail) ([]placedElement, error) {
			for _, each := range cut {
				delete(tower.grown[run.grown], each.index)
			}
			if level == hullLevel {
				return tower.lights(run, whole), tower.drawing.trouble
			}
			return tower.bandFaces(run, whole, level), tower.drawing.trouble
		}}
	if len(cut) < leastRun {
		return made
	}
	made.Opens = func(level Detail, levelOf func(maths.Box) Detail) bool {
		return tower.bandsDiffer(run, cut, level, levelOf)
	}
	made.Unfurl = func(Detail) ([]placedElement, error) {
		half := len(cut) / halves
		return []placedElement{deferred(tower.bandsNode(run, cut[:half])),
			deferred(tower.bandsNode(run, cut[half:]))}, nil
	}
	return made
}

// lights are what of a band's faces gives light when it is drawn bare,
// as a city's distant lit level keeps its lit panes: the parts that glow
// and the lamps, without their trees. A tower drawn for the hull keeps
// none, as it never had them.
func (tower *Tower) lights(run segmentRun, each storeyBand) []placedElement {
	if tower.finest == hullLevel {
		return nil
	}
	faces := tower.bandFaces(run, each, quickLevel)
	delete(tower.grown[run.grown], each.index)
	var lit []placedElement
	for _, part := range faces {
		solid, ok := part.Element.(solid)
		if ok && (solid.Lamp != nil || solid.Material != nil &&
			solid.Material.Glow != (maths.Light{})) {
			lit = append(lit, part)
		}
	}
	return lit
}

// cut is a band's plans at its foot and its head, where it stands and
// how high it is. The whole segment is its own plans, unchanged.
func (tower *Tower) cut(run segmentRun, each storeyBand) (bottom,
	top [][]outlineCorner, base, height float64) {
	storeys := float64(tower.segment(run).Storeys)
	storey := tower.storeyHeight()
	foot := float64(each.from) / storeys
	head := float64(each.from+each.storeys) / storeys
	return between(run.bottom, run.top, foot),
		between(run.bottom, run.top, head),
		run.base + float64(each.from)*storey,
		float64(each.storeys) * storey
}

// between is the plan a share of the way from a segment's foot to its
// head: the prism is straight between them, so a band of it is too.
func between(bottom, top [][]outlineCorner, share float64) [][]outlineCorner {
	switch share {
	case 0:
		return bottom
	case 1:
		return top
	}
	plan := make([][]outlineCorner, len(bottom))
	for part := range bottom {
		plan[part] = make([]outlineCorner, len(bottom[part]))
		for index, low := range bottom[part] {
			high := top[part][index]
			plan[part][index] = outlineCorner{low.X + (high.X-low.X)*share,
				low.Z + (high.Z-low.Z)*share}
		}
	}
	return plan
}

// bandFaces is a band's faces dressed at a level, its trees kept by the
// band. A face is drawn part by part only where both the band and the
// design are at the demo level, since only then are its fine materials
// made; its rooms are as deep as the body was drawn for either way.
func (tower *Tower) bandFaces(run segmentRun, each storeyBand,
	level Detail) []placedElement {
	shaft, segment := tower.Shafts[run.shaft], tower.segment(run)
	bottom, top, base, height := tower.cut(run, each)
	crowned := each.from+each.storeys == segment.Storeys
	var parts []placedElement
	var trees []Tree
	for face, frame := range frames(bottom, top, base, height) {
		job := tower.job(frame, run.shaft, run.face+face,
			run.first+each.from, each.storeys, shaft.Storeys, run.looks)
		job.segment, job.use, job.style = run.number, segment.Use,
			segment.Style
		job.last = crowned && run.number == len(shaft.Segments)-1
		if crowned {
			job.ledge = tower.ledge(shaft, run.number)
		}
		job.seconds = run.seconds
		job.beneath = base - run.base
		job.demo = job.demo && level == demoLevel
		job.cavity = run.cavity
		drawn, grown := tower.faceParts(job)
		parts, trees = append(parts, drawn...), append(trees, grown...)
	}
	tower.grown[run.grown][each.index] = trees
	if level != demoLevel {
		parts = append(parts, tower.filling(run, each)...)
	}
	return parts
}

// filling is the wall of a band at the face's recess, for a band drawn
// below the demo level in a design made for it. The body of such a design
// is drawn in by the rooms' cavity, and only the demo level makes the
// rooms, so without a wall at the recess the glass of the band would show
// an empty cavity behind it. A design made for no rooms has none.
func (tower *Tower) filling(run segmentRun, each storeyBand) []placedElement {
	if !(run.cavity > 0) {
		return nil
	}
	bottom, top, base, height := tower.cut(run, each)
	wall, err := tower.body(bottom, top, base, height, run.looks, 0)
	tower.drawing.fail(err)
	return wall
}

// bandBox is the box a band of a segment fills, in the tower's frame.
func (tower *Tower) bandBox(run segmentRun, each storeyBand) maths.Box {
	bottom, top, base, height := tower.cut(run, each)
	low, high := math.Inf(1), math.Inf(-1)
	west, east := low, high
	for _, plan := range [][][]outlineCorner{bottom, top} {
		for _, part := range plan {
			for _, corner := range part {
				west, east = math.Min(west, corner.X), math.Max(east, corner.X)
				low, high = math.Min(low, corner.Z), math.Max(high, corner.Z)
			}
		}
	}
	return maths.Box{Min: maths.V(west, base, low),
		Max: maths.V(east, base+height, high)}
}
