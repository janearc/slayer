//go:build draw && darwin && arm64

package manufacture

import (
	"github.com/janearc/slayer/manufacture/animals/vertebrates/mammals/pinnipeds"
	"testing"
)

// This records actual geometry and material samples before removal of the
// unused alternate fold algorithms. The changed profile checks that cleanup
// does not accidentally substitute the default rig for an authored profile.
func TestWalrusCleanupWitness(t *testing.T) {
	for _, sample := range []struct {
		name     string
		pose     string
		modified bool
		want     string
	}{
		{"lying", "lying", false, "1ee2ffe1a1097de486b0b2c13826a3281ab713a3ba76ef3f2c0545a8cff0699b"},
		{"hauled-out", "hauled-out", false, "a98a11424337007a1e5cad7be21cac976f3cb07da72d97446451c814ea645dd8"},
		{"swimming", "swimming", false, "97a9cb4b6636c37ac9e8acf7fa55361086c8bffa0267397057500bb415d8c0f2"},
		{"longer-lumbar", "lying", true, "437836859410c680429a3f288342180727cbaef4473f840465b30bbd2cef0039"},
	} {
		t.Run(sample.name, func(t *testing.T) {
			profile := defaultPinniped("walrus")
			if sample.modified {
				profile.Lumbar++
			}
			recipe, err := pinnipeds.NewWalrus(profile, sample.pose, "skin")
			if err != nil {
				t.Fatal(err)
			}
			kind := Kind{name: "walrus-profile", how: sealKind{form: sealWalrus, walrus: recipe}}
			made, err := MakeWithComplexity(kind, Sow(5), "quick")
			if err != nil {
				t.Fatal(err)
			}
			got := pinnipedWitness(t, made)
			if got != sample.want {
				t.Fatalf("geometry/material witness got %s, want %s", got, sample.want)
			}
		})
	}
}
