package budget

// admissionState supplies checked current values, not proof of a held lock.
type admissionState struct {
	stop         stopState
	reservations reservationLedger
}

// admissionRequest carries supplied facts and one ordinary memory request.
// It has no resume field; ordinary observations cannot resume a latched stop.
type admissionRequest struct {
	policy      hostPolicy
	sample      hostSample
	reservation memoryReservation
}

// admissionDisposition distinguishes a value proposal from every hold.
type admissionDisposition uint8

const (
	admissionUnknown admissionDisposition = iota
	admissionStopped
	admissionHostHeld
	admissionReservationHeld
	admissionProposed
)

// admissionProposal retains the stop transition and each decision's reason.
// Only a proposed addition carries a next ledger. No result is a live grant.
type admissionProposal struct {
	stop        stopTransition
	host        hostDecision
	reservation reservationDisposition
	next        *reservationLedger
	disposition admissionDisposition
}

// proposing checks the current stop and host before identified byte capacity.
func (state admissionState) proposing(
	request admissionRequest,
) admissionProposal {
	sample := request.sample
	sample.stopped = false
	host := request.policy.assess(sample)
	stop := state.stop.after(stopEvent{observed: host})
	result := admissionProposal{stop: stop, host: host}
	if stop.state.stopped {
		result.disposition = admissionStopped
		return result
	}
	if host != hostReady {
		result.disposition = admissionHostHeld
		return result
	}
	reservation := state.reservations.reserving(request.reservation)
	result.reservation = reservation.disposition
	if reservation.disposition != reservationProposed {
		result.disposition = admissionReservationHeld
		return result
	}
	result.next = reservation.next
	result.disposition = admissionProposed
	return result
}
