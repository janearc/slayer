package pinnipeds

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
)

// Pelt and reference values were recorded before extraction at 83f4f84b.
func TestLoftProfileOwnsItsOriginalPeltAndReference(t *testing.T) {
	book, err := Read("pinnipeds.yaml", profileRaw)
	if err != nil {
		t.Fatal(err)
	}
	wantSites := []sealSourceSite{
		{"ears", 16.1, 11.9}, {"neck", 41.2, 21.8}, {"axillary", 76.7, 32.2},
		{"sternum", 107.7, 33.7}, {"middle", 145.7, 32.8}, {"umbilicus", 182.5, 28.8},
		{"pelvis", 223.6, 20.2}, {"ankles", 245.1, 12.1},
	}
	for _, tc := range []struct {
		name     string
		axillary float64
		want     sealPelt
	}{
		{"harbor-seal", .2017, sealPelt{back: maths.RGB(.115, .108, .098), belly: maths.RGB(.420, .400, .360),
			mark: maths.RGB(.030, .028, .025), flipper: maths.RGB(.085, .080, .074), pad: maths.RGB(.022, .020, .019),
			backLevel: .17, bellyLevel: .37, depth: .92, ringLight: .40}},
		{"hawaiian-monk-seal", .189, sealPelt{back: maths.RGB(.080, .074, .066), belly: maths.RGB(.380, .340, .260),
			flipper: maths.RGB(.070, .064, .058), pad: maths.RGB(.022, .020, .019)}},
	} {
		profile, ok := book.Get(tc.name)
		if !ok || profile.Loft == nil || profile.Loft.Axillary != tc.axillary ||
			profile.Loft.AxillaryBasis != "derived" || profile.Loft.AxillarySource == "" || profile.Loft.PeltSource == "" ||
			profile.Loft.Reference.Citation == "" {
			t.Fatalf("%s: missing coat or source", tc.name)
		}
		row, err := NewLoft(profile, "hauled")
		if err != nil {
			t.Fatal(err)
		}
		if row.form.pelt != tc.want || row.form.source.Length != 262.3 || row.form.source.Axillary != 32.2 ||
			!reflect.DeepEqual(row.form.source.Sites, wantSites) {
			t.Fatalf("%s: original pelt or reference changed", tc.name)
		}
		profile.Loft.Pelt.Back[0] = 1
		profile.Loft.Reference.Sites[0].Round = 1
		again, _ := book.Get(tc.name)
		if peltFromProfile(again.Loft.Pelt) != tc.want || again.Loft.Reference.Sites[0].Round != 11.9 {
			t.Fatalf("%s: returned profile exposed book data", tc.name)
		}
	}
}

func TestLoftProfileDrivesTheRenderedPelt(t *testing.T) {
	raw := bytes.Replace(profileRaw, []byte("back: [0.115, 0.108, 0.098]"), []byte("back: [0.315, 0.208, 0.198]"), 1)
	if bytes.Equal(raw, profileRaw) {
		t.Fatal("missing harbor pelt fixture")
	}
	book, err := Read("changed.yaml", raw)
	if err != nil {
		t.Fatal(err)
	}
	original, _ := Default().Get("harbor-seal")
	changed, _ := book.Get("harbor-seal")
	a, err := NewLoft(original, "hauled")
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewLoft(changed, "hauled")
	if err != nil {
		t.Fatal(err)
	}
	if a.form.outline.at(.1) != b.form.outline.at(.1) || a.form.axillary != b.form.axillary {
		t.Fatal("changing the pelt changed the anatomy")
	}
	point := maths.V(.1, .2, 0)
	first := (sealHide{pelt: a.form.pelt, outline: a.form.outline, seed: 407}).At(point)
	second := (sealHide{pelt: b.form.pelt, outline: b.form.outline, seed: 407}).At(point)
	if first == second {
		t.Fatal("changed pelt did not change the rendered texture")
	}
	changed.Loft.Pelt.Back[0] = 1
	changed.Loft.Reference.Sites[0].Round = 1
	if b.form.profile.Loft.Pelt.Back[0] != .315 || b.form.source.Sites[0].Round != 11.9 {
		t.Fatal("loft retained its caller's mutable profile")
	}
	// A second profile must not change the default pelt.
	defaultAgain, _ := Default().Get("harbor-seal")
	if defaultAgain.Loft.Pelt.Back != original.Loft.Pelt.Back {
		t.Fatal("changed book altered the default pelt")
	}
}

func TestLoftProfileAcceptsSourcedPeltBasis(t *testing.T) {
	for _, basis := range []string{"read", "derived"} {
		raw := bytes.Replace(profileRaw, []byte("pelt-basis: picked"), []byte("pelt-basis: "+basis), 1)
		book, err := Read("sourced.yaml", raw)
		if err != nil {
			t.Fatal(err)
		}
		profile, _ := book.Get("harbor-seal")
		if profile.Loft.PeltBasis != basis {
			t.Fatalf("lost %s pelt basis", basis)
		}
	}
}

func TestLoftProfileRefusesMalformedPeltAndSource(t *testing.T) {
	for _, tc := range []struct{ old, replacement, field string }{
		{"axillary: 0.2017", "axillary: -1", "axillary"},
		{"axillary: 0.2017", "axillary: .nan", "axillary"},
		{"axillary-basis: derived", "axillary-basis: imagined", "axillary-basis"},
		{"pelt-basis: picked", "pelt-basis: unknown", "pelt-basis"},
		{"back: [0.115, 0.108, 0.098]", "back: [1.1, 0.108, 0.098]", "pelt back"},
		{"back: [0.115, 0.108, 0.098]", "back: [0.115, 0.108]", "back"},
		{"back-level: 0.17", "back-level: null", "back-level"},
		{"      back-level: 0.17\n", "", "back-level"},
		{"ring-light: 0.40", "ring-light: .inf", "ring-light"},
		{"length: 262.3", "length: 0", "reference length"},
		{"name: ears", "name: ''", "site 0 name"},
		{"behind: 16.1", "behind: 300", "site ears behind"},
		{"round: 11.9", "round: 0", "site ears round"},
		{"pelt-source:", "unexpected-source:", "unexpected-source"},
		{`pelt-source: "National Museum of Nature and Science, Tokyo, Marine Mammals Information Database, harbor seal"`, "pelt-source: ''", "pelt-source"},
	} {
		raw := bytes.Replace(profileRaw, []byte(tc.old), []byte(tc.replacement), 1)
		if bytes.Equal(raw, profileRaw) {
			t.Fatalf("missing fixture %q", tc.old)
		}
		_, err := Read("invalid.yaml", raw)
		if err == nil || !strings.Contains(err.Error(), tc.field) {
			t.Fatalf("%s: got %v", tc.field, err)
		}
	}
}
