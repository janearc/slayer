package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/sources"
)

// A later filesystem or missing-key error must not mask the actual first
// refusal. Exercise real owned catalogues and the actual shelf environment.
func TestPinnedAssetKeepsActualShelfAndCatalogueRefusalCauses(t *testing.T) {
	for _, fault := range []string{"missing-shelf", "malformed-catalogue"} {
		t.Run(fault, func(t *testing.T) {
			path := onFixture(t, "", "")
			near := filepath.Dir(path)
			catalogue := filepath.Join(filepath.Dir(near), "sources.yaml")
			shelf := os.Getenv("SLAYER_SHELF")
			good, err := pinnedAssetAt("fixture", near)
			if err != nil || good.shelf != shelf || good.catalogue != catalogue ||
				good.path != filepath.Join(shelf, "fixture.tif") {
				t.Fatalf("positive real pin control failed: %+v, %v", good, err)
			}
			var cause error
			if fault == "missing-shelf" {
				t.Setenv("SLAYER_SHELF", "")
				_, cause = sources.Here()
				if cause == nil || !strings.Contains(cause.Error(), "sources: no shelf") {
					t.Fatalf("actual environment lost its shelf refusal: %v", cause)
				}
			} else {
				if err := os.WriteFile(catalogue, []byte("sources:\n  - key: [\n"), 0600); err != nil {
					t.Fatal(err)
				}
				_, cause = sources.Read(catalogue)
				if cause == nil || !strings.Contains(cause.Error(), catalogue+": yaml:") {
					t.Fatalf("owned malformed catalogue lost its YAML refusal: %v", cause)
				}
			}
			asset, err := pinnedAssetAt("fixture", near)
			if err == nil || err.Error() != cause.Error() ||
				(fault == "missing-shelf" && !errors.Is(err, cause)) {
				t.Fatalf("%s was hidden by a later unrelated refusal: %v, want %v", fault, err, cause)
			}
			if asset != (pinnedAsset{}) {
				t.Fatal("failed pin returned an apparently usable asset")
			}
		})
	}
}
