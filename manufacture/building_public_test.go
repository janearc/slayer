package manufacture_test

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
)

// Captured from the pre-building-vocabulary Griffith maker at seed 5, quick
// detail. It hashes material names and every ordered mesh point and face.
const griffithGeometryWitness = "0c6b65c679819a39c219f72e0b640bd67186e96f3a4708fa46c16a86f09cf1d3"

func TestGriffithBuildingVocabularyPreservesIndependentGeometryWitness(test *testing.T) {
	kind, err := manufacture.KindNamed("griffith")
	if err != nil {
		test.Fatal(err)
	}
	object, err := manufacture.MakeWithComplexity(kind, manufacture.Sow(5), "quick")
	if err != nil {
		test.Fatal(err)
	}
	things, err := object.Things(maths.Local())
	if err != nil {
		test.Fatal(err)
	}
	if got := buildingGeometryDigest(test, things); got != griffithGeometryWitness {
		test.Fatalf("Griffith geometry changed: got %s want %s", got, griffithGeometryWitness)
	}
	if len(things) != 4 {
		test.Fatalf("Griffith material groups changed: got %d", len(things))
	}
	wantMaterials := []string{"civic walls", "refractor tube", "copper dome proxy", "Zeiss objective proxy"}
	for index, want := range wantMaterials {
		if things[index].Material.Name != want {
			test.Fatalf("material group %d = %q, want %q", index, things[index].Material.Name, want)
		}
	}
}

func TestBuildingSpecMakesDifferentPavilionFromSharedParts(test *testing.T) {
	kind, err := manufacture.Building(pavilionSpec())
	if err != nil {
		test.Fatal(err)
	}
	wantBounds := maths.Box{Min: maths.V(-6, 0, -5), Max: maths.V(6, 8, 5)}
	for _, detail := range manufacture.Complexities() {
		test.Run(detail, func(test *testing.T) {
			object, err := manufacture.MakeWithComplexity(kind, manufacture.Sow(19), detail)
			if err != nil {
				test.Fatal(err)
			}
			if got := object.Bounds(); got != wantBounds {
				test.Fatalf("pavilion bounds = %+v, want %+v", got, wantBounds)
			}
			things, err := object.Things(maths.Local())
			if err != nil {
				test.Fatal(err)
			}
			materials := map[string]bool{}
			for _, thing := range things {
				materials[thing.Material.Name] = true
			}
			for _, name := range []string{"pavilion stone", "pavilion copper", "pavilion door"} {
				if !materials[name] {
					test.Errorf("pavilion is missing configured material %q", name)
				}
			}
			(meshContract{Tolerance: 1e-8, MaximumFaces: 10000}).assertMeshes(test,
				things, wantBounds)
		})
	}
}

func TestBuildingCopiesSpecSlices(test *testing.T) {
	spec := pavilionSpec()
	kind, err := manufacture.Building(spec)
	if err != nil {
		test.Fatal(err)
	}
	before, err := manufacture.MakeWithComplexity(kind, manufacture.Sow(19), "quick")
	if err != nil {
		test.Fatal(err)
	}
	beforeThings, err := before.Things(maths.Local())
	if err != nil {
		test.Fatal(err)
	}
	want := buildingGeometryDigest(test, beforeThings)
	spec.Materials[0].Name = "mutated wall"
	spec.Materials[0].Colour = maths.RGB(1, 0, 1)
	spec.Parts[0].Position = maths.V(20, 20, 20)
	spec.Parts[0].Width = measure.Metres(1)
	object, err := manufacture.MakeWithComplexity(kind, manufacture.Sow(19), "quick")
	if err != nil {
		test.Fatal(err)
	}
	things, err := object.Things(maths.Local())
	if err != nil {
		test.Fatal(err)
	}
	if got := buildingGeometryDigest(test, things); got != want {
		test.Fatalf("mutating source slices changed recipe geometry: got %s want %s", got, want)
	}
	for _, thing := range things {
		if thing.Material.Name == "mutated wall" {
			test.Fatal("recipe retained the caller's material slice")
		}
	}
	if got, want := object.Bounds(), (maths.Box{Min: maths.V(-6, 0, -5), Max: maths.V(6, 8, 5)}); got != want {
		test.Fatalf("mutating source spec changed bounds: %+v", got)
	}
}

func TestBuildingRejectsInvalidSpecs(test *testing.T) {
	valid := pavilionSpec()
	tests := []struct {
		name   string
		change func(*manufacture.BuildingSpec)
	}{
		{"empty name", func(spec *manufacture.BuildingSpec) { spec.Name = "" }},
		{"nonpositive envelope", func(spec *manufacture.BuildingSpec) { spec.Width = measure.Metres(0) }},
		{"nonfinite envelope", func(spec *manufacture.BuildingSpec) { spec.Height = measure.Metres(math.Inf(1)) }},
		{"unknown form", func(spec *manufacture.BuildingSpec) { spec.Parts[0].Form = "spire" }},
		{"mismatched dimensions", func(spec *manufacture.BuildingSpec) { spec.Parts[0].Radius = measure.Metres(1) }},
		{"outside envelope", func(spec *manufacture.BuildingSpec) { spec.Parts[0].Position.X = -7 }},
		{"missing material role", func(spec *manufacture.BuildingSpec) { spec.Parts[0].Surface = "glass" }},
		{"duplicate role", func(spec *manufacture.BuildingSpec) { spec.Materials = append(spec.Materials, spec.Materials[0]) }},
		{"invalid reflectance", func(spec *manufacture.BuildingSpec) { spec.Materials[0].Colour.R = math.NaN() }},
		{"slit too thick", func(spec *manufacture.BuildingSpec) {
			spec.Parts = append(spec.Parts, manufacture.BuildingPart{Form: manufacture.BuildingSlitDome,
				Position: maths.V(0, 7, 0), Radius: measure.Metres(1), Thickness: measure.Metres(1),
				Opening: measure.Degrees(30), Surface: manufacture.BuildingRoof})
		}},
		{"too many parts", func(spec *manufacture.BuildingSpec) {
			for len(spec.Parts) <= 512 {
				spec.Parts = append(spec.Parts, spec.Parts[0])
			}
		}},
	}
	for _, row := range tests {
		test.Run(row.name, func(test *testing.T) {
			spec := valid
			spec.Materials = append([]manufacture.BuildingMaterial(nil), valid.Materials...)
			spec.Parts = append([]manufacture.BuildingPart(nil), valid.Parts...)
			row.change(&spec)
			if _, err := manufacture.Building(spec); err == nil {
				test.Fatal("invalid building spec was accepted")
			}
		})
	}
}

func pavilionSpec() manufacture.BuildingSpec {
	return manufacture.BuildingSpec{
		Name: "courtyard pavilion", Width: measure.Metres(12), Depth: measure.Metres(10), Height: measure.Metres(8),
		Materials: []manufacture.BuildingMaterial{
			{Surface: manufacture.BuildingWalls, Name: "pavilion stone", Colour: maths.RGB(.55, .52, .45)},
			{Surface: manufacture.BuildingRoof, Name: "pavilion copper", Colour: maths.RGB(.42, .21, .13), Gloss: .2, Index: 1.4},
			{Surface: manufacture.BuildingAccent, Name: "pavilion door", Colour: maths.RGB(.12, .08, .05)},
		},
		Parts: []manufacture.BuildingPart{
			{Form: manufacture.BuildingBox, Position: maths.V(-6, 0, -5), Width: measure.Metres(12), Height: measure.Metres(1), Depth: measure.Metres(10), Surface: manufacture.BuildingWalls},
			{Form: manufacture.BuildingDrum, Position: maths.V(0, 1, 0), Radius: measure.Metres(3), InnerRadius: measure.Metres(2.7), Height: measure.Metres(4), Surface: manufacture.BuildingWalls},
			{Form: manufacture.BuildingDome, Position: maths.V(0, 5, 0), Radius: measure.Metres(3), Height: measure.Metres(2), DomeCutsPerLevel: 4, Surface: manufacture.BuildingRoof},
			{Form: manufacture.BuildingBox, Position: maths.V(-2, 1, -5), Width: measure.Metres(4), Height: measure.Metres(2), Depth: measure.Metres(1), Surface: manufacture.BuildingAccent},
		},
	}
}

func buildingGeometryDigest(test *testing.T, things []thing.Thing) string {
	test.Helper()
	hash := sha256.New()
	var word [8]byte
	write := func(value uint64) {
		binary.LittleEndian.PutUint64(word[:], value)
		_, _ = hash.Write(word[:])
	}
	for _, thing := range things {
		_, _ = hash.Write([]byte(thing.Material.Name))
		_, _ = hash.Write([]byte{0})
		polygon, ok := thing.Shape.(*mesh.Mesh)
		if !ok {
			test.Fatalf("unexpected shape type %T", thing.Shape)
		}
		points, faces := polygon.Points(), polygon.Faces()
		pointCount := uint64(len(points))
		write(pointCount)
		for _, point := range points {
			write(math.Float64bits(point.X))
			write(math.Float64bits(point.Y))
			write(math.Float64bits(point.Z))
		}
		faceCount := uint64(len(faces))
		write(faceCount)
		for _, face := range faces {
			write(uint64(face[0]))
			write(uint64(face[1]))
			write(uint64(face[2]))
		}
	}
	return fmt.Sprintf("%x", hash.Sum(nil))
}
