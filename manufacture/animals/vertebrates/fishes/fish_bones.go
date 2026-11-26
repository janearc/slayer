package fishes

import (
	"fmt"
	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/maths"
	"math"
)

// A bony fish's skeleton, laid out from its row: the skull, its jaws, gill
// cover and the girdle the pectoral fins hang from; the column, its
// precaudal and caudal vertebrae by the species' counts, each a centrum
// with its neural spine, the caudal ones with haemal spines below and the
// precaudal ones with ribs.
//
// It has also the tail's own skeleton, a fan of hypurals carrying the
// caudal rays; and for each fin its rays on their supports, finlets where
// the species has them. Each bone has its form from the skeleton package,
// and the fins' rays carry their webs.

// Where the skull's bones lie, as shares of the head from the snout, and
// how big they are, as shares of the half depth or width there. Picked,
// to the plan of a perch-like skull; no skull measured was reached.
const (
	fishCondyle     = .8   // picked: the joint of skull and column
	fishSkullFront  = .12  // picked: the front of the braincase
	fishSkullHigh   = .35  // picked: its height there
	fishSkullWide   = .4   // picked: the braincase's depth
	fishSkullThick  = .3   // picked: its thickness, of the half width
	fishSkullStart  = .35  // picked: its depth at the front, of the back
	fishJawSide     = .7   // picked: how far out the jaws' hinges lie
	fishJawLow      = .3   // picked: the lower jaw's hinge below the axis
	fishJawChin     = .18  // picked: the chin below the axis
	fishJawThick    = .05  // picked: a jaw bone's half thickness
	fishCoverFront  = .78  // picked: the gill cover's front
	fishCoverBack   = .97  // picked: its back
	fishCoverHigh   = .35  // picked: its top
	fishCoverLow    = -.25 // picked: its bottom
	fishCoverWide   = .3   // picked: its width
	fishCoverThin   = .02  // picked: its thickness
	fishGirdleTop   = .45  // picked: the cleithrum's top
	fishGirdleLow   = -.65 // picked: its foot
	fishGirdleSlant = .92  // picked: its foot, of the head
	fishThin        = .015 // picked: a slender bone's half thickness
)

// The column's vertebrae: their lengths relative to one another, and
// their sizes. Picked: the caudal ones shorten toward the tail.
const (
	fishPrecaudalFirst = .85  // picked
	fishCaudalLast     = .55  // picked
	fishCentrum        = .14  // picked: a centrum's radius, of a half depth
	fishCentrumLeast   = .004 // picked: its least, of the length
	fishNeural         = .7   // picked: a neural spine's reach to the back
	fishHaemal         = .7   // picked: a haemal spine's reach to the belly
	fishSpineRake      = .25  // picked: a spine's lean back, of its length
	fishRibInset       = .85  // picked: a rib inside the body wall
	fishRibDown        = .85  // picked: how far down a rib reaches
	fishShaft          = .6   // picked: a slender rod's waist
)

// The fins' bones. Picked, to read as a fish's.
const (
	fishFinRoot      = .9    // picked: a ray's base, of the half depth
	fishSupportDeep  = .35   // picked: a support's inner end, likewise
	fishRake         = .7    // picked: a ray's lean back, in radians
	fishFinCurve     = .6    // picked: how a fin's height falls along it
	fishRayRadius    = .0012 // picked: a ray's half thickness, of length
	fishSpineThicker = 1.6   // picked: a spine to a soft ray
	fishSupportWide  = 1.5   // picked: a support to a soft ray
	fishFinletHigh   = .018  // picked: a finlet's height, of the length
	fishFinletGap    = .015  // picked: the gap before the first finlet
	fishFinletRake   = .6    // picked: a finlet's lean back
	fishHypurals     = 5     // picked: the plates of a perch-like tail
	fishHypuralRun   = .012  // picked: their length, of the fish's
	fishHypuralFan   = .45   // picked: their fan, of the stalk's depth
	fishTailRoot     = .55   // picked: the caudal rays' fan at the base
	fishLunate       = 1.4   // picked: how the tail's edge curves
	fishPectoralBase = .22   // picked: the pectoral's base, of a depth
	fishFalcate      = .7    // picked: how much shorter its lower rays
	fishFalcateBend  = .8    // picked: how its rays shorten
	fishPectoralDrop = .1    // picked: its rays' fall
	fishPectoralFan  = .4    // picked: its fan downward
	fishPectoralOut  = .35   // picked: its rays' spread from the body
	fishFinInset     = .95   // picked: a paired fin's base in the skin
	fishThoracic     = 1.3   // picked: pelvic fins this near the head hang
	fishPelvisRun    = .02   // picked: the pelvic bone's length
	fishPelvicLow    = .88   // picked: the pelvic fin's base
	fishPelvicDrop   = .35   // picked: its rays' fall
	fishPelvicOut    = .25   // picked: their spread
	fishPelvicShort  = .5    // picked: its last ray, of the first
	fishPelvicSpread = .25   // picked: its base across the belly
)

// The holders of the skin: how far from its middle line each bone holds.
const (
	fishSkullReach    = .5  // picked: of the skull's length
	fishVertebraReach = 1.5 // picked: of a vertebra's length
)

// fishRay is one ray of a fin: the bone it is, and its base and tip.
type fishRay struct {
	bone      string
	base, tip maths.Vec
}

// fishFin is a fin: its rays in order, the look its web wears, and the
// way its faces look at rest.
type fishFin struct {
	look   string
	rays   []fishRay
	facing maths.Vec
}

// fishBody is a fish's bones and what is hung on them: its plan, its
// proportions, its skeleton and each bone's form, its column's vertebrae
// in order, and its fins.
type fishBody struct {
	plan    fishPlan
	outline fishOutline
	bones   *skeleton.Skeleton
	forms   map[string]skeleton.BoneForm
	column  []string
	fins    []fishFin
}

// fishSides are the fish's left and right, as the sign of z.
var fishSides = [...]float64{1, -1}

// fishSideNames are their names, in the same order.
var fishSideNames = [...]string{"left", "right"}

// buildFish is the skeleton of a planned fish, and its fins.
func buildFish(plan fishPlan) (*fishBody, error) {
	body := &fishBody{plan: plan, outline: plan.outline(),
		bones: skeleton.New(), forms: map[string]skeleton.BoneForm{}}
	steps := []func() error{body.skull, body.spine, body.tail,
		body.medianFins, body.finlets, body.pectorals, body.pelvics}
	for _, step := range steps {
		if err := step(); err != nil {
			return nil, err
		}
	}
	return body, nil
}

// at is the point at share s along the fish, at height y and out z.
func (body *fishBody) at(s, y, z float64) maths.Vec {
	return maths.V(body.outline.x(s), y, z)
}

// head is the share of the length the head is.
func (body *fishBody) head() float64 { return body.plan.figure("head") }

// add puts a bone on the skeleton, from its head to its tail, thick at
// both ends, in its form.
func (body *fishBody) add(name, parent string, head, tail maths.Vec,
	thick float64, form skeleton.BoneForm) error {
	if err := body.bones.Add(name, parent, head, tail, thick,
		thick); err != nil {
		return err
	}
	body.forms[name] = form
	return nil
}

// rod adds a straight bone, round, its form a rod of radius radius.
func (body *fishBody) rod(name, parent string, head, tail maths.Vec,
	radius float64) error {
	return body.add(name, parent, head, tail, radius,
		skeleton.NewRod(head, tail, radius, fishShaft))
}

// arc adds a curved bone through four points, head to tail.
func (body *fishBody) arc(name, parent string, points [4]maths.Vec,
	wide, deep float64) error {
	form := skeleton.NewArc(points, wide, deep, maths.V(0, 0, 1))
	return body.add(name, parent, points[0], points[3], math.Max(wide, deep),
		form)
}

// fishCurve is four points from a to b, bowed out by bow at its middle.
func fishCurve(a, b, bow maths.Vec) [4]maths.Vec {
	third := b.Sub(a).Scale(1. / 3)
	return [4]maths.Vec{a, a.Add(third).Add(bow), a.Add(third.Scale(2)).
		Add(bow), b}
}

// skull adds the braincase at the root, and the jaws, gill covers and
// shoulder girdles on it.
func (body *fishBody) skull() error {
	head, out := body.head(), body.outline
	back := head * fishCondyle
	condyle := body.at(back, out.spineY(back), 0)
	front := body.at(head*fishSkullFront, fishSkullHigh*out.halfDepth(back), 0)
	form := skeleton.NewBlade(front, condyle,
		fishSkullWide*out.halfDepth(back),
		fishSkullThick*out.halfWidth(back), fishSkullStart, maths.V(0, 0, 1))
	if err := body.add("neurocranium", "", condyle, front,
		fishSkullThick*out.halfWidth(back), form); err != nil {
		return err
	}
	for i, side := range fishSides {
		if err := body.jaws(fishSideNames[i], side); err != nil {
			return err
		}
		if err := body.cover(fishSideNames[i], side); err != nil {
			return err
		}
	}
	return nil
}

// jaws adds one side's upper and lower jaw, hinged at the back of the
// gape and meeting their fellows at the snout.
func (body *fishBody) jaws(name string, side float64) error {
	out := body.outline
	gape := body.plan.figure("gape") * body.head()
	half, wide := out.halfDepth(gape), out.halfWidth(gape)
	hinge := body.at(gape, -fishJawChin*half, side*fishJawSide*wide)
	thick := fishJawThick * half
	// each side's tip stands its own thickness off the middle, so the two
	// sides meet at the snout without sharing a point
	tip := body.at(0, 0, side*thick)
	bow := maths.V(0, 0, side*fishJawThick*wide)
	if err := body.arc("premaxilla-"+name, "neurocranium",
		fishCurve(hinge, tip, bow), thick, thick); err != nil {
		return err
	}
	low := body.at(gape, -fishJawLow*half, side*fishJawSide*wide)
	chin := body.at(0, -fishJawChin*half*fishSkullFront, side*thick)
	return body.arc("dentary-"+name, "neurocranium",
		fishCurve(low, chin, bow), thick, thick)
}

// cover adds one side's gill cover and the cleithrum behind it, from
// which the pectoral fin hangs.
func (body *fishBody) cover(name string, side float64) error {
	out, head := body.outline, body.head()
	middle := head * (fishCoverFront + fishCoverBack) / 2
	half, wide := out.halfDepth(middle), out.halfWidth(middle)
	top := body.at(head*fishCoverFront, fishCoverHigh*half, side*wide)
	foot := body.at(head*fishCoverBack, fishCoverLow*half, side*wide)
	form := skeleton.NewBlade(top, foot, fishCoverWide*half,
		fishCoverThin*half, fishSkullStart, maths.V(0, 0, 1))
	if err := body.add("opercle-"+name, "neurocranium", top, foot,
		fishCoverThin*half, form); err != nil {
		return err
	}
	ridge := out.halfDepth(head)
	girdle := body.at(head, fishGirdleTop*ridge,
		side*out.side(head, fishGirdleTop*ridge)*fishRibInset)
	low := body.at(head*fishGirdleSlant, fishGirdleLow*ridge,
		side*out.side(head, fishGirdleLow*ridge)*fishRibInset)
	return body.arc("cleithrum-"+name, "neurocranium", fishCurve(girdle,
		low, maths.Vec{}), fishThin*ridge*2, fishThin*ridge*2)
}

// spine adds the column: the precaudal and caudal vertebrae by the
// species' counts, from the skull's joint to the tail's base, each in
// its form, with its haemal spine or its ribs.
func (body *fishBody) spine() error {
	precaudal := body.plan.whole("precaudal")
	caudal := body.plan.whole("vertebrae") - precaudal
	regions := []skeleton.Region{
		skeleton.NewRegion("P", precaudal, fishPrecaudalFirst, 1),
		skeleton.NewRegion("Ca", caudal, 1, fishCaudalLast)}
	if err := body.bones.Column("neurocranium", regions,
		body.columnPath); err != nil {
		return err
	}
	for n := 1; n <= precaudal; n++ {
		body.column = append(body.column, fmt.Sprintf("P%d", n))
	}
	for n := 1; n <= caudal; n++ {
		body.column = append(body.column, fmt.Sprintf("Ca%d", n))
	}
	for i, name := range body.column {
		if err := body.vertebra(name, i < precaudal, i+1); err != nil {
			return err
		}
	}
	return nil
}

// columnPath is where the column runs: from the skull's joint to the
// tail's base, high in the body, and a centrum's radius along it.
func (body *fishBody) columnPath(share float64) (maths.Vec, float64) {
	out := body.outline
	from := body.head() * fishCondyle
	s := from + share*(out.tailAt-from)
	radius := math.Max(fishCentrum*out.halfDepth(s),
		fishCentrumLeast*out.length)
	return body.at(s, out.spineY(s), 0), radius
}

// vertebra gives one vertebra its centrum and neural spine, and its
// haemal spine where it is caudal or its ribs where it is precaudal and
// far enough back to carry them.
func (body *fishBody) vertebra(name string, precaudal bool, n int) error {
	at, _ := body.bones.Find(name)
	bone := body.bones.Bone(at)
	radius, _ := bone.Thickness()
	middle := bone.Head().Add(bone.Tail()).Scale(fishHalf)
	s := body.outline.shareAt(middle.X)
	half := body.outline.halfDepth(s)
	neural := fishNeural * (half - middle.Y)
	body.forms[name] = skeleton.NewCentrum(bone.Head(), bone.Tail(), radius,
		neural, 0)
	if !precaudal {
		foot := middle.Sub(maths.V(0, radius, 0))
		reach := fishHaemal * (half + middle.Y)
		tip := foot.Add(maths.V(reach*fishSpineRake, -reach, 0))
		return body.rod("haemal-"+name, name, foot, tip, fishSpineRatio*radius)
	}
	if n < body.plan.whole("ribs-from") {
		return nil
	}
	return body.ribs(name, middle, radius, s)
}

// fishSpineRatio is a haemal spine's or a rib's half thickness, of its
// centrum's radius. Picked.
const fishSpineRatio = .22

// ribs adds a pair of ribs to a precaudal vertebra, curving down inside
// the body wall.
func (body *fishBody) ribs(name string, middle maths.Vec, radius,
	s float64) error {
	out := body.outline
	half := out.halfDepth(s)
	low := middle.Y - fishRibDown*(half+middle.Y)
	for i, side := range fishSides {
		start := middle.Add(maths.V(0, 0, side*radius))
		foot := maths.V(middle.X-fishSpineRake*half, low,
			side*out.side(s, low)*fishRibInset)
		bow := maths.V(0, 0, side*out.side(s, middle.Y)*fishRibInset/2)
		rib := fmt.Sprintf("rib-%s-%s", fishSideNames[i], name)
		thick := fishSpineRatio * radius
		if err := body.arc(rib, name, fishCurve(start, foot, bow), thick,
			thick); err != nil {
			return err
		}
	}
	return nil
}

// nearest is the vertebra nearest share s along the fish.
func (body *fishBody) nearest(s float64) string {
	x := body.outline.x(s)
	best, far := body.column[0], math.Inf(1)
	for _, name := range body.column {
		at, _ := body.bones.Find(name)
		bone := body.bones.Bone(at)
		off := math.Abs(bone.Head().Add(bone.Tail()).Scale(fishHalf).X - x)
		if off < far {
			best, far = name, off
		}
	}
	return best
}

// tail adds the tail's own skeleton: the hypurals fanned from the last
// vertebra, and the caudal rays on them, their tips along the tail's
// forked edge, which carry the tail's web.
func (body *fishBody) tail() error {
	last := body.column[len(body.column)-1]
	at, _ := body.bones.Find(last)
	root := body.bones.Bone(at).Tail()
	out := body.outline
	for k := 0; k < fishHypurals; k++ {
		fan := fishFan(k, fishHypurals)
		tip := body.at(out.tailAt+fishHypuralRun, root.Y+fan*fishHypuralFan*
			out.stalk, 0)
		if err := body.rod(fmt.Sprintf("hypural-%d", k+1), last, root, tip,
			fishThin*out.stalk); err != nil {
			return err
		}
	}
	return body.caudalRays(root.Y)
}

// fishFan is the place of item k of count across a fan, from minus one
// to one.
func fishFan(k, count int) float64 {
	return 2*float64(k)/float64(count-1) - 1
}

// caudalRays adds the tail's rays and its fin.
func (body *fishBody) caudalRays(middle float64) error {
	out, count := body.outline, body.plan.whole("caudal-rays")
	span := body.plan.figure("tail-span") * out.length
	sweep := body.plan.figure("tail-sweep")
	fin := fishFin{look: "tail", facing: maths.V(0, 0, 1)}
	for i := 0; i < count; i++ {
		t := fishFan(i, count)
		base := body.at(out.tailAt+fishHypuralRun/2,
			middle+t*fishTailRoot*out.stalk, 0)
		tip := body.at(1+sweep*math.Pow(math.Abs(t), fishLunate),
			middle+t*span/2, 0)
		plate := int(math.Round((t + 1) / 2 * (fishHypurals - 1)))
		name := fmt.Sprintf("caudal-ray-%d", i+1)
		if err := body.rod(name, fmt.Sprintf("hypural-%d", plate+1), base,
			tip, fishRayRadius*out.length); err != nil {
			return err
		}
		fin.rays = append(fin.rays, fishRay{bone: name, base: base, tip: tip})
	}
	body.fins = append(body.fins, fin)
	return nil
}

// medianFins adds the dorsal and anal fins the species has: each ray on
// its support, which stands on the vertebra beneath it.
func (body *fishBody) medianFins() error {
	for _, name := range fishMedianFins {
		spines := body.plan.whole(name + "-spines")
		count := spines + body.plan.whole(name+"-rays")
		if count == 0 {
			continue
		}
		side, look := 1., "fin"
		if name == "anal" {
			side, look = -1, "low-fin"
		}
		fin := fishFin{look: look, facing: maths.V(0, 0, 1)}
		for k := 0; k < count; k++ {
			ray, err := body.medianRay(name, k, count, spines, side)
			if err != nil {
				return err
			}
			fin.rays = append(fin.rays, ray)
		}
		body.fins = append(body.fins, fin)
	}
	return nil
}

// medianRay adds ray k of a median fin of count rays, the first spines of
// them spines, on the back where side is one and the belly where it is
// minus one, and its support.
func (body *fishBody) medianRay(fin string, k, count, spines int,
	side float64) (fishRay, error) {
	plan, out := body.plan, body.outline
	share := float64(k) / float64(count-1)
	s := plan.figure(fin+"-start") + share*(plan.figure(fin+"-end")-
		plan.figure(fin+"-start"))
	front, back := plan.figure(fin+"-front"), plan.figure(fin+"-back")
	high := (front + (back-front)*math.Pow(share, fishFinCurve)) * out.length
	base := body.at(s, side*fishFinRoot*out.halfDepth(s), 0)
	inner := body.at(s, side*fishSupportDeep*out.halfDepth(s), 0)
	support := fmt.Sprintf("%s-support-%d", fin, k+1)
	radius := fishRayRadius * out.length
	if err := body.rod(support, body.nearest(s), inner, base,
		fishSupportWide*radius); err != nil {
		return fishRay{}, err
	}
	kind, thick := "ray", radius
	if k < spines {
		kind, thick = "spine", fishSpineThicker*radius
	}
	tip := base.Add(maths.V(-math.Sin(fishRake), side*math.Cos(fishRake),
		0).Scale(high))
	name := fmt.Sprintf("%s-%s-%d", fin, kind, k+1)
	return fishRay{bone: name, base: base, tip: tip},
		body.rod(name, support, base, tip, thick)
}

// The paired fins and the finlets. A pectoral fin hangs from the
// cleithrum behind the gill cover, its rays fanned back and down, the
// upper ones longest; a pelvic fin hangs from its own small bone, on the
// girdle where the species' pelvic fins lie under the pectorals and on
// the vertebra above where they lie further back. A finlet is a small
// flag of skin on one ray, in a row along the stalk above and below.

// fishFinletRays are the rays a finlet's web is stretched between: its
// front edge, its middle and its back, all on its one bone.
const fishFinletRays = 3 // defined: the web's fewest

// finlets adds the rows of finlets above and below the stalk.
func (body *fishBody) finlets() error {
	plan := body.plan
	rows := []struct {
		name  string
		after string
		side  float64
	}{{"dorsal", plan.lastDorsal(), 1}, {"anal", "anal", -1}}
	for _, row := range rows {
		count := plan.whole(row.name + "-finlets")
		from := plan.figure(row.after+"-end") + fishFinletGap
		pitch := (body.outline.stalkAt - from) / float64(max(count, 1))
		for k := 0; k < count; k++ {
			s := from + (float64(k)+fishHalf)*pitch
			name := fmt.Sprintf("%s-finlet-%d", row.name, k+1)
			if err := body.finlet(name, s, pitch, row.side); err != nil {
				return err
			}
		}
	}
	return nil
}

// lastDorsal is the dorsal fin the finlets follow: the second where the
// species has one, else the first.
func (plan fishPlan) lastDorsal() string {
	if plan.species.Figures["second-dorsal-rays"].Most > 0 {
		return "second-dorsal"
	}
	return "first-dorsal"
}

// finlet adds one finlet at share s, pitch long, on the back where side
// is one and the belly where it is minus one.
func (body *fishBody) finlet(name string, s, pitch, side float64) error {
	out := body.outline
	high := fishFinletHigh * out.length
	fin := fishFin{look: "finlet", facing: maths.V(0, 0, 1)}
	for k := 0; k < fishFinletRays; k++ {
		share := float64(k) / (fishFinletRays - 1)
		at := s + (share-fishHalf)*pitch*fishRibInset
		base := body.at(at, side*fishFinRoot*out.halfDepth(at), 0)
		lift := high * (1 - share*fishRibInset)
		tip := base.Add(maths.V(-fishFinletRake*high, side*lift, 0))
		fin.rays = append(fin.rays, fishRay{bone: name, base: base, tip: tip})
	}
	first := fin.rays[0]
	if err := body.rod(name, body.nearest(s), first.base, first.tip,
		fishRayRadius*out.length); err != nil {
		return err
	}
	body.fins = append(body.fins, fin)
	return nil
}

// pectorals adds a pectoral fin on each side.
func (body *fishBody) pectorals() error {
	for i, side := range fishSides {
		if err := body.pectoral(fishSideNames[i], side); err != nil {
			return err
		}
	}
	return nil
}

// pectoral adds one side's pectoral fin, its rays on the cleithrum.
func (body *fishBody) pectoral(name string, side float64) error {
	plan, out := body.plan, body.outline
	count := plan.whole("pectoral-rays")
	s := plan.figure("pectoral-at")
	half := out.halfDepth(s)
	top := (plan.figure("pectoral-height") + fishPectoralBase/2) * half
	longest := plan.figure("pectoral") * body.head() * out.length
	fin := fishFin{look: "fin"}
	for k := 0; k < count; k++ {
		share := float64(k) / float64(count-1)
		y := top - share*fishPectoralBase*half
		base := body.at(s, y, side*out.side(s, y)*fishFinInset)
		reach := longest * (1 - fishFalcate*math.Pow(share, fishFalcateBend))
		way := maths.V(-1, -fishPectoralDrop-fishPectoralFan*share,
			side*fishPectoralOut).Unit()
		ray := fmt.Sprintf("pectoral-%s-ray-%d", name, k+1)
		tip := base.Add(way.Scale(reach))
		if err := body.rod(ray, "cleithrum-"+name, base, tip,
			fishRayRadius*out.length); err != nil {
			return err
		}
		fin.rays = append(fin.rays, fishRay{bone: ray, base: base, tip: tip})
	}
	fin.facing = fishFacing(fin.rays)
	body.fins = append(body.fins, fin)
	return nil
}

// fishFacing is the way a paired fin's faces look: square to its base
// and to its middle ray.
func fishFacing(rays []fishRay) maths.Vec {
	first, last := rays[0], rays[len(rays)-1]
	middle := rays[len(rays)/2]
	return last.base.Sub(first.base).Cross(middle.tip.Sub(middle.base)).Unit()
}

// pelvics adds a pelvic fin on each side.
func (body *fishBody) pelvics() error {
	for i, side := range fishSides {
		if err := body.pelvic(fishSideNames[i], side); err != nil {
			return err
		}
	}
	return nil
}

// pelvic adds one side's pelvic bone and fin, the bone on the girdle or
// on the vertebra above, by how far back the species carries it.
func (body *fishBody) pelvic(name string, side float64) error {
	plan, out := body.plan, body.outline
	s := plan.figure("pelvic-at")
	parent := body.nearest(s)
	if s <= body.head()*fishThoracic {
		parent = "cleithrum-" + name
	}
	half, wide := out.halfDepth(s), out.halfWidth(s)
	low := -fishPelvicLow * half
	pelvis := "pelvis-" + name
	inner := body.at(s-fishPelvisRun, low*fishHalf, side*wide*fishPelvicSpread)
	foot := body.at(s, low, side*wide*fishPelvicSpread)
	if err := body.add(pelvis, parent, inner, foot, fishThin*half,
		skeleton.NewBlade(inner, foot, fishThin*half*2, fishThin*half,
			fishSkullStart, maths.V(0, 1, 0))); err != nil {
		return err
	}
	return body.pelvicRays(name, pelvis, s, side)
}

// pelvicRays adds one side's pelvic rays on its pelvic bone, and its fin.
func (body *fishBody) pelvicRays(name, pelvis string, s, side float64) error {
	plan, outline := body.plan, body.outline
	count := plan.whole("pelvic-rays")
	half, wide := outline.halfDepth(s), outline.halfWidth(s)
	longest := plan.figure("pelvic") * outline.length
	fin := fishFin{look: "low-fin"}
	for k := 0; k < count; k++ {
		share := float64(k) / float64(count-1)
		spread := wide * fishPelvicSpread * (1 + share)
		base := body.at(s, -fishPelvicLow*half, side*spread)
		way := maths.V(-1, -fishPelvicDrop, side*fishPelvicOut).Unit()
		tip := base.Add(way.Scale(longest * (1 - fishPelvicShort*share)))
		ray := fmt.Sprintf("pelvic-%s-ray-%d", name, k+1)
		if err := body.rod(ray, pelvis, base, tip,
			fishRayRadius*outline.length); err != nil {
			return err
		}
		fin.rays = append(fin.rays, fishRay{bone: ray, base: base, tip: tip})
	}
	fin.facing = fishFacing(fin.rays)
	body.fins = append(body.fins, fin)
	return nil
}

// holders are the bones that hold the body's skin: the skull, along its
// middle from the snout to the column, and each vertebra along its own.
func (body *fishBody) holders() []skeleton.Holder {
	skull, _ := body.bones.Find("neurocranium")
	condyle := body.bones.Bone(skull).Head()
	snout := body.at(0, 0, 0)
	holders := []skeleton.Holder{skeleton.NewHolder(skull,
		[]maths.Vec{snout, condyle}, fishSkullReach*snout.Sub(condyle).Len())}
	for _, name := range body.column {
		at, _ := body.bones.Find(name)
		bone := body.bones.Bone(at)
		line := []maths.Vec{bone.Head(), bone.Tail()}
		holders = append(holders, skeleton.NewHolder(at, line,
			fishVertebraReach*bone.Length()))
	}
	return holders
}
