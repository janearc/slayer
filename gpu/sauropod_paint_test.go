package gpu

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/manufacture/animals/vertebrates/dinosaurs/sauropods"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
)

// The actual four profile constructions must encode with separate local-space
// paints. Each body is a closed, unposed mesh.
func TestSauropodPaintSceneEncodingKeepsFourSpecies(t *testing.T) {
	file, raw := sauropods.Embedded()
	book, err := sauropods.Read(file, raw)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"alamosaurus-sanjuanensis", "apatosaurus-louisae", "brachiosaurus-altithorax", "diplodocus-carnegii"}
	if len(book.Species()) != len(want) {
		t.Fatalf("got species %v", book.Species())
	}
	type sample struct {
		profile sauropods.Profile
		body    thing.Thing
	}
	rows := make([]sample, 0, len(want))
	world := thing.NewWorld()
	for _, name := range want {
		p, err := book.Profile(name)
		if err != nil {
			t.Fatal(err)
		}
		made, err := sauropods.Build(p)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if len(made.Parts) != 3 {
			t.Fatalf("%s: expected body and eyes", name)
		}
		body := made.Parts[0]
		skin, ok := body.Shape.(*mesh.Mesh)
		if !ok || !skin.Closed() || skin.Posed() || body.Material == nil || body.Material.Texture == nil {
			t.Fatalf("%s: expected closed unposed painted body", name)
		}
		for _, part := range made.Parts {
			world.Add(part)
		}
		rows = append(rows, sample{p, body})
	}
	encoded := newScene()
	if err := encoded.things(world, 0); err != nil {
		t.Fatalf("static sauropod encoding: %v", err)
	}
	indices := map[int]bool{}
	colors := map[maths.Light]bool{}
	for _, row := range rows {
		index, ok := encoded.materialOf[row.body.Material]
		if !ok || indices[index] {
			t.Fatalf("%s: missing or collapsed material", row.profile.Kind)
		}
		indices[index] = true
		paint := encoded.paints[index].paint
		if paint == nil {
			t.Fatalf("%s: missing paint", row.profile.Kind)
		}
		s := row.profile.Stations[row.profile.Hind.Station]
		point := maths.V(s.X, s.Y+s.Height, 0)
		got := paint.At(point)
		if got != row.body.Material.Texture.At(point) || got != row.profile.Skin || colors[got] {
			t.Fatalf("%s: encoded skin %v is lost or not distinct", row.profile.Kind, got)
		}
		colors[got] = true
	}
}

// A posed copy must still be refused, even when its rest points are equal.
func TestSauropodPaintSceneRefusesRestPointMesh(t *testing.T) {
	file, raw := sauropods.Embedded()
	book, err := sauropods.Read(file, raw)
	if err != nil {
		t.Fatal(err)
	}
	p, err := book.Profile("alamosaurus-sanjuanensis")
	if err != nil {
		t.Fatal(err)
	}
	made, err := sauropods.Build(p)
	if err != nil {
		t.Fatal(err)
	}
	body := made.Parts[0]
	skin := body.Shape.(*mesh.Mesh)
	points := skin.Points()
	posed, err := mesh.NewWithRestPoints(points, points, skin.Faces(), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	body.Shape = posed
	world := thing.NewWorld()
	world.Add(body)
	err = newScene().things(world, 0)
	if err == nil || !strings.Contains(err.Error(), "CPU-only") {
		t.Fatalf("posed sauropod paint accepted: %v", err)
	}
}
