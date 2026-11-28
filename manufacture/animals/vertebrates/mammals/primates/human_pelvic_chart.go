package primates

import (
	"container/heap"
	"math"
	"slices"

	"github.com/janearc/slayer/manufacture/geometry"
	"github.com/janearc/slayer/maths"
)

// pelvicChart measures one midsagittal line through the indexed skin.
// Its longitudinal coordinate continues past the ends of the pubic rows.
type pelvicChart struct {
	arc  []pelvicStation
	zero float64
	turn float64
	anal float64
}

type pelvicStation struct {
	vertex   int
	front    []int
	point    maths.Vec
	distance float64
}

func (chart *pelvicChart) add(vertex int, front []int, point maths.Vec) {
	point.X = 0
	next := pelvicStation{vertex: vertex, front: front, point: point}
	if count := len(chart.arc); count != 0 {
		last := chart.arc[count-1]
		next.distance = last.distance + point.Sub(last.point).Len()
	}
	chart.arc = append(chart.arc, next)
}

func (model *humanMesh) pelvicChart(trunk [][]int, firstCuff int, bridge []int, back int) pelvicChart {
	var chart pelvicChart
	for row := len(trunk) - 1; row >= 0; row-- {
		rim := trunk[row]
		point := model.vertices[rim[0]].point
		if point.Y <= .878 {
			chart.add(rim[0], rim, point)
		}
	}
	chart.zero = chart.arc[len(chart.arc)-1].distance - .009
	for index := firstCuff; index < bridge[0]; index += humanAround {
		rim := make([]int, humanAround)
		for column := range rim {
			rim[column] = index + column
		}
		chart.add(index, rim, model.vertices[index].point)
	}
	chart.turn = chart.arc[len(chart.arc)-1].distance - chart.zero
	for _, index := range bridge {
		chart.add(index, nil, model.vertices[index].point)
	}
	chart.add(back, nil, model.vertices[back].point)
	chart.anal = chart.posteriorCourse(-.056)
	return chart
}

func (chart pelvicChart) posteriorCourse(depth float64) float64 {
	for i := len(chart.arc) - 1; i > 0; i-- {
		prior, next := chart.arc[i-1], chart.arc[i]
		if prior.point.Z >= depth && next.point.Z < depth {
			share := (depth - prior.point.Z) / (next.point.Z - prior.point.Z)
			return mix(prior.distance, next.distance, share) - chart.zero
		}
	}
	return chart.arc[len(chart.arc)-1].distance - chart.zero
}

type pelvicSkinSite struct {
	course, anteriorDistance, sagittalDistance float64
}

type pelvicSeed struct {
	node   int
	course float64
}

type pelvicReach struct {
	course, distance float64
}

type pelvicVisit struct {
	node     int
	distance float64
}

type pelvicQueue []pelvicVisit

func (queue pelvicQueue) Len() int           { return len(queue) }
func (queue pelvicQueue) Less(i, j int) bool { return queue[i].distance < queue[j].distance }
func (queue pelvicQueue) Swap(i, j int)      { queue[i], queue[j] = queue[j], queue[i] }

func (queue *pelvicQueue) Push(value any) {
	*queue = append(*queue, value.(pelvicVisit))
}

func (queue *pelvicQueue) Pop() any {
	last := len(*queue) - 1
	visit := (*queue)[last]
	*queue = (*queue)[:last]
	return visit
}

// walkPelvicSkin measures distance along the shared skin's indexed edges.
// Two surfaces can be close in space while the skin path between them is long.
func walkPelvicSkin(nodes []pelvicNode, seeds []pelvicSeed) []pelvicReach {
	reach := make([]pelvicReach, len(nodes))
	for i := range reach {
		reach[i].distance = math.Inf(1)
	}
	var queue pelvicQueue
	for _, seed := range seeds {
		reach[seed.node] = pelvicReach{course: seed.course}
		queue = append(queue, pelvicVisit{node: seed.node})
	}
	heap.Init(&queue)
	for queue.Len() != 0 {
		visit := heap.Pop(&queue).(pelvicVisit)
		from := reach[visit.node]
		if visit.distance > from.distance {
			continue
		}
		for _, edge := range nodes[visit.node].edges {
			if edge.node == 0 {
				continue
			}
			to := edge.node - 1
			distance := from.distance + 1/edge.weight
			if distance >= reach[to].distance {
				continue
			}
			reach[to] = pelvicReach{course: from.course, distance: distance}
			heap.Push(&queue, pelvicVisit{node: to, distance: distance})
		}
	}
	return reach
}

func (chart pelvicChart) skinSites(nodes []pelvicNode, lookup []int) []pelvicSkinSite {
	var sagittal, anterior []pelvicSeed
	anchored := make([]bool, len(nodes))
	for _, station := range chart.arc {
		course := station.distance - chart.zero
		if index := lookup[station.vertex]; index != 0 {
			sagittal = append(sagittal, pelvicSeed{node: index - 1, course: course})
			anchored[index-1] = true
		}
		// Only the authored anterior meridians carry the pubic row's
		// course across the mons and the inguinal attachment.
		for column, vertex := range station.front {
			const frontArc = humanAround / 11
			if column > frontArc && column < humanAround-frontArc {
				continue
			}
			if index := lookup[vertex]; index != 0 {
				anterior = append(anterior, pelvicSeed{node: index - 1, course: course})
				anchored[index-1] = true
			}
		}
	}
	fromSagittal := walkPelvicSkin(nodes, sagittal)
	fromAnterior := walkPelvicSkin(nodes, anterior)
	initial := make([]float64, len(nodes))
	for i, landmark := range fromSagittal {
		if fromAnterior[i].distance < landmark.distance {
			landmark = fromAnterior[i]
		}
		initial[i] = landmark.course
	}

	// Relax the course between the marked rows and centreline. A small
	// rest term keeps its values local to the neighboring landmarks.
	course := slices.Clone(initial)
	next := make([]float64, len(nodes))
	for sweep := 0; sweep < 56; sweep++ {
		for i, node := range nodes {
			next[i] = initial[i]
			if anchored[i] || math.IsInf(fromSagittal[i].distance, 1) {
				continue
			}
			sum, total := 0.0, 0.0
			for _, edge := range node.edges {
				if edge.node == 0 {
					continue
				}
				sum += edge.weight * course[edge.node-1]
				total += edge.weight
			}
			if total != 0 {
				next[i] = mix(sum/total, initial[i], .035)
			}
		}
		course, next = next, course
	}

	sites := make([]pelvicSkinSite, len(nodes))
	for i, value := range course {
		sites[i] = pelvicSkinSite{
			course:           value,
			anteriorDistance: fromAnterior[i].distance,
			sagittalDistance: fromSagittal[i].distance,
		}
	}
	return sites
}

// pelvicSurfaceNormals carries the rest surface's outward direction across
// the shared attachment. Averaging over skin edges softens abrupt
// direction changes where the cuff curves into the saddle.
func (model *humanMesh) pelvicSurfaceNormals(nodes []pelvicNode) []maths.Vec {
	points := make([]maths.Vec, len(model.vertices))
	for i, vertex := range model.vertices {
		points[i] = vertex.point
	}
	rest := geometry.UnitNormals(points, model.faces)
	current := make([]maths.Vec, len(nodes))
	next := make([]maths.Vec, len(nodes))
	for i, node := range nodes {
		current[i] = rest[node.vertex]
	}
	for sweep := 0; sweep < 5; sweep++ {
		for i, node := range nodes {
			var sum maths.Vec
			total := 0.0
			for _, edge := range node.edges {
				normal := rest[edge.vertex]
				if edge.node != 0 {
					normal = current[edge.node-1]
				}
				sum = sum.Add(normal.Scale(edge.weight))
				total += edge.weight
			}
			next[i] = rest[node.vertex]
			if total > 0 {
				next[i] = mixPoint(next[i], sum.Scale(1/total), .72).Unit()
			}
		}
		current, next = next, current
	}
	return current
}

func (chart pelvicChart) displacement(x float64, site pelvicSkinSite, normal maths.Vec, form VulvarForm) maths.Vec {
	// Keep outer support on the anterior skin and adjacent groin. The
	// inner relief follows the narrow skin path around the underside.
	outerSupport := (1 - maths.Smooth(.010, .044, site.anteriorDistance)) *
		(1 - maths.Smooth(.061, .091, site.sagittalDistance))
	innerSupport := 1 - maths.Smooth(.016, .042, site.sagittalDistance)
	if math.Abs(x) >= .072 || (outerSupport == 0 && innerSupport == 0) {
		return maths.Vec{}
	}

	relief := form.relief(x, site.course, chart.anal)
	innerTurn := (math.Pi / 2) * maths.Smooth(chart.turn-.037, chart.turn+.050, site.course)
	down, forward := math.Sincos(innerTurn)
	sagittal := maths.V(0, -down, forward)
	// The actual outward direction changes across the pad as well as
	// along it. Keep a narrow sagittal margin at the meeting contours.
	normal.X *= maths.Smooth(0, .011, math.Abs(x))
	normal = normal.Unit()
	if normal == (maths.Vec{}) {
		normal = sagittal
	}
	inner := mixPoint(normal, sagittal, .22).Unit()
	front := mixPoint(sagittal, maths.ZAxis, .44).Unit()
	outer := normal.Add(front.Scale(.32)).Unit()
	return outer.Scale(outerSupport * relief.outer).Add(inner.Scale(innerSupport * relief.inner))
}

// fitPelvicRelief fits the authored displacements on the same mesh edges
// used to fit the base skin. Holding the relief target on its own chart
// keeps this solve from rounding away the narrow central features.
func (model *humanMesh) fitPelvicRelief(nodes []pelvicNode, lookup []int, chart pelvicChart, form VulvarForm) {
	sites := chart.skinSites(nodes, lookup)
	normals := model.pelvicSurfaceNormals(nodes)
	authored := make([]maths.Vec, len(nodes))
	for i, node := range nodes {
		x := model.vertices[node.vertex].point.X
		authored[i] = chart.displacement(x, sites[i], normals[i], form)
	}
	current := slices.Clone(authored)
	next := make([]maths.Vec, len(nodes))
	for sweep := 0; sweep < 8; sweep++ {
		for i, node := range nodes {
			var sum maths.Vec
			total := 0.0
			for _, edge := range node.edges {
				if edge.node != 0 {
					sum = sum.Add(current[edge.node-1].Scale(edge.weight))
				}
				total += edge.weight
			}
			next[i] = authored[i]
			if total > 0 {
				point := model.vertices[node.vertex].point
				ease := .05 + .075*maths.Smooth(.005, .020, math.Abs(point.X))
				next[i] = mixPoint(authored[i], sum.Scale(1/total), ease)
			}
		}
		current, next = next, current
	}
	for i, node := range nodes {
		point := model.vertices[node.vertex].point
		offset := current[i]
		// Preserve the shared midline and softly limit inward movement on
		// either side. This is a rest-shape bound for the cleft.
		if point.X == 0 {
			offset.X = 0
		} else if point.X*offset.X < 0 {
			limit := .65 * math.Abs(point.X)
			offset.X = math.Copysign(limit*(-math.Expm1(-math.Abs(offset.X)/limit)), offset.X)
		}
		model.movePelvicPoint(node.vertex, point.Add(offset))
	}
}
