package budget

import "math"

// lastStopGeneration is the uint64 boundary, reserved for a final stop.
const lastStopGeneration uint64 = math.MaxUint64

// stopState holds the proposed value of a host stop record.
// Its generation names the state a resume requester inspected.
// Its zero state is clear at generation zero. It cannot authorize work.
type stopState struct {
	stopped    bool
	generation uint64
}

// stopEvent supplies a fresh decision and an optional explicit resume.
// The resume names the generation the requester inspected.
// The caller checks current conditions separately from the latched stop.
type stopEvent struct {
	observed   hostDecision
	resume     bool
	generation uint64
}

// stopOutcome explains a proposed transition. Its zero value is held.
// It is not a successful resume or confirmation of persistence.
type stopOutcome uint8

const (
	stopHeld      stopOutcome = iota // stop remains in force
	stopUnchanged                    // an already clear state stays clear
	stopLatched                      // new disk loss proposes a stop
	// stopResumed means an eligible explicit request proposes clear.
	stopResumed
	stopStale // resume names a different generation
	// stopExhausted means no generation may safely clear the stop.
	stopExhausted
)

// stopTransition holds a proposed next value and its disposition.
// Producing it writes no file and signals no process.
type stopTransition struct {
	state   stopState
	outcome stopOutcome
}

// after computes the value transition from the stop design.
// Persistence, synchronization, freshness and process actions are external.
func (state stopState) after(event stopEvent) stopTransition {
	if state.generation == lastStopGeneration {
		state.stopped = true
		return stopTransition{state: state, outcome: stopExhausted}
	}
	if event.observed == hostDiskLow {
		if state.stopped {
			return stopTransition{state: state, outcome: stopHeld}
		}
		state.stopped = true
		state.generation++
		return stopTransition{state: state, outcome: stopLatched}
	}
	if !state.stopped {
		return stopTransition{state: state, outcome: stopUnchanged}
	}
	if !event.resume || event.observed != hostReady {
		return stopTransition{state: state, outcome: stopHeld}
	}
	if event.generation != state.generation {
		return stopTransition{state: state, outcome: stopStale}
	}
	state.generation++
	if state.generation == lastStopGeneration {
		return stopTransition{state: state, outcome: stopExhausted}
	}
	state.stopped = false
	return stopTransition{state: state, outcome: stopResumed}
}
