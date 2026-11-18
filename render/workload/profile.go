// Package workload keeps measured scene costs for subsequent render jobs.
// Observations inform scheduling; they never certify an upper memory bound.
package workload

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// Identity names all dependencies of a particular scene probe. Each field is
// a lowercase SHA256, computed from actual content by the runtime owner.
// Source includes recursive inputs, recipe, seed, detail and scene clocks;
// Build includes executable and measurement algorithm; Device includes OS,
// driver and hardware; Render includes raster, quality, backend and worker
// settings; Probe includes exact requested timestamps and measurement scope.
//
// A job picture key alone is insufficient if it omits transitive input files.
type Identity struct {
	Source string `json:"source_sha256"`
	Build  string `json:"build_sha256"`
	Device string `json:"device_sha256"`
	Render string `json:"render_sha256"`
	Probe  string `json:"probe_sha256"`
}

// Key returns a versioned, content-derived profile key, refusing partial keys.
func (identity Identity) Key() (string, error) {
	for _, field := range []struct{ name, value string }{
		{"source", identity.Source}, {"build", identity.Build},
		{"device", identity.Device}, {"render", identity.Render},
		{"probe", identity.Probe},
	} {
		decoded, err := hex.DecodeString(field.value)
		if err != nil || len(decoded) != sha256.Size || hex.EncodeToString(decoded) != field.value {
			return "", fmt.Errorf("workload: %s must be a lowercase SHA256", field.name)
		}
	}
	raw, err := json.Marshal(identity)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("v1:%x", sha256.Sum256(raw)), nil
}

// Observation is one measured frame or probe interval. Run names the scheduler
// attempt; Sample is a stable slot within that attempt, including repeat/role.
// Cache describes the observed cache state, including
// unknowns; it must not assert a hit from model cacheability alone.
// Optional measurements are nil when unavailable, not invented zeroes.
//
// CPU is consumed CPU time; GPU is measured device execution, not host wait.
// Stages can overlap and must not be added to Wall without an interval model.
// PhysicalPeak is the process lifetime high-water mark at observation time;
// it may include an earlier scene. GroupRSS and physical footprint are distinct.
type Observation struct {
	Run          string                   `json:"run"`
	Sample       string                   `json:"sample"`
	Frame        int                      `json:"frame"`
	SceneTime    time.Duration            `json:"scene_time_ns"`
	At           time.Time                `json:"observed_at"`
	Cache        string                   `json:"cache_state"`
	Wall         time.Duration            `json:"wall_ns"`
	CPU          *time.Duration           `json:"cpu_ns,omitempty"`
	GPU          *time.Duration           `json:"gpu_ns,omitempty"`
	Stages       map[string]time.Duration `json:"stages_ns,omitempty"`
	Physical     *uint64                  `json:"physical_bytes,omitempty"`
	PhysicalPeak *uint64                  `json:"physical_lifetime_peak_bytes,omitempty"`
	GroupRSS     *uint64                  `json:"group_rss_bytes,omitempty"`
	Native       *uint64                  `json:"native_bytes,omitempty"`
	Scratch      *uint64                  `json:"scratch_bytes,omitempty"`
	Pressured    bool                     `json:"pressured"`
}

// validate refuses malformed evidence before it reaches a database transaction.
func (sample Observation) validate() error {
	if sample.Run == "" || sample.Sample == "" || sample.Frame < 0 || sample.SceneTime < 0 || sample.At.IsZero() || sample.Cache == "" || sample.Wall <= 0 {
		return errors.New("workload: observation needs attempt, sample, frame, clock, timestamp, cache state and positive wall time")
	}
	if sample.CPU != nil && *sample.CPU < 0 || sample.GPU != nil && *sample.GPU < 0 {
		return errors.New("workload: measured durations cannot be negative")
	}
	for stage, elapsed := range sample.Stages {
		if stage == "" || elapsed < 0 {
			return errors.New("workload: stages need names and nonnegative measured durations")
		}
	}
	if sample.Physical != nil && sample.PhysicalPeak != nil && *sample.Physical > *sample.PhysicalPeak {
		return errors.New("workload: physical footprint exceeds its reported lifetime peak")
	}
	return nil
}

// slot is stable across retries of one scheduler attempt's measured sample.
func (sample Observation) slot() [sha256.Size]byte {
	raw, _ := json.Marshal([2]string{sample.Run, sample.Sample})
	return sha256.Sum256(raw)
}

// Profile is independent of database transaction memory. Samples from cold,
// retained and pressured runs remain separate rather than overwriting peaks.
type Profile struct {
	Identity     Identity      `json:"identity"`
	Observations []Observation `json:"observations"`
}

// Peaks are independent observed maxima, never a sum or a reservation promise.
// A nil field means no observation measured that quantity.
type Peaks struct {
	Physical, PhysicalPeak, GroupRSS, Native, Scratch *uint64
}

// Peaks reports the largest measurement of each memory quantity on record.
func (profile Profile) Peaks() Peaks {
	var peaks Peaks
	keep := func(into **uint64, value *uint64) {
		if value != nil && (*into == nil || **into < *value) {
			copy := *value
			*into = &copy
		}
	}
	for _, sample := range profile.Observations {
		keep(&peaks.Physical, sample.Physical)
		keep(&peaks.PhysicalPeak, sample.PhysicalPeak)
		keep(&peaks.GroupRSS, sample.GroupRSS)
		keep(&peaks.Native, sample.Native)
		keep(&peaks.Scratch, sample.Scratch)
	}
	return peaks
}
