// Package hair grows geometry from an indexed rest surface. The same
// follicle stays on its triangle when that surface is posed.
package hair

import (
	"fmt"
	"math"
	"slices"

	"github.com/janearc/slayer/manufacture/geometry"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
)

// Surface is an outward-oriented triangular skin in the bioform's local
// space. Its topology and vertex order are fixed for the life of a groom.
type Surface struct {
	Points []maths.Vec
	// Chart optionally gives authoring coordinates for each rest point.
	Chart []maths.Vec
	Faces [][3]int
}

// Site locates one candidate follicle on the rest skin. Point and Normal
// are in local coordinates. Weights are ordered by the source face's corners.
type Site struct {
	Point, Normal maths.Vec
	// Chart is the mapped authoring point. It defaults to Point.
	Chart maths.Vec
	// ChartNormal is the outward normal in the chart, or Normal without a chart.
	ChartNormal maths.Vec
	Face        int
	Weights     [3]float64
	chart       chartFrame
}

// Strand describes the shaft at a site. Length and Radius are in metres.
// RootLift and TipLift are elevations in radians relative to the local
// tangent plane. Sweep is the signed change in heading along the shaft,
// and Wave adds one period of alternating heading, both in radians.
type Strand struct {
	Coverage          float64 // Fraction of candidate roots to keep, from zero to one.
	Flow              maths.Vec
	Length, Radius    float64
	RootLift, TipLift float64
	Sweep, Wave       float64
	Guide             Guide
	// GuideEnd stops a cubic at this fraction of its path parameter.
	// Zero uses the whole path.
	GuideEnd float64
	Pins     GuidePins
	Course   SurfaceCourse
	Taper    ShaftTaper
	// SkinFollow blends the shaft centre toward its authored lift above each nearby face.
	SkinFollow float64
}

// Layer gives one population of fibres a material and a rest-space map.
// Density is the nominal root count per square metre before Coverage.
type Layer struct {
	Seed            uint64
	Density         float64
	Segments, Sides int
	Material        *thing.Material
	Profile         func(Site) Strand
	Units           FollicleUnits
	// FaceFilter may reject a complete candidate triangle using its chart corners.
	FaceFilter func([3]maths.Vec) bool
	// SkinGap optionally keeps curved shaft samples this far above the posed skin.
	SkinGap float64
}

// Groom retains the follicles and topology of a supplied rest skin.
type Groom struct {
	pointCount int
	faces      [][3]int
	layers     []growth
}

type growth struct {
	segments, sides int
	skinGap         float64
	needsSkin       bool
	material        thing.Material
	roots           []follicle
}

type follicle struct {
	paint   maths.Vec
	face    int
	weights [3]float64
	strand  Strand
	flow    [2]float64
}

const (
	maxCandidates = 2_000_000
	maxVertices   = 4_000_000
	leastVector   = 1e-12
)

// New seeds follicles by rest-surface area. A region may blend into bare
// skin by reducing its coverage. Profile is evaluated only at construction.
func New(surface Surface, layers []Layer) (*Groom, error) {
	faces := slices.Clone(surface.Faces)
	if err := checkSurface(surface.Points, faces); err != nil {
		return nil, err
	}
	if err := surface.checkChart(); err != nil {
		return nil, err
	}
	normals := geometry.UnitNormals(surface.Points, faces)
	var chartNormals []maths.Vec
	if surface.Chart != nil {
		chartNormals = geometry.UnitNormals(surface.Chart, faces)
	}
	made := &Groom{pointCount: len(surface.Points), faces: faces}
	candidates, vertices := 0, 0
	for number, layer := range layers {
		if err := checkLayer(layer); err != nil {
			return nil, fmt.Errorf("hair layer %d: %w", number, err)
		}
		group := growth{segments: layer.Segments, sides: layer.Sides, skinGap: layer.SkinGap, needsSkin: layer.SkinGap > 0, material: *layer.Material}
		members := max(1, layer.Units.Size)
		for index, face := range faces {
			if layer.FaceFilter != nil && !layer.FaceFilter(surface.chartCorners(face)) {
				continue
			}
			basis := edges(surface.Points, face)
			expected := basis.area().Len() * layer.Density / (2 * float64(members))
			if expected > float64((maxCandidates-candidates)/members) {
				return nil, fmt.Errorf("hair: more than %d candidate roots", maxCandidates)
			}
			count := int(expected)
			if noise.Hash3(layer.Seed, int64(index), -1, 0) < expected-float64(count) {
				count++
			}
			candidates += count * members
			for sample := range count * members {
				center := weights(layer.Seed, index, sample/members)
				root := layer.Units.member(center, basis, layer.Seed, index, sample, sample%members)
				site := locate(surface.Points, normals, index, face, root, basis)
				site = surface.chartSite(site, face, basis, chartNormals)
				strand := layer.Profile(site)
				if !fraction(strand.Coverage) {
					return nil, fmt.Errorf("hair: invalid coverage on face %d", index)
				}
				if noise.Hash3(layer.Seed, int64(index), int64(sample), 3) >= strand.Coverage {
					continue
				}
				if err := checkStrand(strand, faces); err != nil {
					return nil, fmt.Errorf("hair face %d: %w", index, err)
				}
				if !strand.Course.valid(faces, layer.Segments) {
					return nil, fmt.Errorf("hair face %d: course needs one attachment per shaft segment", index)
				}
				vertices += 2 + (layer.Segments-1)*layer.Sides
				if vertices > maxVertices {
					return nil, fmt.Errorf("hair: more than %d fibre vertices", maxVertices)
				}
				group.roots = append(group.roots, follicle{site.Chart, site.Face, site.Weights, strand, basis.coordinates(strand.Flow)})
				group.needsSkin = group.needsSkin || strand.SkinFollow > 0
			}
		}
		if len(group.roots) != 0 {
			made.layers = append(made.layers, group)
		}
	}
	return made, nil
}

// At builds native shaft meshes from skin points in the original vertex
// order. It reads each root at the original barycentric weights and carries
// the rest heading along the posed edges of the same triangle. The shaft
// follows this local frame. Pins also read their own posed faces when
// present. Distances are in metres, and texture coordinates retain the
// root's chart point at every shaft vertex.
func (made *Groom) At(points []maths.Vec) ([]thing.Thing, error) {
	if len(points) != made.pointCount {
		return nil, fmt.Errorf("hair: got %d posed points, need %d", len(points), made.pointCount)
	}
	if err := checkSurface(points, made.faces); err != nil {
		return nil, err
	}
	normals := geometry.UnitNormals(points, made.faces)
	parts := make([]thing.Thing, 0, len(made.layers))
	var skin *mesh.Mesh
	for _, group := range made.layers {
		if group.needsSkin && skin == nil {
			var err error
			skin, err = mesh.New(points, made.faces, nil, false)
			if err != nil {
				return nil, fmt.Errorf("hair: posed clearance skin: %w", err)
			}
		}
		part, err := group.at(points, normals, made.faces, skin)
		if err != nil {
			return nil, err
		}
		parts = append(parts, part)
	}
	return parts, nil
}

func checkSurface(points []maths.Vec, faces [][3]int) error {
	if len(faces) == 0 {
		return fmt.Errorf("hair: skin has no faces")
	}
	for index, point := range points {
		if !finite(point.Len()) {
			return fmt.Errorf("hair: skin point %d is not finite", index)
		}
	}
	for index, face := range faces {
		for _, corner := range face {
			if corner < 0 || corner >= len(points) {
				return fmt.Errorf("hair: face %d names missing point %d", index, corner)
			}
		}
		area := edges(points, face).area().Len()
		if !(area > 0 && finite(area)) {
			return fmt.Errorf("hair: face %d has no finite area", index)
		}
	}
	return nil
}

func checkLayer(layer Layer) error {
	if layer.Profile == nil || layer.Material == nil {
		return fmt.Errorf("profile and material are required")
	}
	if !finite(layer.Density) || layer.Density < 0 {
		return fmt.Errorf("invalid density")
	}
	if !layer.Units.valid() {
		return fmt.Errorf("follicle units require 2 to 4 members and a positive spread")
	}
	if !finite(layer.SkinGap) || layer.SkinGap < 0 {
		return fmt.Errorf("invalid clearance")
	}
	if layer.Segments < 2 || layer.Segments > 64 || layer.Sides < 3 || layer.Sides > 12 {
		return fmt.Errorf("need 2 to 64 segments and 3 to 12 sides")
	}
	return nil
}

func checkStrand(strand Strand, faces [][3]int) error {
	if !finite(strand.Length) || !finite(strand.Radius) ||
		!(strand.Length > 0 && strand.Radius > 0 && strand.Radius < strand.Length/2) {
		return fmt.Errorf("invalid strand length or radius")
	}
	if !finite(strand.Flow.Len()) ||
		!(strand.RootLift >= 0 && strand.RootLift <= math.Pi/2) ||
		!(strand.TipLift >= -math.Pi/2 && strand.TipLift <= math.Pi/2) ||
		!(math.Abs(strand.Sweep) <= 16*math.Pi) || !(math.Abs(strand.Wave) <= 16*math.Pi) {
		return fmt.Errorf("invalid strand direction")
	}
	if !strand.Guide.valid() {
		return fmt.Errorf("invalid strand guide")
	}
	if strand.GuideEnd != 0 &&
		(!strand.Guide.active() || !(strand.GuideEnd > 0 && strand.GuideEnd <= 1)) {
		return fmt.Errorf("guide end requires a cubic and a fraction above zero")
	}
	if strand.Pins != (GuidePins{}) &&
		(!strand.Guide.active() || !strand.Pins.Lead.valid(faces) || !strand.Pins.Bend.valid(faces)) {
		return fmt.Errorf("guide pins require a cubic and matching skin faces")
	}
	if !strand.Taper.valid() {
		return fmt.Errorf("invalid shaft taper")
	}
	if !fraction(strand.SkinFollow) {
		return fmt.Errorf("invalid skin following share")
	}
	return nil
}

func finite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }

func fraction(value float64) bool { return value >= 0 && value <= 1 }

// weights reflects a square sample back into the unit triangle. This is
// the same area sampling used by the harbor seal's groom.
func weights(seed uint64, face, sample int) [3]float64 {
	u := noise.Hash3(seed, int64(face), int64(sample), 1)
	v := noise.Hash3(seed, int64(face), int64(sample), 2)
	if u+v > 1 {
		u, v = 1-u, 1-v
	}
	return [3]float64{1 - u - v, u, v}
}

func locate(points, normals []maths.Vec, index int, face [3]int,
	w [3]float64, basis triangle,
) Site {
	point := points[face[0]].Add(basis.first.Scale(w[1])).Add(basis.second.Scale(w[2]))
	normal := basis.normal(normals, face, w)
	return Site{Point: point, Normal: normal, Chart: point, ChartNormal: normal, Face: index, Weights: w}
}

func (basis triangle) normal(normals []maths.Vec, face [3]int, w [3]float64) maths.Vec {
	normal := normals[face[0]].Scale(w[0]).Add(normals[face[1]].Scale(w[1])).
		Add(normals[face[2]].Scale(w[2])).Unit()
	outward := basis.area().Unit()
	if normal.Dot(outward) <= 0 {
		normal = outward
	}
	return normal
}

type triangle struct{ first, second maths.Vec }

func edges(points []maths.Vec, face [3]int) triangle {
	base := points[face[0]]
	return triangle{points[face[1]].Sub(base), points[face[2]].Sub(base)}
}

func (basis triangle) area() maths.Vec { return basis.first.Cross(basis.second) }

// coordinates writes the in-plane part of a flow vector in the triangle's
// reciprocal basis, so it can be read along the posed edges.
func (basis triangle) coordinates(flow maths.Vec) [2]float64 {
	area := basis.area()
	normal := area.Unit()
	magnitude := area.Len()
	result := [2]float64{flow.Cross(basis.second).Dot(normal) / magnitude,
		basis.first.Cross(flow).Dot(normal) / magnitude}
	if !finite(result[0]) || !finite(result[1]) ||
		basis.flow(result).Len() < leastVector {
		return [2]float64{1, 0}
	}
	return result
}

func (basis triangle) flow(coords [2]float64) maths.Vec {
	return basis.first.Scale(coords[0]).Add(basis.second.Scale(coords[1]))
}
