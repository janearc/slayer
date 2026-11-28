//go:build !cgo

package main

import (
	"context"
	"strings"
	"testing"

	"github.com/janearc/slayer/ferruccio/native/internal/stagewire"
)

func TestStageWithoutCgoExplainsUnavailableHelper(t *testing.T) {
	if _, err := stage(context.Background(), "/scene.yaml", "source", "set", stagewire.Camera{}, ""); err == nil || !strings.Contains(err.Error(), "Cgo-enabled") {
		t.Fatalf("non-Cgo helper returned %v", err)
	}
}
