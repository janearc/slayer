package budget

import "testing"

func TestWorkAdmissionHoldsAndLatchesLowDisk(t *testing.T) {
	state := workAdmissionState{ledger: workLedger{limits: workLimits{memory: 100, workers: 4, gpuSlots: 1}}}
	owner := reservationOwner{name: "worker", incarnation: "incarnation"}
	request := workAdmissionRequest{policy: hostPolicy{loadLimit: 4, swapLimit: 100, diskFloor: 35}, sample: hostSample{load: 1, swap: 1, freeDisk: 34, loadKnown: true, swapKnown: true, diskKnown: true}, reservation: workReservation{id: "composite", owner: owner, need: workNeed{memory: 20, workers: 1, gpuSlots: 1}}}
	result := state.proposing(request)
	if result.disposition != admissionStopped || !result.stop.state.stopped || result.next != nil {
		t.Fatalf("low disk: %+v", result)
	}
}

func TestWorkAdmissionSharesCapacity(t *testing.T) {
	owner := reservationOwner{name: "worker", incarnation: "incarnation"}
	state := workAdmissionState{ledger: workLedger{limits: workLimits{memory: 100, workers: 4, gpuSlots: 1}}}
	request := workAdmissionRequest{policy: hostPolicy{loadLimit: 4, swapLimit: 100, diskFloor: 35}, sample: hostSample{load: 1, swap: 1, freeDisk: 40, loadKnown: true, swapKnown: true, diskKnown: true}, reservation: workReservation{id: "render", owner: owner, need: workNeed{memory: 20, workers: 2, gpuSlots: 1}}}
	first := state.proposing(request)
	if first.disposition != admissionProposed || first.next == nil {
		t.Fatalf("render: %+v", first)
	}
	state.ledger = *first.next
	request.reservation.id = "composite"
	second := state.proposing(request)
	if second.disposition != admissionReservationHeld || second.reservation != workCapacityHeld || second.next != nil {
		t.Fatalf("composite: %+v", second)
	}
}
