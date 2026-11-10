package set_test

import (
	"github.com/janearc/slayer/set"
	"strings"
	"testing"
)

func TestFullNativeRendererIsExplicit(t *testing.T) {
	raw := strings.Replace(ecosystemFixture(t), "renderer: gpu", "renderer: gpu-native", 1)
	read, err := set.Read("full-native.yaml", []byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if read.Renderer() != "gpu-native" {
		t.Fatalf("renderer lost: %s", read.Renderer())
	}
}

func TestFullNativeRendererRejectsNearMiss(t *testing.T) {
	raw := strings.Replace(ecosystemFixture(t), "renderer: gpu", "renderer: gpu-natives", 1)
	if _, err := set.Read("misspelled.yaml", []byte(raw)); err == nil {
		t.Fatal("unknown mode accepted")
	}
}
