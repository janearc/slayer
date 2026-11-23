// Package queue sequences foreground Slayer jobs submitted by local agents.
package queue

import (
	"errors"
	"time"
)

var (
	ErrConflict     = errors.New("agent and key already identify a different request")
	ErrOwner        = errors.New("job owner does not match")
	ErrActive       = errors.New("an active attempt requires completion or explicit recovery")
	ErrPaused       = errors.New("queue is paused")
	ErrEmpty        = errors.New("queue has no waiting jobs")
	ErrMissing      = errors.New("job does not exist")
	ErrUnsafe       = errors.New("unsafe queue location or record")
	ErrExpiredLease = errors.New("render lease expired")
	// ErrWorkerServing says recovery needs the lock a serving worker holds.
	ErrWorkerServing = errors.New("a worker is serving this queue and holds its lock; stop the worker, then recover")
)

type State string

const (
	Queued      State = "queued"
	Running     State = "running"
	Succeeded   State = "succeeded"
	Failed      State = "failed"
	Cancelled   State = "cancelled"
	Interrupted State = "interrupted"
)

// Config pins the Game executable used for every job's shared lane.
type Config struct {
	Game string `json:"game" yaml:"game"`
}

// Request wraps the existing tool's arguments without interpreting its scene.
type Request struct {
	Agent          string            `json:"agent" yaml:"agent"`
	Key            string            `json:"key" yaml:"key"`
	Command        string            `json:"command" yaml:"command"`
	Args           []string          `json:"args" yaml:"args"`
	Directory      string            `json:"directory" yaml:"directory"`
	Environment    map[string]string `json:"environment,omitempty" yaml:"environment"`
	Inputs         []string          `json:"inputs,omitempty" yaml:"inputs"`
	Lane           string            `json:"lane" yaml:"lane"`
	TimeoutSeconds int64             `json:"timeout_seconds" yaml:"timeout_seconds"`
	MemoryBytes    uint64            `json:"memory_bytes,omitempty" yaml:"memory_bytes"` // Optional estimate, never an RSS ceiling.
	// Pictures is how many pictures the job's render folder holds when the
	// whole render is done, for barkeep's total. Absent means unknown.
	Pictures *int64 `json:"pictures,omitempty" yaml:"pictures"`
}

// Fingerprint binds an input or executable to its submitted bytes.
type Fingerprint struct {
	Path   string `json:"path" yaml:"path"`
	SHA256 string `json:"sha256" yaml:"sha256"`
}

// Result records process completion separately from output verification.
type Result struct {
	Deferred  bool   `json:"deferred,omitempty" yaml:"deferred"`
	Uncertain bool   `json:"uncertain,omitempty" yaml:"uncertain"`
	Stop      bool   `json:"stop,omitempty" yaml:"stop"`
	State     State  `json:"state" yaml:"state"`
	ExitCode  int    `json:"exit_code" yaml:"exit_code"`
	Reason    string `json:"reason,omitempty" yaml:"reason"`
}

// Job is an acknowledged request and its retained execution history.
type Job struct {
	LeaseUntil               time.Time     `json:"lease_until,omitempty" yaml:"lease_until"`
	PeakResidentBytes        uint64        `json:"peak_resident_bytes" yaml:"peak_resident_bytes"`
	MemoryPressure           string        `json:"memory_pressure" yaml:"memory_pressure"`
	Attempt                  uint64        `json:"attempt" yaml:"attempt"`
	Held                     string        `json:"held,omitempty" yaml:"held"`
	Executing                bool          `json:"executing" yaml:"executing"`
	ProcessGroup             int           `json:"process_group,omitempty" yaml:"process_group"`
	ProcessStartSeconds      uint64        `json:"process_start_seconds,omitempty" yaml:"process_start_seconds"`
	ProcessStartMicroseconds uint64        `json:"process_start_microseconds,omitempty" yaml:"process_start_microseconds"`
	ProcessBootID            string        `json:"process_boot_id,omitempty" yaml:"process_boot_id"`
	Sequence                 uint64        `json:"sequence" yaml:"sequence"`
	Request                  Request       `json:"request" yaml:"request"`
	Files                    []Fingerprint `json:"files" yaml:"files"`
	State                    State         `json:"state" yaml:"state"`
	Submitted                time.Time     `json:"submitted" yaml:"submitted"`
	Started                  time.Time     `json:"started,omitempty" yaml:"started"`
	Finished                 time.Time     `json:"finished,omitempty" yaml:"finished"`
	Worker                   string        `json:"worker,omitempty" yaml:"worker"`
	CancelRequested          bool          `json:"cancel_requested" yaml:"cancel_requested"`
	Result                   Result        `json:"result" yaml:"result"`
}

// Snapshot remains readable when work is paused or requires recovery.
type Snapshot struct {
	Tuning     Tuning      `json:"tuning"`
	Version    int         `json:"version" yaml:"version"`
	Config     Config      `json:"config" yaml:"config"`
	Paused     string      `json:"paused,omitempty" yaml:"paused"`
	Jobs       []Job       `json:"jobs" yaml:"jobs"`
	Scheduling *Scheduling `json:"scheduling,omitempty" yaml:"scheduling"`
}

// Store has no held descriptors; each operation owns a short transaction.
type Store struct {
	root    string
	gpuJobs int
	// readLimit is how long supervision reads may keep failing before the
	// render is stopped; zero means defaultReadAllowance.
	readLimit time.Duration
}

// At selects the same absolute queue directory for every participating agent.
func At(root string) Store { return Store{root: root} }
