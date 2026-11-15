package budget

import (
	"context"
	"crypto/sha256"
)

type releaseQuiescenceState uint8

const releaseWorkAndDescendantsQuiescent releaseQuiescenceState = 1

type reservationQuiescenceProof struct {
	binding     admissionBinding
	release     reservationRelease
	predecessor [sha256.Size]byte
	state       releaseQuiescenceState
}

type transactionReleaseRequest struct {
	binding admissionBinding
	release reservationRelease
	proof   reservationQuiescenceProof
}

type transactionReleaseOutcome struct {
	disposition releaseDisposition
	stateDigest [sha256.Size]byte
}

// release removes the reservation named in request and publishes the new
// state. It runs once, after observe, and returns errAdmissionUnsafe for a
// binding or owner that is not the transaction's, a proof that does not match
// the request and state, or an owner mismatch. A reservation that is not held
// still publishes a new state, with a new predecessor and the same ledger.
func (tx *admissionTransaction) release(ctx context.Context, request transactionReleaseRequest) (out transactionReleaseOutcome, resultErr error) {
	if tx == nil {
		return out, errAdmissionUnsafe
	}
	tx.mutex.Lock()
	defer tx.mutex.Unlock()
	if ctx == nil || tx.closed || !tx.ready || tx.attempted {
		return out, errAdmissionUnsafe
	}
	tx.attempted = true
	if err := ctx.Err(); err != nil {
		return out, err
	}
	if err := tx.stableLock(); err != nil {
		return out, err
	}
	binding := admissionBinding{boot: tx.state.boot, policy: tx.state.policy, limit: tx.state.ledger.limit}
	if request.binding != binding || request.release.owner != tx.owner || !admissionOwnerValid(request.release.owner) || !admissionTokenValid(string(request.release.id)) {
		return out, errAdmissionUnsafe
	}
	proof := request.proof
	if proof.state != releaseWorkAndDescendantsQuiescent || proof.binding != binding || proof.release != request.release || proof.predecessor != tx.digest {
		return out, errAdmissionUnsafe
	}
	stateData, err := admissionRead(tx.directory, "state")
	if err != nil {
		return out, err
	}
	if sha256.Sum256(stateData) != tx.digest {
		return out, errAdmissionUnsafe
	}
	state, err := decodeAdmissionRecord(stateData)
	if err != nil {
		return out, err
	}
	intentData, err := admissionRead(tx.directory, "intent")
	if err != nil {
		return out, err
	}
	intent, err := decodeAdmissionIntent(intentData)
	if err != nil {
		return out, err
	}
	if intent.phase != intentObserving || intent.boot != binding.boot || intent.owner != tx.owner || intent.predecessor != tx.digest {
		return out, errAdmissionUnsafe
	}
	proposal := state.ledger.releasing(request.release)
	if proposal.disposition != releaseProposed && proposal.disposition != releaseNotFound {
		return out, errAdmissionUnsafe
	}
	next := state
	next.predecessor = tx.digest
	if proposal.disposition == releaseProposed {
		next.ledger = *proposal.next
	}
	nextData, err := encodeAdmissionRecord(next)
	if err != nil {
		return out, err
	}
	result := transactionReleaseOutcome{disposition: proposal.disposition, stateDigest: sha256.Sum256(nextData)}
	prepared, err := encodeAdmissionIntent(admissionIntent{version: admissionRecordVersion, phase: intentPrepared, boot: binding.boot, owner: tx.owner, predecessor: tx.digest, stateDigest: result.stateDigest, next: &next})
	if err != nil {
		return out, err
	}
	if err := tx.publishTransition(ctx, nextData, prepared, nil); err != nil {
		return out, err
	}
	return result, nil
}
