package set

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/manufacture/cities"
)

func TestGateHangersConnectBothVoidsToGround(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "gate-composition.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, void := range []int{220, 300} {
		t.Run(fmt.Sprint(void), func(t *testing.T) {
			source := strings.Replace(string(raw), "parameters: {void: 220}", fmt.Sprintf("parameters: {void: %d}", void), 1)
			composition, _, err := readStructureComposition("gate.yaml", structureNode(t, "inline:\n"+indentStructure(source)))
			if err != nil {
				t.Fatal(err)
			}
			parts := make(map[string]cities.CompositionPart, len(composition.Parts))
			for _, part := range composition.Parts {
				parts[part.ID] = part
			}
			lower, upper, tower := parts["lower-deck"], parts["upper-deck"], parts["tower"]
			if lower.Size != [3]float64{2000, 3, 60} || upper.Size != lower.Size || lower.At[1] != 59 || upper.At[1] != 75 {
				t.Fatalf("deck dimensions or separation changed: lower=%+v upper=%+v", lower, upper)
			}
			if tower.At[1] != float64(void) || tower.At[1]+tower.Size[1] != 750 {
				t.Fatalf("tower opening/top = %v/%v", tower.At[1], tower.At[1]+tower.Size[1])
			}
			if len(lower.SupportedBy) != 0 || len(lower.HangsFrom) != 80 || len(upper.SupportedBy) != 4 {
				t.Fatalf("unexpected deck supports: lower=%+v upper=%+v", lower, upper)
			}
			for i, id := range lower.HangsFrom {
				side, z := "north", -31.0
				if i >= 40 {
					side, z = "south", 29.5
				}
				wantID := fmt.Sprintf("hanger-%s-%02d", side, i%40)
				p, ok := parts[id]
				if !ok || id != wantID || p.Type != "body" || p.At != [3]float64{-975 + 50*float64(i%40) - .75, 62, z} || p.Size != [3]float64{1.5, 13, 1.5} || len(p.HangsFrom) != 1 || p.HangsFrom[0] != "upper-deck" {
					t.Fatalf("incorrect hanger %s: %+v", id, p)
				}
			}
			for _, id := range upper.SupportedBy {
				fan := parts[id]
				if fan.Type != "stay-fan" || fan.Deck != "upper-deck" || fan.Tower != "tower" || fan.Count != 8 {
					t.Fatalf("incorrect upper stay %s: %+v", id, fan)
				}
			}
			if len(tower.RestsOn) != 4 {
				t.Fatalf("tower supports: %v", tower.RestsOn)
			}
			for _, id := range tower.RestsOn {
				leg := parts[id]
				if leg.Type != "leg" || !leg.Ground || leg.At[1] != 0 || leg.Size[1] != float64(void) {
					t.Fatalf("tower leg %s: %+v", id, leg)
				}
			}
			// Moving one hanger breaks physical contact, even though all IDs remain.
			for i := range composition.Parts {
				if composition.Parts[i].ID == lower.HangsFrom[0] {
					composition.Parts[i].At[1] += .1
					break
				}
			}
			if err := composition.Validate(); err == nil {
				t.Fatal("accepted a detached hanger")
			}
		})
	}
}
