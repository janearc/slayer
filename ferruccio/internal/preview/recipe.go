package preview

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
)

// writeRecipe records exact request intent before any submission can be accepted.
func (plan *Plan) writeRecipe(tools Tools, request queueRequest) error {
	profile, err := plan.Profile.resolved()
	if err != nil {
		return err
	}
	argv := append([]string{request.Command}, request.Args...)
	exact, err := json.Marshal(argv)
	if err != nil {
		return err
	}
	renderer := "producer settings (no client override)"
	if profile.CPURenderer && plan.Kind != "catalogue" {
		renderer = "cpu (explicit request override)"
	}
	if profile.GPU && plan.Kind != "catalogue" {
		renderer = "gpu (explicit request override)"
	}
	quality := profile.quality() + " (explicit request flag)"
	if plan.Kind == "catalogue" {
		quality = "producer-authored quality (catalogue settings)"
	}
	var text bytes.Buffer
	fmt.Fprintln(&text, "Prepared preview submission; acceptance and execution require owner acknowledgement/status.")
	fmt.Fprintf(&text, "\nExact producer argv (JSON array, never a shell):\n%s\n", exact)
	fmt.Fprintf(&text, "Working directory: %q\nQueue: %q\nAgent: %s\nRequest key: %s\n", request.Directory, tools.Queue, request.Agent, request.Key)
	fmt.Fprintf(&text, "Requested image: %dx%d; rays: %d per axis.\nRequested quality: %s\nRequested renderer: %s\nLane: %s\nTimeout: %d seconds.\n", profile.Width, profile.Height, profile.samples(), quality, renderer, request.Lane, request.TimeoutSeconds)
	fmt.Fprintf(&text, "\nSource document: %q\nSource text SHA256: %s\nExecuted frozen YAML: %q\n", plan.Source, plan.Digest, plan.Input)
	fmt.Fprintln(&text, "Audit YAML: scene.yaml")
	fmt.Fprintln(&text, "Exact request and declared inputs: azazel-request.json")
	fmt.Fprintln(&text, "Owner acknowledgement and executable/input fingerprints: azazel-acknowledgement.json (when acknowledged).")
	fmt.Fprintln(&text, "Owner terminal outcome: azazel-terminal.json (when confirmed).")
	fmt.Fprintln(&text, "Measured PNG dimensions and verified producer identity: receipt.json (only after successful verification).")
	fmt.Fprintln(&text, "\nKnown limitations: requested sampling is producer intent; measured dimensions require receipt.json.")
	fmt.Fprintln(&text, "Only known direct inputs are declared; complete transitive asset/data discovery is not claimed.")
	fmt.Fprintln(&text, "No branch/build identity is inferred from client settings; worker fingerprints remain in the owner acknowledgement.")
	fmt.Fprintln(&text, "Azazel owns partial images, live pages, indexes and progress publication. This client starts no publishers.")
	return retainSubmissionFile(filepath.Join(plan.Directory, "recipe.txt"), text.Bytes())
}
