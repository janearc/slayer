package preview

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// TranscriptName is the file beside direct-request.json that holds
// everything the child Slayer wrote, whole and in order, whether it
// succeeded or not.
const TranscriptName = "child-output.txt"

// directLane is what a direct render records for its lane: it has none,
// because Slayer is started by Ferruccio itself and not through Game.
const directLane = "none"

// writeDirectIntent preserves the explicit transport and exact foreground argv.
func (plan *Plan) writeDirectIntent(_ Tools, producer []string) error {
	width, height := plan.Profile.OutputSize()
	intent := map[string]any{"transport": "slayer", "producer": producer, "lane": directLane, "source_sha256": plan.Digest, "width": width, "height": height, "quality": plan.Profile.quality()}
	if plan.Profile.Full && plan.Profile.Samples > 0 {
		intent["samples_per_pixel"] = plan.Profile.Samples
	} else {
		intent["rays_per_axis"] = plan.Profile.samples()
	}
	raw, err := json.MarshalIndent(intent, "", "  ")
	if err != nil {
		return err
	}
	if err := retainSubmissionFile(filepath.Join(plan.Directory, "direct-request.json"), raw); err != nil {
		return err
	}
	exact, err := json.Marshal(producer)
	if err != nil {
		return err
	}
	profile, err := plan.Profile.resolved()
	if err != nil {
		return err
	}
	sampling := fmt.Sprintf("rays per axis: %d", profile.samples())
	if profile.Full && profile.Samples > 0 {
		sampling = fmt.Sprintf("samples per pixel: %d", profile.Samples)
	}
	text := fmt.Sprintf("Explicit shared-Slayer render, started directly: no Game lane.\nExact argv (JSON): %s\nRequested image: %dx%d; %s; quality: %s.\nSource text SHA256: %s\nFrozen input: %q\nAudit YAML: scene.yaml\nInvocation: direct-request.json\nMeasured image and actual executable paths: receipt.json after completion.\nProcess completion is not artistic approval.\n", exact, width, height, sampling, profile.quality(), plan.Digest, plan.Input)
	return retainSubmissionFile(filepath.Join(plan.Directory, "recipe.txt"), []byte(text))
}

// checkCurrent detects a persistent change to the referenced set whose
// film settings informed this request.
func (reference filmReference) checkCurrent() error {
	if reference.path == "" {
		return nil
	}
	file, err := os.Open(reference.path)
	if err != nil {
		return fmt.Errorf("cannot verify referenced set %q: %w", reference.path, err)
	}
	hash := sha256.New()
	_, readErr := io.Copy(hash, file)
	if err := errors.Join(readErr, file.Close()); err != nil {
		return fmt.Errorf("cannot verify referenced set %q: %w", reference.path, err)
	}
	if hex.EncodeToString(hash.Sum(nil)) != reference.digest {
		return fmt.Errorf("referenced set changed since render preparation: %s", reference.path)
	}
	return nil
}

// acceptDirect verifies the reference and output before recording success.
func (plan *Plan) acceptDirect(tools Tools, argv []string) error {
	if err := plan.Profile.reference.checkCurrent(); err != nil {
		return err
	}
	if err := plan.record(tools); err != nil {
		return err
	}
	return plan.recordDirectIdentity(tools, argv)
}

// recordDirectIdentity names the actual producer invoked, with no Game and no
// queue IDs, and says the render had no lane.
func (plan *Plan) recordDirectIdentity(tools Tools, producer []string) error {
	path := filepath.Join(plan.Directory, "receipt.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var receipt map[string]any
	if err := json.Unmarshal(raw, &receipt); err != nil {
		return err
	}
	receipt["transport"] = "slayer"
	receipt["source_shelf"], receipt["cache_directory"] = tools.Shelf, tools.Cache
	receipt["producer"], receipt["producer_arguments"] = producer[0], producer[1:]
	receipt["lane"] = directLane
	raw, err = json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0600)
}
