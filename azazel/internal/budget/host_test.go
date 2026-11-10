package budget

import (
	_ "embed"
	"math"
	"testing"

	"github.com/janearc/puzzlemouth/yamlfile"
	"pgregory.net/rapid"
)

// authoredHostData preserves original guard examples as embedded inputs.
//
//go:embed testdata/host.yaml
var authoredHostData []byte

// noHostExamples is defined as the cardinality of an empty example group.
const noHostExamples = 0

// noMeasuredLoad is defined as the origin of nonnegative load averages.
const noMeasuredLoad = 0.0

// hostPolicyData decodes thresholds without converting byte counts to floats.
type hostPolicyData struct {
	LoadLimit float64 `yaml:"load_limit"`
	SwapLimit uint64  `yaml:"swap_limit"`
	DiskFloor uint64  `yaml:"disk_floor"`
}

// value retains invalid examples so the production guard must refuse them.
func (example hostPolicyData) value() hostPolicy {
	return hostPolicy{loadLimit: example.LoadLimit,
		swapLimit: byteCount(example.SwapLimit),
		diskFloor: byteCount(example.DiskFloor)}
}

// hostSampleData keeps availability independent of the numerical fields.
type hostSampleData struct {
	Load      float64 `yaml:"load"`
	Swap      uint64  `yaml:"swap"`
	FreeDisk  uint64  `yaml:"free_disk"`
	LoadKnown bool    `yaml:"load_known"`
	SwapKnown bool    `yaml:"swap_known"`
	DiskKnown bool    `yaml:"disk_known"`
	Stopped   bool    `yaml:"stopped"`
}

// value copies all fields, including unknown, nonfinite and stopped examples.
// It returns a hostSample with the same values.
func (example hostSampleData) value() hostSample {
	return hostSample{load: example.Load, swap: byteCount(example.Swap),
		freeDisk: byteCount(example.FreeDisk), loadKnown: example.LoadKnown,
		swapKnown: example.SwapKnown, diskKnown: example.DiskKnown,
		stopped: example.Stopped}
}

// unsafeHostData names the seven original boundary and precedence cases.
type unsafeHostData struct {
	LoadAtCeiling     hostSampleData `yaml:"load_at_ceiling"`
	SwapAtCeiling     hostSampleData `yaml:"swap_at_ceiling"`
	DiskBelowFloor    hostSampleData `yaml:"disk_below_floor"`
	LoadAboveCeiling  hostSampleData `yaml:"load_above_ceiling"`
	SwapAboveCeiling  hostSampleData `yaml:"swap_above_ceiling"`
	LoadPrecedesSwap  hostSampleData `yaml:"load_precedes_swap"`
	DiskPrecedesHolds hostSampleData `yaml:"disk_precedes_holds"`
}

// configuredHostData preserves the three caller-configured observations.
type configuredHostData struct {
	Policy        hostPolicyData `yaml:"policy"`
	Ready         hostSampleData `yaml:"ready"`
	LoadAtCeiling hostSampleData `yaml:"load_at_ceiling"`
	SwapAtCeiling hostSampleData `yaml:"swap_at_ceiling"`
}

// hostAvailability retains an original bit label and readiness witness.
type hostAvailability struct {
	Label     string `yaml:"label"`
	LoadKnown bool   `yaml:"load_known"`
	SwapKnown bool   `yaml:"swap_known"`
	DiskKnown bool   `yaml:"disk_known"`
	Ready     bool   `yaml:"ready"`
}

// hostFixtures holds authored inputs, never installed host policy.
type hostFixtures struct {
	Policy               hostPolicyData     `yaml:"policy"`
	Measured             hostSampleData     `yaml:"measured"`
	NearCeilings         hostSampleData     `yaml:"near_ceilings"`
	Unsafe               unsafeHostData     `yaml:"unsafe"`
	InvalidLoads         []float64          `yaml:"invalid_loads"`
	Configured           configuredHostData `yaml:"configured"`
	InvalidPolicies      []hostPolicyData   `yaml:"invalid_policies"`
	Availability         []hostAvailability `yaml:"availability"`
	KnownDiskWithUnknown hostSampleData     `yaml:"known_disk_with_unknown"`
	MinimumDiskFloor     uint64             `yaml:"minimum_disk_floor"`
}

// hostExamples refuses empty groups and a narrowed positive disk-floor domain.
func hostExamples(t *testing.T) hostFixtures {
	t.Helper()
	var examples hostFixtures
	if err := yamlfile.Read("testdata/host.yaml", authoredHostData,
		&examples); err != nil {
		t.Fatal(err)
	}
	if len(examples.InvalidLoads) == noHostExamples ||
		len(examples.InvalidPolicies) == noHostExamples ||
		examples.MinimumDiskFloor != minimumPositiveBytes {
		t.Fatal("authored host examples are incomplete")
	}
	near := examples.NearCeilings
	edgeLoad := math.Nextafter(examples.Policy.LoadLimit, noMeasuredLoad)
	if examples.Policy.SwapLimit < minimumPositiveBytes ||
		near.Load != edgeLoad ||
		near.Swap != examples.Policy.SwapLimit-minimumPositiveBytes ||
		near.FreeDisk != examples.Policy.DiskFloor {
		t.Fatal("authored host examples lost an exact ready boundary")
	}
	examples.requireAvailability(t)
	return examples
}

// hostKnownFlags names one member of the three-boolean availability domain.
type hostKnownFlags struct {
	load, swap, disk bool
}

// requireAvailability retains every distinct tuple and its original readiness.
func (examples hostFixtures) requireAvailability(t *testing.T) {
	t.Helper()
	seen := make(map[hostKnownFlags]struct{})
	for _, example := range examples.Availability {
		flags := hostKnownFlags{
			load: example.LoadKnown,
			swap: example.SwapKnown,
			disk: example.DiskKnown,
		}
		_, duplicate := seen[flags]
		ready := flags.load && flags.swap && flags.disk
		if duplicate || example.Ready != ready {
			t.Fatalf("invalid or repeated availability tuple: %+v", example)
		}
		seen[flags] = struct{}{}
	}
	for _, loadKnown := range []bool{false, true} {
		for _, swapKnown := range []bool{false, true} {
			for _, diskKnown := range []bool{false, true} {
				flags := hostKnownFlags{
					load: loadKnown, swap: swapKnown, disk: diskKnown,
				}
				if _, present := seen[flags]; !present {
					t.Fatalf("missing availability tuple: %+v", flags)
				}
			}
		}
	}
}

// hostObservation keeps one authored sample and its expected decision together.
type hostObservation struct {
	sample hostSample
	want   hostDecision
}

// observations retains typed expected decisions without an enum parser.
func (examples unsafeHostData) observations() map[string]hostObservation {
	return map[string]hostObservation{
		"load at ceiling": {
			examples.LoadAtCeiling.value(), hostLoadHigh},
		"swap at ceiling": {
			examples.SwapAtCeiling.value(), hostSwapHigh},
		"disk below floor": {
			examples.DiskBelowFloor.value(), hostDiskLow},
		"load above ceiling": {
			examples.LoadAboveCeiling.value(), hostLoadHigh},
		"swap above ceiling": {
			examples.SwapAboveCeiling.value(), hostSwapHigh},
		"load precedes swap": {
			examples.LoadPrecedesSwap.value(), hostLoadHigh},
		"disk stop precedes both holds": {
			examples.DiskPrecedesHolds.value(), hostDiskLow},
	}
}

// hostWitness retains the same policy the enclosing boundary test selected.
type hostWitness struct {
	policy      hostPolicy
	observation hostObservation
}

// check applies one authored observation without deriving its expected value.
func (witness hostWitness) check(t *testing.T) {
	got := witness.policy.assess(witness.observation.sample)
	if got != witness.observation.want {
		t.Errorf("got %v, want %v", got, witness.observation.want)
	}
}

// TestHostReadyAtSafeBoundaries holds the inclusive disk floor.
// It also holds the permitted side of both exclusive ceilings.
// It includes measured zeros.
func TestHostReadyAtSafeBoundaries(t *testing.T) {
	examples := hostExamples(t)
	policy := examples.Policy.value()
	for _, example := range []hostSampleData{
		examples.Measured, examples.NearCeilings,
	} {
		sample := example.value()
		if got := policy.assess(sample); got != hostReady {
			t.Errorf("safe host %+v: %v, want ready", sample, got)
		}
	}
}

// TestHostHoldsAtUnsafeBoundaries is the anti-test: an exact ceiling
// holds work. One byte below the disk floor reports a stop condition.
// An invalid load is unknown rather than favorable.
func TestHostHoldsAtUnsafeBoundaries(t *testing.T) {
	examples := hostExamples(t)
	policy := examples.Policy.value()
	for name, each := range examples.Unsafe.observations() {
		witness := hostWitness{policy: policy, observation: each}
		t.Run(name, witness.check)
	}
	for _, load := range examples.InvalidLoads {
		sample := examples.Measured.value()
		sample.load = load
		if got := policy.assess(sample); got != hostUnknown {
			t.Errorf("invalid measured load %v: %v, want unknown", load, got)
		}
	}
}

// TestHostUsesConfiguredPolicy checks that no hidden nightly defaults
// replace the caller's limits.
func TestHostUsesConfiguredPolicy(t *testing.T) {
	examples := hostExamples(t)
	policy := examples.Configured.Policy.value()
	sample := examples.Configured.Ready.value()
	if got := policy.assess(sample); got != hostReady {
		t.Errorf("configured safe host: %v, want ready", got)
	}
	sample = examples.Configured.LoadAtCeiling.value()
	if got := policy.assess(sample); got != hostLoadHigh {
		t.Errorf("configured load ceiling: %v, want load hold", got)
	}
	sample = examples.Configured.SwapAtCeiling.value()
	if got := policy.assess(sample); got != hostSwapHigh {
		t.Errorf("one-byte swap ceiling: %v, want swap hold", got)
	}
}

// TestHostRefusesInvalidPolicies is the anti-test. Omitted, nonfinite
// and nonpositive limits cannot become an unlimited policy.
func TestHostRefusesInvalidPolicies(t *testing.T) {
	examples := hostExamples(t)
	for _, example := range examples.InvalidPolicies {
		policy := example.value()
		sample := examples.Measured.value()
		if got := policy.assess(sample); got != hostInvalidPolicy {
			t.Errorf("invalid policy %+v: %v, want policy refusal", policy, got)
		}
	}
	sample := examples.Measured.value()
	sample.stopped = true
	if got := (hostPolicy{}).assess(sample); got != hostInvalidPolicy {
		t.Errorf("invalid policy and stop latch: %v, want policy refusal", got)
	}
}

// TestHostRequiresMeasuredValues checks all measurement availability
// combinations. Numbers left by a failed read grant no authority.
func TestHostRequiresMeasuredValues(t *testing.T) {
	examples := hostExamples(t)
	policy := examples.Policy.value()
	for _, example := range examples.Availability {
		sample := examples.Measured.value()
		sample.loadKnown = example.LoadKnown
		sample.swapKnown = example.SwapKnown
		sample.diskKnown = example.DiskKnown
		want := hostUnknown
		if example.Ready {
			want = hostReady
		}
		if got := policy.assess(sample); got != want {
			t.Errorf("availability %s: %v, want %v", example.Label, got, want)
		}
	}
}

// TestHostNeverTreatsMissingAsZero is the anti-test: the zero sample
// and zero decision do not mean favorable measurements or a grant.
func TestHostNeverTreatsMissingAsZero(t *testing.T) {
	examples := hostExamples(t)
	policy := examples.Policy.value()
	if got := policy.assess(hostSample{}); got != hostUnknown {
		t.Errorf("unmeasured host: %v, want unknown", got)
	}
	var omitted hostDecision
	if omitted == hostReady {
		t.Error("the zero decision grants work")
	}
}

// TestHostKeepsLatchedStop checks that favorable conditions do not
// silently resume a stopped host.
func TestHostKeepsLatchedStop(t *testing.T) {
	examples := hostExamples(t)
	policy := examples.Policy.value()
	sample := examples.Measured.value()
	sample.stopped = true
	if got := policy.assess(sample); got != hostStopped {
		t.Errorf("latched stop: %v, want stopped", got)
	}
}

// TestHostReportsKnownDiskStopBeforeUnknownMeasurements is the anti-test.
// Failed load/swap reads do not mask known disk exhaustion.
// Unknown disk bytes are never used to declare exhaustion.
func TestHostReportsKnownDiskStopBeforeUnknownMeasurements(t *testing.T) {
	examples := hostExamples(t)
	policy := examples.Policy.value()
	sample := examples.KnownDiskWithUnknown.value()
	if got := policy.assess(sample); got != hostDiskLow {
		t.Errorf("known low disk, unknown load/swap: %v, want disk stop", got)
	}
	sample.diskKnown = false
	if got := policy.assess(sample); got != hostUnknown {
		t.Errorf("unknown disk: %v, want unknown", got)
	}
	sample.stopped = true
	if got := policy.assess(sample); got != hostStopped {
		t.Errorf("stopped and unmeasured: %v, want stopped", got)
	}
	sample.diskKnown = true
	if got := policy.assess(sample); got != hostStopped {
		t.Errorf("stopped with known low disk: %v, want stopped", got)
	}
}

// hostProperty retains value copies of the original valid policy and sample.
type hostProperty struct {
	policy           hostPolicy
	measured         hostSample
	minimumDiskFloor uint64
}

// moreDisk checks monotonicity without reproducing the decision branches.
func (property hostProperty) moreDisk(t *rapid.T) {
	floor := rapid.Uint64Range(property.minimumDiskFloor,
		math.MaxUint64).Draw(t, "disk floor")
	more := rapid.Uint64Range(floor, math.MaxUint64).Draw(t, "more disk")
	policy := property.policy
	policy.diskFloor = byteCount(floor)
	sample := property.measured
	sample.freeDisk = byteCount(floor)
	if got := policy.assess(sample); got != hostReady {
		t.Fatalf("at disk floor: %v, want ready", got)
	}
	sample.freeDisk = byteCount(more)
	if got := policy.assess(sample); got != hostReady {
		t.Fatalf("increasing disk withdrew readiness: %v", got)
	}
}

// keepsStop holds the latch independently of generated numerical conditions.
func (property hostProperty) keepsStop(t *rapid.T) {
	sample := property.measured
	sample.load = rapid.Float64().Draw(t, "load")
	sample.swap = byteCount(rapid.Uint64().Draw(t, "swap"))
	sample.freeDisk = byteCount(rapid.Uint64().Draw(t, "free disk"))
	sample.stopped = true
	if got := property.policy.assess(sample); got != hostStopped {
		t.Fatalf("numeric conditions cleared stop: %v", got)
	}
}

// TestHostReadySurvivesMoreDisk checks monotonicity across the unsigned
// byte range. It does not make a second copy of the decision rules.
func TestHostReadySurvivesMoreDisk(t *testing.T) {
	examples := hostExamples(t)
	property := hostProperty{policy: examples.Policy.value(),
		measured:         examples.Measured.value(),
		minimumDiskFloor: examples.MinimumDiskFloor}
	rapid.Check(t, property.moreDisk)
}

// TestHostNeverResumesLatchedStop is the anti-test: no generated
// conditions can clear the explicitly set stop latch.
func TestHostNeverResumesLatchedStop(t *testing.T) {
	examples := hostExamples(t)
	property := hostProperty{policy: examples.Policy.value(),
		measured: examples.Measured.value()}
	rapid.Check(t, property.keepsStop)
}
