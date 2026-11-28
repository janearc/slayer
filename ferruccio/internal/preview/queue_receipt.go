package preview

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// verifyFilmReference checks the file identity that informed the film estimate.
func (plan *Plan) verifyFilmReference(job queueJob) error {
	reference := plan.Profile.reference
	if reference.path == "" {
		return nil
	}
	for _, file := range job.Files {
		if file.Path == reference.path && file.SHA256 == reference.digest {
			return nil
		}
	}
	return fmt.Errorf("Azazel did not verify the referenced set used for the render settings")
}

// recordQueueIdentity augments measured image evidence with its verified job.
func (plan *Plan) recordQueueIdentity(tools Tools, job queueJob) error {
	path := filepath.Join(plan.Directory, "receipt.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var receipt map[string]any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&receipt); err != nil {
		return err
	}
	delete(receipt, "game")
	delete(receipt, "slayer")
	delete(receipt, "catalogue")
	receipt["producer"], receipt["producer_arguments"] = job.Request.Command, job.Request.Args
	receipt["scheduler"] = "Azazel"
	receipt["worker_provenance"] = "azazel-acknowledgement.json"
	receipt["lane"] = job.Request.Lane
	receipt["azazel"], receipt["queue"], receipt["agent"] = tools.Azazel, tools.Queue, job.Request.Agent
	receipt["queue_sequence"], receipt["queue_key"] = job.Sequence, job.Request.Key
	raw, err = json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0600)
}
