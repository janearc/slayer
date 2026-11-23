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
	source := ecosystemFixture(t)
	plain, err := set.Read("software.yaml", []byte(source))
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
