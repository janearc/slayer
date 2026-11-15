//go:build draw

package gpuocean

import (
	"context"
	"testing"

	"github.com/janearc/slayer/improv/moonlake/ocean"
)

func TestRenderRejectsInvalidContextAndDimensions(t *testing.T) {
	config := ocean.DefaultConfig()
	if _, err := Render(nil, config, ocean.RGB{}, nil); err == nil {
		t.Fatal("nil context accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Render(ctx, config, ocean.RGB{}, nil); err == nil {
		t.Fatal("cancelled context accepted")
	}
	if _, err := Render(context.Background(), config, ocean.RGB{}, nil); err == nil {
		t.Fatal("empty frame accepted")
	}
}
