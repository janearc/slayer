//go:build draw

package session_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/session"
	"github.com/janearc/slayer/sources"
)

// TendTerrain, the function the door's flags call, fetches what the
// shelf lacks, says what it did, and then a check of the shelf holds:
// each of the set's two datasets is fetched once, with a server of the
// test's own.
func TestTendTerrainFetchesThenChecks(t *testing.T) {
	path, shelf, server := terrainSet(t)
	var said bytes.Buffer
	if err := session.TendTerrain(path, sources.Fetching, &said); err != nil {
		t.Fatalf("%v: %s", err, said.String())
	}
	mustSay(t, said.String(), "fixture: fetched", "moon-face: fetched")
	for _, name := range []string{"fixture.tif", "moon-face.tif"} {
		if _, err := os.Stat(filepath.Join(shelf, name)); err != nil {
			t.Error(err)
		}
	}
	said.Reset()
	if err := session.TendTerrain(path, sources.Checking, &said); err != nil {
		t.Fatalf("a good shelf: %v: %s", err, said.String())
	}
	if strings.Count(said.String(), "same as sources.yaml") != twice {
		t.Errorf("the check does not say both are the source's:\n%s", &said)
	}
	if got := server.requests.Load(); got < twice {
		t.Errorf("%d requests for two datasets", got)
	}
}

// the anti-test: a check of a shelf with a file that is not the source's
// fails and says which, and replaces nothing.
func TestTendTerrainCheckFailsOnAWrongFileAndChangesNothing(t *testing.T) {
	path, shelf, _ := terrainSet(t)
	if err := session.TendTerrain(path, sources.Fetching,
		&bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	bent := filepath.Join(shelf, "fixture.tif")
	write(t, bent, "an old terrain")
	var said bytes.Buffer
	err := session.TendTerrain(path, sources.Checking, &said)
	if err == nil || !strings.Contains(said.String(), "sha256 differs") {
		t.Errorf("a bent terrain: %v: %s", err, said.String())
	}
	if got, _ := os.ReadFile(bent); string(got) != "an old terrain" {
		t.Errorf("a check changed the shelf: %q", got)
	}
}
