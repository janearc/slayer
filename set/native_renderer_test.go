package set_test

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

func TestNativeShadowsAreAnExplicitRendererSelection(t *testing.T) {
	raw := strings.Replace(ecosystemFixture(t), "renderer: gpu",
		"renderer: gpu-native-shadows", 1)
	made, err := set.Read("native.yaml", []byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if made.Renderer() != "gpu-native-shadows" {
		t.Fatal("native selection lost")
	}
	plain, err := set.Read("software.yaml", []byte(ecosystemFixture(t)))
	if err != nil || plain.Renderer() != "gpu" {
		t.Fatalf("software selection changed: %s, %v", plain.Renderer(), err)
	}
}

func TestNativeRendererDoesNotAcceptAMisspelledMode(t *testing.T) {
	raw := strings.Replace(ecosystemFixture(t), "renderer: gpu",
		"renderer: gpu-native-shadow", 1)
	if _, err := set.Read("misspelled.yaml", []byte(raw)); err == nil {
		t.Fatal("accepted an unknown rendering mode")
	}
}
