package birds

import (
	"strings"

	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/maths"
)

// The parts round the outline, each a closed form carried by one bone:
// the bare face, the eyes with their rings and the bill's sheaths by the
// skull's bones, the thighs, the bare legs, their scutes and the claws by
// the leg's, the small coverts of the wing by its bones.

// Stations of the braincase, back to front, as shares of its half
// sizes; picked.
var (
	birdBraincaseWide = []float64{.55, .95, 1, .8, .5}
	birdBraincaseDeep = []float64{.6, 1, .95, .7, .45}
	birdSleeve        = []float64{.85, 1, 1, .85}
	birdBall          = []float64{.35, .85, 1, .85, .35}
	birdThigh         = []float64{.7, 1, .8, .4}
)

// Stations of the three bills, base to tip, as shares of the bill's
// half width and half depth at its base; picked. The spatula's last
// four are its spoon, widened to the spoon's width read for the row.
var (
	birdSpatulaWide = []float64{1, .65, .45, .42, .6, .95, 1, .65}
	birdSpatulaDeep = []float64{1, .55, .35, .25, .18, .14, .12, .08}
	birdDaggerWide  = []float64{1, .75, .55, .4, .28, .16, .05}
	birdDaggerDeep  = []float64{1, .8, .62, .46, .32, .18, .05}
	birdHookedWide  = []float64{1, .8, .62, .5, .4, .32, .2}
	birdHookedDeep  = []float64{1, .8, .66, .55, .5, .6, .3}
)

// Shares that size the masses; picked.
const (
	birdBoneShare  = .8   // a bill's bone, of its sheath
	birdJawShare   = .6   // the lower bill, of the upper's depth
	birdNeckBase   = .28  // the neck's half thickness at the chest, of W/2
	birdNeckTop    = .75  // and at the head, of the braincase's half width
	birdHeadShare  = 1.25 // the head's feathers, of the braincase
	birdEyeShare   = .09  // the eye's radius, of the head's length
	birdLegSkin    = 1.5  // the bare leg, of the bone's thickness
	birdThighShare = .30  // the thigh's half thickness, of W/2
)

// The eye ring is the bare skin round the eye, a disc lying over the
// head's side with the eye standing out of it: its radius of the eye's,
// and its thickness outward of the eye's radius. Picked, as a ring is
// seen round a bird's eye; no width of a ring was read.
const (
	birdEyeRing      = 1.8
	birdEyeRingThick = .5
)

// birdPiece is one closed form of a bird, the bone that carries it,
// and the region of plumage or bare part it is coloured as.
type birdPiece struct {
	bone, region string
	form         skeleton.BoneForm
}

// billStations are the bill's half widths and half depths, base to
// tip, its sheath's times share.
func (body *birdBody) billStations(share float64) ([]float64, []float64) {
	m := body.measures
	wide, deep := birdDaggerWide, birdDaggerDeep
	switch body.species.Bill {
	case birdSpatula:
		wide, deep = birdSpatulaWide, birdSpatulaDeep
	case birdHooked:
		wide, deep = birdHookedWide, birdHookedDeep
	}
	wide = birdScaled(wide, share*m.billWidth/2)
	if body.species.Bill == birdSpatula && m.billWidth > 0 {
		spoon := m.spatula / m.billWidth
		for i := len(wide) - 4; i < len(wide); i++ {
			wide[i] *= spoon
		}
	}
	return wide, birdScaled(deep, share*m.billDepth/2)
}

// bonePieces are the skeleton's bones alone, each in its form, the
// feathers left off.
func (body *birdBody) bonePieces() []birdPiece {
	feathers := map[string]bool{}
	for _, feather := range body.feathers {
		feathers[feather.name] = true
	}
	pieces := []birdPiece{}
	for i := 0; i < body.bones.Len(); i++ {
		name := body.bones.Bone(i).Name()
		if form, ok := body.forms[name]; ok && !feathers[name] {
			pieces = append(pieces, birdPiece{bone: name, region: "bone",
				form: form})
		}
	}
	return pieces
}

// barePieces are the parts round the outline that are not feathers: the
// bare face, the eyes and their rings, the bill, the thighs, the legs
// with their scutes, and the toes with their claws and webs.
func (body *birdBody) barePieces() []birdPiece {
	pieces := append(body.headPieces(), body.legPieces()...)
	pieces = append(pieces, body.scutePieces()...)
	pieces = append(pieces, body.clawPieces()...)
	return append(pieces, body.webPieces()...)
}

// skinPieces are the bird as it is seen round its outline: its bare
// parts, the small coverts of the wings, and every feather.
func (body *birdBody) skinPieces() []birdPiece {
	pieces := append(body.barePieces(), body.wingPieces()...)
	for _, feather := range body.feathers {
		pieces = append(pieces, birdPiece{bone: feather.name,
			region: birdRegionOf(feather.group),
			form:   body.forms[feather.name]})
	}
	return pieces
}

// birdRegionOf is the region a feather of group is coloured as.
func birdRegionOf(group string) string {
	switch group {
	case birdPrimary:
		return "primaries"
	case birdSecondary:
		return "secondaries"
	case birdRectrix:
		return "tail"
	case birdUpperCovert, birdScapular:
		return "back"
	case birdUnderCovert:
		return "belly"
	case birdPlume:
		return "plumes"
	}
	return "wing"
}

// headPieces are the bare face before the eyes, the eyes and the ring
// of bare skin round each, and the bill's upper and lower sheaths; the
// feathered head is the outline's.
func (body *birdBody) headPieces() []birdPiece {
	m := body.measures
	at, _ := body.bones.Find("cranium")
	cranium := body.bones.Bone(at)
	condyle, front := cranium.Head(), cranium.Tail()
	wide, deep := birdSkullWide*m.head, birdSkullDeep*m.head
	face := skeleton.NewProfile(front.Add(birdAlong.Scale(-.3*m.head)),
		front.Add(birdAlong.Scale(.08*m.bill)), birdUp,
		birdScaled(birdSleeve, .8*wide), birdScaled(birdSleeve, .7*deep))
	pieces := []birdPiece{{bone: "cranium", region: "face", form: face}}
	radius := birdEyeShare * m.head
	for _, side := range birdSides {
		eye := condyle.Add(birdAlong.Scale(.62 * m.head)).
			Add(birdUp.Scale(.25 * deep)).
			Add(birdAcross.Scale(side.sign * .9 * birdHeadShare * wide))
		ball := birdScaled(birdBall, radius)
		pieces = append(pieces, birdPiece{bone: "cranium", region: "eye",
			form: skeleton.NewProfile(eye.Sub(birdAlong.Scale(radius)),
				eye.Add(birdAlong.Scale(radius)), birdUp, ball, ball)})
		pieces = append(pieces, body.eyeRing(eye, side.sign, radius))
	}
	return append(pieces, body.billPieces(front)...)
}

// eyeRing is the ring of bare skin round the eye at eye, on the side of
// sign: a disc as wide across as the ring, from the eye's middle out
// along the side by its thickness, so that the eye stands out of it.
func (body *birdBody) eyeRing(eye maths.Vec, sign, radius float64) birdPiece {
	across := birdEyeRing * radius
	outer := eye.Add(birdAcross.Scale(sign * birdEyeRingThick * radius))
	disc := []float64{across, across}
	return birdPiece{bone: "cranium", region: "face",
		form: skeleton.NewProfile(eye, outer, birdUp, disc, disc)}
}

// billPieces are the sheaths of the upper bill and the lower.
func (body *birdBody) billPieces(front maths.Vec) []birdPiece {
	m := body.measures
	wide, deep := body.billStations(1)
	tip := front.Add(birdAlong.Scale(m.bill))
	at, _ := body.bones.Find("mandible")
	under := body.bones.Bone(at).Tail()
	lower := front.Add(birdUp.Scale(-m.billDepth / 4))
	return []birdPiece{{bone: "premaxilla", region: "bill",
		form: skeleton.NewProfile(front, tip, birdUp, wide, deep)},
		{bone: "mandible", region: "bill",
			form: skeleton.NewProfile(lower, under, birdUp, wide,
				birdScaled(deep, birdJawShare))}}
}

// legPieces are the feathered thighs, the bare shanks and tarsi, and
// the toes.
func (body *birdBody) legPieces() []birdPiece {
	m := body.measures
	skin := birdLegSkin * birdLimbBone * m.length
	pieces := []birdPiece{}
	for i := 0; i < body.bones.Len(); i++ {
		bone := body.bones.Bone(i)
		name := bone.Name()
		head, tail := bone.Head(), bone.Tail()
		switch {
		case strings.HasPrefix(name, "femur-"):
			thigh := birdScaled(birdThigh, birdThighShare*m.width/2)
			pieces = append(pieces, birdPiece{bone: name, region: "belly",
				form: skeleton.NewProfile(head, tail.Add(birdTibiaWay.
					Unit().Scale(.4*m.tibia)), birdUp, thigh, thigh)})
		case strings.HasPrefix(name, "tibiotarsus-"):
			middle := head.Add(tail.Sub(head).Scale(.45))
			pieces = append(pieces, birdPiece{bone: name, region: "legs",
				form: birdRod(middle, tail, skin)})
		case strings.HasPrefix(name, "tarsometatarsus-"):
			pieces = append(pieces, birdPiece{bone: name, region: "legs",
				form: birdRod(head, tail, skin)})
		case strings.HasPrefix(name, "toe"):
			thick, _ := bone.Thickness()
			pieces = append(pieces, birdPiece{bone: name, region: "feet",
				form: birdRod(head, tail, birdLegSkin*thick)})
		}
	}
	return pieces
}

// wingPieces are the small coverts over the arm and hand, as a blade
// on each of the humerus, the ulna and the carpometacarpus.
func (body *birdBody) wingPieces() []birdPiece {
	m := body.measures
	thin := birdVaneThick * m.length * 4
	longest := body.longestPrimary()
	pieces := []birdPiece{}
	for _, side := range birdSides {
		for _, bone := range []struct {
			name  string
			width float64
		}{{"humerus-", .15 * m.humerus}, {"ulna-", .22 * m.secondary *
			longest}, {"carpometacarpus-", .12 * longest}} {
			at, _ := body.bones.Find(bone.name + side.name)
			b := body.bones.Bone(at)
			pieces = append(pieces, birdPiece{bone: bone.name + side.name,
				region: "wing", form: skeleton.NewBlade(b.Head(), b.Tail(),
					bone.width, thin, .7, birdUp)})
		}
	}
	return pieces
}
