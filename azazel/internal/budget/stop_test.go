package budget

import (
	_ "embed"
	"math"
	"testing"

	"github.com/janearc/puzzlemouth/yamlfile"
	"pgregory.net/rapid"
)

// authoredStopData stores exact generation inputs with state witnesses.
//
//go:embed testdata/stop.yaml
var authoredStopData []byte

// noGenerationAdvance is defined as the difference for an unchanged counter.
const noGenerationAdvance uint64 = 0

// oneGenerationAdvance is defined as the difference to the next counter value.
const oneGenerationAdvance uint64 = 1

// initialStopGeneration is defined by the clear zero state of the protocol.
const initialStopGeneration uint64 = 0

// noStopEvents is defined as the cardinality of an empty history.
const noStopEvents = 0

// stopFixtures decodes full-width counters without a floating-point conversion.
type stopFixtures struct {
	BeforeDiskLoss             uint64   `yaml:"before_disk_loss"`
	AfterDiskLoss              uint64   `yaml:"after_disk_loss"`
	ResumeGenerations          []uint64 `yaml:"resume_generations"`
	StoppedGeneration          uint64   `yaml:"stopped_generation"`
	StaleGenerations           []uint64 `yaml:"stale_generations"`
	LastGeneration             uint64   `yaml:"last_generation"`
	BeforeLastGeneration       uint64   `yaml:"before_last_generation"`
	ClearBeforeRedundantResume uint64   `yaml:"clear_before_redundant_resume"`
	RedundantResumeGeneration  uint64   `yaml:"redundant_resume_generation"`
	MinimumEvents              int      `yaml:"minimum_events"`
	MaximumEvents              int      `yaml:"maximum_events"`
	OldResumeGeneration        uint64   `yaml:"old_resume_generation"`
	SecondStopGeneration       uint64   `yaml:"second_stop_generation"`
}

// stopExamples refuses lost unsigned boundaries or invalid history bounds.
func stopExamples(t *testing.T) stopFixtures {
	t.Helper()
	var examples stopFixtures
	if err := yamlfile.Read("testdata/stop.yaml", authoredStopData,
		&examples); err != nil {
		t.Fatal(err)
	}
	if examples.LastGeneration != math.MaxUint64 ||
		examples.BeforeLastGeneration != math.MaxUint64-oneGenerationAdvance ||
		examples.BeforeDiskLoss == math.MaxUint64 ||
		examples.AfterDiskLoss != examples.BeforeDiskLoss+oneGenerationAdvance ||
		examples.MinimumEvents <= noStopEvents ||
		examples.MaximumEvents < examples.MinimumEvents {
		t.Fatal("authored stop examples lost a boundary or event range")
	}
	examples.requireResumeGenerations(t)
	examples.requireStaleGenerations(t)
	return examples
}

// requireResumeGenerations preserves the ordinary and last eligible examples.
func (examples stopFixtures) requireResumeGenerations(t *testing.T) {
	t.Helper()
	seen := make(map[uint64]bool)
	for _, generation := range examples.ResumeGenerations {
		if seen[generation] || generation >= examples.BeforeLastGeneration {
			t.Fatalf("invalid or repeated resume generation: %d", generation)
		}
		seen[generation] = true
	}
	lastEligible := examples.BeforeLastGeneration - oneGenerationAdvance
	if examples.StoppedGeneration >= lastEligible ||
		!seen[examples.StoppedGeneration] || !seen[lastEligible] {
		t.Fatal("authored resume examples lost distinct ordinary and final cases")
	}
}

// requireStaleGenerations keeps zero, prior, later and exhausted requests.
func (examples stopFixtures) requireStaleGenerations(t *testing.T) {
	t.Helper()
	if examples.StoppedGeneration == initialStopGeneration ||
		examples.StoppedGeneration == math.MaxUint64 {
		t.Fatal("authored stopped generation needs two unsigned neighbours")
	}
	seen := make(map[uint64]bool)
	for _, generation := range examples.StaleGenerations {
		if seen[generation] || generation == examples.StoppedGeneration {
			t.Fatalf("invalid or repeated stale generation: %d", generation)
		}
		seen[generation] = true
	}
	prior := examples.StoppedGeneration - oneGenerationAdvance
	later := examples.StoppedGeneration + oneGenerationAdvance
	required := make(map[uint64]bool)
	for _, generation := range []uint64{initialStopGeneration, prior, later,
		examples.LastGeneration} {
		if required[generation] || !seen[generation] {
			t.Fatalf("missing or collapsed stale generation boundary: %d", generation)
		}
		required[generation] = true
	}
}

// TestStopLatchesKnownDiskLoss checks the first durable-state proposal:
// known low disk stops a clear host and advances its generation.
func TestStopLatchesKnownDiskLoss(t *testing.T) {
	examples := stopExamples(t)
	before := stopState{generation: examples.BeforeDiskLoss}
	want := stopTransition{state: stopState{stopped: true,
		generation: examples.AfterDiskLoss},
		outcome: stopLatched}
	if got := before.after(stopEvent{observed: hostDiskLow}); got != want {
		t.Errorf("disk loss: %+v, want %+v", got, want)
	}
}

// TestStopDiskLossDefeatsResume is the anti-test: resume during disk loss
// cannot override the loss. A repeated stop keeps state and generation.
func TestStopDiskLossDefeatsResume(t *testing.T) {
	examples := stopExamples(t)
	before := stopState{generation: examples.BeforeDiskLoss}
	event := stopEvent{observed: hostDiskLow, resume: true,
		generation: examples.BeforeDiskLoss}
	first := before.after(event)
	want := stopTransition{state: stopState{stopped: true,
		generation: examples.AfterDiskLoss},
		outcome: stopLatched}
	if first != want {
		t.Fatalf("resume during disk loss: %+v, want %+v", first, want)
	}
	want.outcome = stopHeld
	if got := first.state.after(event); got != want {
		t.Errorf("repeated disk loss: %+v, want %+v", got, want)
	}
}

// TestStopResumesExplicitReadyGeneration checks that the operator's
// current-generation request can resume a measured ready host.
func TestStopResumesExplicitReadyGeneration(t *testing.T) {
	examples := stopExamples(t)
	for _, generation := range examples.ResumeGenerations {
		before := stopState{stopped: true, generation: generation}
		want := stopTransition{state: stopState{
			generation: generation + oneGenerationAdvance}, outcome: stopResumed}
		event := stopEvent{observed: hostReady, resume: true, generation: generation}
		if got := before.after(event); got != want {
			t.Errorf("eligible resume: %+v, want %+v", got, want)
		}
	}
}

// TestStopRefusesIneligibleResume is the anti-test: failed guards,
// unknown decisions and outdated requests cannot clear a stop.
func TestStopRefusesIneligibleResume(t *testing.T) {
	examples := stopExamples(t)
	before := stopState{stopped: true, generation: examples.StoppedGeneration}
	want := stopTransition{state: before, outcome: stopHeld}
	for _, observed := range []hostDecision{hostUnknown, hostInvalidPolicy,
		hostStopped, hostLoadHigh, hostSwapHigh, hostDecision(math.MaxUint8)} {
		event := stopEvent{observed: observed, resume: true,
			generation: examples.StoppedGeneration}
		if got := before.after(event); got != want {
			t.Errorf("ineligible decision %v: %+v, want %+v", observed, got, want)
		}
	}
	want.outcome = stopStale
	for _, generation := range examples.StaleGenerations {
		event := stopEvent{observed: hostReady, resume: true, generation: generation}
		if got := before.after(event); got != want {
			t.Errorf("stale generation %v: %+v, want %+v", generation, got, want)
		}
	}
}

// TestStopObservationsPreserveState checks that favorable observations
// cannot perform an operator's resume, and ordinary holds do not invent
// a stop on an already clear host.
func TestStopObservationsPreserveState(t *testing.T) {
	examples := stopExamples(t)
	for _, stopped := range []bool{false, true} {
		before := stopState{stopped: stopped,
			generation: examples.StoppedGeneration}
		want := stopTransition{state: before, outcome: stopUnchanged}
		if stopped {
			want.outcome = stopHeld
		}
		for _, observed := range []hostDecision{hostUnknown, hostReady,
			hostLoadHigh, hostSwapHigh} {
			if got := before.after(stopEvent{observed: observed}); got != want {
				t.Errorf("observation %v, stopped %v: %+v, want %+v",
					observed, stopped, got, want)
			}
		}
	}
}

// TestStopGenerationNeverWraps is the anti-test: an exhausted sequence
// forces a stopped result, regardless of the supplied state or request.
func TestStopGenerationNeverWraps(t *testing.T) {
	examples := stopExamples(t)
	want := stopTransition{
		state:   stopState{stopped: true, generation: examples.LastGeneration},
		outcome: stopExhausted}
	for _, stopped := range []bool{false, true} {
		before := stopState{stopped: stopped, generation: examples.LastGeneration}
		for _, observed := range []hostDecision{hostUnknown, hostReady, hostDiskLow} {
			event := stopEvent{observed: observed, resume: true,
				generation: examples.LastGeneration}
			if got := before.after(event); got != want {
				t.Errorf("exhausted generation: %+v, want %+v", got, want)
			}
		}
	}
}

// TestStopKeepsTheLastGenerationForStopping checks the permitted side
// of the boundary. The last generation can record a stop.
// A resume cannot consume the last chance to stop again.
func TestStopKeepsTheLastGenerationForStopping(t *testing.T) {
	examples := stopExamples(t)
	before := stopState{generation: examples.BeforeLastGeneration}
	want := stopTransition{
		state:   stopState{stopped: true, generation: examples.LastGeneration},
		outcome: stopLatched}
	if got := before.after(stopEvent{observed: hostDiskLow}); got != want {
		t.Errorf("last generation stop: %+v, want %+v", got, want)
	}
	before.stopped = true
	want.outcome = stopExhausted
	event := stopEvent{observed: hostReady, resume: true, generation: before.generation}
	if got := before.after(event); got != want {
		t.Errorf("resume must leave a future stop generation: %+v, want %+v", got, want)
	}
}

// TestStopResumeOnAClearHostIsIdempotent is the anti-test: a redundant
// request changes neither generation nor state.
func TestStopResumeOnAClearHostIsIdempotent(t *testing.T) {
	examples := stopExamples(t)
	before := stopState{generation: examples.ClearBeforeRedundantResume}
	want := stopTransition{state: before, outcome: stopUnchanged}
	event := stopEvent{observed: hostReady, resume: true,
		generation: examples.RedundantResumeGeneration}
	if got := before.after(event); got != want {
		t.Errorf("redundant resume: %+v, want %+v", got, want)
	}
}

// stopSequenceProperty owns the generated history's independent invariants.
type stopSequenceProperty struct {
	examples stopFixtures
}

// check requires an eligible explicit resume before any stop can clear.
func (property stopSequenceProperty) check(t *rapid.T) {
	state := stopState{}
	steps := rapid.IntRange(property.examples.MinimumEvents,
		property.examples.MaximumEvents).Draw(t, "events")
	for range steps {
		event := stopEvent{
			observed: rapid.SampledFrom([]hostDecision{hostUnknown,
				hostDiskLow, hostReady, hostLoadHigh}).Draw(t, "decision"),
			resume: rapid.Bool().Draw(t, "explicit resume"),
			generation: rapid.SampledFrom([]uint64{state.generation,
				initialStopGeneration, property.examples.LastGeneration}).Draw(t,
				"expected generation"),
		}
		next := state.after(event)
		if state.stopped && !next.state.stopped &&
			(!event.resume || event.observed != hostReady ||
				event.generation != state.generation) {
			t.Fatalf("ineligible event %+v cleared %+v", event, state)
		}
		if event.observed == hostDiskLow && !next.state.stopped {
			t.Fatalf("disk loss did not latch: %+v", next)
		}
		if next.state.generation < state.generation {
			t.Fatalf("generation went backwards: %+v -> %+v", state, next)
		}
		wantAdvance := noGenerationAdvance
		if next.outcome == stopLatched || next.outcome == stopResumed ||
			(next.outcome == stopExhausted &&
				state.generation != property.examples.LastGeneration) {
			wantAdvance = oneGenerationAdvance
		}
		if advanced := next.state.generation - state.generation; advanced != wantAdvance {
			t.Fatalf("outcome %v advanced %d generations, want %d",
				next.outcome, advanced, wantAdvance)
		}
		state = next.state
	}
}

// TestStopSequencesRequireEligibleExplicitResume checks a safety invariant.
// It tests event histories without depending on branch order.
// Only an explicit resume can clear an existing stop.
// The resume must be ready and match the current generation.
func TestStopSequencesRequireEligibleExplicitResume(t *testing.T) {
	property := stopSequenceProperty{examples: stopExamples(t)}
	rapid.Check(t, property.check)
}

// TestStopSequenceRejectsAResumePreparedBeforeANewStop is the anti-test.
// Stop, resume and stop again invalidate the first stop's resume token.
func TestStopSequenceRejectsAResumePreparedBeforeANewStop(t *testing.T) {
	examples := stopExamples(t)
	state := stopState{}
	first := state.after(stopEvent{observed: hostDiskLow})
	oldResume := stopEvent{observed: hostReady, resume: true,
		generation: examples.OldResumeGeneration}
	resumed := first.state.after(oldResume)
	second := resumed.state.after(stopEvent{observed: hostDiskLow})
	if got := second.state.after(oldResume); got != (stopTransition{
		state: stopState{stopped: true,
			generation: examples.SecondStopGeneration}, outcome: stopStale}) {
		t.Errorf("outdated request cleared a new stop: %+v", got)
	}
}
