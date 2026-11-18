package preview

// queueRequest follows Azazel's published request schema, captured in this package.
type queueRequest struct {
	Agent          string            `json:"agent" yaml:"agent"`
	Key            string            `json:"key" yaml:"key"`
	Command        string            `json:"command" yaml:"command"`
	Args           []string          `json:"args" yaml:"args"`
	Directory      string            `json:"directory" yaml:"directory"`
	Environment    map[string]string `json:"environment,omitempty" yaml:"environment"`
	Inputs         []string          `json:"inputs,omitempty" yaml:"inputs"`
	Lane           string            `json:"lane" yaml:"lane"`
	MemoryBytes    uint64            `json:"memory_bytes,omitempty" yaml:"memory_bytes"`
	TimeoutSeconds int64             `json:"timeout_seconds" yaml:"timeout_seconds"`
}

// queueJob is the consumed subset of the owner's durable job record.
type queueJob struct {
	Attempt   uint64       `json:"attempt" yaml:"attempt"`
	Sequence  uint64       `json:"sequence" yaml:"sequence"`
	Request   queueRequest `json:"request" yaml:"request"`
	State     string       `json:"state" yaml:"state"`
	Held      string       `json:"held,omitempty" yaml:"held"`
	Executing bool         `json:"executing" yaml:"executing"`
	Result    queueResult  `json:"result" yaml:"result"`
}

// queueResult distinguishes a confirmed process outcome from an uncertain stop.
type queueResult struct {
	State     string `json:"state" yaml:"state"`
	ExitCode  int    `json:"exit_code" yaml:"exit_code"`
	Reason    string `json:"reason,omitempty" yaml:"reason"`
	Uncertain bool   `json:"uncertain,omitempty" yaml:"uncertain"`
}
