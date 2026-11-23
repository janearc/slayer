package manufacture

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/maths"
)

// The walrus skeleton is laid out at rest lying on land, in the hide's own
// space: x forward, y up, z to the right, the ground at y of minus .41.
// Every count and length says where it is from; "chosen" means no source
// was read for it and the number was picked to fit the others.
const (
	// walrusCervicalCount is the neck's vertebrae. Chosen: the count
	// nearly every mammal has; no walrus source read.
	walrusCervicalCount = 7
	// walrusThoracicCount and walrusLumbarCount are from Pierce, Clack and
	// Hutchinson 2011, J. Anat. 219(4), 502-514: "except in Odobenus
	// rosmarus and Hydrurga leptonyx which have 14 thoracic and six lumbar
	// vertebrae". Thoracic there means rib-bearing: one rib pair to each.
	walrusThoracicCount = 14
	walrusLumbarCount   = 6
	// walrusSacralCount and walrusCaudalCount are chosen.
	walrusSacralCount = 3
	walrusCaudalCount = 9
	// walrusSternebraCount is the bones of the breastbone. Chosen.
	walrusSternebraCount = 8
	// walrusSternalRibs is how many rib pairs reach the breastbone; the
	// rest float. Chosen.
	walrusSternalRibs = 9
)

// Lengths along the backbone, in metres. Chosen, so that the hide over
// skull and column comes to about the 315.2 cm asymptotic standard length
// of bulls (Garlich-Miller and Stewart 1998, as quoted in the COSEWIC
// Atlantic walrus report), inside the 2.2 to 3.5 m of Fay 1985.
const (
	walrusCervicalLength = .075
	walrusThoracicLength = .080
	walrusLumbarLength   = .090
	walrusSacralLength   = .065
	// walrusCaudalLength is the first tail vertebra; the rest shrink.
	walrusCaudalLength = .040
	// walrusSkullLength is condyle to the front of the premaxilla.
	walrusSkullLength = .40
	// walrusSkullTilt is how far the skull's axis drops toward the snout,
	// in radians, in the rest pose.
	walrusSkullTilt = .25
)

// Fore limb lengths, in metres. The four are a measured Pacific female,
// museum specimen UWBM 35479, on a virtual zooarchaeology page, given
// there as scapula 255.882, humerus 253.959, ulna 265.384 and radius
// 210.408, read as millimetres; walrusBullScale makes them a bull's.
const (
	walrusScapulaLength = .255882 * walrusBullScale
	walrusHumerusLength = .253959 * walrusBullScale
	walrusUlnaLength    = .265384 * walrusBullScale
	walrusRadiusLength  = .210408 * walrusBullScale
	// walrusBullScale is a bull's limb bone over that female's. Chosen.
	walrusBullScale = 1.15
	// walrusFemurLength and walrusTibiaLength are chosen: a short, broad
	// thigh bone and a long shank, as a pinniped has.
	walrusFemurLength = .20
	walrusTibiaLength = .38
)

// The ankle. Otariids and the walrus turn the hind flipper forward under
// the body and walk on it; phocids cannot "because of their massive ankle
// bones and flatter heels" (Riedman 1990, p. 64, as quoted by Wikipedia's
// Pinniped page). In this skeleton the turn is at the joint of the shank
// and the tarsus, and walrusAnkleForward is how far it may go.
const (
	// walrusAnkleForward is the most the hind flipper turns forward from
	// lying in line with the shank, in radians. Chosen.
	walrusAnkleForward = 2.6
)

// walrusRegion is a run of vertebrae: its name, how many, the length of
// each, the centrum's radius, the neural spine's height, the transverse
// process's reach, and how much the run curls down at each joint.
type walrusRegion struct {
	name                 string
	count                int
	length               float64
	radius, spine, reach float64
	curl                 float64
}

// walrusThoracolumbar is the joint between the last thoracic vertebra and
// the first lumbar: the root of the skeleton.
var walrusThoracolumbar = maths.V(-.29, .14, 0)

// walrusBones is the walrus's skeleton: the shared tree of bones, the form
// of each bone, and the first fault met in laying them out.
type walrusBones struct {
	tree  *skeleton.Skeleton
	forms map[string]skeleton.BoneForm
	fault error
}

// add hangs a bone from the bone named parent, or makes it the root when
// parent is empty: its joint, its far end, its half thickness and its
// form. It gives back the bone's name; a fault is kept, not lost.
func (body *walrusBones) add(name, parent string, joint, tail maths.Vec,
	thick float64, form skeleton.BoneForm) string {
	if err := body.tree.Add(name, parent, joint, tail, thick, thick); err != nil {
		if body.fault == nil {
			body.fault = err
		}
		return name
	}
	body.forms[name] = form
	return name
}

// bone is the bone of that name, and its place in the tree.
func (body *walrusBones) bone(name string) (skeleton.Bone, int) {
	at, _ := body.tree.Find(name)
	return body.tree.Bone(at), at
}

// rod is the two ends of the straight bone of that name.
func (body *walrusBones) rod(name string) (maths.Vec, maths.Vec) {
	piece, _ := body.forms[name].(skeleton.Rod)
	return piece.Ends()
}

// walrusSkeleton lays out the whole walrus at rest.
func walrusSkeleton() *walrusBones {
	body := &walrusBones{tree: skeleton.New(),
		forms: map[string]skeleton.BoneForm{}}
	neck := walrusForeColumn(body)
	walrusSkull(body, neck)
	sacrum := walrusHindColumn(body)
	walrusRibcage(body)
	for _, side := range []float64{-1, 1} {
		walrusForeLimb(body, side)
		walrusHindLimb(body, side, sacrum)
	}
	return body
}

// walrusForeColumn lays the thoracic vertebrae from the root forward, then
// the neck, rising to the skull, and gives back the name of the atlas.
func walrusForeColumn(body *walrusBones) string {
	thorax := walrusRegion{name: "thoracic", count: walrusThoracicCount,
		length: walrusThoracicLength, radius: .033, spine: .07, reach: .03,
		curl: .0015}
	neck := walrusRegion{name: "cervical", count: walrusCervicalCount,
		length: walrusCervicalLength, radius: .030, spine: .03, reach: .05,
		curl: -.012}
	heading := maths.V(1, .02, 0).Unit()
	last, at := walrusRun(body, "", walrusThoracolumbar, heading, thorax, true)
	atlas, _ := walrusRun(body, last, at, maths.V(.9255, .3786, 0).Unit(),
		neck, true)
	return atlas
}

// walrusHindColumn lays the lumbar, sacral and tail vertebrae from the
// root back, and gives back the name of the first sacral vertebra. The
// headings are chosen so the column falls with the back of the outline
// over it, from the loins down over the rump to the tail.
func walrusHindColumn(body *walrusBones) string {
	loins := walrusRegion{name: "lumbar", count: walrusLumbarCount,
		length: walrusLumbarLength, radius: .038, spine: .045, reach: .07,
		curl: .006}
	sacrum := walrusRegion{name: "sacral", count: walrusSacralCount,
		length: walrusSacralLength, radius: .035, spine: .03, reach: .06,
		curl: .01}
	tail := walrusRegion{name: "caudal", count: walrusCaudalCount,
		length: walrusCaudalLength, radius: .024, spine: .012, reach: .025,
		curl: .06}
	last, at := walrusRun(body, "thoracic 14", walrusThoracolumbar,
		maths.V(-1, -.14, 0).Unit(), loins, false)
	sacral, end := walrusRun(body, last, at, maths.V(-1, -.45, 0).Unit(),
		sacrum, false)
	walrusRun(body, sacral, end, maths.V(-1, -.7, 0).Unit(), tail, false)
	return "sacral 1"
}

// walrusRun lays one run of vertebrae from at along heading, each hung
// from the one before, curling heading down by the run's curl at each
// joint and tapering toward the run's end when the run is the tail.
// Numbering runs away from the root, except that a run counted toward the
// root (reversed) numbers its last bone first. It gives back the name of
// the run's last bone and where the run ends.
func walrusRun(body *walrusBones, after string, at, heading maths.Vec,
	run walrusRegion, reversed bool) (string, maths.Vec) {
	parent := after
	down := -run.curl
	if heading.X < 0 {
		down = run.curl
	}
	for i := 0; i < run.count; i++ {
		number := i + 1
		if reversed {
			number = run.count - i
		}
		taper := walrusTailTaper(run, i)
		tail := at.Add(heading.Scale(run.length * taper))
		name := fmt.Sprintf("%s %d", run.name, number)
		radius := run.radius * taper
		parent = body.add(name, parent, at, tail, radius,
			skeleton.NewCentrum(at, tail, radius, run.spine*taper,
				run.reach*taper))
		at = tail
		heading = maths.Rotation(maths.V(0, 0, 1), down).Out(heading)
	}
	return parent, at
}

// walrusTailTaper is how much the i-th vertebra of the run is shrunk: not
// at all, except down the tail, which ends at a third of its first size.
func walrusTailTaper(run walrusRegion, i int) float64 {
	const end = .35
	if run.name != "caudal" {
		return 1
	}
	return 1 - (1-end)*float64(i)/float64(run.count-1)
}

// walrusRibcage hangs a rib pair from every thoracic vertebra and the
// breastbone under the first ribs.
func walrusRibcage(body *walrusBones) {
	const ribThickness = .015
	for number := 1; number <= walrusThoracicCount; number++ {
		owner := fmt.Sprintf("thoracic %d", number)
		vertebra, _ := body.bone(owner)
		at := vertebra.Head().Add(vertebra.Tail()).Scale(.5)
		for _, side := range []float64{-1, 1} {
			name := fmt.Sprintf("rib %d %s", number, walrusSideName(side))
			rib := walrusRib(number, at, side)
			body.add(name, owner, at, rib.At(1), ribThickness, rib)
		}
	}
	walrusSternum(body)
}

// walrusRib is the rib of the given number on one side, from its head at
// the vertebra round the barrel of the chest: the sternal ribs to the
// breastbone, the floating ribs short of it. The chest is narrow at the
// first rib and widest from the sixth. Sizes chosen.
func walrusRib(number int, at maths.Vec, side float64) skeleton.Arc {
	const (
		centre = -.05
		breast = -.30
		sweep  = .12
	)
	half := walrusRibHalf(number)
	head := maths.V(at.X, at.Y, side*.045)
	end := maths.V(at.X-sweep, breast, side*.03)
	if number > walrusSternalRibs {
		beyond := float64(number - walrusSternalRibs)
		end = maths.V(at.X-sweep-.02, -.20+.025*beyond, side*half*.85)
	}
	out := side * (half - .125*(.045+math.Abs(end.Z))) / .75
	return skeleton.NewArc([4]maths.Vec{head,
		maths.V(at.X-.02, at.Y, out), maths.V(at.X-sweep*.7, centre-.15, out),
		end}, .010, .021, maths.V(1, 0, 0))
}

// walrusRibHalf is the half width of the chest at the rib of the given
// number: narrow at the first rib, widest from the sixth. Chosen.
func walrusRibHalf(number int) float64 {
	const halfWidth, firstRibShare = .30, .55
	grow := firstRibShare +
		(1-firstRibShare)*maths.Smooth(0, 5, float64(number-1))
	return halfWidth * grow
}

// walrusSternum is the breastbone: a row of sternebrae along the floor of
// the chest, from under the first rib to under the last sternal rib, hung
// from the fifth thoracic vertebra.
func walrusSternum(body *walrusBones) {
	const breast, radius = -.31, .020
	first, _ := body.bone("thoracic 1")
	last, _ := body.bone(fmt.Sprintf("thoracic %d", walrusSternalRibs))
	front, back := first.Head().X-.02, last.Head().X-.14
	step := (front - back) / walrusSternebraCount
	for i := 0; i < walrusSternebraCount; i++ {
		from := maths.V(front-float64(i)*step, breast, 0)
		to := from.Sub(maths.V(step*.85, 0, 0))
		body.add(fmt.Sprintf("sternebra %d", i+1), "thoracic 5", from, to,
			radius, skeleton.NewRod(from, to, radius, .7))
	}
}

// walrusSideName is the word for one side.
func walrusSideName(side float64) string {
	if side < 0 {
		return "left"
	}
	return "right"
}
