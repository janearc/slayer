package reptiles

import (
	"bytes"
	"slices"
	"testing"
)

func TestBookOwnsAuthoredSpeciesAndControls(t *testing.T) {
	_, raw := Embedded()
	book, err := Read("reptiles.yaml", raw)
	if err != nil {
		t.Fatal(err)
	}
	for i := range raw {
		raw[i] = 0
	}
	if got := book.Species(); !slices.Equal(got, []string{"hatzegopteryx", "pliosaur", "purussaurus"}) {
		t.Fatalf("species %v", got)
	}
	for _, tc := range []struct {
		species, scaffold                  string
		phase, yaw, posture, fold, seconds float64
	}{
		{"pliosaur", "pliosaur", 0, 0, 0, 0, 0},
		{"purussaurus", "crocodylian", 0, .15, .2, 0, 0},
		{"hatzegopteryx", "pterosaur", 0, 0, 0, .5, 0},
	} {
		row, ok := book.Get(tc.species)
		if !ok || row.Scaffold != tc.scaffold || row.Phase != tc.phase || row.Yaw != tc.yaw || row.Posture != tc.posture || row.Fold != tc.fold || row.Seconds != tc.seconds {
			t.Errorf("profile %s: %+v", tc.species, row)
		}
	}
	if _, ok := book.Get("unknown"); ok {
		t.Fatal("unknown species accepted")
	}
	var absent *Book
	if _, ok := absent.Get("pliosaur"); ok || len(absent.Species()) != 0 {
		t.Fatal("nil book accepted")
	}
}

func TestBookRejectsInvalidAndAmbiguousProfiles(t *testing.T) {
	_, raw := Embedded()
	for _, mutation := range [][2]string{
		{"kind: pliosaur", "kind: ''"}, {"scaffold: pliosaur", "scaffold: unknown"},
		{"phase: 0", "phase: .nan"}, {"phase: 0", "phase: null"},
		{"fold: 0.5", "fold: 2"}, {"yaw: 0.15", "yaw: .inf"},
		{"posture: 0.2", "posture: 2"}, {"status: picked", "status: read"},
		{"scale: 2.8", "scale: 0"}, {"scale: 2.8", "scale: .nan"},
		{"[-5.00, 0, 0.015, 0.015]", "[-5.00, 0, 0, 0.015]"},
		{"[-4.50, 0, 0.09, 0.08]", "[-5.50, 0, 0.09, 0.08]"},
		{"[-0.24, 0, 0, 0.79, 0.26, 0.34]", "[-0.24, 0, 0, 0.79, 0, 0.34]"},
		{"continuous-head: [0.66, 1.33, 0.035, 0.085, 0.025, 0.105, 0.036]", "continuous-head: [0.66, 1.33, 0, 0.085, 0.025, 0.105, 0.036]"},
		{"continuous-fins: [0.18, -0.72, 0.20, 0.90, 0.17, 0.23]", "continuous-fins: [0.18, -0.72, 0.20, 0.20, 0.17, 0.23]"},
		{"continuous-fins: [0.18, -0.72, 0.20, 0.90, 0.17, 0.23]", "continuous-fins: [0.18, .nan, 0.20, 0.90, 0.17, 0.23]"},
		{"fin-roots: [0.65, -2.10]", "fin-roots: [0.65, 2.10]"},
		{"eye: [2.67, 0.14, 0.37, 0.045]", "eye: [2.67, 0.14, 0.37, 0]"},
		{"[0.95, 0.59, 0.12, -0.06]", "[0.95, 0, 0.12, -0.06]"},
		{"[-1.25, 2.55, 0, 0.04, 0.05]", "[-1.25, 2.55, 0, 0, 0.05]"},
		{"[-0.92, 2.58, 0.29, 0.25]", "[-1.92, 2.58, 0.29, 0.25]"},
		{"[-0.80, 0, 0, 0.35, 0.20, 0.16, 0.95]", "[-0.80, 0, 0, 0.35, 0, 0.16, 0.95]"},
		{"wing-lengths: [1.35, 1.45, 2.85]", "wing-lengths: [0.1, 0.1, 2.85]"},
		{"[0.15, 2.80, 2.70]", "[0.15, 2.80, -2.70]"},
		{"cage-wing-lengths: [1.22, 1.63, 1.25, 1.0]", "cage-wing-lengths: [0.1, 0.1, 1.25, 1.0]"},
		{"cage-leg-lengths: [1.18, 1.40]", "cage-leg-lengths: [0.1, 0.1]"},
		{"[0.40, 0.20, 0.15]", "[0, 0, 0]"},
		{"[-1.70, -1.28, 0.018, 0.095, 0.012, 0.055]", "[-1.70, -1.28, 0, 0.095, 0.012, 0.055]"},
		{"[-1.28, -0.80, 0.095, 0.17, 0.055, 0.12]", "[-1.20, -0.80, 0.095, 0.17, 0.055, 0.12]"},
		{"limb-roots: [0.43, -0.40]", "limb-roots: [-0.43, 0.40]"},
		{"limb-radii: [0.10, 0.08, 0.06]", "limb-radii: [0.10, 0, 0.06]"},
		{"[-0.04, -0.16, 0.40]", "[0, -0.075, 0.26]"},
		{"by: Pliosaur study", "by: ''"}, {"phase: 0", "unexpected: 1"},
	} {
		bad := bytes.Replace(raw, []byte(mutation[0]), []byte(mutation[1]), 1)
		if bytes.Equal(bad, raw) {
			t.Fatalf("missing fixture %q", mutation[0])
		}
		if _, err := Read("bad.yaml", bad); err == nil {
			t.Errorf("accepted %q", mutation[1])
		}
	}
	for _, bad := range [][]byte{nil, make([]byte, (1<<20)+1), []byte("profiles: []\n"), append(bytes.Clone(raw), []byte("\n---\nprofiles: []\n")...)} {
		if _, err := Read("bad.yaml", bad); err == nil {
			t.Fatal("accepted invalid profile document")
		}
	}
	duplicate := append(bytes.Clone(raw), raw[bytes.Index(raw, []byte("  - kind:")):]...)
	if _, err := Read("duplicate.yaml", duplicate); err == nil {
		t.Fatal("accepted duplicate species")
	}
}
