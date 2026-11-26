package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A real posed, painted animal reaches the CPU tracer and records its path.
func TestCPUTraceWritesAnAnimalWithHonestReceipt(test *testing.T) {
	raw := strings.Replace(string(planRaw), "width: 640", "width: 80", 1)
	raw = strings.Replace(raw, "height: 480", "height: 60", 1)
	raw = strings.Replace(raw, "rays: 2", "rays: 1", 1)
	plan, err := readPlan("small.yaml", []byte(raw))
	if err != nil {
		test.Fatal(err)
	}
	plan.renderer = "cpu"
	menu, err := readMenu()
	if err != nil {
		test.Fatal(err)
	}
	entries, err := menu.selectEntries([]string{"harbor-seal"})
	if err != nil {
		test.Fatal(err)
	}
	maker, err := compiledProducer()
	if err != nil {
		test.Fatal(err)
	}
	root := test.TempDir()
	view := card{Entry: entries[0], Detail: "quick", Folder: "seal"}
	if err := plan.draw(root, &view, maker); err != nil {
		test.Fatal(err)
	}
	recordBytes, err := os.ReadFile(filepath.Join(root, "seal", "render.json"))
	if err != nil {
		test.Fatal(err)
	}
	var record receipt
	if err := json.Unmarshal(recordBytes, &record); err != nil {
		test.Fatal(err)
	}
	image, err := os.ReadFile(filepath.Join(root, "seal", "specimen.png"))
	if err != nil {
		test.Fatal(err)
	}
	if view.Status != cpuRenderedStatus || record.Card.Status != cpuRenderedStatus ||
		record.Backend != "cpu" || record.Profile != nil || record.CPUTraceWall <= 0 ||
		record.ImageSHA256 != fmt.Sprintf("%x", sha256.Sum256(image)) ||
		record.SettingsSHA256 != fmt.Sprintf("%x", sha256.Sum256([]byte(raw))) {
		test.Fatalf("incorrect CPU record: %+v", record)
	}
}
