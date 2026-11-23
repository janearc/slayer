package door

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/gpu"
)

// refusedParts are refusals of two reasons, the first of them twice.
var refusedParts = []gpu.Refusal{
	{Part: "thing", Index: 3, Kind: "shape.Tube", Material: "neon", Reason: "a tube"},
	{Part: "world", Index: -1, Reason: "the world's haze"},
	{Part: "thing", Index: 9, Kind: "shape.Tube", Reason: "a tube"},
}

// refusals are grouped by reason, in the order each first appears, each
// with its count and its first as the example.
func TestRefusalsAreGroupedByReason(t *testing.T) {
	groups := grouped(refusedParts)
	if len(groups) != 2 || groups[0].reason != "a tube" || groups[0].count != 2 ||
		groups[0].example.Index != 3 || groups[1].reason != "the world's haze" ||
		groups[1].count != 1 {
		t.Errorf("groups %+v", groups)
	}
}

// what is said names each part by place, kind and material, the world
// with no place, and says a clean world is taken.
// anti-test of TestRefusalsAreGroupedByReason
func TestRefusalLinesSayEveryPartAndAnEmptyListIsTaken(t *testing.T) {
	said := strings.Join(refusalLines("street.yaml", refusedParts), "\n")
	for _, want := range []string{
		"would refuse 3 parts for 2 reasons",
		"2  a tube (for example thing 3 shape.Tube (neon))",
		"1  the world's haze (for example world)",
		"  thing 9 shape.Tube: a tube",
	} {
		if !strings.Contains(said, want) {
			t.Errorf("%q is not in %q", want, said)
		}
	}
	clean := refusalLines("ball.yaml", nil)
	if len(clean) != 1 || !strings.Contains(clean[0], "would take every part") {
		t.Errorf("a clean file said %q", clean)
	}
}

// --refusals stages a file and draws nothing: a still of a sphere is
// one the GPU path takes. With --census it is refused by name, and
// without --refusals assess says nothing of the GPU.
func TestAssessRefusalsOfAStillAndItsStrays(t *testing.T) {
	directory := t.TempDir()
	path := written(t, directory, "still.yaml", stillJob(t, ""))
	got := assessed(t, directory, path, "--refusals")
	if got.code != nothing || strings.Contains(got.said, "will draw") {
		t.Fatalf("assess --refusals gave %d: %q %q", got.code, got.said,
			got.wrongSaid)
	}
	if !strings.Contains(got.said, "still.yaml: the GPU path would ") {
		t.Errorf("assess said %q", got.said)
	}
	stray := assessed(t, directory, path, "--refusals", "--census", "c.json")
	if stray.code != one || !strings.Contains(stray.wrongSaid,
		"--census does nothing with refusals") {
		t.Errorf("--refusals --census gave %d: %q", stray.code, stray.wrongSaid)
	}
	plain := assessed(t, directory, path)
	if strings.Contains(plain.said, "GPU") {
		t.Errorf("plain assess spoke of the GPU: %q", plain.said)
	}
}
