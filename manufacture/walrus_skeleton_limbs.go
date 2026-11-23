package manufacture

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/maths"
)

// A flipper is a hand: a row of carpals or tarsals, five long bones of the
// palm or sole, and the bones of five digits, two in the first digit and
// three in each other, as a carnivoran has (chosen; no walrus source read
// for the count). Lengths in metres, chosen. In the fore flipper "the
// digits are shorter after the first"; in the hind flipper "the first and
// fifth digits are longer" (OBIS-SEAMAP species profile, as read).
//
// The order is sourced; the lengths are chosen to keep it, and short
// enough that the flipper stays inside the hide's mesh box.
var (
	walrusForePalm   = [walrusDigitCount]float64{.11, .115, .11, .105, .10}
	walrusForeDigits = [walrusDigitCount][]float64{
		{.12, .10}, {.09, .07, .05}, {.085, .065, .045},
		{.08, .06, .04}, {.075, .055, .035},
	}
	walrusHindSole   = [walrusDigitCount]float64{.14, .12, .11, .12, .14}
	walrusHindDigits = [walrusDigitCount][]float64{
		{.13, .10}, {.09, .07, .05}, {.085, .065, .045},
		{.09, .07, .05}, {.12, .09, .06},
	}
)

// How the rays of a flipper are spread. Chosen, so that a flipper comes
// out broad and nearly square, about as wide at the ends of its digits
// as it is long from the wrist or the ankle.
const (
	// walrusForeSpacing and walrusHindSpacing are the distance between
	// neighbouring rays where they leave the wrist or the ankle, in
	// metres.
	walrusForeSpacing = .050
	walrusHindSpacing = .052
	// walrusForeFan and walrusHindFan are the angle between neighbouring
	// rays, in radians.
	walrusForeFan = .09
	walrusHindFan = .12
)

// walrusHand is how a flipper's rays are laid: the joint at its root
// (wrist or ankle), along the flipper and across it toward the first
// digit; which flipper and side, and the name of its palm bones; the
// lengths of each ray's bones; the spacing and fan of the rays; and the
// carpal or tarsal each ray hangs from.
type walrusHand struct {
	root, along, across  maths.Vec
	kind, side, palmName string
	palm                 [walrusDigitCount]float64
	digits               [walrusDigitCount][]float64
	spacing, fan         float64
	carriers             [walrusDigitCount]string
}

// walrusFlipperFrame is along and across for a flipper pointing along
// direction, its sole down: across is level and square to along, toward
// the first digit.
func walrusFlipperFrame(direction maths.Vec) (along, across maths.Vec) {
	along = direction.Unit()
	across = maths.V(along.Z, 0, -along.X).Unit()
	return along, across
}

// walrusForeLimb lays one fore limb: shoulder blade on the chest wall,
// upper arm back and down inside the body, forearm forward, down and out
// to the wrist on the ground beside the chest, and the hand of bones
// turned forward and outward, flat on the ground, where it carries the
// chest. Places and thicknesses chosen: the wrist lies where the outline's
// underside comes down to the ground.
func walrusForeLimb(body *walrusBones, side float64) {
	const bladeThickness = .012
	name := walrusSideName(side)
	shoulder := maths.V(.85, -.04, side*.22)
	blade := shoulder.Add(maths.V(-.40, .88, -side*.20).Unit().
		Scale(walrusScapulaLength))
	body.add("scapula "+name, "thoracic 4", shoulder, blade, bladeThickness,
		skeleton.NewBlade(shoulder, blade, .220282*walrusBullScale/2,
			bladeThickness, .25, maths.V(0, 0, 1)))
	elbow := shoulder.Add(maths.V(-.70, -.55, side*.45).Unit().
		Scale(walrusHumerusLength))
	walrusRod(body, "humerus "+name, "scapula "+name, shoulder, shoulder,
		elbow, .052, .55)
	forearm := maths.V(.52, -.62, side*.50).Unit()
	along, across := walrusFlipperFrame(maths.V(.80, -.02, side*.60))
	wrist := elbow.Add(forearm.Scale(walrusRadiusLength)).Add(across.Scale(.02))
	behind := across.Scale(-.025)
	walrusRod(body, "radius "+name, "humerus "+name, elbow,
		elbow.Add(across.Scale(.02)), wrist, .026, .6)
	// The ulna hangs from the radius, so the forearm bends at the elbow
	// as one.
	walrusRod(body, "ulna "+name, "radius "+name, elbow,
		elbow.Sub(forearm.Scale(.06)).Add(behind),
		elbow.Add(forearm.Scale(walrusUlnaLength-.06)).Add(behind), .030, .6)
	walrusCarpus(body, wrist, along, across, name)
	walrusDigits(body, walrusHand{root: wrist, along: along, across: across,
		kind: "fore", side: name, palmName: "metacarpal", palm: walrusForePalm,
		digits: walrusForeDigits, spacing: walrusForeSpacing, fan: walrusForeFan,
		carriers: [walrusDigitCount]string{"trapezium", "trapezoid", "magnum",
			"unciform", "unciform"}})
}

// walrusCarpus lays the wrist bones, as a carnivoran has them: the
// scapholunar, cuneiform and pisiform, then the trapezium, trapezoid,
// magnum and unciform. The scapholunar hangs from the radius and turns at
// the wrist; the others hang from it. Places chosen.
func walrusCarpus(body *walrusBones, wrist, along, across maths.Vec, side string) {
	hub := "scapholunar " + side
	walrusSmallBone(body, hub, "radius "+side, wrist, wrist.Add(along.Scale(.03)),
		along, .022)
	rows := []struct {
		name          string
		ahead, toward float64
	}{
		{"cuneiform", .03, -.04}, {"pisiform", .02, -.075},
		{"trapezium", .075, .09}, {"trapezoid", .075, .03},
		{"magnum", .075, -.03}, {"unciform", .075, -.09},
	}
	for _, row := range rows {
		at := wrist.Add(along.Scale(row.ahead)).Add(across.Scale(row.toward))
		walrusSmallBone(body, row.name+" "+side, hub, at, at, along, .018)
	}
}

// walrusSmallBone hangs a short block of bone, a carpal or a tarsal,
// centred at centre and lying along along, from parent, turning at joint.
func walrusSmallBone(body *walrusBones, name, parent string, joint, centre,
	along maths.Vec, radius float64) {
	reach := along.Scale(radius)
	walrusRod(body, name, parent, joint, centre.Sub(reach), centre.Add(reach),
		radius, .85)
}

// walrusRod hangs a straight bone from parent, turning at joint, from from
// to to, of that radius and shaft. Its far end in the tree is to, or from
// when to is the joint itself.
func walrusRod(body *walrusBones, name, parent string, joint, from,
	to maths.Vec, radius, shaft float64) {
	tail := to
	if tail == joint {
		tail = from
	}
	body.add(name, parent, joint, tail, radius,
		skeleton.NewRod(from, to, radius, shaft))
}

// walrusDigits lays the five rays of a flipper: each a long bone of the
// palm or sole hung from its carpal or tarsal, then its digit's bones, the
// rays fanned out from the middle one.
func walrusDigits(body *walrusBones, hand walrusHand) {
	const (
		start  = .10
		radius = .018
		thin   = .82
	)
	normal := hand.along.Cross(hand.across)
	for ray := 0; ray < walrusDigitCount; ray++ {
		offset := float64(2 - ray)
		heading := maths.Rotation(normal, offset*hand.fan).Out(hand.along)
		at := hand.root.Add(hand.along.Scale(start)).
			Add(hand.across.Scale(offset * hand.spacing))
		parent := hand.carriers[ray] + " " + hand.side
		lengths := append([]float64{hand.palm[ray]}, hand.digits[ray]...)
		size := radius
		for bone, length := range lengths {
			name := fmt.Sprintf("%s phalanx %d.%d %s", hand.kind, ray+1, bone, hand.side)
			if bone == 0 {
				name = fmt.Sprintf("%s %d %s", hand.palmName, ray+1, hand.side)
			}
			tip := at.Add(heading.Scale(length))
			walrusRod(body, name, parent, at, at, tip, size, .55)
			parent, at, size = name, tip, size*thin
		}
	}
}

// walrusHindLimb lays one hind limb from the sacrum: the hip bone, a
// short thigh bone and knee inside the body, the long shank back to the
// ankle on the ground beside the tail's root, and the hand of bones
// turned forward along the flank, as the walrus holds it on land. Places
// and thicknesses chosen: hip bone, thigh and knee lie inside the outline
// of the rump.
func walrusHindLimb(body *walrusBones, side float64, sacrum string) {
	const bladeThickness = .015
	name := walrusSideName(side)
	joint := maths.V(-.86, .03, side*.075)
	hip := maths.V(-.90, -.06, side*.11)
	seat := maths.V(-1.12, -.14, side*.08)
	body.add("innominate "+name, sacrum, joint, seat, bladeThickness,
		skeleton.NewBlade(joint, seat, .045, bladeThickness, .8,
			maths.V(0, 0, 1)))
	knee := hip.Add(maths.V(.25, -.55, side*.78).Unit().Scale(walrusFemurLength))
	walrusRod(body, "femur "+name, "innominate "+name, hip, hip, knee, .040, .65)
	kneecap := knee.Add(maths.V(1, .2, 0).Unit().Scale(.035))
	walrusSmallBone(body, "patella "+name, "femur "+name, knee, kneecap,
		maths.V(0, 1, 0), .02)
	ankle := knee.Add(maths.V(-.86, -.48, side*.05).Unit().
		Scale(walrusTibiaLength))
	walrusRod(body, "tibia "+name, "femur "+name, knee, knee, ankle, .030, .6)
	outside := maths.V(0, 0, side*.035)
	walrusRod(body, "fibula "+name, "tibia "+name, knee, knee.Add(outside),
		ankle.Add(outside), .017, .6)
	along, across := walrusFlipperFrame(maths.V(.80, -.05, side*.60))
	walrusTarsus(body, ankle, along, across, name)
	walrusDigits(body, walrusHand{root: ankle, along: along, across: across,
		kind: "hind", side: name, palmName: "metatarsal", palm: walrusHindSole,
		digits: walrusHindDigits, spacing: walrusHindSpacing, fan: walrusHindFan,
		carriers: [walrusDigitCount]string{"medial cuneiform",
			"intermediate cuneiform", "lateral cuneiform", "cuboid", "cuboid"}})
}

// walrusTarsus lays the ankle bones: the astragalus, which hangs from the
// shank and turns at the ankle, and from it the calcaneus behind and below,
// the navicular, the cuboid and the three cuneiforms. Places chosen.
func walrusTarsus(body *walrusBones, ankle, along, across maths.Vec, side string) {
	hub := "astragalus " + side
	walrusSmallBone(body, hub, "tibia "+side, ankle,
		ankle.Add(along.Scale(.02)), along, .026)
	heel := ankle.Sub(along.Scale(.03)).Sub(maths.V(0, .02, 0)).
		Sub(across.Scale(.02))
	reach := along.Scale(.045)
	walrusRod(body, "calcaneus "+side, hub, heel, heel.Sub(reach),
		heel.Add(reach), .022, .7)
	rows := []struct {
		name          string
		ahead, toward float64
	}{
		{"navicular", .06, .015}, {"cuboid", .06, -.03},
		{"medial cuneiform", .095, .045}, {"intermediate cuneiform", .095, .015},
		{"lateral cuneiform", .095, -.005},
	}
	for _, row := range rows {
		at := ankle.Add(along.Scale(row.ahead)).Add(across.Scale(row.toward))
		walrusSmallBone(body, row.name+" "+side, hub, at, at, along, .016)
	}
}

// walrusAnkleTurn is the angle at one ankle between the shank and the sole
// in the pose given by frames: nought when the flipper lies straight on
// from the shank, as it does trailing behind in the water.
func walrusAnkleTurn(body *walrusBones, frames []skeleton.Frame,
	side string) float64 {
	_, shin := body.bone("tibia " + side)
	_, foot := body.bone("metatarsal 3 " + side)
	knee, ankle := body.rod("tibia " + side)
	base, tip := body.rod("metatarsal 3 " + side)
	shank := frames[shin].Apply(ankle).Sub(frames[shin].Apply(knee)).Unit()
	ray := frames[foot].Apply(tip).Sub(frames[foot].Apply(base)).Unit()
	return math.Acos(math.Max(-1, math.Min(1, shank.Dot(ray))))
}
