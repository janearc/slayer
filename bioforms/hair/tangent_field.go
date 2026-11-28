package hair

import (
	"fmt"
	"slices"

	"github.com/janearc/slayer/manufacture/geometry"
	"github.com/janearc/slayer/maths"
)

// TangentFieldSpec supplies rest-chart domains and directional
// constraints. Negative domain indices exclude vertices. Constraint
// returns a chart vector and an influence from zero to one. A full
// constraint fixes the vector, including a deliberate zero.
type TangentFieldSpec struct {
	Surface Surface
	Domain  func(maths.Vec) int
	// Barrier reports whether the segment between chart points crosses
	// a part or another finite seam. It must be symmetric in its inputs.
	Barrier    func(from, to maths.Vec) bool
	Constraint func(point, normal maths.Vec) (direction maths.Vec, weight float64)
}

// TangentField retains a piecewise-interpolated direction field on
// the authored skin. Its graph excludes edges that cross domains.
type TangentField struct {
	pins                      *PinSurface
	points, normals, headings []maths.Vec
	rest                      []maths.Vec
	faces                     [][3]int
	domains                   []int
	domain                    func(maths.Vec) int
	barrier                   func(from, to maths.Vec) bool
}

const weakTangent = .065

// NewTangentField smooths constraints over the chart's connected edges
// while transporting vectors between adjoining tangent planes.
func NewTangentField(spec TangentFieldSpec) (*TangentField, error) {
	if spec.Domain == nil || spec.Constraint == nil {
		return nil, fmt.Errorf("hair: tangent fields require domains and constraints")
	}
	surface := spec.Surface
	pins, err := NewPinSurface(surface, func(corners [3]maths.Vec) bool {
		return spec.Domain(corners[0]) >= 0 || spec.Domain(corners[1]) >= 0 || spec.Domain(corners[2]) >= 0
	})
	if err != nil {
		return nil, err
	}
	if pins.chart == nil {
		return nil, fmt.Errorf("hair: tangent field has no eligible faces")
	}
	points := surface.Chart
	if points == nil {
		points = surface.Points
	}
	faces := slices.Clone(surface.Faces)
	field := &TangentField{
		pins: pins, points: slices.Clone(points), rest: slices.Clone(surface.Points), faces: faces,
		normals: geometry.UnitNormals(points, faces), headings: make([]maths.Vec, len(points)),
		domains: make([]int, len(points)), domain: spec.Domain, barrier: spec.Barrier,
	}
	authored := make([]maths.Vec, len(points))
	weights := make([]float64, len(points))
	for i, point := range field.points {
		field.domains[i] = field.domain(point)
		if field.domains[i] < 0 {
			continue
		}
		direction, weight := spec.Constraint(point, field.normals[i])
		if !finite(direction.Len()) || !fraction(weight) {
			return nil, fmt.Errorf("hair: invalid tangent constraint at vertex %d", i)
		}
		authored[i] = tangentTo(direction, field.normals[i]).Unit()
		weights[i] = weight
		field.headings[i] = authored[i]
	}
	graph := guideEdges(surface, field.domains)
	if field.barrier != nil {
		for i, neighbours := range graph {
			graph[i] = slices.DeleteFunc(neighbours, func(edge guideEdge) bool {
				return field.blocked(field.points[i], field.points[edge.to])
			})
		}
	}
	field.smooth(graph, authored, weights)
	return field, nil
}

func (field *TangentField) smooth(graph [][]guideEdge, authored []maths.Vec, weights []float64) {
	next := make([]maths.Vec, len(field.headings))
	for range 28 {
		for i, neighbours := range graph {
			next[i] = authored[i]
			if field.domains[i] < 0 || weights[i] == 1 || len(neighbours) == 0 {
				continue
			}
			up := field.normals[i]
			var sum maths.Vec
			total := 0.
			for _, edge := range neighbours {
				weight := 1 / max(leastVector, edge.distance)
				direction := transport(field.headings[edge.to], field.normals[edge.to], up)
				sum = sum.Add(tangentTo(direction, up).Scale(weight))
				total += weight
			}
			next[i] = authored[i].Scale(weights[i]).Add(sum.Scale((1 - weights[i]) / total))
		}
		field.headings, next = next, field.headings
	}
}

func tangentTo(direction, normal maths.Vec) maths.Vec {
	return direction.Sub(normal.Scale(direction.Dot(normal)))
}

func (field *TangentField) blocked(from, to maths.Vec) bool {
	return field.barrier != nil && field.barrier(from, to)
}

// WeakVertices identifies retained chart vertices with little or no
// resolved direction. This includes authored singularities and
// cancellation between competing constraints.
func (field *TangentField) WeakVertices() []int {
	var vertices []int
	for i, heading := range field.headings {
		if field.domains[i] >= 0 && heading.Len() < weakTangent {
			vertices = append(vertices, i)
		}
	}
	return vertices
}

// Heading reads the chart tangent at a follicle. The vector's length
// indicates how strongly the neighbouring constraints agree.
func (field *TangentField) Heading(site Site) maths.Vec {
	if site.Face < 0 || site.Face >= len(field.faces) {
		return maths.Vec{}
	}
	return field.direction(site, field.domain(site.Chart))
}

func (field *TangentField) direction(site Site, domain int) maths.Vec {
	var sum maths.Vec
	total := 0.
	up := site.ChartNormal
	for i, vertex := range field.faces[site.Face] {
		if field.domains[vertex] != domain || field.blocked(site.Chart, field.points[vertex]) {
			continue
		}
		direction := transport(field.headings[vertex], field.normals[vertex], up)
		sum = sum.Add(tangentTo(direction, up).Scale(site.Weights[i]))
		total += site.Weights[i]
	}
	if total <= leastVector {
		return maths.Vec{}
	}
	return sum.Scale(1 / total)
}

func (field *TangentField) project(point maths.Vec, domain int, maxGap float64) (Site, bool) {
	near, ok := field.pins.chart.SurfaceNear(point)
	if !ok || field.domain(near.Point) != domain || near.Point.Sub(point).Len() > maxGap {
		return Site{}, false
	}
	index := field.pins.indices[near.Face]
	face := field.faces[index]
	return locate(field.points, field.normals, index, face, near.Weights, edges(field.points, face)), true
}
