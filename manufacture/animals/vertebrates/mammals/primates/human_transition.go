package primates

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// skinBoundary gives the shared root vertices and the cross-boundary
// directions of the adjoining skin.
type skinBoundary struct {
	rim    []int
	inward []maths.Vec
}

type shoulderRegion struct {
	angle, y, angleHalf, yHalf float64
}

func shoulderRegionAt(stations []float64, opening socket) shoulderRegion {
	bottom, top := stations[opening.bottom], stations[opening.top]
	return shoulderRegion{
		angle:     math.Pi * float64(opening.lo+opening.hi) / humanAround,
		angleHalf: math.Pi * float64(opening.hi-opening.lo) / humanAround,
		y:         .5 * (bottom + top),
		yHalf:     .5 * (top - bottom),
	}
}

func shoulderChart(angle, y float64, regions [2]shoulderRegion) (float64, float64) {
	for _, region := range regions {
		across, height := (angle-region.angle)/region.angleHalf, (y-region.y)/region.yHalf
		boxRadius := math.Max(math.Abs(across), math.Abs(height))
		if boxRadius == 0 || boxRadius >= 1.77 {
			continue
		}
		// Move the cut's perimeter onto an ellipse while preserving each
		// point's direction from its centre. Fade into the outer wall.
		round := boxRadius / math.Hypot(across, height)
		scale := mix(round, 1, maths.Smooth(1, 1.77, boxRadius))
		return region.angle + (angle-region.angle)*scale, region.y + (y-region.y)*scale
	}
	return angle, y
}

func (anatomy humanAnatomy) shoulderBoundary(model *humanMesh, trunk [][]int,
	stations []float64, opening socket, regions [2]shoulderRegion,
) skinBoundary {
	keys := opening.rim()
	result := skinBoundary{rim: make([]int, len(keys)), inward: make([]maths.Vec, len(keys))}
	region := shoulderRegionAt(stations, opening)
	for i, key := range keys {
		row, column := key/humanAround, key%humanAround
		index := trunk[row][column]
		result.rim[i] = index
		root := model.vertices[index].point
		angle := 2 * math.Pi * float64(column) / humanAround
		angle, y := shoulderChart(angle, stations[row], regions)
		// Sample radially across the smooth boundary in the authored wall.
		// Rectangular index directions would change abruptly at its corners.
		outsideAngle := region.angle + 1.014*(angle-region.angle)
		outsideY := region.y + 1.014*(y-region.y)
		outside, _, _ := anatomy.chest(outsideAngle, outsideY)
		result.inward[i] = root.Sub(lowerTorso(outside, outsideAngle))
	}
	return result
}

type skinJoin struct {
	root, startTangent, target, endTangent maths.Vec
}

func (join skinJoin) at(share float64) maths.Vec {
	return maths.V(
		hermite(join.root.X, join.target.X, join.startTangent.X, join.endTangent.X, share),
		hermite(join.root.Y, join.target.Y, join.startTangent.Y, join.endTangent.Y, share),
		hermite(join.root.Z, join.target.Z, join.startTangent.Z, join.endTangent.Z, share),
	)
}

// connectedSweep joins an existing skin to a distal profile using matching
// cross-boundary tangents. The distal sweep takes over after the join.
func (model *humanMesh) connectedSweep(limb limbSweep, boundary skinBoundary, rootTravel float64) skinBoundary {
	first := limb.stations[0]
	angles := model.angles(boundary.rim, model.centre(boundary.rim), limb.across(first.bend),
		limb.forward(first.bend), first.width, first.depth)
	targetAt := func(at float64, index int) maths.Vec {
		ring := limbSectionAt(limb.stations, at)
		return limb.point(ring, angles.angle(index, maths.Smooth(0, .52, at)))
	}
	joinAt := limb.blendEnd
	joins := make([]skinJoin, len(boundary.rim))
	const difference = .0004
	for i, index := range boundary.rim {
		target := targetAt(joinAt, i)
		tangent := targetAt(joinAt+difference, i).Sub(targetAt(joinAt-difference, i)).
			Scale(joinAt / (2 * difference))
		root := model.vertices[index].point
		travel := math.Min(rootTravel, .91*target.Sub(root).Len())
		joins[i] = skinJoin{root, boundary.inward[i].Unit().Scale(travel), target, tangent}
	}
	previous := boundary.rim
	var upstream []int
	for row := 1; row <= limb.rows; row++ {
		at := float64(row) / float64(limb.rows)
		ring := limbSectionAt(limb.stations, at)
		round := maths.Smooth(0, .52, at)
		share := maths.Smooth(0, joinAt, at)
		next := make([]int, len(previous))
		for i, index := range boundary.rim {
			point := limb.point(ring, angles.angle(i, round))
			if at < joinAt {
				point = joins[i].at(at / joinAt)
			}
			root := model.vertices[index]
			site, tissue := limb.site, root.tissue
			site.attachment = share
			tissue.shoulder *= 1 - share
			tissue.gravity *= 1 - share
			next[i] = model.add(humanVertex{point: point, chart: point, site: site,
				weights: limb.weight(at, point, root.weights), tissue: tissue})
		}
		model.connect(previous, next)
		upstream, previous = previous, next
	}
	rim := turnRim(previous, angles.front)
	result := skinBoundary{rim: rim, inward: make([]maths.Vec, len(rim))}
	for i, index := range rim {
		near := upstream[(i+angles.front)%len(rim)]
		result.inward[i] = model.vertices[index].point.Sub(model.vertices[near].point)
	}
	return result
}
