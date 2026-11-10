package set_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/set"
)

func TestShippedEcosystemScenesPassSchemaAndDomainChecks(t *testing.T) {
	for _, name := range []string{"observatory-ecosystem.yaml", "observatory-4k.yaml",
		"fire-study.yaml", "fire-cluster.yaml", "fire-night.yaml", "fire-fuel.yaml",
		"fire-woodland.yaml"} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join("../doc", name)
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var header struct {
				Schema string `yaml:"$schema"`
				Scene  any    `yaml:"scene"`
			}
			if err = yamlfile.Read(path, raw, &header); err != nil {
				t.Fatal(err)
			}
			if header.Schema != set.EcosystemSchema {
				t.Fatal("shipped scene bypasses its declared schema")
			}
			if _, err = set.Read(path, raw); err != nil {
				t.Fatal(err)
			}
		})
	}
}
