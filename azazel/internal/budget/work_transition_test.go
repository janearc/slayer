package budget

import (
	"crypto/sha256"
	"testing"
)

func TestWorkTransitionCarriesStopAndPredecessor(t *testing.T) {
	policy := hostPolicy{loadLimit: 4, swapLimit: 100, diskFloor: 35}
	state := workRecord{boot: "boot", policy: policy, ledger: workLedger{limits: workLimits{memory: 100, workers: 4, gpuSlots: 1}}}
	before, err := encodeWorkRecord(state)
	if err != nil {
		t.Fatal(err)
	}
	request := workAdmissionRequest{policy: policy, sample: hostSample{load: 1, swap: 1, freeDisk: 34, loadKnown: true, swapKnown: true, diskKnown: true}, reservation: workReservation{id: "render", owner: reservationOwner{name: "worker", incarnation: "boot-process"}, need: workNeed{memory: 20, workers: 2, gpuSlots: 1}}}
	transition, err := proposeWorkTransition(state, request)
	if err != nil {
		t.Fatal(err)
	}
	if transition.outcome != admissionStopped || !transition.next.stop.stopped || transition.previous != sha256.Sum256(before) || len(transition.next.ledger.held) != 0 {
		t.Fatalf("transition: %+v", transition)
	}
}

func TestWorkTransitionRejectsChangedPolicyAndUnknownHost(t *testing.T) {
	policy := hostPolicy{loadLimit: 4, swapLimit: 100, diskFloor: 35}
	state := workRecord{boot: "boot", policy: policy, ledger: workLedger{limits: workLimits{memory: 100, workers: 4, gpuSlots: 1}}}
	if _, err := proposeWorkTransition(state, workAdmissionRequest{}); err == nil {
		t.Fatal("different policy accepted")
	}
	if _, err := proposeWorkTransition(state, workAdmissionRequest{policy: policy}); err == nil {
		t.Fatal("unknown host accepted")
	}
}
