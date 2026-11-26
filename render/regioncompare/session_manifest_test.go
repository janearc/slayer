package regioncompare

import (
	"testing"

	"github.com/janearc/slayer/render/regioninput"
	"github.com/janearc/slayer/render/regionmanifest"
	"github.com/janearc/slayer/render/regionset"
)

func manifestFixture() (regionmanifest.Manifest, regionset.Region, []regioninput.Entry) {
	zero := int64(0)
	full := false
	entry := regioninput.Entry{Name: "job.yaml", Bytes: 1, SHA256: [32]byte{1}}
	region := regionset.Region{Name: "view", Left: 1, Top: 1, Width: 2, Height: 2}
	manifest := regionmanifest.Manifest{ClosureAsserted: true,
		Inputs:     []regionmanifest.Input{{Name: entry.Name, Bytes: entry.Bytes, SHA256: entry.SHA256}},
		Executable: [32]byte{2}, Width: 4, Height: 4,
		Region:   regionmanifest.Rectangle{Left: 1, Top: 1, Width: 2, Height: 2},
		Renderer: "cpu", Rays: 1, Camera: "view", TimeNS: &zero,
		Settings: [32]byte{3}, Development: [32]byte{4}, Scrooge: &full}
	return manifest, region, []regioninput.Entry{entry}
}

func TestDetailModeMustMatchTheEnvironment(t *testing.T) {
	manifest, _, _ := manifestFixture()
	t.Setenv("SLAYER_SCROOGE", "1")
	if err := matchDetail(manifest); err == nil {
		t.Fatal("accepted a different detail mode")
	}
	on := true
	manifest.Scrooge = &on
	if err := matchDetail(manifest); err != nil {
		t.Fatal(err)
	}
}

func TestTrialAndNativeGPUAreNotAdmittedByThisAdapter(t *testing.T) {
	manifest, _, _ := manifestFixture()
	t.Setenv("SLAYER_SCROOGE", "")
	t.Setenv("SLAYER_TRIAL_PLAN", "1")
	if err := matchDetail(manifest); err == nil {
		t.Fatal("trial plan was accepted")
	}
	t.Setenv("SLAYER_TRIAL_PLAN", "")
	manifest.Renderer = "gpu-native"
	if err := matchDetail(manifest); err == nil {
		t.Fatal("native GPU was accepted")
	}
}

func TestSoftwareGPUIsAdmittedByThisAdapter(t *testing.T) {
	manifest, _, _ := manifestFixture()
	t.Setenv("SLAYER_SCROOGE", "")
	t.Setenv("SLAYER_TRIAL_PLAN", "")
	manifest.Renderer = "gpu"
	if err := matchDetail(manifest); err != nil {
		t.Fatal(err)
	}
}

func TestManifestBindsTheRequestedRegionAndFiles(t *testing.T) {
	manifest, region, entries := manifestFixture()
	if err := matchManifest(manifest, region, entries, 4, 4, 1); err != nil {
		t.Fatal(err)
	}
}

func TestManifestRefusesSwappedRegionAndChangedInputs(t *testing.T) {
	manifest, region, entries := manifestFixture()
	region.Left++
	if err := matchManifest(manifest, region, entries, 4, 4, 1); err == nil {
		t.Fatal("different region accepted")
	}
	manifest, region, entries = manifestFixture()
	entries[0].SHA256[0]++
	if err := matchManifest(manifest, region, entries, 4, 4, 1); err == nil {
		t.Fatal("different snapshot accepted")
	}
	manifest, region, entries = manifestFixture()
	if err := matchManifest(manifest, region, entries, 4, 4, 2); err == nil {
		t.Fatal("different sampling accepted")
	}
}
