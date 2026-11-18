package set_test

import (
	"os"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/set"
)

const beamSchemaV2 = "https://github.com/janearc/slayer/schemas/ecosystem-v2.json"

// beamSource is a small authored world with its beam, under a schema label
// when the caller asks for one.
func beamSource(t *testing.T, schema string) string {
	t.Helper()
	raw, err := os.ReadFile("testdata/built.yaml")
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Replace(string(raw), "  world:\n", "  world:\n"+
		"    unit: 2\n"+
		"    beams:\n"+
		"      column:\n"+
		"        at: [1, 2, 3]\n"+
		"        aim: [0, 2, 0]\n"+
		"        radius: 0.5\n"+
		"        reach: 8\n"+
		"        divergence: 0.8\n"+
		"        temperature: 5200\n"+
		"        strength: 4\n"+
		"        source: eye\n", 1)
	if text == string(raw) {
		t.Fatal("built fixture has no world block")
	}
	if schema != "" {
		text = "$schema: " + schema + "\n" + text
	}
	return text
}

func TestBeamReadsInLegacyAndBothSchemaVersions(t *testing.T) {
	for _, schema := range []string{"", set.EcosystemSchema, beamSchemaV2} {
		t.Run(schema, func(t *testing.T) {
			made, err := set.Read("beam.yaml", []byte(beamSource(t, schema)))
			if err != nil {
				t.Fatal(err)
			}
			world, has := made.World()
			if !has || world.Unit() != 2 || len(world.Beams()) != 1 {
				t.Fatalf("world unit %v, beams %+v", world.Unit(), world.Beams())
			}
			beam := world.Beams()[0]
			radius, reach, divergence := beam.Envelope()
			heat, strength := beam.Shine()
			if beam.Name() != "column" || beam.At() != maths.V(1, 3, -2) ||
				beam.Aim() != maths.V(0, 0, -2) || radius != 0.5 || reach != 8 ||
				divergence != 0.8 || heat != 5200 || strength != 4 {
				t.Errorf("beam changed: %+v", beam)
			}
			beams := world.Beams()
			beams[0] = set.Beam{}
			if world.Beams()[0].Name() != "column" {
				t.Error("caller changed the retained beam list")
			}
		})
	}
}

func TestBeamRefusesInvalidEnvelopeAndDirectionInEveryReaderMode(t *testing.T) {
	invalid := []struct{ before, after string }{
		{"radius: 0.5", "radius: 0"},
		{"reach: 8", "reach: 0"},
		{"divergence: 0.8", "divergence: 90"},
		{"aim: [0, 2, 0]", "aim: [0, 0, 0]"},
		{"strength: 4", "strength: -1"},
		{"source: eye", "source: eye\n        mystery: true"},
	}
	for _, schema := range []string{"", set.EcosystemSchema, beamSchemaV2} {
		for _, change := range invalid {
			raw := beamSource(t, schema)
			changed := strings.Replace(raw, change.before, change.after, 1)
			if changed == raw {
				t.Fatalf("fixture did not contain %q", change.before)
			}
			if _, err := set.Read("bad-beam.yaml", []byte(changed)); err == nil {
				t.Errorf("schema %q accepted %s", schema, change.after)
			}
		}
	}
}
