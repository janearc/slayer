package observability

import "os"

// MemoryStagesVariable names the environment variable that turns the
// memory stage checkpoints on. Every checkpoint in slayer reads it
// through MemoryStagesEnabled, so the switch has one owner.
const MemoryStagesVariable = "SLAYER_MEMORY_STAGES"

// MemoryStagesEnabled reports whether memory stage checkpoints are on.
// They are off unless the variable is set to a non-empty value.
func MemoryStagesEnabled() bool {
	return os.Getenv(MemoryStagesVariable) != ""
}
