package hair

import (
	"container/heap"
	"fmt"
	"math"
	"slices"

	"github.com/janearc/slayer/manufacture/geometry"
	"github.com/janearc/slayer/maths"
)

// GuideFieldSpec lays sparse guides on an indexed skin. Spacing is the
// chart-space cell width in metres. Region returns a nonnegative
// guide region, or a negative number outside the groom. Guide paths
// can cross only between vertices of the same region.
type GuideFieldSpec struct {
	Surface Surface
	Region  func(maths.Vec) int
	Spacing float64
	Seed    uint64
}

// GuideAnchor is a sparse point on the authored surface and its region.
type GuideAnchor struct {
	Point, Normal maths.Vec
	Region        int
}

// GuideWeight gives one anchor's share at a follicle.
type GuideWeight struct {
	Anchor int
	Share  float64
}

const guidesPerVertex = 4

// GuideWeights holds the barycentric contributions from the three
// corners of a follicle's triangle.
type GuideWeights [3 * guidesPerVertex]GuideWeight

// Leaders returns the two largest anchor weights after combining
// contributions from multiple corners. Missing leaders have zero share.
func (weights GuideWeights) Leaders() (first, second GuideWeight) {
	for i, weight := range weights {
		if weight.Share == 0 {
			continue
		}
		for j := i + 1; j < len(weights); j++ {
			if weights[j].Anchor == weight.Anchor {
				weight.Share += weights[j].Share
				weights[j].Share = 0
			}
		}
		if weight.Share > first.Share {
			first, second = weight, first
		} else if weight.Share > second.Share {
			second = weight
		}
	}
	return first, second
}

// SurfaceGuides retains the anchors and nearby guide weights on the
// connected surface. It can be shared by many seeded strand layers.
type SurfaceGuides struct {
	faces   [][3]int
	anchors []GuideAnchor
	weights [][guidesPerVertex]GuideWeight
}

// NewSurfaceGuides selects sparse vertices and propagates them along
// mesh edges. Distances are shortest paths on the edge graph.
func NewSurfaceGuides(spec GuideFieldSpec) (*SurfaceGuides, error) {
	surface := spec.Surface
	if spec.Region == nil || !(spec.Spacing > 0 && finite(spec.Spacing)) {
		return nil, fmt.Errorf("hair: guide field needs a region and positive spacing")
	}
	if err := checkSurface(surface.Points, surface.Faces); err != nil {
		return nil, err
	}
	if err := surface.checkChart(); err != nil {
		return nil, err
	}
	chart := surface.Chart
	if chart == nil {
		chart = surface.Points
	}
	normals := geometry.UnitNormals(chart, surface.Faces)
	regions := make([]int, len(chart))
	for i, point := range chart {
		regions[i] = spec.Region(point)
	}
	roots, err := guideRoots(chart, regions, spec)
	if err != nil {
		return nil, err
	}
	if len(roots) == 0 {
		return nil, fmt.Errorf("hair: guide region has no source vertices")
	}
	anchors := make([]GuideAnchor, len(roots))
	front := make(guideQueue, len(roots))
	for anchor, point := range roots {
		anchors[anchor] = GuideAnchor{chart[point], normals[point], regions[point]}
		front[anchor] = guideVisit{vertex: point, anchor: anchor}
	}
	graph := guideEdges(surface, regions)
	return &SurfaceGuides{
		faces: slices.Clone(surface.Faces), anchors: anchors,
		weights: guideDistances(graph, front, spec.Spacing),
	}, nil
}

// Anchors returns a copy in the order used by GuideWeight.
func (field *SurfaceGuides) Anchors() []GuideAnchor { return slices.Clone(field.anchors) }

// Weights blends the guides of the triangle's three corners using the
// follicle's barycentric weights. The region filter keeps adjacent
// strands on opposite sides of a part from borrowing one another's
// guides. Positive weights sum to one and can repeat an anchor.
func (field *SurfaceGuides) Weights(site Site, region int) GuideWeights {
	var weights GuideWeights
	if site.Face < 0 || site.Face >= len(field.faces) || region < 0 {
		return weights
	}
	total := 0.
	for corner, vertex := range field.faces[site.Face] {
		cornerShare := site.Weights[corner]
		if !(cornerShare > 0 && cornerShare <= 1) {
			continue
		}
		for index, each := range field.weights[vertex] {
			if each.Share == 0 || field.anchors[each.Anchor].Region != region {
				continue
			}
			share := cornerShare * each.Share
			weights[corner*guidesPerVertex+index] = GuideWeight{each.Anchor, share}
			total += share
		}
	}
	if total > 0 {
		for i := range weights {
			weights[i].Share /= total
		}
	}
	return weights
}

type guideCell struct {
	x, y, z int64
	region  int
}

// guideRoots picks the nearest eligible vertex to each seeded cell
// centre. The first encounter order is kept independently of map order.
func guideRoots(points []maths.Vec, regions []int, spec GuideFieldSpec) ([]int, error) {
	cells := make(map[guideCell]int)
	var roots []int
	var distances []float64
	for i, point := range points {
		if regions[i] < 0 {
			continue
		}
		at := point.Scale(1 / spec.Spacing)
		if !(at.Len() < 1<<48) {
			return nil, fmt.Errorf("hair: chart scale too large for guide spacing")
		}
		cell := guideCell{int64(math.Floor(at.X)), int64(math.Floor(at.Y)), int64(math.Floor(at.Z)), regions[i]}
		grid := Clumps{Seed: spec.Seed + uint64(cell.region)}
		gap := at.Sub(grid.centre(cell.x, cell.y, cell.z)).Len()
		slot, found := cells[cell]
		if !found {
			cells[cell] = len(roots)
			roots, distances = append(roots, i), append(distances, gap)
		} else if gap < distances[slot] {
			roots[slot], distances[slot] = i, gap
		}
	}
	return roots, nil
}

type guideEdge struct {
	to       int
	distance float64
}

func guideEdges(surface Surface, regions []int) [][]guideEdge {
	graph := make([][]guideEdge, len(surface.Points))
	for _, face := range surface.Faces {
		for i, from := range face {
			to := face[(i+1)%len(face)]
			if regions[from] < 0 || regions[from] != regions[to] {
				continue
			}
			distance := surface.Points[to].Sub(surface.Points[from]).Len()
			graph[from] = append(graph[from], guideEdge{to, distance})
			graph[to] = append(graph[to], guideEdge{from, distance})
		}
	}
	return graph
}

type guideVisit struct {
	vertex, anchor int
	distance       float64
}

type guideQueue []guideVisit

func (queue guideQueue) Len() int      { return len(queue) }
func (queue guideQueue) Swap(i, j int) { queue[i], queue[j] = queue[j], queue[i] }
func (queue guideQueue) Less(i, j int) bool {
	left, right := queue[i], queue[j]
	if left.distance != right.distance {
		return left.distance < right.distance
	}
	return left.anchor < right.anchor
}
func (queue *guideQueue) Push(value any) { *queue = append(*queue, value.(guideVisit)) }
func (queue *guideQueue) Pop() any {
	last := len(*queue) - 1
	value := (*queue)[last]
	*queue = (*queue)[:last]
	return value
}

// guideDistances visits each vertex once per accepted guide. After the
// nearest four labels fill a vertex, farther labels cannot improve a
// shortest route through that vertex. The compact weight falls to zero
// at the support radius. Each reached vertex keeps at least its nearest guide.
func guideDistances(graph [][]guideEdge, front guideQueue, spacing float64) [][guidesPerVertex]GuideWeight {
	weights := make([][guidesPerVertex]GuideWeight, len(graph))
	counts := make([]int, len(graph))
	heap.Init(&front)
	for front.Len() != 0 {
		visit := heap.Pop(&front).(guideVisit)
		vertex, used := visit.vertex, counts[visit.vertex]
		if used == guidesPerVertex || hasGuide(weights[vertex][:used], visit.anchor) {
			continue
		}
		share := 1 - maths.Smooth(.28*spacing, 1.65*spacing, visit.distance)
		if share == 0 {
			if used != 0 {
				continue
			}
			share = 1
		}
		weights[vertex][used] = GuideWeight{visit.anchor, share}
		counts[vertex]++
		for _, edge := range graph[vertex] {
			if counts[edge.to] < guidesPerVertex {
				heap.Push(&front, guideVisit{edge.to, visit.anchor, visit.distance + edge.distance})
			}
		}
	}
	for vertex, items := range weights {
		total := 0.
		for _, item := range items {
			total += item.Share
		}
		if total > 0 {
			for i := range items {
				weights[vertex][i].Share /= total
			}
		}
	}
	return weights
}

func hasGuide(choices []GuideWeight, anchor int) bool {
	for _, choice := range choices {
		if choice.Anchor == anchor {
			return true
		}
	}
	return false
}
