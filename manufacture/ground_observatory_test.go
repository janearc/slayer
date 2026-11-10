package manufacture_test

import (
	_ "embed"
	"math"
	"testing"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
)

//go:embed testdata/ground-observatory.yaml
var groundContractRaw []byte

type groundProfile struct {
	Design                     string
	Diameter, Height, Aperture float64
	Panels                     int
}

type groundContract struct {
	Seed          uint64
	Tolerance     float64
	MaximumFaces  int       `yaml:"maximum-faces"`
	MirrorTangent maths.Vec `yaml:"mirror-tangent"`
	Profiles      []groundProfile
}

func readGroundContract(test *testing.T) groundContract {
	test.Helper()
	var data groundContract
	if err := yamlfile.Read("ground-observatory.yaml", groundContractRaw,
		&data); err != nil {
		test.Fatal(err)
	}
	return data
}

func TestGroundObservatoryDimensionsAndClosedMeshes(test *testing.T) {
	data := readGroundContract(test)
	for _, profile := range data.Profiles {
		profile.checkDetails(test, data)
	}
}

func (profile groundProfile) checkDetails(test *testing.T,
	data groundContract,
) {
	test.Helper()
	row, err := manufacture.KindNamed(profile.Design)
	if err != nil {
		test.Fatal(err)
	}
	for _, detail := range manufacture.Complexities() {
		made, err := manufacture.MakeWithComplexity(row,
			manufacture.Sow(data.Seed), detail)
		if err != nil {
			test.Fatal(profile.Design, detail, err)
		}
		bounds := made.Bounds()
		if bounds.Max.X-bounds.Min.X != profile.Diameter ||
			bounds.Max.Y != profile.Height || bounds.Min.Y != spacecraftNear {
			test.Fatal("wrong physical support", profile.Design, detail, bounds)
		}
		parts, err := made.Things(maths.Local())
		if err != nil {
			test.Fatal(profile.Design, detail, err)
		}
		limits := meshContract{Tolerance: data.Tolerance,
			MaximumFaces: data.MaximumFaces}
		limits.assertMeshes(test, parts, bounds)
		profile.checkPrimary(test, data, parts)
	}
}

func (profile groundProfile) checkPrimary(test *testing.T, data groundContract,
	parts []thing.Thing,
) {
	test.Helper()
	for _, part := range parts {
		if part.Material.Name != "primary mirror" {
			continue
		}
		polygon := part.Shape.(*mesh.Mesh)
		low := math.Inf(spacecraftPositiveSign)
		high := math.Inf(-spacecraftPositiveSign)
		for _, point := range polygon.Points() {
			along := point.Dot(data.MirrorTangent)
			low, high = min(low, along), max(high, along)
		}
		if math.Abs(high-low-profile.Aperture) > data.Tolerance {
			test.Fatal("wrong primary aperture", profile.Design, high-low)
		}
		if pieces := connectedPanels(polygon); pieces != profile.Panels {
			test.Fatal("wrong mirror panel count", profile.Design, pieces)
		}
		return
	}
	test.Fatal("missing primary mirror", profile.Design)
}

// connectedPanels counts welded connected components, independently of how
// a panel was triangulated. It does not assume a fixed face count per tile.
func connectedPanels(polygon *mesh.Mesh) int {
	welded := map[maths.Vec]int{}
	points := polygon.Points()
	canonical := make([]int, len(points))
	for index, point := range points {
		key, found := welded[point]
		if !found {
			key = len(welded)
			welded[point] = key
		}
		canonical[index] = key
	}
	neighbours := map[int][]int{}
	for _, face := range polygon.Faces() {
		for index, point := range face {
			face[index] = canonical[point]
		}
		for _, here := range face {
			neighbours[here] = append(neighbours[here], face[:]...)
		}
	}
	seen := map[int]bool{}
	components := spacecraftNear
	for root := range neighbours {
		if seen[root] {
			continue
		}
		components++
		pending := []int{root}
		for len(pending) > spacecraftNear {
			here := pending[spacecraftNear]
			pending = pending[spacecraftPositiveSign:]
			if seen[here] {
				continue
			}
			seen[here] = true
			pending = append(pending, neighbours[here]...)
		}
	}
	return components
}

func TestGroundObservatoryRefusalsExposeNoThing(test *testing.T) {
	if _, err := manufacture.KindNamed("missing observatory"); err == nil {
		test.Fatal("unknown kind exposed a construction")
	}
	data := readGroundContract(test)
	for _, profile := range data.Profiles {
		row, err := manufacture.KindNamed(profile.Design)
		if err != nil {
			test.Fatal(err)
		}
		from := unsown(manufacture.Sow(data.Seed))
		made, err := manufacture.MakeWithComplexity(row, from, "quick")
		if err == nil {
			test.Fatal("unsown seed made a ground observatory")
		}
		if parts, err := made.Things(maths.Local()); err == nil ||
			len(parts) != spacecraftNear {
			test.Fatal("refused construction exposed geometry", parts, err)
		}
	}
}
