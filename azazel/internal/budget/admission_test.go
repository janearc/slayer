package budget

import (
	"fmt"
	"math/big"
	"slices"
	"testing"

	"pgregory.net/rapid"
)

// admissionFixture composes checked existing examples from their own sources.
type admissionFixture struct {
	state   admissionState
	request admissionRequest
}

// ordinaryAdmissionTest supplies an exact fit at a checked ready host boundary.
func ordinaryAdmissionTest(t *testing.T) admissionFixture {
	t.Helper()
	host, stop := hostExamples(t), stopExamples(t)
	reservation := reservationExamples(t).Ownership
	return admissionFixture{
		state: admissionState{
			stop:         stopState{generation: stop.BeforeDiskLoss},
			reservations: reservation.ledger()},
		request: admissionRequest{policy: host.Policy.value(),
			sample:      host.NearCeilings.value(),
			reservation: reservation.Request.value()},
	}
}

// checkReady requires the exact next ledger and the unchanged clear stop.
func (fixture admissionFixture) checkReady(t *testing.T) {
	t.Helper()
	before := slices.Clone(fixture.state.reservations.reservations)
	got := fixture.state.proposing(fixture.request)
	wantStop := stopTransition{state: fixture.state.stop, outcome: stopUnchanged}
	wantEntries := append(slices.Clone(before), fixture.request.reservation)
	if got.disposition != admissionProposed || got.host != hostReady ||
		got.reservation != reservationProposed || got.stop != wantStop ||
		got.next == nil || got.next.limit != fixture.state.reservations.limit ||
		!slices.Equal(got.next.reservations, wantEntries) {
		t.Fatalf("ordinary ready proposal: %+v, want entries %+v", got, wantEntries)
	}
	if !slices.Equal(fixture.state.reservations.reservations, before) {
		t.Fatal("ordinary proposal changed the input ledger")
	}
}

// TestAdmissionProposesReadyExactFit checks the complete value composition.
func TestAdmissionProposesReadyExactFit(t *testing.T) {
	ordinaryAdmissionTest(t).checkReady(t)
}

// TestAdmissionOrdinaryReadyCannotResume pairs ready host facts with a latch.
func TestAdmissionOrdinaryReadyCannotResume(t *testing.T) {
	fixture := ordinaryAdmissionTest(t)
	fixture.state.stop = stopState{stopped: true,
		generation: stopExamples(t).StoppedGeneration}
	fixture.request.sample.stopped = false
	got := fixture.state.proposing(fixture.request)
	want := stopTransition{state: fixture.state.stop, outcome: stopHeld}
	if got.disposition != admissionStopped || got.next != nil ||
		got.stop != want || got.host != hostReady {
		t.Fatalf("ordinary observation resumed a stop: %+v", got)
	}
}

// TestAdmissionUsesTheCurrentStopInsteadOfCachedSample checks a stale sample.
func TestAdmissionUsesTheCurrentStopInsteadOfCachedSample(t *testing.T) {
	fixture := ordinaryAdmissionTest(t)
	fixture.request.sample.stopped = true
	fixture.checkReady(t)
}

// TestAdmissionDiskLossProposesTheStop requires a new generation and no ledger.
func TestAdmissionDiskLossProposesTheStop(t *testing.T) {
	fixture := ordinaryAdmissionTest(t)
	fixture.request.sample = hostExamples(t).Unsafe.DiskBelowFloor.value()
	got := fixture.state.proposing(fixture.request)
	want := stopTransition{state: stopState{stopped: true,
		generation: stopExamples(t).AfterDiskLoss}, outcome: stopLatched}
	if got.disposition != admissionStopped || got.host != hostDiskLow ||
		got.next != nil || got.stop != want {
		t.Fatalf("disk stop proposal: %+v, want %+v", got, want)
	}
}

// TestAdmissionBadReservationsCannotMaskDiskLoss combines both invalid inputs.
func TestAdmissionBadReservationsCannotMaskDiskLoss(t *testing.T) {
	fixture := ordinaryAdmissionTest(t)
	fixture.request.sample = hostExamples(t).KnownDiskWithUnknown.value()
	fixture.request.reservation = memoryReservation{}
	fixture.state.reservations.reservations = []memoryReservation{{}}
	before := slices.Clone(fixture.state.reservations.reservations)
	got := fixture.state.proposing(fixture.request)
	want := stopTransition{state: stopState{stopped: true,
		generation: stopExamples(t).AfterDiskLoss}, outcome: stopLatched}
	if got.disposition != admissionStopped || got.host != hostDiskLow ||
		got.next != nil || got.stop != want {
		t.Fatalf("invalid reservation hid a known stop: %+v", got)
	}
	if !slices.Equal(fixture.state.reservations.reservations, before) {
		t.Fatal("stopping observation changed the invalid input ledger")
	}
}

// TestAdmissionExhaustedGenerationStops refuses an exhausted clear state too.
func TestAdmissionExhaustedGenerationStops(t *testing.T) {
	for _, stopped := range []bool{false, true} {
		fixture := ordinaryAdmissionTest(t)
		fixture.state.stop = stopState{stopped: stopped,
			generation: stopExamples(t).LastGeneration}
		got := fixture.state.proposing(fixture.request)
		want := stopTransition{state: stopState{stopped: true,
			generation: fixture.state.stop.generation}, outcome: stopExhausted}
		if got.disposition != admissionStopped ||
			got.next != nil || got.stop != want {
			t.Fatalf("exhausted state: %+v, want %+v", got, want)
		}
	}
}

// admissionHostWitness preserves a supplied hold reason with no reservation.
type admissionHostWitness struct{ observation hostObservation }

// check refuses resource changes and preserves the current clear generation.
func (witness admissionHostWitness) check(t *testing.T) {
	fixture := ordinaryAdmissionTest(t)
	fixture.request.sample = witness.observation.sample
	before := slices.Clone(fixture.state.reservations.reservations)
	got := fixture.state.proposing(fixture.request)
	want := stopTransition{state: fixture.state.stop, outcome: stopUnchanged}
	if got.disposition != admissionHostHeld || got.next != nil ||
		got.host != witness.observation.want || got.stop != want {
		t.Fatalf("host hold: %+v, want %v", got, witness.observation.want)
	}
	if !slices.Equal(fixture.state.reservations.reservations, before) {
		t.Fatal("host hold changed the input ledger")
	}
}

// TestAdmissionHostPressureHolds retains the authored exclusive ceilings.
func TestAdmissionHostPressureHolds(t *testing.T) {
	examples := hostExamples(t).Unsafe
	witnesses := map[string]hostObservation{
		"load": {sample: examples.LoadAtCeiling.value(), want: hostLoadHigh},
		"swap": {sample: examples.SwapAtCeiling.value(), want: hostSwapHigh},
	}
	for name, observation := range witnesses {
		witness := admissionHostWitness{observation: observation}
		t.Run(name, witness.check)
	}
}

// TestAdmissionNeedsEveryMeasurement includes all authored availability tuples.
func TestAdmissionNeedsEveryMeasurement(t *testing.T) {
	for _, availability := range hostExamples(t).Availability {
		fixture := ordinaryAdmissionTest(t)
		sample := fixture.request.sample
		sample.loadKnown, sample.swapKnown, sample.diskKnown = availability.LoadKnown,
			availability.SwapKnown, availability.DiskKnown
		if availability.Ready {
			fixture.request.sample = sample
			fixture.checkReady(t)
		} else {
			witness := admissionHostWitness{observation: hostObservation{
				sample: sample, want: hostUnknown}}
			witness.check(t)
		}
	}
}

// TestAdmissionInvalidPolicyCannotPropose pairs a ready sample with bad policy.
func TestAdmissionInvalidPolicyCannotPropose(t *testing.T) {
	for _, policy := range hostExamples(t).InvalidPolicies {
		fixture := ordinaryAdmissionTest(t)
		fixture.request.policy = policy.value()
		got := fixture.state.proposing(fixture.request)
		if got.disposition != admissionHostHeld || got.host != hostInvalidPolicy ||
			got.next != nil || got.stop.state != fixture.state.stop {
			t.Fatalf("invalid host policy: %+v", got)
		}
	}
}

// admissionReservationWitness binds an authored ledger refusal to composition.
type admissionReservationWitness struct {
	example reservationExample
	want    reservationDisposition
}

// check requires the reservation reason and unchanged input on a ready host.
func (witness admissionReservationWitness) check(t *testing.T) {
	fixture := ordinaryAdmissionTest(t)
	fixture.state.reservations = witness.example.ledger()
	fixture.request.reservation = witness.example.Request.value()
	before := slices.Clone(fixture.state.reservations.reservations)
	got := fixture.state.proposing(fixture.request)
	if got.disposition != admissionReservationHeld || got.host != hostReady ||
		got.reservation != witness.want || got.next != nil ||
		got.stop.state != fixture.state.stop {
		t.Fatalf("reservation hold: %+v, want %v", got, witness.want)
	}
	if !slices.Equal(fixture.state.reservations.reservations, before) {
		t.Fatal("refused admission changed its input ledger")
	}
}

// TestAdmissionReadyStillNeedsAnIdentifiedCapacityProposal keeps each refusal.
func TestAdmissionReadyStillNeedsAnIdentifiedCapacityProposal(t *testing.T) {
	examples := reservationExamples(t)
	groups := []struct {
		cases []reservationExample
		want  reservationDisposition
	}{
		{examples.Missing, reservationInvalid},
		{examples.Invalid, reservationInvalid},
		{examples.Duplicates, reservationDuplicate},
		{examples.Capacity, reservationCapacityHeld},
		{examples.Wrapped, reservationCapacityHeld},
	}
	for _, group := range groups {
		for _, example := range group.cases {
			witness := admissionReservationWitness{example: example, want: group.want}
			t.Run(example.Name, witness.check)
		}
	}
}

// TestAdmissionProposalDoesNotAliasInput checks the actual composing result.
func TestAdmissionProposalDoesNotAliasInput(t *testing.T) {
	fixture := ordinaryAdmissionTest(t)
	before := slices.Clone(fixture.state.reservations.reservations)
	got := fixture.state.proposing(fixture.request)
	if got.disposition != admissionProposed || got.next == nil {
		t.Fatalf("independent admission proposal: %+v", got)
	}
	got.next.reservations[firstReservation].id = "changed proposal"
	if !slices.Equal(fixture.state.reservations.reservations, before) {
		t.Fatal("composed proposal shares input reservation storage")
	}
}

// TestAdmissionZeroProposalGrantsNothing refuses the omitted-result boundary.
func TestAdmissionZeroProposalGrantsNothing(t *testing.T) {
	zero := admissionProposal{}
	if zero.disposition == admissionProposed || zero.next != nil {
		t.Fatal("zero result claims ordinary admission")
	}
}

// admissionHistoryProperty carries checked samples and existing history bounds.
type admissionHistoryProperty struct {
	fixture admissionFixture
	samples []hostSample
	bounds  stopFixtures
}

// check independently latches the first disk loss without any resume operation.
func (property admissionHistoryProperty) check(trial *rapid.T) {
	state := property.fixture.state
	state.reservations.reservations = slices.Clone(state.reservations.reservations)
	steps := rapid.SliceOfN(rapid.SampledFrom(property.samples),
		property.bounds.MinimumEvents, property.bounds.MaximumEvents).
		Draw(trial, "history")
	stopped := false
	generation := state.stop.generation
	for index, sample := range steps {
		lowDisk := sample.diskKnown &&
			sample.freeDisk < property.fixture.request.policy.diskFloor
		if lowDisk && !stopped {
			stopped = true
			generation += oneGenerationAdvance
		}
		request := property.fixture.request
		request.sample = sample
		request.reservation.id = fmt.Sprintf("observation-%d", index)
		got := state.proposing(request)
		if got.stop.state.stopped != stopped ||
			got.stop.state.generation != generation {
			trial.Fatalf("observation lost stop history: %+v", got)
		}
		if stopped && (got.disposition != admissionStopped || got.next != nil) {
			trial.Fatalf("ordinary observation resumed after disk loss: %+v", got)
		}
		state.stop = got.stop.state
		if got.next != nil {
			state.reservations = *got.next
		}
	}
}

// TestAdmissionObservationHistoriesCannotResume is the latched-stop property.
func TestAdmissionObservationHistoriesCannotResume(t *testing.T) {
	host := hostExamples(t)
	unknown := host.NearCeilings.value()
	unknown.loadKnown = false
	property := admissionHistoryProperty{fixture: ordinaryAdmissionTest(t),
		bounds: stopExamples(t), samples: []hostSample{host.NearCeilings.value(),
			host.Unsafe.DiskBelowFloor.value(),
			host.Unsafe.LoadAtCeiling.value(), unknown}}
	rapid.Check(t, property.check)
}

// admissionCapacityProperty compares the ready-host composition with a big sum.
type admissionCapacityProperty struct{ fixture admissionFixture }

// check varies full-width byte counts and a deliberately stale sample stop bit.
func (property admissionCapacityProperty) check(trial *rapid.T) {
	state, request := property.fixture.state, property.fixture.request
	limit := rapid.Uint64().Draw(trial, "limit")
	held := rapid.Uint64().Draw(trial, "held")
	amount := rapid.Uint64().Draw(trial, "requested")
	state.reservations = reservationLedger{limit: byteCount(limit)}
	if held != noRequestedBytes {
		state.reservations.reservations = []memoryReservation{
			{id: "held", amount: byteCount(held)}}
	}
	request.reservation = memoryReservation{id: "new", amount: byteCount(amount)}
	request.sample.stopped = rapid.Bool().Draw(trial, "cached stop")
	left, right := new(big.Int).SetUint64(held), new(big.Int).SetUint64(amount)
	total := new(big.Int).Add(left, right)
	want, disposition := reservationCapacityHeld, admissionReservationHeld
	if limit == noRequestedBytes || amount == noRequestedBytes || held > limit {
		want = reservationInvalid
	} else if total.Cmp(new(big.Int).SetUint64(limit)) <= equalIntegerComparison {
		want, disposition = reservationProposed, admissionProposed
	}
	got := state.proposing(request)
	hasProposal := disposition == admissionProposed
	if got.disposition != disposition || got.reservation != want ||
		got.host != hostReady || (got.next != nil) != hasProposal ||
		got.stop.state != state.stop {
		trial.Fatalf("capacity proposal: %+v, want %v/%v", got, disposition, want)
	}
}

// TestAdmissionReadyCapacityMatchesAnUnboundedSum uses the independent oracle.
func TestAdmissionReadyCapacityMatchesAnUnboundedSum(t *testing.T) {
	property := admissionCapacityProperty{fixture: ordinaryAdmissionTest(t)}
	rapid.Check(t, property.check)
}
