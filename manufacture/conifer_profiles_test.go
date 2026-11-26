package manufacture

import (
	"bytes"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/puzzlemouth/numbers"
)

func TestExternalConiferProfilesKeepExistingSpecimens(t *testing.T) {
	raw, err := os.ReadFile("plants/conifers/conifer-profiles.yaml")
	if err != nil {
		t.Fatal(err)
	}
	book, err := ReadConiferProfiles("conifers.yaml", raw)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		make func() (Kind, error)
	}{
		{"spruce", book.NewSpruce}, {"norway", book.NewNorwaySpruce},
		{"coast-redwood", book.NewCoastRedwood}, {"giant-sequoia", book.NewGiantSequoia},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			made, err := tc.make()
			if err != nil {
				t.Fatal(err)
			}
			// Compare against the former row constructors; the separate
			// recorded pre-profile witness checks their historical output.
			var original Kind
			switch tc.name {
			case "spruce":
				original = kind{name: tc.name, how: physicalSpruce{legacy: spruceRow()}}
			case "norway":
				original = kind{name: tc.name, how: physicalSpruce{legacy: norwayRow()}}
			case "coast-redwood":
				original = kind{name: tc.name, how: redwoodRow()}
			case "giant-sequoia":
				original = kind{name: tc.name, how: sequoiaRow()}
			}
			for _, detail := range []string{"hull", "quick"} {
				old, err := MakeWithComplexity(original, Sow(5), detail)
				if err != nil {
					t.Fatal(err)
				}
				newTree, err := MakeWithComplexity(made, Sow(5), detail)
				if err != nil {
					t.Fatal(err)
				}
				if referenceDigest(t, old) != referenceDigest(t, newTree) {
					t.Fatalf("%s geometry changed", detail)
				}
			}
		})
	}
}

func TestExternalConiferProfilesRefuseBadData(t *testing.T) {
	raw, err := os.ReadFile("plants/conifers/conifer-profiles.yaml")
	if err != nil {
		t.Fatal(err)
	}
	for _, mutation := range [][2]string{
		{"kind: \"spruce\"", "kind: ''"},
		{"status: \"picked\"", "status: read"},
		{"between: [0.4, 0.6]", "between: [1e-150, 0.6]"},
		{"pieces: 3", "pieces: 1000000000"},
		{"spacing: 0.004", "spacing: 1e-150"},
		{"length-source: osu-coast-redwood-foliage", "length-source: wrong-source"},
		{"length: [0.01, 0.02]", "length: [0.011, 0.02]"},
		{"length-status: read", "length-status: picked"},
	} {
		changed := bytes.Replace(raw, []byte(mutation[0]), []byte(mutation[1]), 1)
		if bytes.Equal(changed, raw) {
			t.Fatalf("fixture missing %q", mutation[0])
		}
		if _, err := ReadConiferProfiles("bad.yaml", changed); err == nil {
			t.Errorf("accepted %q", mutation[1])
		}
	}
	book, err := ReadConiferProfiles("conifers.yaml", raw)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := book.NewConifer("absent"); err == nil || !strings.Contains(err.Error(), "absent") {
		t.Fatalf("unknown species: %v", err)
	}
}

func TestConiferProfilesDistinguishNullMissingAndZero(t *testing.T) {
	for _, edit := range [][2]string{
		{"    droop: 0.22\n", "    droop: null\n"},
		{"    wander: 0\n", ""},
		{"  bark-colour: [0.26, 0.17, 0.1]", "  bark-colour: [null, 0.17, 0.1]"},
		{"    joints: [[1, 0], [0, 0.01]]", "    joints: [[1, null], [0, 0.01]]"},
	} {
		changed := bytes.Replace(defaultConiferProfilesRaw, []byte(edit[0]), []byte(edit[1]), 1)
		if bytes.Equal(changed, defaultConiferProfilesRaw) {
			t.Fatalf("missing fixture %q", edit[0])
		}
		if _, err := ReadConiferProfiles("conifers.yaml", changed); err == nil {
			t.Errorf("accepted %q", edit[1])
		}
	}
	for _, edit := range [][2]string{
		{"    droop: 0.22\n", "    droop: 0\n"},
		{"  bark-colour: [0.26, 0.17, 0.1]", "  bark-colour: [0, 0.17, 0.1]"},
	} {
		changed := bytes.Replace(defaultConiferProfilesRaw, []byte(edit[0]), []byte(edit[1]), 1)
		if _, err := ReadConiferProfiles("conifers.yaml", changed); err != nil {
			t.Errorf("refused zero %q: %v", edit[1], err)
		}
	}
	shootStart := bytes.Index(defaultConiferProfilesRaw, []byte("  shoots:\n"))
	shootEnd := bytes.Index(defaultConiferProfilesRaw[shootStart:], []byte("- kind:")) + shootStart
	shootNull := append(bytes.Clone(defaultConiferProfilesRaw[:shootStart]), []byte("  shoots: null\n")...)
	shootNull = append(shootNull, defaultConiferProfilesRaw[shootEnd:]...)
	if _, err := ReadConiferProfiles("conifers.yaml", shootNull); err == nil || !strings.Contains(err.Error(), "null field profiles[") {
		t.Fatalf("null shoots: %v", err)
	}
	for _, edit := range [][2]string{
		{"length-references: [redwood-needle-minimum, redwood-needle-maximum]", "length-references: [null, null]"},
		{"    droop: 0.22\n", "    <<: {droop: null}\n"},
	} {
		changed := bytes.Replace(defaultConiferProfilesRaw, []byte(edit[0]), []byte(edit[1]), 1)
		if bytes.Equal(changed, defaultConiferProfilesRaw) {
			t.Fatalf("missing fixture %q", edit[0])
		}
		if _, err := ReadConiferProfiles("conifers.yaml", changed); err == nil {
			t.Errorf("accepted %q", edit[1])
		}
	}
	for _, extra := range []string{"\n---\nprofiles: []\n", "\n---\nprofiles: [\n"} {
		if _, err := ReadConiferProfiles("conifers.yaml", append(bytes.Clone(defaultConiferProfilesRaw), extra...)); err == nil {
			t.Fatalf("accepted trailing document %q", extra)
		}
	}
}

func TestConiferProfileRequiresAtLeastOneWhorlAcrossRange(t *testing.T) {
	for _, tc := range []struct {
		between string
		valid   bool
	}{
		{"between: [0.4, 8]", true},
		{"between: [0.4, 8.01]", false},
	} {
		raw := bytes.Replace(defaultConiferProfilesRaw, []byte("between: [0.4, 0.6]"), []byte(tc.between), 1)
		if bytes.Equal(raw, defaultConiferProfilesRaw) {
			t.Fatal("missing spruce spacing fixture")
		}
		book, err := ReadConiferProfiles("conifers.yaml", raw)
		if !tc.valid {
			if err == nil || !strings.Contains(err.Error(), "whorl budget") {
				t.Fatalf("accepted a range allowing zero whorls: %v", err)
			}
			continue
		}
		if err != nil {
			t.Fatalf("refused the minimum whorl boundary: %v", err)
		}
		row, err := book.NewSpruce()
		if err != nil {
			t.Fatal(err)
		}
		for seed := 0; seed < 20; seed++ {
			if _, err := MakeWithComplexity(row, Sow(uint64(seed)), "hull"); err != nil {
				t.Fatalf("boundary seed %d: %v", seed, err)
			}
		}
	}
}

func TestExternalConiferFoliageUsesTheSelectedProfile(t *testing.T) {
	raw, err := os.ReadFile("plants/conifers/conifer-profiles.yaml")
	if err != nil {
		t.Fatal(err)
	}
	wide := bytes.Replace(raw, []byte("width: 0.0012"), []byte("width: 0.003"), 1)
	if bytes.Equal(raw, wide) {
		t.Fatal("fixture has no spruce width")
	}
	first, err := ReadConiferProfiles("first.yaml", raw)
	if err != nil {
		t.Fatal(err)
	}
	second, err := ReadConiferProfiles("second.yaml", wide)
	if err != nil {
		t.Fatal(err)
	}
	a, err := first.NewSpruce()
	if err != nil {
		t.Fatal(err)
	}
	b, err := second.NewSpruce()
	if err != nil {
		t.Fatal(err)
	}
	oldTree, err := MakeWithComplexity(a, Sow(5), "quick")
	if err != nil {
		t.Fatal(err)
	}
	newTree, err := MakeWithComplexity(b, Sow(5), "quick")
	if err != nil {
		t.Fatal(err)
	}
	if referenceDigest(t, oldTree) == referenceDigest(t, newTree) {
		t.Fatal("different needle widths produced identical geometry")
	}
}

func TestConiferFoliageRefusesAnExcessiveAxisBeforeAllocating(t *testing.T) {
	axis, err := newTube(measure.Metres(1e100), measure.Metres(1), measure.Metres(.5))
	if err != nil {
		t.Fatal(err)
	}
	_, err = recursiveFoliage(good{parts: []part{{of: leaf, terminal: true, shape: axis}}}, Sow(1), 1, "quick")
	if err == nil || !strings.Contains(err.Error(), "budget") {
		t.Fatalf("excessive axis: %v", err)
	}
}

func TestConiferBookCacheBoundsAndRetainsExactGeometry(t *testing.T) {
	raw, err := os.ReadFile("plants/conifers/conifer-profiles.yaml")
	if err != nil {
		t.Fatal(err)
	}
	book, err := ReadConiferProfiles("conifers.yaml", raw)
	if err != nil {
		t.Fatal(err)
	}
	store := &shootStore{limit: 1}
	profile, _, err := book.book.Select("spruce")
	if err != nil {
		t.Fatal(err)
	}
	spec := profile.Needle.Geometry()
	first, err := store.prototype(3, 0, 0, "quick", spec)
	if err != nil {
		t.Fatal(err)
	}
	again, err := store.prototype(3, 0, 0, "quick", spec)
	if err != nil || first != again {
		t.Fatalf("same profile did not reuse its prototype: %v", err)
	}
	spec.Width *= 2
	second, err := store.prototype(3, 0, 0, "quick", spec)
	if err != nil {
		t.Fatal(err)
	}
	third, err := store.prototype(3, 0, 0, "quick", spec)
	if err != nil {
		t.Fatal(err)
	}
	if first == second || second == third || store.count != 1 {
		t.Fatal("full cache reused wrong geometry or grew past its limit")
	}
	if second.physical != third.physical {
		t.Fatal("uncached geometry changed")
	}
}

func TestConiferProfileMaturityNotesOnlyVerifiedSources(t *testing.T) {
	raw, err := os.ReadFile("plants/conifers/conifer-profiles.yaml")
	if err != nil {
		t.Fatal(err)
	}
	profile, err := ReadConiferProfiles("conifers.yaml", raw)
	if err != nil {
		t.Fatal(err)
	}
	// Reporting serializes the global observer for the duration of a report.
	reporting.Lock()
	defer reporting.Unlock()
	book := &ledger{read: map[string]map[string]numbers.Kind{}}
	previous := watching.Swap(book)
	defer watching.Store(previous)
	book.scope([]string{"coast-redwood"})
	tree, err := profile.NewCoastRedwood()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := MakeWithComplexity(tree, Sow(1), "hull"); err != nil {
		t.Fatal(err)
	}
	measured, picked := book.names("coast-redwood")
	if !slices.Contains(measured, "redwood-needle-minimum") || !slices.Contains(measured, "redwood-needle-maximum") {
		t.Fatalf("verified measurements missing: %v", measured)
	}
	if !slices.Contains(picked, "coast-redwood.needle.width") {
		t.Fatalf("picked foliage missing: %v", picked)
	}
}

func TestConiferProfileCountsShootWhorlMinimumInBudget(t *testing.T) {
	raw, err := os.ReadFile("plants/conifers/conifer-profiles.yaml")
	if err != nil {
		t.Fatal(err)
	}
	changed := bytes.Replace(raw, []byte("      fewest: 1\n"), []byte("      fewest: 64\n"), 1)
	if bytes.Equal(raw, changed) {
		t.Fatal("fixture has no shoot habit")
	}
	if _, err := ReadConiferProfiles("conifers.yaml", changed); err == nil || !strings.Contains(err.Error(), "branch budget") {
		t.Fatalf("excessive shoot whorls: %v", err)
	}
}

func TestConiferProfilesRejectUnmeshableNeedles(t *testing.T) {
	raw, err := os.ReadFile("plants/conifers/conifer-profiles.yaml")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ name, from, to string }{
		{"tiny width", "width: 0.0012", "width: 1e-300"},
		{"tiny thickness", "thickness: 0.0012", "thickness: 1e-300"},
		{"tiny length", "length: [0.014, 0.022]", "length: [1e-300, 1e-300]"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if !bytes.Contains(raw, []byte(tc.from)) {
				t.Fatal("fixture absent")
			}
			changed := bytes.Replace(raw, []byte(tc.from), []byte(tc.to), 1)
			if _, err := ReadConiferProfiles("bad.yaml", changed); err == nil {
				t.Fatal("accepted unmeshable foliage")
			}
		})
	}
}
