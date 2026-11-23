package manufacture

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/maths"
)

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
