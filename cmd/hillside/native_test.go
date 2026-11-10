package main

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

func TestSceneSelectsNativeShadowsWithoutChangingOtherModes(t *testing.T) {
	for _, test := range []struct {
		name                 string
		gpu, native, primary bool
	}{
		{"gpu-native", true, false, true},
		{"gpu-native-shadows", true, true, false},
		{"gpu", true, false, false},
		{"cpu", false, false, false},
	} {
		raw := strings.Replace(fixtureScene, "scene:", "scene:\n  renderer: "+test.name, 1)
		read, err := set.Read("mode.yaml", []byte(raw))
		if err != nil {
			t.Fatal(err)
		}
		given := mustParse(t)
		given.onGPU, given.nativeShadows, given.nativeMeshes = true, true, true
		got, err := filmed(read, given)
		if err != nil {
			t.Fatal(err)
		}
		if got.onGPU != test.gpu || got.nativeShadows != test.native ||
			got.nativeMeshes != test.primary {
			t.Fatalf("%s selected gpu=%v, shadows=%v, primary=%v",
				test.name, got.onGPU, got.nativeShadows, got.nativeMeshes)
		}
	}
}
