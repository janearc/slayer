package preview

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/janearc/puzzlemouth/yamlfile"
)

// errNoMatchingJob is a readable ledger that holds no job for this plan.
var errNoMatchingJob = errors.New("Azazel ledger has no matching preview job")

// The dispatcher normally refreshes every second; allow for a busy host.
const schedulerStaleAfter = 30 * time.Second

// decodeReply checks duplicate keys and JSON syntax before consuming owner data.
func decodeReply(raw []byte, target any) error {
	var plain map[string]any
	if err := yamlfile.Read("Azazel reply", raw, &plain); err != nil {
		return err
	}
	return json.Unmarshal(raw, target)
}

// decodeJob binds an acknowledgement or ledger job to the exact submitted request.
func decodeJob(raw []byte, request queueRequest, sequence uint64) (queueJob, error) {
	var job queueJob
	if err := decodeReply(raw, &job); err != nil {
		return job, err
	}
	var present struct {
		Executing *bool `json:"executing"`
		Result    *struct {
			ExitCode *int `json:"exit_code"`
		} `json:"result"`
	}
	if err := json.Unmarshal(raw, &present); err != nil {
		return job, err
	}
	if present.Executing == nil {
		return job, fmt.Errorf("Azazel job omits its executing state")
	}
	if job.Sequence == 0 || sequence != 0 && job.Sequence != sequence || !sameRequest(job.Request, request) {
		return job, fmt.Errorf("Azazel job does not match the submitted identity and content")
	}
	switch job.State {
	case "queued", "running":
	case "succeeded", "failed", "cancelled":
		if present.Result == nil || present.Result.ExitCode == nil {
			return job, fmt.Errorf("Azazel terminal result omits its exit status")
		}
		if !terminalJob(job) {
			return job, fmt.Errorf("Azazel terminal result is unconfirmed")
		}
	case "interrupted":
		if present.Result == nil || present.Result.ExitCode == nil || job.Result.State != "interrupted" {
			return job, fmt.Errorf("Azazel interrupted result is unconfirmed")
		}
	default:
		return job, fmt.Errorf("unknown Azazel job state %q", job.State)
	}
	return job, nil
}

// terminalJob trusts the owner-confirmed outcome; Executing is a historical claim.
func terminalJob(job queueJob) bool {
	switch job.State {
	case "succeeded", "failed", "cancelled":
		return job.Result.State == job.State && !job.Result.Uncertain
	}
	return false
}

type observedJob struct {
	job          queueJob
	raw          []byte
	paused       string
	lastSchedule time.Time
	hasSnapshot  bool
}

// readObservation filters the retained ledger by this plan's sequence and owner.
func readObservation(raw []byte, request queueRequest, sequence uint64) (observedJob, error) {
	var snapshot struct {
		Version    int    `json:"version"`
		Paused     string `json:"paused"`
		Scheduling *struct {
			Observed time.Time `json:"observed"`
		} `json:"scheduling"`
		Jobs []json.RawMessage `json:"jobs"`
	}
	if err := decodeReply(raw, &snapshot); err != nil {
		return observedJob{}, err
	}
	switch snapshot.Version {
	case 1, 2, 3:
		// All three versions keep the identity, state and result fields used here.
	default:
		return observedJob{}, fmt.Errorf("unsupported Azazel ledger version %d", snapshot.Version)
	}
	var found *observedJob
	for _, rawJob := range snapshot.Jobs {
		var identity struct {
			Sequence uint64 `json:"sequence"`
			Request  struct {
				Agent string `json:"agent"`
				Key   string `json:"key"`
			} `json:"request"`
		}
		if err := json.Unmarshal(rawJob, &identity); err != nil {
			return observedJob{}, err
		}
		matches := identity.Sequence == sequence
		if sequence == 0 {
			matches = identity.Request.Agent == request.Agent && identity.Request.Key == request.Key
		}
		if !matches {
			continue
		}
		if found != nil {
			return observedJob{}, fmt.Errorf("Azazel ledger repeats this job identity")
		}
		job, err := decodeJob(rawJob, request, sequence)
		if err != nil {
			return observedJob{}, err
		}
		found = &observedJob{job: job, raw: rawJob, paused: snapshot.Paused, hasSnapshot: true}
		if snapshot.Scheduling != nil {
			found.lastSchedule = snapshot.Scheduling.Observed
		}
	}
	if found == nil {
		return observedJob{}, errNoMatchingJob
	}
	return *found, nil
}

// update reports execution only after the owner's executing flag confirms it.
func (observation observedJob) update() Update {
	return observation.updateAt(time.Now())
}

// updateAt reports stale scheduler evidence without claiming the worker is dead.
func (observation observedJob) updateAt(now time.Time) Update {
	state := observation.job.State
	if state == "running" && !observation.job.Executing {
		state = "waiting"
	}
	held := observation.job.Held
	if held == "" && (state == "queued" || state == "waiting" || state == "running") {
		held = observation.paused
	}
	if held == "" && state == "queued" && observation.hasSnapshot && (observation.lastSchedule.IsZero() || now.Sub(observation.lastSchedule) > schedulerStaleAfter) {
		held = "No recent queue worker activity · check the queue worker"
	}
	return Update{State: state, Held: held, Sequence: observation.job.Sequence, Terminal: terminalJob(observation.job)}
}
