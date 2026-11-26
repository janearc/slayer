package manufacture

import (
	"errors"
	"testing"

	"github.com/janearc/slayer/manufacture/animals/vertebrates/dinosaurs/sauropods"
)

func TestSauropodAnimalBookSelectsEachProfileAndRejectsOptions(t *testing.T) {
	file, raw, err := EmbeddedAnimalBook("sauropod")
	if err != nil {
		t.Fatal(err)
	}
	family, err := ReadAnimalBook("sauropod", file, raw)
	if err != nil {
		t.Fatal(err)
	}
	profiles, err := sauropods.Read(file, raw)
	if err != nil {
		t.Fatal(err)
	}
	names := profiles.Species()
	if len(names) != 4 {
		t.Fatalf("got %d sauropod profiles, want 4", len(names))
	}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			if _, err := KindNamed(name); err == nil {
				t.Fatal("species became a global kind alias")
			}
			row, err := family.New(AnimalOptions{Species: name})
			if err != nil {
				t.Fatal(err)
			}
			if Deforms(row) {
				t.Fatal("a static sauropod claims time-dependent deformation")
			}
			if _, err := Make(row, seed{}); !errors.Is(err, errGood) {
				t.Fatalf("unsown make: %v", err)
			}
			if _, err := ExtentOf(row, seed{}); !errors.Is(err, errGood) {
				t.Fatalf("unsown extent: %v", err)
			}
			box, err := ExtentOf(row, sow(7))
			if err != nil {
				t.Fatal(err)
			}
			made, err := Make(row, sow(7))
			if err != nil {
				t.Fatal(err)
			}
			if len(made.native) == 0 {
				t.Fatal("sauropod has no parts")
			}
			for _, part := range made.native {
				b := transformed(part.Shape.Bounds(), part.At(0))
				if b.Min.X < box.Min.X || b.Min.Y < box.Min.Y || b.Min.Z < box.Min.Z || b.Max.X > box.Max.X || b.Max.Y > box.Max.Y || b.Max.Z > box.Max.Z {
					t.Fatalf("planned box %v omits shape bounds %v", box, b)
				}
			}
		})
	}
	pose, repr, study, blank := "swim", "skin", "study", ""
	count, seconds := 0, 0.0
	for _, o := range []AnimalOptions{
		{Pose: &pose}, {Pose: &blank}, {Count: &count}, {Seconds: &seconds},
		{Phase: &seconds}, {Representation: &repr}, {Study: &study},
	} {
		o.Species = names[0]
		if _, err := family.New(o); err == nil {
			t.Errorf("accepted unsupported option %+v", o)
		}
	}
	if _, err := family.New(AnimalOptions{Species: "no-such-sauropod"}); err == nil {
		t.Fatal("accepted missing species")
	}
	first := raw[0]
	raw[0] ^= 0xff
	_, again, err := EmbeddedAnimalBook("sauropod")
	if err != nil || again[0] != first {
		t.Fatalf("embedded book is mutable: %v", err)
	}
}

func TestNamedSauropodConstructorsSelectTheirProfiles(t *testing.T) {
	file, raw, err := EmbeddedAnimalBook("sauropod")
	if err != nil {
		t.Fatal(err)
	}
	book, err := ReadSauropodProfiles(file, raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		make func() (Kind, error)
	}{
		{"alamosaurus-sanjuanensis", book.NewAlamosaurus},
		{"diplodocus-carnegii", book.NewDiplodocus},
		{"apatosaurus-louisae", book.NewApatosaurus},
		{"brachiosaurus-altithorax", book.NewBrachiosaurus},
	} {
		row, err := tc.make()
		if err != nil || row.name != tc.name {
			t.Errorf("%s selected %q: %v", tc.name, row.name, err)
		}
	}
	var missing *SauropodProfiles
	if _, err := missing.NewSauropod("alamosaurus-sanjuanensis"); err == nil {
		t.Fatal("nil book accepted")
	}
	if _, err := ReadSauropodProfiles("bad.yaml", []byte("profiles: null\n")); err == nil {
		t.Fatal("malformed book accepted")
	}
}
