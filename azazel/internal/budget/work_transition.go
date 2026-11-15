package budget

import (
	"crypto/sha256"
	"errors"
)

var errWorkTransition = errors.New("invalid shared work transition")

type workTransition struct {
	previous [sha256.Size]byte
	next     workRecord
	encoded  []byte
	outcome  admissionDisposition
}

// proposeWorkTransition returns the record that would follow state after
// request, with its encoding and the admission outcome, and leaves state as
// it is. The request's policy must equal state's. In the new record the stop
// state is updated, the reservation is added only if the outcome is proposed,
// and the predecessor is the SHA-256 of state's encoding, which previous also
// holds.
func proposeWorkTransition(state workRecord, request workAdmissionRequest) (workTransition, error) {
	encoded, err := encodeWorkRecord(state)
	if err != nil {
		return workTransition{}, err
	}
	if request.policy != state.policy {
		return workTransition{}, errWorkTransition
	}
	proposal := (workAdmissionState{stop: state.stop, ledger: state.ledger}).proposing(request)
	if proposal.disposition == admissionUnknown || proposal.host == hostUnknown || proposal.host == hostInvalidPolicy {
		return workTransition{}, errWorkTransition
	}
	next := state
	next.ledger.held = append([]workReservation(nil), state.ledger.held...)
	next.predecessor = sha256.Sum256(encoded)
	next.stop = proposal.stop.state
	if proposal.disposition == admissionProposed {
		if proposal.next == nil {
			return workTransition{}, errWorkTransition
		}
		next.ledger = *proposal.next
	}
	data, err := encodeWorkRecord(next)
	if err != nil {
		return workTransition{}, err
	}
	return workTransition{previous: next.predecessor, next: next, encoded: data, outcome: proposal.disposition}, nil
}
