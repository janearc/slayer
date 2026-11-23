package preview

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strconv"
)

// Resolve asks the queue again about a job this plan submitted and whose
// observation ended without a confirmed outcome. It reports ended when the
// queue says the job finished, was cancelled, or is not in its ledger. While
// the job is still live it asks for the cancellation again and returns what
// it saw. An unreadable reply is an error and the job stays unresolved.
func (plan *Plan) Resolve(ctx context.Context, tools Tools, sequence uint64) (Update, bool, error) {
	client := queueClient{runner: processRunner{}}
	return client.resolve(ctx, tools, plan, sequence)
}

// resolve is Resolve with the client's own runner.
func (client queueClient) resolve(ctx context.Context, tools Tools, plan *Plan, sequence uint64) (Update, bool, error) {
	raw, err := os.ReadFile(filepath.Join(plan.Directory, requestName))
	if err != nil {
		return Update{}, false, err
	}
	var request queueRequest
	if err := json.Unmarshal(raw, &request); err != nil {
		return Update{}, false, err
	}
	reply, err := client.invoke(ctx, tools, plan, io.Discard, "status", "--queue", tools.Queue)
	if err != nil {
		return Update{}, false, err
	}
	observation, err := readObservation(reply, request, sequence)
	if err == errNoMatchingJob {
		return Update{Sequence: sequence}, true, nil
	}
	if err != nil {
		return Update{}, false, err
	}
	update := observation.update()
	if terminalJob(observation.job) {
		return update, true, nil
	}
	_, err = client.invoke(ctx, tools, plan, io.Discard, "cancel", "--queue", tools.Queue, "--agent", request.Agent, strconv.FormatUint(sequence, 10))
	return update, false, err
}
