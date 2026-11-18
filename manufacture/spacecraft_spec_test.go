package manufacture_test

import (
	"math"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

func TestComponentSpacecraftComposesAtEveryDetail(test *testing.T) {
	kind, err := manufacture.SpacecraftFrom(componentCraft())
	if err != nil {
		test.Fatal(err)
	}
	want := maths.Box{Min: maths.V(-5, 0, -6), Max: maths.V(5, 8, 6)}
	for _, detail := range manufacture.Complexities() {
		test.Run(detail, func(test *testing.T) {
			made, err := manufacture.MakeWithComplexity(*kind, manufacture.Sow(27), detail)
			if err != nil {
				test.Fatal(err)
			}
			if got := made.Bounds(); got != want {
				test.Fatalf("custom spacecraft bounds = %+v, want %+v", got, want)
			}
			things, err := made.Things(maths.Local())
			if err != nil {
				test.Fatal(err)
			}
			(meshContract{Tolerance: 1e-8, MaximumFaces: 10000}).assertMeshes(test, things, want)
			materials := map[string]bool{}
			for _, thing := range things {
				materials[thing.Material.Name] = true
			}
			for _, name := range []string{"courier structure", "courier mirror", "courier solar"} {
				if !materials[name] {
					test.Errorf("custom spacecraft lacks material %q", name)
				}
			}
		})
	}
}

func TestComponentSpacecraftCopiesPartsAndProfiles(test *testing.T) {
	spec := componentCraft()
	spec.Materials[0].Look = &thing.Material{Name: "owned top-level value", Colour: maths.RGB(.2, .3, .4)}
	kind, err := manufacture.SpacecraftFrom(spec)
	if err != nil {
		test.Fatal(err)
	}
	before, err := manufacture.MakeWithComplexity(*kind, manufacture.Sow(27), "quick")
	if err != nil {
		test.Fatal(err)
	}
	beforeThings, err := before.Things(maths.Local())
	if err != nil {
		test.Fatal(err)
	}
	want := buildingGeometryDigest(test, beforeThings)
	spec.Materials[0].Name = "mutated structure"
	spec.Materials[0].Look.Name = "mutated look name"
	spec.Materials[0].Look.Colour = maths.RGB(.9, .8, .7)
	spec.Parts[1].Profile[0].Radius = measure.Metres(100)
	spec.Parts[1].Frame.Origin = maths.V(100, 100, 100)
	after, err := manufacture.MakeWithComplexity(*kind, manufacture.Sow(27), "quick")
	if err != nil {
		test.Fatal(err)
	}
	afterThings, err := after.Things(maths.Local())
	if err != nil {
		test.Fatal(err)
	}
	if got := buildingGeometryDigest(test, afterThings); got != want {
		test.Fatalf("mutating caller's spacecraft spec changed recipe: got %s want %s", got, want)
	}
	if got := afterThings[0].Material.Name; got != "owned top-level value" {
		test.Fatalf("mutating caller's top-level Look changed recipe material name: %q", got)
	}
	if got := afterThings[0].Material.Colour; got != maths.RGB(.2, .3, .4) {
		test.Fatalf("mutating caller's top-level Look changed recipe colour: %v", got)
	}
}

func TestComponentSpacecraftRejectsInvalidSpecs(test *testing.T) {
	valid := componentCraft()
	tests := []struct {
		name   string
		change func(*manufacture.SpacecraftSpec)
	}{
		{"missing name", func(spec *manufacture.SpacecraftSpec) { spec.Name = "" }},
		{"bad envelope", func(spec *manufacture.SpacecraftSpec) { spec.Width = measure.Metres(math.NaN()) }},
		{"unknown form", func(spec *manufacture.SpacecraftSpec) { spec.Parts[0].Form = "warp-core" }},
		{"bad profile", func(spec *manufacture.SpacecraftSpec) { spec.Parts[1].Profile[0].Radius = measure.Metres(-1) }},
		{"invalid frame", func(spec *manufacture.SpacecraftSpec) { spec.Parts[1].Frame.Origin.X = math.Inf(1) }},
		{"missing material", func(spec *manufacture.SpacecraftSpec) { spec.Parts[0].Surface = "unknown" }},
		{"duplicate material role", func(spec *manufacture.SpacecraftSpec) { spec.Materials = append(spec.Materials, spec.Materials[0]) }},
		{"part outside envelope", func(spec *manufacture.SpacecraftSpec) { spec.Parts[0].Position.X = 100 }},
		{"smooth segmented dish", func(spec *manufacture.SpacecraftSpec) { spec.Parts[2].Segmented = true }},
		{"zero rod", func(spec *manufacture.SpacecraftSpec) { spec.Parts[3].To = spec.Parts[3].From }},
	}
	for _, row := range tests {
		test.Run(row.name, func(test *testing.T) {
			spec := valid
			spec.Materials = append([]manufacture.ComponentMaterial(nil), valid.Materials...)
			spec.Parts = cloneComponentParts(valid.Parts)
			row.change(&spec)
			if _, err := manufacture.SpacecraftFrom(spec); err == nil {
				test.Fatal("invalid spacecraft spec was accepted")
			}
		})
	}
}

func componentCraft() manufacture.SpacecraftSpec {
	return manufacture.SpacecraftSpec{
		Name: "courier observatory", Width: measure.Metres(10), Depth: measure.Metres(12), Height: measure.Metres(8),
		Materials: []manufacture.ComponentMaterial{
			{Surface: "structure", Name: "courier structure", Colour: maths.RGB(.42, .45, .48)},
			{Surface: "optics", Name: "courier mirror", Colour: maths.Grey(.8), Gloss: .9, Index: 1.5},
			{Surface: "solar", Name: "courier solar", Colour: maths.RGB(.08, .12, .2)},
		},
		Parts: []manufacture.ComponentSpec{
			{Form: manufacture.ComponentDrum, Position: maths.V(0, 0, 0), Radius: measure.Metres(1.5), Height: measure.Metres(1), Cuts: 8, Surface: "structure"},
			{Form: manufacture.ComponentDrum, Position: maths.V(0, 1, 0), Radius: measure.Metres(1), Height: measure.Metres(2), Smooth: true, Surface: "structure",
				Profile: []manufacture.ComponentProfilePoint{
					{Radius: measure.Metres(1), Height: measure.Metres(0)},
					{Radius: measure.Metres(1), Height: measure.Metres(2)},
					{Radius: measure.Metres(.9), Height: measure.Metres(2)},
					{Radius: measure.Metres(.9), Height: measure.Metres(0)},
				}},
			{Form: manufacture.ComponentDish, Position: maths.V(0, 3, 0), Radius: measure.Metres(.9), Depth: measure.Metres(.04), Smooth: true, Surface: "optics"},
			{Form: manufacture.ComponentRod, From: maths.V(-1, 2, 0), To: maths.V(-2.5, 2, 0), Radius: measure.Metres(.06), Surface: "structure"},
			{Form: manufacture.ComponentRod, From: maths.V(1, 2, 0), To: maths.V(2.5, 2, 0), Radius: measure.Metres(.06), Surface: "structure"},
			{Form: manufacture.ComponentBox, Position: maths.V(-5, 1, -.5), Width: measure.Metres(2), Height: measure.Metres(.1), Depth: measure.Metres(1), Surface: "solar"},
			{Form: manufacture.ComponentBox, Position: maths.V(3, 1, -.5), Width: measure.Metres(2), Height: measure.Metres(.1), Depth: measure.Metres(1), Surface: "solar"},
		},
	}
}

func cloneComponentParts(parts []manufacture.ComponentSpec) []manufacture.ComponentSpec {
	copyOf := append([]manufacture.ComponentSpec(nil), parts...)
	for index := range copyOf {
		copyOf[index].Profile = append([]manufacture.ComponentProfilePoint(nil), parts[index].Profile...)
	}
	return copyOf
}
