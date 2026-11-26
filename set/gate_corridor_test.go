package set

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/manufacture/cities"
)

// gateCorridorChecks is deliberately fixture-specific, not a vehicle sweep.
// Its stay calculation applies to chords with both ends in the same x/y plane:
// chord's square cross section then extends thickness/2 in the z direction.
func gateCorridorChecks(c cities.Composition) error {
	const half = 58.5 / 2
	if len(c.Roads) != 14 {
		return fmt.Errorf("expected 14 road segments, got %d", len(c.Roads))
	}
	for _, road := range c.Roads {
		if road.Width != 2*half || road.From[2] != 0 || road.To[2] != 0 {
			return fmt.Errorf("road %s does not follow the centred corridor", road.ID)
		}
	}
	var hangers, fans, ribbons int
	for _, p := range c.Parts {
		switch {
		case strings.HasPrefix(p.ID, "hanger-"):
			hangers++
			if p.At[2] < half && p.At[2]+p.Size[2] > -half {
				return fmt.Errorf("hanger %s enters corridor", p.ID)
			}
		case p.Type == "window-ribbon":
			ribbons++
			if p.At[2] < half && p.At[2]+p.Size[2] > -half {
				return fmt.Errorf("ribbon %s enters corridor", p.ID)
			}
		case p.Type == "stay-fan":
			fans++
			z := p.TowerFrom[2]
			if p.TowerTo[2] != z || p.DeckFrom[2] != z || p.DeckTo[2] != z || p.TowerFrom[0] == p.DeckFrom[0] || p.TowerTo[0] == p.DeckTo[0] || math.Abs(z)-p.Thickness/2 < half {
				return fmt.Errorf("stay %s is not clear of corridor", p.ID)
			}
		}
	}
	if hangers != 80 || fans != 4 || ribbons != 44 {
		return fmt.Errorf("unexpected checked geometry: %d hangers, %d fans, %d ribbons", hangers, fans, ribbons)
	}
	return nil
}

func TestGateRoadCorridorAvoidsKnownEdgeGeometry(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "gate-composition.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, void := range []int{220, 300} {
		t.Run(fmt.Sprint(void), func(t *testing.T) {
			source := strings.Replace(string(raw), "parameters: {void: 220}", fmt.Sprintf("parameters: {void: %d}", void), 1)
			c, _, err := readStructureComposition("gate.yaml", structureNode(t, "inline:\n"+indentStructure(source)))
			if err != nil {
				t.Fatal(err)
			}
			if err := gateCorridorChecks(c); err != nil {
				t.Fatal(err)
			}
			for _, id := range []string{"hanger-north-00", "upper-west-north", "lower-north-road-window-0"} {
				found := false
				for i := range c.Parts {
					if c.Parts[i].ID != id {
						continue
					}
					found = true
					original := c.Parts[i]
					if original.Type == "stay-fan" {
						c.Parts[i].Thickness = 2
					} else {
						c.Parts[i].At[2] = 0
					}
					if err := gateCorridorChecks(c); err == nil {
						t.Errorf("accepted %s inside the corridor", id)
					}
					c.Parts[i] = original
					break
				}
				if !found {
					t.Errorf("missing geometry for anti-case %s", id)
				}
			}
			// The ordinary composition validator still rejects bad grades and joins.
			for i := range c.Roads {
				if c.Roads[i].ID == "lower-approach-west" {
					original := c.Roads[i]
					c.Roads[i].To[1] += 100
					if err := c.Validate(); err == nil || !strings.Contains(err.Error(), "grade") {
						t.Errorf("steep road was not rejected as a grade: %v", err)
					}
					c.Roads[i] = original
					c.Roads[i].Next = "upper-shore-west"
					if err := c.Validate(); err == nil {
						t.Error("accepted a disconnected road")
					}
					c.Roads[i] = original
					break
				}
			}
		})
	}
}
