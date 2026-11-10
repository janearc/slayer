package budget

// additionalReservation is defined as the one claim appended by this operation.
const additionalReservation = 1

// memoryReservation pairs an opaque claim identity with its promised bytes.
// The identity alone proves neither owner liveness nor permission to release.
type memoryReservation struct {
	id     string
	amount byteCount
}

// reservationLedger holds proposed byte accounting, not persisted authority.
type reservationLedger struct {
	limit        byteCount
	reservations []memoryReservation
}

// reservationDisposition distinguishes a proposal from every kind of refusal.
type reservationDisposition uint8

const (
	reservationUnknown reservationDisposition = iota
	reservationInvalid
	reservationDuplicate
	reservationCapacityHeld
	reservationProposed
)

// reservationProposal has a next ledger only for a permitted value addition.
// Persistence and other admission checks are the transaction's responsibility.
type reservationProposal struct {
	next        *reservationLedger
	disposition reservationDisposition
}

// reserving validates existing claims before proposing an independent addition.
func (ledger reservationLedger) reserving(
	request memoryReservation,
) reservationProposal {
	if ledger.limit == noBytes || request.id == "" ||
		request.amount == noBytes {
		return reservationProposal{disposition: reservationInvalid}
	}
	seen := make(map[string]struct{}, len(ledger.reservations))
	used := noBytes
	for _, held := range ledger.reservations {
		if held.id == "" {
			return reservationProposal{
				disposition: reservationInvalid}
		}
		if _, duplicate := seen[held.id]; duplicate {
			return reservationProposal{
				disposition: reservationInvalid}
		}
		budget := memoryBudget{limit: ledger.limit, reserved: used}
		if !budget.admits(memoryRequest{amount: held.amount}) {
			return reservationProposal{
				disposition: reservationInvalid}
		}
		used += held.amount
		seen[held.id] = struct{}{}
	}
	if _, duplicate := seen[request.id]; duplicate {
		return reservationProposal{disposition: reservationDuplicate}
	}
	budget := memoryBudget{limit: ledger.limit, reserved: used}
	if !budget.admits(memoryRequest{amount: request.amount}) {
		return reservationProposal{disposition: reservationCapacityHeld}
	}
	entries := make([]memoryReservation,
		len(ledger.reservations)+additionalReservation)
	copy(entries, ledger.reservations)
	entries[len(ledger.reservations)] = request
	next := reservationLedger{limit: ledger.limit, reservations: entries}
	return reservationProposal{
		next: &next, disposition: reservationProposed}
}
