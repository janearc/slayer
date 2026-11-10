package set_test

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

const mixedRegionBlock = `  ecosystems:
    mixed grove:
      seed: 1
      cell: {value: 100, unit: m}
      density: {value: 1, unit: ha^-1}
      variety: 2
      bounds: {west: 0, east: 200, north: 0, south: -200}
      mixture: {redwoods: 1}
      atmosphere:
        weather: grove mist
        extent: canopy-and-ground
        fade-at-boundary: false
`

func mixedFixture(test *testing.T, schema bool) string {
	test.Helper()
	raw := ecosystemFixture(test)
	if !schema {
		raw = strings.Replace(raw, "$schema: "+set.EcosystemSchema+"\n", "", 1)
	}
	changed := strings.Replace(raw, "  terrain:", mixedRegionBlock+"  terrain:", 1)
	if changed == raw {
		test.Fatal("terrain insertion anchor moved")
	}
	return changed
}

func TestMixedRegionSchemaAndLegacyReaderKeepTheSameControls(test *testing.T) {
	for _, schema := range []bool{false, true} {
		read, err := set.Read("mixture.yaml", []byte(mixedFixture(test, schema)))
		if err != nil {
			test.Fatal(schema, err)
		}
		regions := read.Ecosystems()
		if len(regions) != 1 || regions[0].Variety() != 2 || regions[0].Seed() != 1 {
			test.Fatal("mixed region controls were lost", regions)
		}
		weather, fade := regions[0].Atmosphere()
		if weather != "grove mist" || fade {
			test.Fatal("explicit false fade was not retained", weather, fade)
		}
	}
}

func TestMixedRegionRefusalsAgreeWithAndWithoutTheSchema(test *testing.T) {
	for _, schema := range []bool{false, true} {
		for _, change := range [][2]string{
			{"cell: {value: 100", "cell: {value: 0"},
			{"cell: {value: 100", "cell: {value: .inf"},
			{"variety: 2", "variety: 0"},
			{"      mixture: {redwoods: 1}", "      mixture: {redwoods: 0}"},
			{"      mixture: {redwoods: 1}", "      mixture: {missing: 1}"},
			{"        fade-at-boundary: false\n", ""},
			{"south: -200", "south: 200"},
			{"      density:", "      densitty:"},
			{mixedRegionBlock, "  ecosystems: {}\n"},
		} {
			before := mixedFixture(test, schema)
			raw := strings.Replace(before, change[0], change[1], 1)
			if raw == before {
				test.Fatal("ineffective refusal fixture", change)
			}
			if _, err := set.Read("bad-mixture.yaml", []byte(raw)); err == nil {
				test.Fatal("invalid mixture accepted", schema, change)
			}
		}
	}
}
