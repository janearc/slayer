package budget

type workAdmissionState struct {
	stop   stopState
	ledger workLedger
}

type workAdmissionRequest struct {
	policy      hostPolicy
	sample      hostSample
	reservation workReservation
}

type workAdmissionProposal struct {
	stop        stopTransition
	host        hostDecision
	reservation workDisposition
	next        *workLedger
	disposition admissionDisposition
}

// proposing decides a work admission request against the state, and leaves
// the state as it is. The sample's stopped flag is cleared and the host
// assessed, then the stop state is updated. The result is stopped if the stop
// state is on, host held if the host is not ready, and otherwise proposed
// with the next ledger if the reservation fits, or reservation held if it
// does not.
func (state workAdmissionState) proposing(request workAdmissionRequest) workAdmissionProposal {
	sample := request.sample
	sample.stopped = false
	host := request.policy.assess(sample)
	stop := state.stop.after(stopEvent{observed: host})
	result := workAdmissionProposal{stop: stop, host: host}
	if stop.state.stopped {
		result.disposition = admissionStopped
		return result
	}
	if host != hostReady {
		result.disposition = admissionHostHeld
		return result
	}
	reservation := state.ledger.reserving(request.reservation)
	result.reservation = reservation.disposition
	if reservation.disposition != workProposed {
		result.disposition = admissionReservationHeld
		return result
	}
	result.next = reservation.next
	result.disposition = admissionProposed
	return result
}
