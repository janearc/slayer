//go:build draw

package regioncompare

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/render/regionmanifest"
	"github.com/janearc/slayer/render/regionref"
)

func TestReferencePlanRoundTripAndRefusals(t *testing.T) {
	source, region := sessionFixture(t)
	original := source.References[region.Name]
	var saved bytes.Buffer
	if err := WriteReferencePlan(&saved, source); err != nil {
		t.Fatal(err)
	}
	if err := ApplyReferencePlan(bytes.NewReader(saved.Bytes()), source, t.TempDir()); err != nil {
		t.Fatal(err)
	}
	if source.References[region.Name].Identity != original.Identity {
		t.Fatal("identity changed")
	}
	for _, invalid := range []string{saved.String() + " {}", `{"version":"bad","regions":[]}`, strings.Repeat(" ", (4<<20)+1)} {
		before := source.References[region.Name].Identity
		if err := ApplyReferencePlan(strings.NewReader(invalid), source, t.TempDir()); err == nil {
			t.Fatal("invalid plan accepted")
		}
		if source.References[region.Name].Identity != before {
			t.Fatal("invalid plan changed references")
		}
	}
}

func TestReferencePlanRefusesIncompatibleDrawBeforeRendering(t *testing.T) {
	for _, change := range []struct {
		name  string
		alter func(*regionmanifest.Manifest)
	}{
		{"rectangle", func(m *regionmanifest.Manifest) { m.Region.Left = 0 }},
		{"input", func(m *regionmanifest.Manifest) { m.Inputs[0].SHA256[0] ^= 1 }},
		{"settings", func(m *regionmanifest.Manifest) { m.Settings[0] ^= 1 }},
		{"camera", func(m *regionmanifest.Manifest) { m.Camera = "elsewhere" }},
		{"time", func(m *regionmanifest.Manifest) { *m.TimeNS = 1 }},
		{"development", func(m *regionmanifest.Manifest) { m.Development[0] ^= 1 }},
		{"renderer", func(m *regionmanifest.Manifest) { m.Renderer = "gpu" }},
		{"detail", func(m *regionmanifest.Manifest) { *m.Scrooge = !*m.Scrooge }},
		{"sampling", func(m *regionmanifest.Manifest) { m.Rays = 1 }},
	} {
		t.Run(change.name, func(t *testing.T) {
			source, region := sessionFixture(t)
			before := source.References[region.Name].Identity
			var saved bytes.Buffer
			if err := WriteReferencePlan(&saved, source); err != nil {
				t.Fatal(err)
			}
			var plan referencePlan
			if err := json.Unmarshal(saved.Bytes(), &plan); err != nil {
				t.Fatal(err)
			}
			change.alter(&plan.Regions[0].Manifest)
			data, err := json.Marshal(plan)
			if err != nil {
				t.Fatal(err)
			}
			if err := ApplyReferencePlan(bytes.NewReader(data), source, t.TempDir()); err == nil {
				t.Fatal("incompatible reference plan accepted")
			}
			if source.References[region.Name].Identity != before {
				t.Fatal("failed plan changed references")
			}
		})
	}
}

func TestReferencePlanCanPinAnEarlierExecutable(t *testing.T) {
	source, region := sessionFixture(t)
	var saved bytes.Buffer
	if err := WriteReferencePlan(&saved, source); err != nil {
		t.Fatal(err)
	}
	var plan referencePlan
	if err := json.Unmarshal(saved.Bytes(), &plan); err != nil {
		t.Fatal(err)
	}
	plan.Regions[0].Manifest.Executable[0] ^= 1
	id, err := regionmanifest.Digest(plan.Regions[0].Manifest)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	ref := filepath.Join(dir, hex.EncodeToString(id[:])+".regionref")
	if err := regionref.Publish(ref, regionref.Identity(id), region.Width, region.Height, make([]byte, region.Width*region.Height*12)); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	if err := ApplyReferencePlan(bytes.NewReader(data), source, dir); err != nil {
		t.Fatal(err)
	}
	if source.References[region.Name].Manifest.Executable != plan.Regions[0].Manifest.Executable {
		t.Fatal("reference executable was not retained")
	}
}

func TestReferencePlanRefusesUnavailableArtifactsBeforeRendering(t *testing.T) {
	for _, oldBuild := range []bool{false, true} {
		t.Run(map[bool]string{false: "damaged", true: "missing earlier build"}[oldBuild], func(t *testing.T) {
			source, region := sessionFixture(t)
			before := source.References[region.Name].Identity
			var saved bytes.Buffer
			if err := WriteReferencePlan(&saved, source); err != nil {
				t.Fatal(err)
			}
			var plan referencePlan
			if err := json.Unmarshal(saved.Bytes(), &plan); err != nil {
				t.Fatal(err)
			}
			if oldBuild {
				plan.Regions[0].Manifest.Executable[0] ^= 1
			}
			id, err := regionmanifest.Digest(plan.Regions[0].Manifest)
			if err != nil {
				t.Fatal(err)
			}
			dir := t.TempDir()
			if !oldBuild {
				if err := os.WriteFile(filepath.Join(dir, hex.EncodeToString(id[:])+".regionref"), []byte("damaged"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			data, err := json.Marshal(plan)
			if err != nil {
				t.Fatal(err)
			}
			if err := ApplyReferencePlan(bytes.NewReader(data), source, dir); err == nil {
				t.Fatal("unavailable reference accepted")
			}
			if source.References[region.Name].Identity != before {
				t.Fatal("failed plan changed references")
			}
		})
	}
}
