package set

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
)

func TestSchemaConstructionFailureIsPreserved(t *testing.T) {
	_, invalid := yamlfile.NewSchema(EcosystemSchema, []byte(`{"type":"not-a-type"}`))
	if invalid == nil {
		t.Fatal("invalid schema compiled")
	}
	previous := ecosystemError
	ecosystemError = invalid
	t.Cleanup(func() { ecosystemError = previous })
	raw, err := os.ReadFile("../doc/observatory-ecosystem.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Read("schema-construction.yaml", raw); !errors.Is(err, invalid) {
		t.Fatalf("lost schema construction cause: %v", err)
	}
}

func TestCameraRequiresItsTargetAndRetainsItsLens(t *testing.T) {
	shooting := camera{name: "test", from: "base", pointTo: "target"}
	_, err := (Set{cameras: []camera{shooting}}).Shots(func(name string) (maths.Vec, bool) {
		return maths.Vec{}, name == "base"
	})
	if err == nil || !strings.Contains(err.Error(), `points to "target"`) {
		t.Fatalf("missing target: %v", err)
	}
	lens := film.Pinhole{Wide: .7, Pixel: 1.3}
	if got := (Shot{lens: lens}).Lens(); got != lens {
		t.Fatalf("shot lens changed: %+v", got)
	}
	if (Stand{standsIn: "the measured stand"}).StandsIn() != "the measured stand" {
		t.Fatal("stand identity was lost")
	}
}
