//go:build draw

package regioncompare

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"image"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/render/regioninput"
	"github.com/janearc/slayer/render/regionmanifest"
	"github.com/janearc/slayer/render/regionref"
	"github.com/janearc/slayer/render/regionset"
	"github.com/janearc/slayer/session"
)

const smallSet = `scene:
  name: small
  light: ambient
  film: {width: 4, height: 4, rays: 1, wide: 45, exposure: 1}
  cameras:
    view:
      location: {at: [0, -5, 1]}
      orientation: {look_at: [0, 0, 1]}
  world:
    materials:
      matte: {colour: {srgb_linear: [0.3, 0.4, 0.5]}, source: eye}
    boxes:
      object:
        material: matte
        middle: [0, 0, 1]
        size: [1, 1, 1]
        source: eye
`

const smallJob = `"$schema": https://github.com/janearc/slayer/schemas/job-v1.json
scene: {set: set.yaml}
camera: {name: view}
film: {width: 4, height: 4, rays: 1}
renderer: cpu
outputs: {picture: picture.png}
`

func sessionFixture(t *testing.T) (*SessionSource, regionset.Region) {
	t.Helper()
	root := t.TempDir()
	shelf := filepath.Join(root, "shelf")
	if err := os.Mkdir(shelf, 0700); err != nil {
		t.Fatal(err)
	}
	cache := t.TempDir()
	t.Setenv("SLAYER_SHELF", shelf)
	t.Setenv("SLAYER_CACHE", cache)
	t.Setenv("SLAYER_SCROOGE", "")
	t.Setenv("SLAYER_TRIAL_PLAN", "")
	var entries []regioninput.Entry
	for name, text := range map[string]string{"job.yaml": smallJob, "set.yaml": smallSet} {
		bytes := []byte(text)
		if err := os.WriteFile(filepath.Join(root, name), bytes, 0600); err != nil {
			t.Fatal(err)
		}
		entries = append(entries, regioninput.Entry{Name: name, Bytes: int64(len(bytes)), SHA256: sha256.Sum256(bytes)})
	}
	wanted, err := job.Read(filepath.Join(root, "job.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	box := image.Rect(1, 1, 3, 3)
	wanted, err = wanted.With(job.Width(4), job.Height(4), job.Rays(1), job.Region(box))
	if err != nil {
		t.Fatal(err)
	}
	opened, err := session.Open(context.Background(), wanted, session.Stamped("fixture", "", "", ""))
	if err != nil {
		t.Fatal(err)
	}
	development, err := ToneDigest(opened.Tone())
	if err != nil {
		t.Fatal(err)
	}
	if err := opened.Close(); err != nil {
		t.Fatal(err)
	}
	exe, err := executableDigest()
	if err != nil {
		t.Fatal(err)
	}
	var inputs []regionmanifest.Input
	for _, entry := range entries {
		inputs = append(inputs, regionmanifest.Input{Name: entry.Name, Bytes: entry.Bytes, SHA256: entry.SHA256})
	}
	zero := int64(0)
	full := false
	manifest := regionmanifest.Manifest{ClosureAsserted: true, Inputs: inputs, Executable: exe,
		Width: 4, Height: 4, Region: regionmanifest.Rectangle{Left: 1, Top: 1, Width: 2, Height: 2},
		Renderer: "cpu", Rays: 1, Camera: "view", TimeNS: &zero, Settings: JobSettingsDigest(wanted), Development: development, Scrooge: &full}
	candidate, err := regionmanifest.Digest(manifest)
	if err != nil {
		t.Fatal(err)
	}
	refManifest := manifest
	refManifest.Rays = 2
	refWanted, err := wanted.With(job.Rays(2))
	if err != nil {
		t.Fatal(err)
	}
	refManifest.Settings = JobSettingsDigest(refWanted)
	refID, err := regionmanifest.Digest(refManifest)
	if err != nil {
		t.Fatal(err)
	}
	refPath := filepath.Join(t.TempDir(), "reference")
	light := make([]byte, 2*2*3*4)
	for at := 0; at < len(light); at += 4 {
		binary.LittleEndian.PutUint32(light[at:], math.Float32bits(0.25))
	}
	if err := regionref.Publish(refPath, regionref.Identity(refID), 2, 2, light); err != nil {
		t.Fatal(err)
	}
	source := &SessionSource{Context: context.Background(), Stamp: session.Stamped("fixture", "", "", ""), Snapshot: root, Entries: entries,
		JobName: "job.yaml", JobFile: "job.yaml", Shelf: shelf, Cache: cache,
		Width: 4, Height: 4, Rays: 1, Candidates: map[string]Identity{"box": Identity(candidate)},
		Manifests:  map[string]regionmanifest.Manifest{"box": manifest},
		References: map[string]ReferenceFile{"box": {Identity: Identity(refID), Manifest: refManifest, Path: refPath, Rays: 2, MaxBytes: 1024}}}
	return source, regionset.Region{Name: "box", Observation: "the box", Left: 1, Top: 1, Width: 2, Height: 2}
}

func TestSessionSourceRequiresAnExplicitBuildStamp(t *testing.T) {
	source, region := sessionFixture(t)
	source.Stamp = session.Stamp{}
	if _, err := source.Draw("job.yaml", region, Identity{1}); err == nil {
		t.Fatal("a source with no build stamp was accepted")
	}
}

func TestSessionSourceDrawsAndReadsThePinnedPair(t *testing.T) {
	source, region := sessionFixture(t)
	drawn, err := source.Draw("job.yaml", region, source.Candidates["box"])
	if err != nil {
		t.Fatal(err)
	}
	if drawn.Identity != source.Candidates["box"] || drawn.Rays != 1 || drawn.Light.W != 2 || drawn.Light.H != 2 ||
		drawn.View.Bounds().Dx() != 2 || drawn.View.Bounds().Dy() != 2 {
		t.Fatalf("candidate: %+v", drawn)
	}
	ref, err := source.Reference("job.yaml", region, source.References["box"].Identity)
	if err != nil {
		t.Fatal(err)
	}
	if ref.Identity != source.References["box"].Identity || ref.Rays != 2 || ref.Light.At(0, 0).R != 0.25 ||
		ref.View.Bounds().Dx() != 2 || ref.View.Bounds().Dy() != 2 {
		t.Fatalf("reference: %+v", ref)
	}
}

func TestSessionSourceDrawsReferenceWithoutReplacingIt(t *testing.T) {
	source, region := sessionFixture(t)
	ref := source.References["box"]
	ref.Path = filepath.Join(t.TempDir(), "drawn-reference")
	source.References["box"] = ref
	if err := source.EnsureReference("job.yaml", region, ref.Identity); err == nil {
		t.Fatal("reference draw before candidate was accepted")
	}
	if _, err := source.Draw("job.yaml", region, source.Candidates["box"]); err != nil {
		t.Fatal(err)
	}
	if err := source.EnsureReference("job.yaml", region, ref.Identity); err != nil {
		t.Fatal(err)
	}
	picture, err := source.Reference("job.yaml", region, ref.Identity)
	if err != nil {
		t.Fatal(err)
	}
	if picture.Rays != 2 || picture.Light.W != 2 || picture.Light.H != 2 {
		t.Fatal("wrong reference")
	}
	before, err := os.ReadFile(ref.Path)
	if err != nil {
		t.Fatal(err)
	}
	if err := source.EnsureReference("job.yaml", region, ref.Identity); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(ref.Path)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("reference changed")
	}
	if err := os.WriteFile(ref.Path, []byte("damaged"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := source.EnsureReference("job.yaml", region, ref.Identity); err == nil {
		t.Fatal("damaged reference was silently replaced")
	}
}

func TestRunDrawsAndReusesSessionReference(t *testing.T) {
	source, _ := sessionFixture(t)
	ref := source.References["box"]
	ref.Path = filepath.Join(t.TempDir(), "drawn-reference")
	source.References["box"] = ref
	set := `job: job.yaml
threshold: 0.1
regions:
  - {name: box, observation: box, left: 1, top: 1, width: 2, height: 2}
`
	for i := 0; i < 2; i++ {
		result, err := Run(strings.NewReader(set), 4, 4, source.Candidates, map[string]Identity{"box": ref.Identity}, source)
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Regions) != 1 || result.Sheet == nil || result.Regions[0].ReferenceRays != 2 {
			t.Fatalf("comparison %d: %+v", i, result)
		}
	}
}

func TestSessionSourceRejectsChangedSnapshotInput(t *testing.T) {
	source, region := sessionFixture(t)
	if err := os.WriteFile(filepath.Join(source.Snapshot, "set.yaml"), []byte(smallSet+"\n# changed\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := source.Draw("job.yaml", region, source.Candidates["box"]); err == nil {
		t.Fatal("draw accepted a changed snapshot input")
	}
}

func TestFailedDrawCannotReusePreviousTone(t *testing.T) {
	source, region := sessionFixture(t)
	if _, err := source.Draw("job.yaml", region, source.Candidates["box"]); err != nil {
		t.Fatal(err)
	}
	if _, err := source.Draw("job.yaml", region, Identity{9}); err == nil {
		t.Fatal("wrong identity was accepted")
	}
	if _, err := source.Reference("job.yaml", region, source.References["box"].Identity); err == nil {
		t.Fatal("reference reused the prior successful draw's tone")
	}
}

func TestSessionSourceRejectsChangedExecutableIdentity(t *testing.T) {
	source, region := sessionFixture(t)
	manifest := source.Manifests["box"]
	manifest.Executable[0] ^= 1
	id, err := regionmanifest.Digest(manifest)
	if err != nil {
		t.Fatal(err)
	}
	source.Manifests["box"] = manifest
	source.Candidates["box"] = Identity(id)
	if _, err := source.Draw("job.yaml", region, Identity(id)); err == nil {
		t.Fatal("different executable was accepted")
	}
}

func TestSessionSourceRejectsReferenceSamplingChange(t *testing.T) {
	source, region := sessionFixture(t)
	if _, err := source.Draw("job.yaml", region, source.Candidates["box"]); err != nil {
		t.Fatal(err)
	}
	ref := source.References["box"]
	ref.Rays++
	source.References["box"] = ref
	if _, err := source.Reference("job.yaml", region, ref.Identity); err == nil {
		t.Fatal("reference rays inconsistent with manifest were accepted")
	}
}

func TestSessionSourceRejectsReferenceDetailModeChange(t *testing.T) {
	source, region := sessionFixture(t)
	if _, err := source.Draw("job.yaml", region, source.Candidates["box"]); err != nil {
		t.Fatal(err)
	}
	ref := source.References["box"]
	on := true
	ref.Manifest.Scrooge = &on
	id, err := regionmanifest.Digest(ref.Manifest)
	if err != nil {
		t.Fatal(err)
	}
	ref.Identity = Identity(id)
	source.References["box"] = ref
	if _, err := source.Reference("job.yaml", region, ref.Identity); err == nil {
		t.Fatal("reference with different detail mode was accepted")
	}
}

func TestSessionSourceRejectsReferenceProvenanceChanges(t *testing.T) {
	for _, change := range []struct {
		name  string
		apply func(*regionmanifest.Manifest)
	}{
		{"input", func(m *regionmanifest.Manifest) { m.Inputs[0].SHA256[0] ^= 1 }},
		{"settings", func(m *regionmanifest.Manifest) { m.Settings[0] ^= 1 }},
		{"renderer", func(m *regionmanifest.Manifest) { m.Renderer = "gpu" }},
	} {
		t.Run(change.name, func(t *testing.T) {
			source, region := sessionFixture(t)
			if _, err := source.Draw("job.yaml", region, source.Candidates["box"]); err != nil {
				t.Fatal(err)
			}
			ref := source.References["box"]
			ref.Manifest.Inputs = append([]regionmanifest.Input(nil), ref.Manifest.Inputs...)
			change.apply(&ref.Manifest)
			id, err := regionmanifest.Digest(ref.Manifest)
			if err != nil {
				t.Fatal(err)
			}
			ref.Identity = Identity(id)
			source.References["box"] = ref
			if _, err := source.Reference("job.yaml", region, ref.Identity); err == nil {
				t.Fatal("reference with changed provenance was accepted")
			}
		})
	}
}

func TestSessionSourceCanReadAnAttestedReferenceFromAnotherBuild(t *testing.T) {
	source, region := sessionFixture(t)
	if _, err := source.Draw("job.yaml", region, source.Candidates["box"]); err != nil {
		t.Fatal(err)
	}
	ref := source.References["box"]
	ref.Manifest.Executable[0] ^= 1
	id, err := regionmanifest.Digest(ref.Manifest)
	if err != nil {
		t.Fatal(err)
	}
	ref.Identity = Identity(id)
	ref.Path = filepath.Join(t.TempDir(), "other-build-reference")
	source.References["box"] = ref
	if err := source.EnsureReference("job.yaml", region, ref.Identity); err == nil {
		t.Fatal("current build generated a reference claiming a different executable")
	}
	light := make([]byte, region.Width*region.Height*12)
	if err := regionref.Publish(ref.Path, regionref.Identity(id), region.Width, region.Height, light); err != nil {
		t.Fatal(err)
	}
	if _, err := source.Reference("job.yaml", region, ref.Identity); err != nil {
		t.Fatal(err)
	}
}
