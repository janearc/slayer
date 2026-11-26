package regioncompare

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/maths/strategies/scrooge"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/render/regioninput"
	"github.com/janearc/slayer/render/regionmanifest"
	"github.com/janearc/slayer/render/regionset"
)

// matchDetail refuses detail modes that differ from the declared draw.
func matchDetail(m regionmanifest.Manifest) error {
	if m.Scrooge == nil || *m.Scrooge != scrooge.Asked() {
		return fmt.Errorf("regioncompare: scrooge detail mode differs")
	}
	if os.Getenv("SLAYER_TRIAL_PLAN") == "1" {
		return fmt.Errorf("regioncompare: trial plan is not supported")
	}
	// The software GPU region drawer checks its captured diagnostics itself.
	if m.Renderer != "cpu" && m.Renderer != "gpu" {
		return fmt.Errorf("regioncompare: only cpu and software gpu are supported")
	}
	return nil
}

// JobSettingsDigest identifies the actual overridden job's canonical settings.
// Snapshot entries and the executable must also be bound separately.
func JobSettingsDigest(wanted job.Job) [32]byte { return sha256.Sum256(wanted.Canonical()) }

// ToneDigest identifies the actual session tone used for both views.
func ToneDigest(tone colour.Tone) ([32]byte, error) {
	bytes, err := json.Marshal(tone)
	if err != nil {
		return [32]byte{}, err
	}
	return sha256.Sum256(bytes), nil
}

// executableDigest hashes the running executable.
func executableDigest() ([32]byte, error) {
	path, err := os.Executable()
	if err != nil {
		return [32]byte{}, err
	}
	f, err := os.Open(path)
	if err != nil {
		return [32]byte{}, err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return [32]byte{}, err
	}
	var result [32]byte
	copy(result[:], h.Sum(nil))
	return result, nil
}

// matchManifest checks a request and its snapshot entries against a manifest.
func matchManifest(m regionmanifest.Manifest, region regionset.Region,
	entries []regioninput.Entry, width, height, rays int) error {
	if _, err := regionmanifest.Digest(m); err != nil {
		return err
	}
	box := m.Region
	if m.Width != width || m.Height != height || m.Rays != rays ||
		box.Left != region.Left || box.Top != region.Top || box.Width != region.Width || box.Height != region.Height ||
		len(m.Inputs) != len(entries) {
		return fmt.Errorf("regioncompare: manifest request differs")
	}
	listed := make(map[string]regioninput.Entry, len(entries))
	for _, entry := range entries {
		if _, exists := listed[entry.Name]; exists {
			return fmt.Errorf("regioncompare: duplicate snapshot entry")
		}
		listed[entry.Name] = entry
	}
	for _, input := range m.Inputs {
		entry, found := listed[input.Name]
		if !found || input.Bytes != entry.Bytes || input.SHA256 != entry.SHA256 {
			return fmt.Errorf("regioncompare: snapshot manifest differs")
		}
	}
	return nil
}
