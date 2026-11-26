package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/ferruccio/native/internal/settings"
)

// noSettings points this process at a home folder with no settings file and
// no cache variable, so a test never reads the person's own.
func noSettings(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SLAYER_CACHE", "")
	return home
}

func TestProofUsingSlayerRefusesWithoutACacheFolderFromAnyPlace(t *testing.T) {
	noSettings(t)
	for _, render := range []string{"--render", "--full"} {
		code, output := runAtelier("proof", "--out", "unused.png", render, "--file", "source.yaml", "--preview-transport", "slayer")
		for _, want := range []string{"--cache", "SLAYER_CACHE", "Settings"} {
			if code == 0 || !strings.Contains(output, want) {
				t.Fatalf("proof %s without a cache folder returned %d, and the line lacks %s: %s", render, code, want, output)
			}
		}
	}
}

func TestDryRunNeedsNoCacheFolder(t *testing.T) {
	noSettings(t)
	if code, output := runAtelier("open", "--dry-run"); code != 0 {
		t.Fatalf("an open dry-run was refused for the cache: %d %s", code, output)
	}
	if code, output := runAtelier("proof", "--out", "unused.png", "--dry-run"); code != 0 {
		t.Fatalf("a proof dry-run was refused for the cache: %d %s", code, output)
	}
}

func TestTheSettingsFileIsUnderTheHomeAndOnlyThere(t *testing.T) {
	home := noSettings(t)
	got, err := os.UserHomeDir()
	if err != nil || settings.Path(got) != filepath.Join(home, ".config", "ferruccio", "config") {
		t.Fatalf("the settings file would be %s, %v", settings.Path(got), err)
	}
}

func TestAProofThatDoesNotRenderIsNotAskedForACache(t *testing.T) {
	noSettings(t)
	code, output := runAtelier("proof", "--out", "unused.png", "--gesture", "orbit", "--dry-run")
	if strings.Contains(output, "cache") {
		t.Fatalf("a proof that draws nothing was asked for a cache (%d): %s", code, output)
	}
}
