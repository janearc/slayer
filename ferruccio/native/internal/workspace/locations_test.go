package workspace

import (
	"github.com/janearc/slayer/ferruccio/internal/preview"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRuntimePathsSurviveRendererWorkingDirectory protects relative CLI settings.
func TestRuntimePathsSurviveRendererWorkingDirectory(t *testing.T) {
	options := Options{Evidence: "shared evidence", Examples: "examples", Tools: preview.Tools{Game: "../bin/engine/game5", Slayer: "../bin/engine/slayer", Catalogue: "../bin/engine/catalogue", GameHome: "../bin/engine/gamehome", Shelf: "../shelf"}}
	resolved, err := locations(options)
	if err != nil {
		t.Fatal(err)
	}
	for label, path := range map[string]string{"evidence": resolved.Evidence, "game": resolved.Tools.Game, "slayer": resolved.Tools.Slayer, "catalogue": resolved.Tools.Catalogue, "home": resolved.Tools.GameHome, "shelf": resolved.Tools.Shelf, "examples": resolved.Examples} {
		if !filepath.IsAbs(path) {
			t.Errorf("%s remains relative: %s", label, path)
		}
	}
	if options.Tools.Game != "../bin/engine/game5" {
		t.Fatal("caller options mutated")
	}
}

// TestAbsentRuntimePathsRemainAbsent keeps an unconfigured window usable.
func TestAbsentRuntimePathsRemainAbsent(t *testing.T) {
	got, err := locations(Options{})
	if err != nil || got.Tools != (preview.Tools{}) || got.Examples != "" {
		t.Fatalf("empty locations changed: %+v %v", got, err)
	}
}

// TestAzazelRuntimePathsAreAbsoluteWithoutRequiringGame preserves the external operator boundary.
func TestAzazelRuntimePathsAreAbsoluteWithoutRequiringGame(t *testing.T) {
	options := Options{Tools: preview.Tools{Azazel: "../operator/azazel", Queue: "../operator/queue", Agent: "studio"}}
	got, err := locations(options)
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(got.Tools.Azazel) || !filepath.IsAbs(got.Tools.Queue) {
		t.Fatalf("relative queue settings: %+v", got.Tools)
	}
	if got.Tools.Game != "" || got.Tools.Agent != "studio" || options.Tools.Queue != "../operator/queue" {
		t.Fatalf("queue settings mutated: %+v / %+v", got.Tools, options.Tools)
	}
}

// TestAzazelCommandLookupRefusesUnknownExecutable does not silently choose Game.
func TestAzazelCommandLookupRefusesUnknownExecutable(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PATH", root)
	if _, err := locations(Options{Tools: preview.Tools{Azazel: "missing-azazel"}}); err == nil || !strings.Contains(err.Error(), "missing-azazel") {
		t.Fatalf("missing operator client accepted: %v", err)
	}
	client := filepath.Join(root, "azazel-client")
	if err := os.WriteFile(client, []byte("#!/bin/sh\nexit 97\n"), 0700); err != nil {
		t.Fatal(err)
	}
	got, err := locations(Options{Tools: preview.Tools{Azazel: "azazel-client", Queue: "queue"}})
	if err != nil || got.Tools.Azazel != client {
		t.Fatalf("operator lookup: %+v %v", got.Tools, err)
	}
}
