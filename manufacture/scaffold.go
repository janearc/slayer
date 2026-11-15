package manufacture

import (
	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

// the stage scaffold: one tower of the scaffolding a festival's stage
// stands on, and hangs its lasers and its pyro from. 2026-11-10:
// "the stage is a quarter mile long and it's built on huge scaffolds".
// like every row, its numbers are made up to sound right; see kinds.go.
//
// a ringlock system's, from memory and not looked up: a 48.3 mm tube,
// bays 2.57 m square, a lift every 2 m. standards stand at every bay's
// corners, ledgers join them level at every lift, and braces cross some
// of the outer bays, as the seed picks. plank decks lie at some lifts
// and at the top, and a guard rail runs a metre over the top deck.

// scaffolding is the row of a stage scaffold: its bay, its lift and
// its tube's radius; how many bays across and deep, and how many lifts
// high; a deck every so many lifts, as well as the top's; how thick a
// deck and how high its rail; the share of the outer bays braced; how
// many sides its tubes are cut into; whether it carries a show's gear;
// and how it is dressed.
type scaffolding struct {
	bay, lift, tube measure.Length
	deck, rail      measure.Length
	across, deep    int
	lifts, decked   int
	braced          float64
	around          int
	geared          bool
	dressed         looks
}

// the scaffold's matters: its steel, its decks' planks, and its gear's
// black housings. the slots are only the keys of its looks, so the
// planks and the gear borrow two of the spacecraft's spare ones.
const (
	scaffoldSteel = body
	scaffoldDeck  = spaceSecondary
	scaffoldGear  = spaceFoil
)

// plankGap is the gap a deck's planks leave around them in a bay, in
// metres: a little light comes through a deck, as it does through a
// real one.
const plankGap = 0.03

// scaffoldLooks is galvanised steel, dark planks and black housings.
func scaffoldLooks() looks {
	return looks{
		scaffoldSteel: plain("galvanised steel", 0.42, 0.43, 0.45),
		scaffoldDeck:  plain("deck planks", 0.09, 0.085, 0.08),
		scaffoldGear:  plain("black housing", 0.04, 0.04, 0.045)}
}

// scaffoldRow is a bare stage scaffold: six bays across and three
// deep, twenty lifts high, about 15 by 8 by 40 m, decked every fifth
// lift, half its outer bays braced.
func scaffoldRow() scaffolding {
	return scaffolding{bay: measure.Metres(2.57), lift: measure.Metres(2),
		tube: measure.Metres(0.024), deck: measure.Metres(0.06),
		rail: measure.Metres(1), across: 6, deep: 3, lifts: 20, decked: 5,
		braced: 0.5, around: 8, dressed: scaffoldLooks()}
}

// stageTowerRow is the tower a show's lasers and pyro stand on: a
// smaller scaffold, four bays across and two deep, ten lifts high,
// about 10 by 5 by 20 m, decked at its middle and its top, and carrying
// its gear (see gear).
func stageTowerRow() scaffolding {
	row := scaffoldRow()
	row.across, row.deep, row.lifts = 4, 2, 10
	row.geared = true
	return row
}

// made is a scaffold tower from a seed, its foot centred on the thing's
// origin, every tube of a length one shared mesh. it refuses a seed
// never sown, a cut that cannot close a ring, and a tube or bay that
// cannot be a tube.
func (row scaffolding) made(from seed) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	fine, err := cutInto(row.around)
	if err != nil {
		return good{}, err
	}
	frame := row.frame()
	assembly := &instrumentAssembly{fine: fine,
		object: good{dressed: row.dressed}}
	frame.standards(assembly)
	frame.ledgers(assembly)
	frame.braces(assembly, from)
	frame.decks(assembly)
	if row.geared {
		frame.gear(assembly)
	}
	made, err := assembly.finish(frame.bounds())
	if err != nil {
		return good{}, err
	}
	if err := assembly.shareComponents(); err != nil {
		return good{}, err
	}
	return *made, nil
}

// tower is a scaffold's measures in the scene's unit, its foot centred
// on the origin: x across, z deep, y up.
type tower struct {
	row                   scaffolding
	bay, lift, tube, deck float64
	wide, depth, top      float64
	rail                  float64
}

// frame is the row's tower in the scene's unit.
func (row scaffolding) frame() tower {
	bay := inScene(row.bay)
	lift := inScene(row.lift)
	return tower{row: row, bay: bay, lift: lift, tube: inScene(row.tube),
		deck: inScene(row.deck), wide: float64(row.across) * bay,
		depth: float64(row.deep) * bay, top: float64(row.lifts) * lift,
		rail: inScene(row.rail)}
}

// corner is the node at bay corner i across and j deep, at a height.
func (at tower) corner(i, j int, height float64) maths.Vec {
	return maths.V(float64(i)*at.bay-at.wide/2, height,
		float64(j)*at.bay-at.depth/2)
}

// pipe is a tube of the tower's steel from a node along a run. the
// runs are whole bays and lifts, not one node less another, so every
// tube of a length is the same mesh.
func (at tower) pipe(assembly *instrumentAssembly, from, run maths.Vec) {
	assembly.rodAlong(from, run, at.tube, scaffoldSteel)
}

// standards are the uprights, one at every bay corner, from the ground
// to the guard rail over the top deck.
func (at tower) standards(assembly *instrumentAssembly) {
	up := maths.V(0, at.top+at.rail, 0)
	for i := 0; i <= at.row.across; i++ {
		for j := 0; j <= at.row.deep; j++ {
			at.pipe(assembly, at.corner(i, j, 0), up)
		}
	}
}

// ledgers are the level tubes: at every lift, joining every corner to
// the next across and the next deep; and the guard rail, round the
// outside only, a rail over the top deck.
func (at tower) ledgers(assembly *instrumentAssembly) {
	for k := 1; k <= at.row.lifts; k++ {
		at.ring(assembly, float64(k)*at.lift, false)
	}
	at.ring(assembly, at.top+at.rail, true)
}

// ring is one level of ledgers at a height: every bay's, or only the
// outer ones'.
func (at tower) ring(assembly *instrumentAssembly, height float64,
	outer bool,
) {
	across, deep := maths.V(at.bay, 0, 0), maths.V(0, 0, at.bay)
	for j := 0; j <= at.row.deep; j++ {
		if outer && j != 0 && j != at.row.deep {
			continue
		}
		for i := 0; i < at.row.across; i++ {
			at.pipe(assembly, at.corner(i, j, height), across)
		}
	}
	for i := 0; i <= at.row.across; i++ {
		if outer && i != 0 && i != at.row.across {
			continue
		}
		for j := 0; j < at.row.deep; j++ {
			at.pipe(assembly, at.corner(i, j, height), deep)
		}
	}
}

// braces cross the outer bays the seed picks, lift by lift, each
// leaning the other way from the bay beside it and the lift below, so
// a braced face climbs in a zigzag.
func (at tower) braces(assembly *instrumentAssembly, from seed) {
	faces := at.faces()
	for k := 0; k < at.row.lifts; k++ {
		low := float64(k) * at.lift
		for face, bays := range faces {
			for b, bay := range bays {
				if from.draw(picking, k*len(faces)+face, b) >=
					at.row.braced {
					continue
				}
				foot, head := bay[0], bay[1]
				if (b+k)%2 == 1 {
					foot, head = head, foot
				}
				run := maths.V(float64(head[0]-foot[0])*at.bay, at.lift,
					float64(head[1]-foot[1])*at.bay)
				at.pipe(assembly, at.corner(foot[0], foot[1], low), run)
			}
		}
	}
}

// faces is each outer face's bays, front, back, left and right, each
// bay its two corners as bay indices across and deep.
func (at tower) faces() [4][][2][2]int {
	var faces [4][][2][2]int
	for i := 0; i < at.row.across; i++ {
		faces[0] = append(faces[0], [2][2]int{{i, 0}, {i + 1, 0}})
		faces[1] = append(faces[1],
			[2][2]int{{i, at.row.deep}, {i + 1, at.row.deep}})
	}
	for j := 0; j < at.row.deep; j++ {
		faces[2] = append(faces[2], [2][2]int{{0, j}, {0, j + 1}})
		faces[3] = append(faces[3],
			[2][2]int{{at.row.across, j}, {at.row.across, j + 1}})
	}
	return faces
}

// decks are a plank in every bay at every decked lift and at the top,
// laid on the ledgers.
func (at tower) decks(assembly *instrumentAssembly) {
	plank := at.bay - plankGap
	for k := 1; k <= at.row.lifts; k++ {
		if k%at.row.decked != 0 && k != at.row.lifts {
			continue
		}
		height := float64(k)*at.lift + at.tube
		for i := 0; i < at.row.across; i++ {
			for j := 0; j < at.row.deep; j++ {
				foot := at.corner(i, j, height).Add(
					maths.V(plankGap/2, 0, plankGap/2))
				assembly.box(plank, at.deck, plank, foot, scaffoldDeck)
			}
		}
	}
}

// the gear's measures, in metres: each housing's width, height and
// depth, and where it stands.
const (
	// the speaker columns hang this far out in front of the front face,
	// and their reach is all the gear stands out of the scaffold
	hangOut   = 0.45
	gearReach = hangOut + 0.55/2
	// a column is ten cabinets, each this tall with a hand's gap
	cabinets, cabinetPitch = 10, 0.34
)

// gear is what a stage tower carries, all in black housings, on the
// face it turns to the audience (its +z): on the top deck two laser
// heads at the front corners, two flame units and their nozzles
// between them, and six moving heads along the front rail; a column of
// speaker cabinets hung under each front corner of the top deck; and
// three road cases on every lower deck.
func (at tower) gear(assembly *instrumentAssembly) {
	put := func(w, h, d float64, foot maths.Vec) {
		assembly.box(w, h, d, foot.Sub(maths.V(w/2, 0, d/2)), scaffoldGear)
	}
	floor := at.top + at.tube + at.deck
	front, half := at.depth/2, at.wide/2
	for _, side := range []float64{-1, 1} {
		put(0.6, 0.45, 0.7, maths.V(side*(half-0.8), floor, front-0.6))
		unit := maths.V(side*1.6, floor, front-0.7)
		put(0.5, 0.5, 0.5, unit)
		assembly.rodAlong(unit.Add(maths.V(0, 0.5, 0)), maths.V(0, 0.35, 0),
			0.08, scaffoldGear)
	}
	for i := 0; i < 6; i++ {
		across := -half + at.bay/2 + float64(i)*(at.wide-at.bay)/5
		put(0.4, 0.55, 0.4, maths.V(across, floor, front-0.3))
	}
	for _, side := range []float64{-1, 1} {
		hang := maths.V(side*(half-0.9), at.top, front+hangOut)
		assembly.rodAlong(hang, maths.V(0, -0.4, 0), 0.02, scaffoldGear)
		for c := 1; c <= cabinets; c++ {
			put(1.1, 0.32, 0.55,
				hang.Sub(maths.V(0, 0.4+float64(c)*cabinetPitch, 0)))
		}
	}
	for k := at.row.decked; k < at.row.lifts; k += at.row.decked {
		lower := float64(k)*at.lift + at.tube + at.deck
		for _, across := range []float64{-half + 1.2, -0.4, half - 1.6} {
			put(1.2, 0.8, 0.6, maths.V(across, lower, -front+0.8))
		}
	}
}

// bounds is the tower's box: its footprint and its rail, and the
// tubes' girth outside them, below the ground too, where a brace's
// foot ring, square to its slope, dips; and the gear hung out in
// front, when it carries any.
func (at tower) bounds() maths.Box {
	reach := at.tube
	if at.row.geared {
		reach = gearReach
	}
	low := maths.V(-at.wide/2-at.tube, -at.tube, -at.depth/2-at.tube)
	high := maths.V(at.wide/2+at.tube, at.top+at.rail+at.tube,
		at.depth/2+reach)
	return maths.SegmentBox(low, high, spaceZero)
}
