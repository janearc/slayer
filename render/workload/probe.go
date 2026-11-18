package workload

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"
)

// ProbeFrame preserves a requested frame's exact clock and all sample roles.
type ProbeFrame struct {
	Index int           `json:"frame"`
	At    time.Duration `json:"at_ns"`
	Roles []string      `json:"roles"`
}

// Probe is one scene's requested measurement plan, independent of execution.
type Probe struct {
	Shot      int          `json:"shot"`
	Scene     string       `json:"scene"`
	Kind      string       `json:"kind,omitempty"`
	Caption   string       `json:"caption"`
	Advancing bool         `json:"advancing_clock"`
	Frames    []ProbeFrame `json:"frames"`
}

// Key hashes the ordered exact frame/role plan, not an FPS reconstruction.
func (probe Probe) Key() (string, error) {
	if probe.Shot < 0 || probe.Scene == "" || len(probe.Frames) == 0 {
		return "", ErrIncomplete
	}
	for i, frame := range probe.Frames {
		if frame.Index < 0 || frame.At < 0 || len(frame.Roles) == 0 || i > 0 && (frame.Index <= probe.Frames[i-1].Index || frame.At < probe.Frames[i-1].At) {
			return "", ErrIncomplete
		}
		for _, role := range frame.Roles {
			if role == "" {
				return "", ErrIncomplete
			}
		}
	}
	raw, err := json.Marshal(struct {
		Version int
		Plan    Probe
	}{1, probe})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha256.Sum256(raw)), nil
}
