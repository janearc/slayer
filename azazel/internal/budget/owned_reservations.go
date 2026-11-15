package budget

type reservationOwnerName string

type reservationOwnerIdentity string

type reservationIdentity string

type reservationOwner struct {
	name        reservationOwnerName
	incarnation reservationOwnerIdentity
}

type ownedMemoryReservation struct {
	id     reservationIdentity
	owner  reservationOwner
	amount byteCount
}

type ownedReservationLedger struct {
	limit        byteCount
	reservations []ownedMemoryReservation
}

type ownedReservationProposal struct {
	next        *ownedReservationLedger
	disposition reservationDisposition
}

type reservationRelease struct {
	id    reservationIdentity
	owner reservationOwner
}

type releaseDisposition uint8

const (
	releaseUnknown releaseDisposition = iota
	releaseInvalid
	releaseNotFound
	releaseOwnerMismatch
	releaseProposed
)

type releaseProposal struct {
	next        *ownedReservationLedger
	disposition releaseDisposition
}

// valid is true when the owner has both a name and an incarnation.
func (owner reservationOwner) valid() bool {
	return owner.name != "" && owner.incarnation != ""
}

// validatedUsage returns the bytes the ledger's reservations add up to, and
// whether the ledger is sound. It is not sound if the limit is zero, a
// reservation has no id or no valid owner, an id repeats, or a reservation
// does not fit under the limit beside those before it.
func (ledger ownedReservationLedger) validatedUsage() (byteCount, bool) {
	if ledger.limit == noBytes {
		return noBytes, false
	}
	seen := make(map[reservationIdentity]struct{}, len(ledger.reservations))
	used := noBytes
	for _, held := range ledger.reservations {
		if held.id == "" || !held.owner.valid() {
			return noBytes, false
		}
		if _, duplicate := seen[held.id]; duplicate {
			return noBytes, false
		}
		budget := memoryBudget{limit: ledger.limit, reserved: used}
		if !budget.admits(memoryRequest{amount: held.amount}) {
			return noBytes, false
		}
		used += held.amount
		seen[held.id] = struct{}{}
	}
	return used, true
}

// reserving proposes the ledger with request added, and leaves the receiver
// as it is. The disposition is invalid for an empty id, an invalid owner, a
// zero amount or an unsound ledger, duplicate if the id is already held, and
// capacity held if the amount does not fit under the limit.
func (ledger ownedReservationLedger) reserving(request ownedMemoryReservation) ownedReservationProposal {
	if request.id == "" || !request.owner.valid() || request.amount == noBytes {
		return ownedReservationProposal{disposition: reservationInvalid}
	}
	used, valid := ledger.validatedUsage()
	if !valid {
		return ownedReservationProposal{disposition: reservationInvalid}
	}
	for _, held := range ledger.reservations {
		if held.id == request.id {
			return ownedReservationProposal{disposition: reservationDuplicate}
		}
	}
	budget := memoryBudget{limit: ledger.limit, reserved: used}
	if !budget.admits(memoryRequest{amount: request.amount}) {
		return ownedReservationProposal{disposition: reservationCapacityHeld}
	}
	entries := make([]ownedMemoryReservation, len(ledger.reservations)+additionalReservation)
	copy(entries, ledger.reservations)
	entries[len(ledger.reservations)] = request
	next := ownedReservationLedger{limit: ledger.limit, reservations: entries}
	return ownedReservationProposal{next: &next, disposition: reservationProposed}
}

// releasing proposes the ledger without the reservation that request names,
// and leaves the receiver as it is. An empty id, an invalid owner or an
// unsound ledger is invalid, an id held by another owner is an owner
// mismatch, and an id not held is not found.
func (ledger ownedReservationLedger) releasing(request reservationRelease) releaseProposal {
	if request.id == "" || !request.owner.valid() {
		return releaseProposal{disposition: releaseInvalid}
	}
	if _, valid := ledger.validatedUsage(); !valid {
		return releaseProposal{disposition: releaseInvalid}
	}
	for index, held := range ledger.reservations {
		if held.id != request.id {
			continue
		}
		if held.owner != request.owner {
			return releaseProposal{disposition: releaseOwnerMismatch}
		}
		entries := make([]ownedMemoryReservation, len(ledger.reservations)-1)
		copy(entries, ledger.reservations[:index])
		copy(entries[index:], ledger.reservations[index+1:])
		next := ownedReservationLedger{limit: ledger.limit, reservations: entries}
		return releaseProposal{next: &next, disposition: releaseProposed}
	}
	return releaseProposal{disposition: releaseNotFound}
}
