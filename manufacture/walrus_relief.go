package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// The relief of the hide, in field units: a depth of one is about
// .6 of a metre of skin at the hide, where the field falls by about
// 1.7 for each metre. Every depth here is chosen, not measured.
const (
	// walrusNeckFoldSpacing is the distance between the rings of skin
	// round the neck.
	walrusNeckFoldSpacing = .115
	// walrusNeckFoldDepth is how deep a neck ring cuts at its deepest.
	walrusNeckFoldDepth = .07
	// walrusFlankFoldDepth is how deep the fold along the flank cuts.
	walrusFlankFoldDepth = .045
	// walrusRootFoldDepth is how deep the fold round a flipper root cuts.
	walrusRootFoldDepth = .05
	// walrusWrinkleDepth is how deep the fine wrinkles cut.
	walrusWrinkleDepth = .022
	// walrusCreaseScale turns the deepest total fold into one.
	walrusCreaseScale = .10
	// walrusBossCell is the side of the lattice the bosses sit on.
	walrusBossCell = .085
	// walrusBossRadius is the width of an average boss.
	walrusBossRadius = .036
	// walrusFloor is the height of the ground the belly lies on.
	walrusFloor = -.41
)

// walrusFoldProfile is a groove at every whole number of u, from one at
// the centre to nothing a few widths away.
func walrusFoldProfile(u, width float64) float64 {
	off := (u - math.Round(u)) / width
	return math.Exp(-off * off)
}

// walrusNeckFolds are the rings of loose skin that wrap the neck. They
// wander with the lie of the body so no two run exactly alike, and are
// deeper over the back and sides than under the throat.
func walrusNeckFolds(p maths.Vec) float64 {
	const (
		start, finish, fadeIn, fadeOut = .60, .80, 1.25, 1.38
		wander, slope, tilt            = .35, 2.2, .45
		width, bellyShare              = .2, .35
		lowest, highest                = -.25, .05
		crinkle, crinkleRate           = .25, 5.
		drift, driftScale              = 1.2, 3.
		unevenLow, unevenSpan          = .45, 1.0
		unevenScale                    = 5.
	)
	zone := maths.Smooth(start, finish, p.X) * (1 - maths.Smooth(fadeIn, fadeOut, p.X))
	if zone == 0 {
		return 0
	}
	u := p.X/walrusNeckFoldSpacing + wander*math.Sin(slope*p.Z+1.1) + tilt*p.Y +
		crinkle*math.Sin(crinkleRate*p.Y+3*p.Z) + drift*walrusNoise(p.Scale(driftScale))
	back := maths.Smooth(lowest, highest, p.Y)
	uneven := unevenLow + unevenSpan*walrusNoise(p.Scale(unevenScale))
	depth := walrusNeckFoldDepth * zone * uneven * (bellyShare + (1-bellyShare)*back)
	return depth * walrusFoldProfile(u, width)
}

// walrusFlankFold is the long crease where the sagging side of the barrel
// folds over the belly.
func walrusFlankFold(p maths.Vec) float64 {
	const (
		rearStart, rearEnd, frontStart, frontEnd = -1.0, -.7, .2, .5
		sideStart, sideEnd                       = .15, .3
		height, sway, width                      = -.17, .04, .032
	)
	zone := maths.Smooth(rearStart, rearEnd, p.X) *
		(1 - maths.Smooth(frontStart, frontEnd, p.X)) *
		maths.Smooth(sideStart, sideEnd, math.Abs(p.Z))
	if zone == 0 {
		return 0
	}
	line := (p.Y - height + sway*math.Sin(3*p.X)) / width
	return walrusFlankFoldDepth * zone * math.Exp(-line*line)
}

// walrusRootFolds are the creases where each flipper grows from the body:
// a ring of skin round its wrist or ankle that opens and closes as it
// swings.
func walrusRootFolds(p maths.Vec) float64 {
	const ringWidth = .03
	total := 0.
	for _, flipper := range walrusRayLines {
		root := flipper[walrusDigitCount/2][0]
		off := (p.Sub(root).Len() - walrusRootRing) / ringWidth
		total += walrusRootFoldDepth * math.Exp(-off*off)
	}
	return total
}

// walrusRootRing is the radius of the crease round a flipper's root,
// centred on the base of its middle ray, in metres.
const walrusRootRing = .16

// The hide is "thick, rough, and heavily marked with creases and folds"
// (OBIS-SEAMAP species profile, as read). Over the body the creases are
// cut as a net of cells, the edges where two cells meet, the cells drawn
// out along the body so the creases run more round it than along it, as
// skin that is bent along the body creases across it. Sizes chosen.
const (
	// walrusCreaseCell is the size of a cell of the crease net across the
	// body; along it the cells are walrusCreaseStretch times as long.
	walrusCreaseCell    = .07
	walrusCreaseStretch = 1.8
	// walrusCreaseWidth is how sharp a crease is, as a share of a cell.
	walrusCreaseWidth = .09
)

// walrusWrinkles are the creases of the hide's net: deep in patches, where a
// slow noise lets them, shallow elsewhere, and none on the muzzle pad.
func walrusWrinkles(p maths.Vec) float64 {
	const (
		patchLow, patchHigh = .25, .7
		patchScale          = 3.
		shallow             = .35
	)
	patch := maths.Smooth(patchLow, patchHigh, walrusNoise(p.Scale(patchScale)))
	bare := 1 - walrusPadShare(p)
	edge := walrusCellEdge(maths.V(p.X/(walrusCreaseCell*walrusCreaseStretch),
		p.Y/walrusCreaseCell, p.Z/walrusCreaseCell))
	return walrusWrinkleDepth * (shallow + (1-shallow)*patch) * bare *
		math.Exp(-(edge/walrusCreaseWidth)*(edge/walrusCreaseWidth))
}

// walrusCellEdge is how far q lies from the nearest edge between the cells
// of a jittered lattice with one point to each unit cube: the difference
// between its distances to the nearest and the second nearest point.
func walrusCellEdge(q maths.Vec) float64 {
	ix, iy, iz := int(math.Floor(q.X)), int(math.Floor(q.Y)), int(math.Floor(q.Z))
	first, second := math.Inf(1), math.Inf(1)
	for dx := -1; dx <= 1; dx++ {
		for dy := -1; dy <= 1; dy++ {
			for dz := -1; dz <= 1; dz++ {
				x, y, z := ix+dx, iy+dy, iz+dz
				point := maths.V(float64(x)+walrusHash(x, y, z, 21),
					float64(y)+walrusHash(x, y, z, 22), float64(z)+walrusHash(x, y, z, 23))
				distance := q.Sub(point).Len()
				if distance < first {
					first, second = distance, first
				} else if distance < second {
					second = distance
				}
			}
		}
	}
	return second - first
}

// walrusFlipperCreases are the fine creases across the bare skin of each
// flipper: grooves square to each ray of bone, close together, on the
// web over the palm, the sole and the digits. Spacing and depth chosen.
func walrusFlipperCreases(p maths.Vec) float64 {
	const (
		spacing, width = .014, .25
		depth, near    = .02, .05
		reach          = .6
	)
	along, nearest := 0., math.Inf(1)
	for _, flipper := range walrusRayLines {
		if p.Sub(flipper[walrusDigitCount/2][0]).Len() > reach {
			continue
		}
		for _, ray := range flipper {
			if d := walrusLineDistance(p, ray); d < nearest {
				nearest, along = d, p.Sub(ray[0]).Dot(ray[len(ray)-1].Sub(ray[0]).Unit())
			}
		}
	}
	if nearest > near {
		return 0
	}
	return depth * walrusFoldProfile(along/spacing, width)
}

// walrusDigitGrooves are the clefts that run down each flipper between its
// five digits, on both faces: a groove wherever two neighbouring rays of
// bone lie equally near, and both near enough to be under the web.
func walrusDigitGrooves(p maths.Vec) float64 {
	const (
		depth, width = .05, .009
		web, reach   = .045, .6
	)
	total := 0.
	for _, flipper := range walrusRayLines {
		middle := flipper[walrusDigitCount/2]
		if p.Sub(middle[0]).Len() > reach {
			continue
		}
		var near [walrusDigitCount]float64
		for ray, line := range flipper {
			near[ray] = walrusLineDistance(p, line)
		}
		for ray := 1; ray < walrusDigitCount; ray++ {
			a, b := near[ray-1], near[ray]
			if math.Max(a, b) > web {
				continue
			}
			off := (a - b) / width
			total += depth * math.Exp(-off*off)
		}
	}
	return total
}

// walrusHeadFolds are the creases of the face: rings of wrinkles round each
// eye, and furrows across the cheek and the bridge of the muzzle.
func walrusHeadFolds(p maths.Vec) float64 {
	const (
		start, end                = 1.2, 1.85
		ringDepth, ringWidth      = .035, .007
		ringFirst, ringStep       = .045, .03
		rings                     = 3
		ringReach                 = .14
		furrowSpacing             = .05
		furrowDepth, furrowWidth  = .03, .2
		furrowFrom, furrowTo      = 1.56, 1.72
		furrowWander, furrowSwing = .5, 6.
	)
	if p.X < start || p.X > end {
		return 0
	}
	total := 0.
	for _, side := range []float64{-1, 1} {
		reach := p.Sub(walrusEyeSocket(side)).Len()
		mask := math.Exp(-(reach / ringReach) * (reach / ringReach))
		for ring := 0; ring < rings; ring++ {
			off := (reach - ringFirst - float64(ring)*ringStep) / ringWidth
			total += ringDepth * mask * math.Exp(-off*off)
		}
	}
	zone := maths.Smooth(furrowFrom-.05, furrowFrom, p.X) *
		(1 - maths.Smooth(furrowTo, furrowTo+.05, p.X))
	u := p.X/furrowSpacing + furrowWander*math.Sin(furrowSwing*p.Z)
	return total + furrowDepth*zone*walrusFoldProfile(u, furrowWidth)
}

// walrusFolds is the total depth of every fold and wrinkle at p.
func walrusFolds(p maths.Vec) float64 {
	return walrusNeckFolds(p) + walrusFlankFold(p) + walrusRootFolds(p) +
		walrusWrinkles(p) + walrusDigitGrooves(p) + walrusHeadFolds(p) +
		walrusFlipperCreases(p)
}

// walrusCrease is walrusFolds as a share, from nought on smooth hide to
// one in the floor of a deep fold. The hide's colour reads it.
func walrusCrease(p maths.Vec) float64 {
	return math.Min(1, walrusFolds(p)/walrusCreaseScale)
}

// walrusBoss is the height of the bosses on a bull at p, from nought to
// one: "many lumps or nodules on the neck and chest" (OBIS-SEAMAP species
// profile, as read). The bosses sit on a jittered lattice and differ in
// size; they cover the neck, the shoulders and the chest down to the
// ground, and fade out behind the chest and at the back of the head.
func walrusBoss(p maths.Vec) float64 {
	const (
		frontStart, frontEnd = .45, .65
		rearStart, rearEnd   = 1.15, 1.30
		lowest, highest      = -.45, -.38
		reach                = 9.
	)
	zone := maths.Smooth(frontStart, frontEnd, p.X) *
		(1 - maths.Smooth(rearStart, rearEnd, p.X)) *
		maths.Smooth(lowest, highest, p.Y)
	if zone == 0 {
		return 0
	}
	cx, cy, cz := math.Floor(p.X/walrusBossCell), math.Floor(p.Y/walrusBossCell),
		math.Floor(p.Z/walrusBossCell)
	best := 0.
	for dx := -1; dx <= 1; dx++ {
		for dy := -1; dy <= 1; dy++ {
			for dz := -1; dz <= 1; dz++ {
				best = math.Max(best, walrusBossAt(p, int(cx)+dx, int(cy)+dy,
					int(cz)+dz, reach))
			}
		}
	}
	return zone * best
}

// walrusBossAt is the height at p of the boss in one lattice cell, or
// nothing if p lies more than reach squared radii from it.
func walrusBossAt(p maths.Vec, ix, iy, iz int, reach float64) float64 {
	const (
		margin, spread = .2, .6
		smallest       = .7
		variation      = .6
	)
	centre := maths.V(
		(float64(ix)+margin+spread*walrusHash(ix, iy, iz, 1))*walrusBossCell,
		(float64(iy)+margin+spread*walrusHash(ix, iy, iz, 2))*walrusBossCell,
		(float64(iz)+margin+spread*walrusHash(ix, iy, iz, 3))*walrusBossCell)
	size := walrusBossRadius * (smallest + variation*walrusHash(ix, iy, iz, 4))
	offset := p.Sub(centre)
	far := offset.Dot(offset) / (size * size)
	if far > reach {
		return 0
	}
	return math.Exp(-far)
}
