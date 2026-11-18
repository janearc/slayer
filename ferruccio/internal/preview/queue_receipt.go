package preview

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// recordQueueIdentity augments measured image evidence with its verified job.
func (plan *Plan) recordQueueIdentity(tools Tools, job queueJob) error {
	path := filepath.Join(plan.Directory, "receipt.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var receipt map[string]any
	if err := json.Unmarshal(raw, &receipt); err != nil {
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
