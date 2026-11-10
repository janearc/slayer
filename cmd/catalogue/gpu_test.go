//go:build darwin

package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/janearc/slayer/metal"
)

func TestCatalogueCommandWritesAnActualNativeSpecimen(test *testing.T) {
	plan := defaultPlan(test)
	menu, err := readMenu()
	if err != nil {
		test.Fatal(err)
	}
	entries, err := menu.selectEntries([]string{"webb"})
	if err != nil {
		test.Fatal(err)
	}
	producer, err := compiledProducer()
	if err != nil {
		test.Fatal(err)
	}
	for _, entry := range entries {
		probe := card{Entry: entry, Detail: "quick", Folder: "probe"}
		err := plan.draw(test.TempDir(), &probe, producer)
		if errors.Is(err, metal.ErrNoDevice) || errors.Is(err, metal.ErrNoRayTracing) {
			test.Skip(err)
		}
		if err != nil {
			test.Fatal(err)
		}
	}
	root := filepath.Join(test.TempDir(), "book")
	var said, wrong bytes.Buffer
	status := tool(context.Background()).Run(
		[]string{"render", "webb", "--out", root}, &said, &wrong)
	if status != successfulExit {
		test.Fatal("real catalogue command failed", said.String(), wrong.String())
	}
	assertNativeSpecimen(test, root, plan)
}

func TestCatalogueCutawayRetainsItsNativeRenderEvidence(test *testing.T) {
	plan := cutawayPlan(test)
	menu, err := readMenu()
	if err != nil {
		test.Fatal(err)
	}
	entries, err := menu.selectEntries([]string{"rubin"})
	if err != nil {
		test.Fatal(err)
	}
	producer, err := compiledProducer()
	if err != nil {
		test.Fatal(err)
	}
	for _, entry := range entries {
		root := test.TempDir()
		view := card{Entry: entry, Detail: "quick", Folder: "probe"}
		err := plan.draw(root, &view, producer)
		if errors.Is(err, metal.ErrNoDevice) ||
			errors.Is(err, metal.ErrNoRayTracing) {
			test.Skip(err)
		}
		if err != nil {
			test.Fatal(err)
		}
		raw, err := os.ReadFile(filepath.Join(root, "probe", "render.json"))
		if err != nil {
			test.Fatal(err)
		}
		var record receipt
		if err := json.Unmarshal(raw, &record); err != nil {
			test.Fatal(err)
		}
		if !view.Cutaway || !record.Card.Cutaway ||
			record.Profile.Native.Refined == noEntries {
			test.Fatal("cutaway output lost its label or native evidence", record)
		}
	}
}

func assertNativeSpecimen(test *testing.T, root string, plan *plan) {
	test.Helper()
	folder := filepath.Join(root, "webb-quick")
	raw, err := os.ReadFile(filepath.Join(folder, "render.json"))
	if err != nil {
		test.Fatal(err)
	}
	// This is the command's output record, not application configuration.
	var record receipt
	if err := json.Unmarshal(raw, &record); err != nil {
		test.Fatal(err)
	}
	if record.Profile.Native.Refined == noEntries ||
		record.Card.Entry.Kind != "webb" || record.Producer == nil ||
		record.Producer.BinarySHA256 == "" {
		test.Fatal("missing specimen or hardware provenance", record)
	}
	expected, err := compiledProducer()
	if err != nil || *record.Producer != *expected {
		test.Fatal("saved producer differs from executing bytes", record.Producer, err)
	}
	imageRaw, err := os.ReadFile(filepath.Join(folder, "specimen.png"))
	if err != nil || record.ImageSHA256 != fmt.Sprintf("%x", sha256.Sum256(imageRaw)) {
		test.Fatal("image hash differs from written PNG", err)
	}
	image, err := os.Open(filepath.Join(folder, "specimen.png"))
	if err != nil {
		test.Fatal(err)
	}
	defer image.Close()
	picture, err := png.DecodeConfig(image)
	if err != nil || picture.Width != plan.Width || picture.Height != plan.Height {
		test.Fatal("missing or wrong-size real PNG", picture, err)
	}
	settings, err := os.ReadFile(filepath.Join(folder, "settings.yaml"))
	if err != nil {
		test.Fatal(err)
	}
	if _, err := readPlan("settings.yaml", settings); err != nil {
		test.Fatal("saved settings cannot be read", err)
	}
	if record.SettingsSHA256 != fmt.Sprintf("%x", sha256.Sum256(settings)) {
		test.Fatal("settings hash differs from saved input")
	}
}
