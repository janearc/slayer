package birds

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/maths"
)

// A feather is a bone of the skeleton in all but name: it is jointed at
// its root to the bone that carries it, a primary to the hand, a
// secondary to the ulna, a tail feather to the pygostyle, so that a
// pose that turns the wing's joints carries its feathers with it.
//
// Each is drawn as a blade: a thin vane either side of its shaft,
// narrow at the root, full along most of its length and rounded at the
// tip, bowed a little along its length.

// The groups a feather belongs to.
const (
	birdPrimary      = "primary"
	birdSecondary    = "secondary"
	birdGreater      = "greater-covert"
	birdMedian       = "median-covert"
	birdLesser       = "lesser-covert"
	birdScapular     = "scapular"
	birdAlula        = "alula"
	birdRectrix      = "rectrix"
	birdUpperCovert  = "upper-tail-covert"
	birdUnderCovert  = "under-tail-covert"
	birdPlume        = "plume"
	birdFeatherShaft = .0016 // a shaft's half thickness, of the length
)

// The primaries fan from pointing back at the wrist to pointing out at
// the tip; the angles from straight back, in radians, and the shortest
// primary's length as a share of the longest, are picked.
const (
	birdPrimaryFirst    = .25
	birdPrimaryLast     = 1.35
	birdPrimaryShortest = .55
	birdPrimaryOuter    = .92 // the outermost primary, of the longest
	birdSecondaryLean   = .12 // a secondary's lean out from straight back
	birdSecondaryTaper  = .25 // how much shorter the innermost is
	birdTailFan         = .45 // the spread tail's outer feather's angle
	birdTailRoot        = .25 // the tail's roots, of the half width
	birdAlulaFeathers   = 3   // feathers on the alula
)

// Feather sizes as shares of its own length or of its remex's; picked.
const (
	birdRemexWidth   = .24   // a flight feather's vane, of its length
	birdCovertWidth  = .50   // a covert's
	birdRectrixWidth = .20   // a tail feather's
	birdPlumeWidth   = .03   // a plume's, a few barbs
	birdGreaterShare = .55   // a greater covert, of its remex
	birdMedianShare  = .32   // a median covert, of its remex
	birdLesserShare  = .18   // a lesser covert, of its remex
	birdTailCovert   = .50   // a tail covert, of its rectrix
	birdVaneThick    = .0007 // a vane's half thickness, of the length
	birdJitter       = .03   // the largest turn a seed gives a feather
)

// birdFeather is one feather: its bone's name, its group, the side it
// is on, and its share across its row from the first to the last.
type birdFeather struct {
	name, group string
	sign, share float64
}

// birdVane is a feather's form: a blade from root to tip, wide across,
// thin through its face, the face looking toward up.
type birdVane struct {
	root, tip    maths.Vec
	width, thick float64
	up           maths.Vec
	bow          float64
}

// Middle is the feather's shaft, root to tip.
func (vane birdVane) Middle() []maths.Vec {
	return []maths.Vec{vane.root, vane.tip}
}

// Build adds the feather to the mesh.
func (vane birdVane) Build(points []maths.Vec, faces [][3]int,
	place func(maths.Vec) maths.Vec) ([]maths.Vec, [][3]int) {
	const rings = 10 // picked: enough for a rounded tip
	span := vane.tip.Sub(vane.root)
	path := func(t float64) maths.Vec {
		sag := vane.up.Scale(-vane.bow * span.Len() * math.Sin(math.Pi*t))
		return vane.root.Add(span.Scale(t)).Add(sag)
	}
	half := func(t float64) (float64, float64) {
		return vane.width / 2 * birdVaneShape(t), vane.thick
	}
	return skeleton.Sweep(points, faces, rings, path, half, vane.up, place)
}

// birdVaneShape is the share of its width a vane has t of the way from
// root to tip: rising from the root, full, then rounding to the tip.
func birdVaneShape(t float64) float64 {
	const rise, round, least = .15, .3, .08 // picked
	shape := math.Min(1, .35+.65*t/rise)
	if over := t - (1 - round); over > 0 {
		shape *= math.Sqrt(math.Max(0, 1-math.Pow(over/round, 2)))
	}
	return math.Max(least, shape)
}

// feather adds one feather as a bone of the skeleton, with its form.
func (body *birdBody) feather(name, parent, group string, sign, share float64,
	root, way maths.Vec, length, width float64) error {
	const bow = .03 // picked: a feather bows a little downward
	m := body.measures
	tip := root.Add(way.Unit().Scale(length))
	vane := birdVane{root: root, tip: tip, width: width * length,
		thick: birdVaneThick * m.length, up: birdUp, bow: bow}
	err := body.add(name, parent, root, tip, birdFeatherShaft*m.length,
		vane)
	if err != nil {
		return err
	}
	body.feathers = append(body.feathers, birdFeather{name: name,
		group: group, sign: sign, share: share})
	return nil
}

// hangFeathers puts the flight feathers, their coverts, the alula's,
// the scapulars, the tail's feathers and any plumes on the bones.
func (body *birdBody) hangFeathers() error {
	for _, side := range birdSides {
		if err := body.primaries(side.name, side.sign); err != nil {
			return err
		}
		if err := body.secondaries(side.name, side.sign); err != nil {
			return err
		}
		if err := body.alula(side.name, side.sign); err != nil {
			return err
		}
		if err := body.scapulars(side.name, side.sign); err != nil {
			return err
		}
	}
	if err := body.tailFeathers(); err != nil {
		return err
	}
	return body.plumes()
}

// longestPrimary is the length of the longest primary, worked so that
// its tip, the next to outermost, reaches the half span.
func (body *birdBody) longestPrimary() float64 {
	m := body.measures
	n := m.primaries
	share := float64(n-2) / float64(max(n-1, 1))
	root := birdShoulderOut*m.width/2 + m.humerus + m.forearm +
		m.hand*share
	angle := birdPrimaryFirst + (birdPrimaryLast-birdPrimaryFirst)*share
	return (m.wingspan/2 - root) / math.Sin(angle)
}

// primaryLength is the length of primary k of n, counted from one at
// the wrist, as a share of the longest.
func primaryLength(k, n int) float64 {
	if k == n {
		return birdPrimaryOuter
	}
	share := float64(k-1) / float64(max(n-2, 1))
	return birdPrimaryShortest + (1-birdPrimaryShortest)*share
}

// primaries are the primaries on one side, from the wrist out along the
// hand, each with its greater and median covert.
func (body *birdBody) primaries(side string, sign float64) error {
	m := body.measures
	longest := body.longestPrimary()
	for k := 1; k <= m.primaries; k++ {
		share := float64(k-1) / float64(max(m.primaries-1, 1))
		along := share * m.hand
		root := body.wrist(sign).Add(birdAcross.Scale(sign * along))
		angle := birdPrimaryFirst + (birdPrimaryLast-birdPrimaryFirst)*share
		way := maths.V(-math.Cos(angle), 0, sign*math.Sin(angle))
		name := fmt.Sprintf("p%d", k)
		parent := body.handBone(along)
		if err := body.remex(name, parent+side, birdPrimary, side, sign,
			share, root, way, longest*primaryLength(k, m.primaries)); err != nil {
			return err
		}
	}
	return nil
}

// handBone is the bone of the hand that carries a feather rooted along
// metres out from the wrist.
func (body *birdBody) handBone(along float64) string {
	m := body.measures
	switch {
	case along <= birdCarpus*m.hand:
		return "carpometacarpus-"
	case along <= (birdCarpus+birdMajorFirst)*m.hand:
		return "major-digit1-"
	}
	return "major-digit2-"
}

// secondaries are the secondaries on one side, from the wrist in along
// the ulna, each with its coverts.
func (body *birdBody) secondaries(side string, sign float64) error {
	m := body.measures
	length := m.secondary * body.longestPrimary()
	way := maths.V(-math.Cos(birdSecondaryLean), 0,
		sign*math.Sin(birdSecondaryLean))
	trailing := birdAlong.Scale(-birdForearmApart * m.length)
	for j := 1; j <= m.secondaries; j++ {
		share := float64(j-1) / float64(max(m.secondaries-1, 1))
		in := birdAcross.Scale(-sign * share * .95 * m.forearm)
		root := body.wrist(sign).Add(in).Add(trailing)
		name := fmt.Sprintf("s%d", j)
		long := length * (1 - birdSecondaryTaper*share)
		if err := body.remex(name, "ulna-"+side, birdSecondary, side, sign,
			share, root, way, long); err != nil {
			return err
		}
		if err := body.lesser(name, "ulna-"+side, side, sign, share, root,
			way, long); err != nil {
			return err
		}
	}
	return body.tertials(side, sign, way, length)
}

// The tertials, the inner flight feathers on the humerus that close the
// wing to the body: how many, and their length of a secondary's; picked.
const (
	birdTertials     = 4
	birdTertialShare = .8
)

// tertials are the inner flight feathers along the humerus, from the
// elbow in toward the shoulder, each with its coverts.
func (body *birdBody) tertials(side string, sign float64, way maths.Vec,
	length float64) error {
	m := body.measures
	elbow := body.shoulder(sign).Add(birdAcross.Scale(sign * m.humerus))
	for k := 1; k <= birdTertials; k++ {
		share := float64(k) / (birdTertials + 1)
		root := elbow.Add(birdAcross.Scale(-sign * share * m.humerus))
		name := fmt.Sprintf("t%d", k)
		long := length * birdTertialShare
		if err := body.remex(name, "humerus-"+side, birdSecondary, side,
			sign, 1, root, way, long); err != nil {
			return err
		}
		if err := body.lesser(name, "humerus-"+side, side, sign, 1, root,
			way, long); err != nil {
			return err
		}
	}
	return nil
}

// The scapulars, the feathers of the shoulder that lie back over the
// base of the folded wing: how many to a side, their length of a
// secondary's, and how far each leans in toward the middle. All picked:
// a row along the blade that covers it, which is how a bird's shoulder
// is seen; no count of scapulars was read.
const (
	birdScapulars     = 6
	birdScapularShare = .7
	birdScapularLean  = .10
)

// scapulars are the scapular feathers of one side, rooted evenly along
// the shoulder blade from the shoulder to the blade's far end and lying
// back, a little toward the middle, over the back.
func (body *birdBody) scapulars(side string, sign float64) error {
	m := body.measures
	at, _ := body.bones.Find("scapula-" + side)
	blade := body.bones.Bone(at)
	length := birdScapularShare * m.secondary * body.longestPrimary()
	way := maths.V(-1, 0, -sign*birdScapularLean)
	raise := birdUp.Scale(birdCovertLift * m.length)
	for i := 0; i < birdScapulars; i++ {
		share := float64(i) / float64(birdScapulars-1)
		root := blade.Head().Add(blade.Tail().Sub(blade.Head()).Scale(share))
		name := fmt.Sprintf("%s%d-%s", birdScapular, i+1, side)
		if err := body.feather(name, "scapula-"+side, birdScapular, sign,
			share, root.Add(raise), way, length, birdRemexWidth); err != nil {
			return err
		}
	}
	return nil
}

// The rows of coverts over a flight feather lie each a little above and
// forward of the one below: the rise of one row, of the bird's length, and
// how many rises forward it lies; picked, enough to read as tiles. The
// lesser coverts are the third row.
const (
	birdCovertLift    = .004
	birdCovertForward = 2
	birdLesserRow     = 3
)

// covert adds the covert of group in row of rows over the flight
// feather rooted at root, on the same bone and lying the same way.
func (body *birdBody) covert(name, parent, group, side string, row int,
	sign, share float64, root, way maths.Vec, length float64) error {
	lift := float64(row) * birdCovertLift * body.measures.length
	at := root.Add(birdAlong.Scale(birdCovertForward * lift)).
		Add(birdUp.Scale(lift))
	bone := fmt.Sprintf("%s-%s-%s", group, name, side)
	return body.feather(bone, parent, group, sign, share, at, way, length,
		birdCovertWidth)
}

// remex adds a flight feather and over it its greater and median
// coverts, each rooted a little forward of and above the one below.
func (body *birdBody) remex(name, parent, group, side string, sign,
	share float64, root, way maths.Vec, length float64) error {
	if err := body.feather(name+"-"+side, parent, group, sign, share, root,
		way, length, birdRemexWidth); err != nil {
		return err
	}
	rows := []struct {
		group string
		share float64
	}{{birdGreater, birdGreaterShare}, {birdMedian, birdMedianShare}}
	for i, row := range rows {
		if err := body.covert(name, parent, row.group, side, i+1, sign,
			share, root, way, length*row.share); err != nil {
			return err
		}
	}
	return nil
}

// lesser adds the lesser covert over the flight feather name: the third
// row of coverts, on the arm only, where the small coverts of the wing
// are. The hand has the primary coverts of its own.
func (body *birdBody) lesser(name, parent, side string, sign, share float64,
	root, way maths.Vec, length float64) error {
	return body.covert(name, parent, birdLesser, side, birdLesserRow, sign,
		share, root, way, length*birdLesserShare)
}

// alula adds the alula's few feathers along the leading edge.
func (body *birdBody) alula(side string, sign float64) error {
	at, _ := body.bones.Find("alula-" + side)
	bone := body.bones.Bone(at)
	length := .18 * body.longestPrimary()
	for i := 0; i < birdAlulaFeathers; i++ {
		share := float64(i) / float64(birdAlulaFeathers-1)
		root := bone.Head().Add(bone.Tail().Sub(bone.Head()).Scale(share))
		way := maths.V(-.2, 0, sign)
		name := fmt.Sprintf("alula%d-%s", i+1, side)
		if err := body.feather(name, "alula-"+side, birdAlula, sign, share,
			root, way, length*(1-.25*share), birdRemexWidth); err != nil {
			return err
		}
	}
	return nil
}

// tailFeathers are the rectrices fanned from the pygostyle, spread as
// in flight, and their coverts above and below.
func (body *birdBody) tailFeathers() error {
	m := body.measures
	at, _ := body.bones.Find("pygostyle")
	base := body.bones.Bone(at).Tail()
	length := m.tail + base.X
	middle := float64(m.rectrices-1) / 2
	for i := 0; i < m.rectrices; i++ {
		across := (float64(i) - middle) / math.Max(middle, 1)
		angle := across * birdTailFan
		way := maths.V(-math.Cos(angle), 0, math.Sin(angle))
		root := base.Add(birdAcross.Scale(across * birdTailRoot * m.width / 2))
		names := []struct {
			group string
			lift  float64
			share float64
		}{{birdRectrix, 0, 1}, {birdUpperCovert, .006, birdTailCovert},
			{birdUnderCovert, -.006, birdTailCovert}}
		pair := int(math.Abs(float64(i) - middle))
		for _, row := range names {
			name := fmt.Sprintf("%s%d", row.group, i+1)
			width, long := birdCovertWidth, length*row.share
			if row.group == birdRectrix {
				width = birdRectrixWidth
				long = m.tail*body.species.Tail.share(pair) + base.X
			}
			if err := body.feather(name, "pygostyle", row.group, 1, across,
				root.Add(birdUp.Scale(row.lift*m.length)), way,
				long, width); err != nil {
				return err
			}
		}
	}
	return nil
}

// plumes are the long plumes of the nape, hung from the back of the
// skull and lying down the neck, as many as the row gives.
func (body *birdBody) plumes() error {
	m := body.measures
	at, _ := body.bones.Find("cranium")
	back := body.bones.Bone(at).Head().Add(birdUp.Scale(birdSkullDeep *
		m.head))
	for i := 0; i < m.plumes; i++ {
		share := float64(i) / float64(max(m.plumes-1, 1))
		way := maths.V(-1, -.25, (share-.5)*.2)
		name := fmt.Sprintf("plume%d", i+1)
		if err := body.feather(name, "cranium", birdPlume, 1, share, back,
			way, m.plumeLength*(1-.2*share), birdPlumeWidth); err != nil {
			return err
		}
	}
	return nil
}
