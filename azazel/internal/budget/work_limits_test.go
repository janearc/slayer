package budget

import "testing"

func TestWorkLedgerSharesMemoryWorkersAndGPU(t *testing.T) {
	owner := reservationOwner{name: "worker", incarnation: "boot-process"}
	ledger := workLedger{limits: workLimits{memory: 100, workers: 4, gpuSlots: 1}}
	first := ledger.reserving(workReservation{id: "render", owner: owner, need: workNeed{memory: 40, workers: 2, gpuSlots: 1}})
	if first.disposition != workProposed || first.next == nil {
		t.Fatalf("first reservation: %+v", first)
	}
	if len(ledger.held) != 0 {
		t.Fatal("original ledger changed")
	}
	second := first.next.reserving(workReservation{id: "composite", owner: owner, need: workNeed{memory: 30, workers: 2, gpuSlots: 1}})
	if second.disposition != workCapacityHeld {
		t.Fatalf("GPU overcommit: %+v", second)
	}
	cpu := first.next.reserving(workReservation{id: "cpu", owner: owner, need: workNeed{memory: 60, workers: 2}})
	if cpu.disposition != workProposed {
		t.Fatalf("fitting CPU work: %+v", cpu)
	}
}

func TestWorkLedgerReleaseRequiresMatchingOwner(t *testing.T) {
	owner := reservationOwner{name: "worker", incarnation: "boot-process"}
	other := reservationOwner{name: "worker", incarnation: "replacement"}
	ledger := workLedger{limits: workLimits{memory: 100, workers: 4, gpuSlots: 1}}
	proposal := ledger.reserving(workReservation{id: "render", owner: owner, need: workNeed{memory: 40, workers: 2, gpuSlots: 1}})
	if _, status := proposal.next.releasing("render", other); status != releaseOwnerMismatch {
		t.Fatalf("wrong owner: %v", status)
	}
	released, status := proposal.next.releasing("render", owner)
	if status != releaseProposed || len(released.held) != 0 || len(proposal.next.held) != 1 {
		t.Fatalf("release: %v %+v", status, released)
	}
	if _, status := released.releasing("render", owner); status != releaseNotFound {
		t.Fatalf("repeated release: %v", status)
	}
}

func TestWorkLedgerRefusesInvalidAndDuplicateClaims(t *testing.T) {
	owner := reservationOwner{name: "worker", incarnation: "boot-process"}
	ledger := workLedger{limits: workLimits{memory: 100, workers: 4, gpuSlots: 1}}
	request := workReservation{id: "job", owner: owner, need: workNeed{memory: 10, workers: 1}}
	first := ledger.reserving(request)
	if first.next == nil {
		t.Fatalf("initial reservation: %+v", first)
	}
	if first.next.reserving(request).disposition != workDuplicate {
		t.Fatal("duplicate accepted")
	}
	request.need.memory = 0
	if ledger.reserving(request).disposition != workInvalid {
		t.Fatal("zero memory accepted")
	}
	request.need.memory = 101
	if ledger.reserving(request).disposition != workCapacityHeld {
		t.Fatal("over-capacity accepted")
	}
}
