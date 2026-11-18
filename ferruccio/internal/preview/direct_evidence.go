package preview

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// writeDirectIntent preserves the explicit transport and exact foreground argv.
func (plan *Plan) writeDirectIntent(tools Tools, producer, gameArgs []string) error {
	game := append([]string{tools.Game}, gameArgs...)
	raw, err := json.MarshalIndent(map[string]any{"transport": "slayer", "producer": producer, "game": game, "source_sha256": plan.Digest, "width": plan.Profile.Width, "height": plan.Profile.Height, "rays_per_axis": plan.Profile.samples(), "quality": plan.Profile.quality()}, "", "  ")
	if err != nil {
		return err
	}
	if err := retainSubmissionFile(filepath.Join(plan.Directory, "direct-request.json"), raw); err != nil {
		return err
	}
	exact, err := json.Marshal(game)
	if err != nil {
		return err
	}
	profile, err := plan.Profile.resolved()
	if err != nil {
		return err
	}
	text := fmt.Sprintf("Explicit shared-Slayer render through Game.\nExact argv (JSON): %s\nRequested image: %dx%d; rays per axis: %d; quality: %s.\nSource text SHA256: %s\nFrozen input: %q\nAudit YAML: scene.yaml\nInvocation: direct-request.json\nMeasured image and actual executable paths: receipt.json after completion.\nProcess completion is not artistic approval.\n", exact, profile.Width, profile.Height, profile.samples(), profile.quality(), plan.Digest, plan.Input)
	return retainSubmissionFile(filepath.Join(plan.Directory, "recipe.txt"), []byte(text))
}

// recordDirectIdentity names the actual producer and Game invoked, with no queue IDs.
func (plan *Plan) recordDirectIdentity(tools Tools, producer, gameArgs []string) error {
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
	receipt["game"], receipt["game_arguments"] = tools.Game, gameArgs
	raw, err = json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0600)
}
