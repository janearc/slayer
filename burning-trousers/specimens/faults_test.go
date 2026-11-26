package specimens

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/janearc/slayer/manufacture"
)

var interrupted = errors.New("injected book failure")

type failingPlant struct {
	plantBook
	fail func(manufacture.PlantOptions) bool
}

func (book failingPlant) New(options manufacture.PlantOptions) (manufacture.Kind, error) {
	if book.fail(options) {
		return manufacture.Kind{}, interrupted
	}
	return book.plantBook.New(options)
}

type failingAnimal struct {
	animalBook
	fail func(manufacture.AnimalOptions) bool
}

type failingAlga struct{ algaBook }

func (book failingAlga) NewKelp(manufacture.KelpOptions) (manufacture.Kind, error) {
	return manufacture.Kind{}, interrupted
}

func (book failingAnimal) New(options manufacture.AnimalOptions) (manufacture.Kind, error) {
	if book.fail(options) {
		return manufacture.Kind{}, interrupted
	}
	return book.animalBook.New(options)
}

// The inventory must not silently omit a specimen when its book or checked
// constructor cannot be read. Break the boundary at each distinct kind of
// input and require an error instead of a partial inventory.
func TestBrokenProfileSourcesAreRefused(t *testing.T) {
	cases := []struct {
		name   string
		change func(*sources)
		want   string
	}{
		{"plant source", func(s *sources) {
			s.plant = func(string) (string, []byte, error) { return "", nil, interrupted }
		}, "injected book failure"},
		{"invalid plant book", func(s *sources) {
			s.plant = func(string) (string, []byte, error) { return "bad-plant.yaml", []byte("not: [valid"), nil }
		}, "bad-plant.yaml"},
		{"animal source", func(s *sources) {
			s.animal = func(string) (string, []byte, error) { return "", nil, interrupted }
		}, "injected book failure"},
		{"invalid animal book", func(s *sources) {
			s.animal = func(string) (string, []byte, error) { return "bad-animal.yaml", []byte("not: [valid"), nil }
		}, "bad-animal.yaml"},
		{"alga source", func(s *sources) {
			s.alga = func(string) (string, []byte, error) { return "", nil, interrupted }
		}, "injected book failure"},
		{"invalid alga book", func(s *sources) {
			s.alga = func(string) (string, []byte, error) { return "bad-alga.yaml", []byte("not: [valid"), nil }
		}, "bad-alga.yaml"},
		{"invalid alga metadata", func(s *sources) {
			s.decodeAlga = func([]byte, *algaDocument) error { return interrupted }
		}, "kelp.yaml"},
		{"invalid inventory metadata", func(s *sources) {
			s.decode = func([]byte, *animalDocument) error { return interrupted }
		}, "fish.yaml"},
		{"missing named school", func(s *sources) {
			decode := s.decode
			s.decode = func(raw []byte, document *animalDocument) error {
				if err := decode(raw, document); err != nil {
					return err
				}
				delete(document.Species, "pacific-sardine")
				return nil
			}
		}, "missing school species pacific-sardine"},
		{"unknown pinniped form", func(s *sources) {
			decode := s.decode
			s.decode = func(raw []byte, document *animalDocument) error {
				if err := decode(raw, document); err != nil {
					return err
				}
				if len(document.Profiles) != 0 {
					document.Profiles[0].Form = "unknown"
				}
				return nil
			}
		}, "unknown pinniped form"},
		{"unknown reptile scaffold", func(s *sources) {
			decode := s.decode
			s.decode = func(raw []byte, document *animalDocument) error {
				if err := decode(raw, document); err != nil {
					return err
				}
				for i := range document.Profiles {
					document.Profiles[i].Scaffold = "unknown"
				}
				return nil
			}
		}, "missing reptile studies for unknown"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := embeddedSources()
			tc.change(&input)
			got, err := profiles(input)
			if err == nil || !strings.Contains(err.Error(), tc.want) || got != nil {
				t.Fatalf("got %d specimens, %v; want no partial inventory and error containing %q", len(got), err, tc.want)
			}
		})
	}
}

func TestFailedAlgaConstructorIsRefused(t *testing.T) {
	input := embeddedSources()
	readAlga := input.readAlga
	input.readAlga = func(file string, raw []byte) (algaBook, error) {
		book, err := readAlga(file, raw)
		return failingAlga{book}, err
	}
	got, err := profiles(input)
	if !errors.Is(err, interrupted) || got != nil {
		t.Fatalf("got %d specimens, %v; want no partial inventory and the constructor error", len(got), err)
	}
}

func TestFailedProfileConstructorsAreRefused(t *testing.T) {
	cases := []struct {
		family string
		plant  func(manufacture.PlantOptions) bool
		animal func(manufacture.AnimalOptions) bool
	}{
		{"seasonal", func(o manufacture.PlantOptions) bool { return o.Species == "alder" }, nil},
		{"conifer", func(o manufacture.PlantOptions) bool { return o.Representation != nil }, nil},
		{"fish", nil, func(o manufacture.AnimalOptions) bool { return o.Pose != nil }},
		{"fish", nil, func(o manufacture.AnimalOptions) bool { return o.Count != nil }},
		{"bird", nil, func(o manufacture.AnimalOptions) bool { return o.Pose != nil }},
		{"pinniped", nil, func(o manufacture.AnimalOptions) bool { return o.Pose != nil }},
		{"theropod", nil, func(manufacture.AnimalOptions) bool { return true }},
		{"whale", nil, func(manufacture.AnimalOptions) bool { return true }},
		{"reptile", nil, func(manufacture.AnimalOptions) bool { return true }},
	}
	for i, tc := range cases {
		t.Run(fmt.Sprintf("%s-%d", tc.family, i), func(t *testing.T) {
			input := embeddedSources()
			readPlant, readAnimal := input.readPlant, input.readAnimal
			input.readPlant = func(family, file string, raw []byte) (plantBook, error) {
				book, err := readPlant(family, file, raw)
				if err != nil || family != tc.family || tc.plant == nil {
					return book, err
				}
				return failingPlant{book, tc.plant}, nil
			}
			input.readAnimal = func(family, file string, raw []byte) (animalBook, error) {
				book, err := readAnimal(family, file, raw)
				if err != nil || family != tc.family || tc.animal == nil {
					return book, err
				}
				return failingAnimal{book, tc.animal}, nil
			}
			got, err := profiles(input)
			if !errors.Is(err, interrupted) || got != nil {
				t.Fatalf("got %d specimens, %v; want no partial inventory and the constructor error", len(got), err)
			}
		})
	}
}
