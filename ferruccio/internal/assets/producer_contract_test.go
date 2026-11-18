package assets

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestProducerContractsMatchSlayer prevents a stale frontend contract copy.
func TestProducerContractsMatchSlayer(t *testing.T) {
	contracts := map[string]string{
		"native-larson.yaml":         "manufacture/saucer-choices.yaml",
		"native-larson-landed.yaml":  "manufacture/saucer-landed.yaml",
		"native-human.yaml":          "manufacture/human-scale.yaml",
		"job.schema.json":            "job/job.schema.json",
		"ecosystem.schema.json":      "set/ecosystem.schema.json",
		"ecosystem-v2.schema.json":   "set/ecosystem-v2.schema.json",
		"catalogue.schema.json":      "cmd/catalogue/plan.schema.json",
		"catalogue-menu.schema.json": "cmd/catalogue/menu.schema.json",
		"catalogue-menu.yaml":        "cmd/catalogue/menu.yaml",
	}
	for asset, source := range contracts {
		t.Run(asset, func(t *testing.T) {
			bundled, err := Read(asset)
			if err != nil {
				t.Fatal(err)
			}
			producer, err := os.ReadFile(filepath.Join("..", "..", "..", source))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(bundled, producer) {
				t.Fatalf("%s differs from Slayer %s", asset, source)
			}
		})
	}
}
