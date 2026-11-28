package primates

import (
	"math"

	"github.com/janearc/slayer/maths"
)

type humanAnatomy struct {
	spec  HumanSpec
	torso Torso
}

func newHumanAnatomy(spec HumanSpec) humanAnatomy {
	torso := spec.Torso
	torso.Width, torso.Depth = 1, 1
	return humanAnatomy{spec: spec, torso: torso}
}

func (anatomy humanAnatomy) build() humanMesh {
	var model humanMesh
	stations, shoulders := trunkStations(anatomy.spec.Anatomy)
	regions := [2]shoulderRegion{
		shoulderRegionAt(stations, shoulders[0]), shoulderRegionAt(stations, shoulders[1]),
	}
	trunk := anatomy.trunk(&model, stations, shoulders, regions)
	anatomy.head(&model, trunk[len(trunk)-1])
	for i, side := range humanSides {
		rim := anatomy.shoulderBoundary(&model, trunk, stations, shoulders[i], regions)
		anatomy.arm(&model, side, rim)
	}
	anatomy.legs(&model, trunk)
	if anatomy.spec.Anatomy == MaleAnatomy {
		model.maleGroin(trunk)
	}
	model.compact()
	return model
}

func (model *humanMesh) compact() {
	used := make([]bool, len(model.vertices))
	for _, face := range model.faces {
		for _, index := range face {
			used[index] = true
		}
	}
	reindex := make([]int, len(used))
	kept := model.vertices[:0]
	for i, vertex := range model.vertices {
		if used[i] {
			reindex[i] = len(kept)
			kept = append(kept, vertex)
		}
	}
	model.vertices = kept
	for i, face := range model.faces {
		for j, index := range face {
			model.faces[i][j] = reindex[index]
		}
	}
}

func trunkStations(form AdultAnatomy) ([]float64, [2]socket) {
	stations := []float64{pelvisY}
	appendSpan := func(end float64, count int) {
		start := stations[len(stations)-1]
		for i := 1; i <= count; i++ {
			stations = append(stations, mix(start, end, float64(i)/float64(count)))
		}
	}
	if form == FemaleAnatomy {
		// Resolve the folds and lower return of the pubic surface.
		appendSpan(.865, 30)
		appendSpan(1.071, 48)
	} else {
		appendSpan(1.071, 62)
	}
	appendSpan(1.253, 128)
	bottom := len(stations) - 1
	appendSpan(1.369, 48)
	top := len(stations) - 1
	appendSpan(1.455, 24)
	const half = 23
	sockets := [2]socket{
		{humanAround/4 - half, humanAround/4 + half, bottom, top, humanAround, 1},
		{3*humanAround/4 - half, 3*humanAround/4 + half, bottom, top, humanAround, -1},
	}
	return stations, sockets
}

func (anatomy humanAnatomy) trunk(model *humanMesh, stations []float64, shoulders [2]socket,
	regions [2]shoulderRegion,
) [][]int {
	trunk := make([][]int, len(stations))
	for row, station := range stations {
		rim := make([]int, humanAround)
		for column := range rim {
			angle := 2 * math.Pi * float64(column) / humanAround
			angle, y := shoulderChart(angle, station, regions)
			if anatomy.spec.Anatomy == MaleAnatomy {
				angle, y = pubicChart(angle, y, stations[pubisTopRow])
			} else {
				angle = femalePubicAngle(angle, y)
			}
			point, chart, tissue := anatomy.chest(angle, y)
			point = lowerTorso(point, angle)
			if anatomy.spec.Anatomy == MaleAnatomy && y < .912 {
				pubic := maths.Smooth(.76, .97, math.Cos(angle)) *
					(1 - maths.Smooth(.86, .912, y))
				chart = mixPoint(chart, point, pubic)
			}
			weights := trunkWeights(y, point)
			weights = shoulderWeights(weights, angle, y, regions)
			rim[column] = model.add(humanVertex{point: point, chart: chart,
				weights: weights, tissue: tissue})
		}
		trunk[row] = rim
		if row == 0 {
			continue
		}
		lower := trunk[row-1]
		for i, first := range lower {
			groin := anatomy.spec.Anatomy == MaleAnatomy && malePubicOpening(i, row-1)
			if !groin && !shoulders[0].covers(i, row-1) && !shoulders[1].covers(i, row-1) {
				next := (i + 1) % len(lower)
				model.quad(first, lower[next], rim[next], rim[i])
			}
		}
	}
	return trunk
}

func lowerTorso(point maths.Vec, angle float64) maths.Vec {
	y, x := point.Y, math.Abs(point.X)
	front := maths.Smooth(.15, .88, math.Cos(angle))
	back := maths.Smooth(.18, .95, -math.Cos(angle))
	// Paired gluteal volumes curve up and lateral from the intergluteal cleft.
	point.Z -= back * .027 * bell((x-.088)/.061) * bell((y-.846)/.078)
	point.Z += back * .019 * bell(x/.019) * bell((y-.814)/.076)
	// The pubic mound and the outer relief continue onto the shared bridge.
	point.Z += front * .016 * bell(x/.036) * bell((y-.813)/.063)
	return point
}

func (anatomy humanAnatomy) legs(model *humanMesh, trunk [][]int) {
	root := [2][]int{trunk[0], trunk[1]}
	first := len(model.vertices)
	if anatomy.spec.Anatomy == FemaleAnatomy {
		root = model.femalePelvicCuff(root)
	}
	front := model.vertices[root[0][0]].point
	back := model.vertices[root[0][humanAround/2]].point
	bridge := make([]int, 31)
	for i := range bridge {
		share := float64(i+1) / float64(len(bridge)+1)
		point := femaleSaddle(front, back, share)
		if anatomy.spec.Anatomy == MaleAnatomy {
			point = mixPoint(front, back, share)
			point.Y -= .057 * math.Pow(math.Sin(math.Pi*share), .85)
			point = malePerineum(point)
		}
		bridge[i] = model.point(point, fixedWeight("pelvis"))
	}
	for _, side := range humanSides {
		anatomy.leg(model, side, model.thighBoundary(root, bridge, side))
	}
	if anatomy.spec.Anatomy == FemaleAnatomy {
		model.finishFemalePelvis(trunk, first, bridge, root[0][humanAround/2], anatomy.spec.Vulvar)
	}
}
