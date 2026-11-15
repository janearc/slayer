package budget

type workNeed struct {
	memory   byteCount
	workers  uint32
	gpuSlots uint32
}

type workLimits struct {
	memory   byteCount
	workers  uint32
	gpuSlots uint32
}

type workReservation struct {
	id    reservationIdentity
	owner reservationOwner
	need  workNeed
}

type workLedger struct {
	limits workLimits
	held   []workReservation
}

type workDisposition uint8

const (
	workInvalid workDisposition = iota
	workDuplicate
	workCapacityHeld
	workProposed
)

type workProposal struct {
	disposition workDisposition
	next        *workLedger
}

// valid is true when the need asks for some memory, at least one worker and
// at most one GPU slot.
func (need workNeed) valid() bool {
	return need.memory != 0 && need.workers != 0 && need.gpuSlots <= 1
}

// releasing returns the ledger without the reservation id held by owner, and
// how that went. An unsound ledger or an invalid id or owner is invalid. An
// id held by another owner is an owner mismatch. An id not held is not found,
// and the ledger comes back as it was.
func (ledger workLedger) releasing(id reservationIdentity, owner reservationOwner) (workLedger, releaseDisposition) {
	if _, valid := ledger.usage(); !valid || !admissionTokenValid(string(id)) || !admissionOwnerValid(owner) {
		return workLedger{}, releaseInvalid
	}
	for index, held := range ledger.held {
		if held.id != id {
			continue
		}
		if held.owner != owner {
			return workLedger{}, releaseOwnerMismatch
		}
		entries := make([]workReservation, 0, len(ledger.held)-1)
		entries = append(entries, ledger.held[:index]...)
		entries = append(entries, ledger.held[index+1:]...)
		return workLedger{limits: ledger.limits, held: entries}, releaseProposed
	}
	return ledger, releaseNotFound
}

// usage returns the memory, workers and GPU slots that the held reservations
// add up to, and whether the ledger is sound. It is not sound with a zero
// memory or worker limit, more than 1024 held, an invalid id, owner or need,
// a repeated id, or a total over a limit.
func (ledger workLedger) usage() (workNeed, bool) {
	if ledger.limits.memory == 0 || ledger.limits.workers == 0 || len(ledger.held) > admissionEntryLimit {
		return workNeed{}, false
	}
	var used workNeed
	seen := make(map[reservationIdentity]struct{}, len(ledger.held))
	for _, held := range ledger.held {
		if !admissionTokenValid(string(held.id)) || !admissionOwnerValid(held.owner) || !held.need.valid() {
			return workNeed{}, false
		}
		if _, duplicate := seen[held.id]; duplicate {
			return workNeed{}, false
		}
		seen[held.id] = struct{}{}
		if held.need.memory > ledger.limits.memory || used.memory > ledger.limits.memory-held.need.memory || held.need.workers > ledger.limits.workers || used.workers > ledger.limits.workers-held.need.workers || held.need.gpuSlots > ledger.limits.gpuSlots || used.gpuSlots > ledger.limits.gpuSlots-held.need.gpuSlots {
			return workNeed{}, false
		}
		used.memory += held.need.memory
		used.workers += held.need.workers
		used.gpuSlots += held.need.gpuSlots
	}
	return used, true
}

// reserving proposes the ledger with request added, and leaves the receiver
// as it is. The disposition is invalid for an unsound ledger, an invalid id,
// owner or need, or a ledger already at 1024 held, duplicate if the id is
// held, and capacity held if memory, workers or GPU slots would pass a limit.
func (ledger workLedger) reserving(request workReservation) workProposal {
	used, valid := ledger.usage()
	if !valid || !admissionTokenValid(string(request.id)) || !admissionOwnerValid(request.owner) || !request.need.valid() {
		return workProposal{disposition: workInvalid}
	}
	for _, held := range ledger.held {
		if held.id == request.id {
			return workProposal{disposition: workDuplicate}
		}
	}
	if request.need.memory > ledger.limits.memory || used.memory > ledger.limits.memory-request.need.memory || request.need.workers > ledger.limits.workers || used.workers > ledger.limits.workers-request.need.workers || request.need.gpuSlots > ledger.limits.gpuSlots || used.gpuSlots > ledger.limits.gpuSlots-request.need.gpuSlots {
		return workProposal{disposition: workCapacityHeld}
	}
	if len(ledger.held) >= admissionEntryLimit {
		return workProposal{disposition: workInvalid}
	}
	entries := make([]workReservation, len(ledger.held)+1)
	copy(entries, ledger.held)
	entries[len(ledger.held)] = request
	return workProposal{disposition: workProposed, next: &workLedger{limits: ledger.limits, held: entries}}
}
