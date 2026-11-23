package budget

import (
	"context"
	"crypto/sha256"
	"errors"
	"io"
	"math"
	"time"

	"golang.org/x/sys/unix"
)

const admissionSampleAge = 5 * time.Second

type transactionAdmissionRequest struct {
	binding     admissionBinding
	reservation ownedMemoryReservation
}

type admissionMeasurement struct {
	binding   admissionBinding
	taken     time.Time
	completed time.Time
	pressure  uint32
	sample    hostSample
}

type transactionAdmissionOutcome struct {
	disposition admissionDisposition
	host        hostDecision
	reservation reservationDisposition
	stop        stopTransition
	stateDigest [sha256.Size]byte
}

// publishCheck is run between the steps of a write. It fails if ctx is done
// or the lock file is no longer the one named lock, calls the publish hook
// with point if there is one, then checks both again.
func (tx *admissionTransaction) publishCheck(ctx context.Context, point string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := tx.stableLock(); err != nil {
		return err
	}
	if tx.publish != nil {
		if err := tx.publish(point); err != nil {
			return err
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return tx.stableLock()
}

// publishFile creates the file name, which must not exist yet, writes data to
// it, syncs it and closes it. publishCheck runs before each step, with prefix
// and write, sync or close as the point. A short write is an error.
func (tx *admissionTransaction) publishFile(ctx context.Context, name, prefix string, data []byte) error {
	if err := tx.publishCheck(ctx, prefix+"-write"); err != nil {
		return err
	}
	file, err := admissionOpen(tx.directory, name, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL)
	if err != nil {
		return err
	}
	count, writeErr := file.Write(data)
	if writeErr == nil && count != len(data) {
		writeErr = io.ErrShortWrite
	}
	if writeErr == nil {
		writeErr = tx.publishCheck(ctx, prefix+"-sync")
	}
	if writeErr == nil {
		writeErr = file.Sync()
	}
	if writeErr == nil {
		writeErr = tx.publishCheck(ctx, prefix+"-close")
	}
	return errors.Join(writeErr, file.Close())
}

// restoreIntent writes data to a new file named intent, syncs it, then syncs
// the directory. publishTransition uses it to put the intent back when a
// later step fails after the intent was removed. It makes none of the
// publishCheck calls.
func (tx *admissionTransaction) restoreIntent(data []byte) error {
	file, err := admissionOpen(tx.directory, "intent", unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL)
	if err != nil {
		return err
	}
	count, writeErr := file.Write(data)
	if writeErr == nil && count != len(data) {
		writeErr = io.ErrShortWrite
	}
	if writeErr == nil {
		writeErr = file.Sync()
	}
	err = errors.Join(writeErr, file.Close())
	directoryDescriptor := int(tx.directory.Fd())
	return errors.Join(err, unix.Fsync(directoryDescriptor))
}

// admit decides one admission request and publishes the result. It runs once
// per transaction, after observe, and returns errAdmissionUnsafe for a
// binding or owner that is not the transaction's, an invalid id, a zero
// amount, changed state or intent files, or a measurement that is stale or
// has abnormal pressure. The reservation is added only if the stop state is
// off and the host is ready; the outcome says which case held.
func (tx *admissionTransaction) admit(ctx context.Context, request transactionAdmissionRequest, measure func(context.Context) (admissionMeasurement, error)) (out transactionAdmissionOutcome, resultErr error) {
	if tx == nil {
		return out, errAdmissionUnsafe
	}
	tx.mutex.Lock()
	defer tx.mutex.Unlock()
	if ctx == nil || tx.closed || !tx.ready || tx.attempted || measure == nil {
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
	if request.binding != binding || request.reservation.owner != tx.owner || !admissionTokenValid(string(request.reservation.id)) || request.reservation.amount == noBytes {
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
	started := time.Now()
	measurement, err := measure(ctx)
	if err != nil {
		return out, err
	}
	now := time.Now()
	if err := ctx.Err(); err != nil {
		return out, err
	}
	if measurement.binding != binding || measurement.taken.Before(started) || measurement.completed.Before(measurement.taken) || measurement.completed.After(now) || now.Sub(measurement.taken) > admissionSampleAge || measurement.pressure != 1 {
		return out, errAdmissionUnsafe
	}
	sample := measurement.sample
	sample.stopped = false
	if !sample.loadKnown || !sample.swapKnown || !sample.diskKnown || !(sample.load >= 0) || math.IsInf(sample.load, 0) {
		return out, errAdmissionUnsafe
	}
	host := state.policy.assess(sample)
	if host == hostUnknown || host == hostInvalidPolicy {
		return out, errAdmissionUnsafe
	}
	stop := state.stop.after(stopEvent{observed: host})
	next := state
	next.stop = stop.state
	next.predecessor = tx.digest
	result := transactionAdmissionOutcome{host: host, stop: stop}
	switch {
	case stop.state.stopped:
		result.disposition = admissionStopped
	case host != hostReady:
		result.disposition = admissionHostHeld
	default:
		proposal := state.ledger.reserving(request.reservation)
		result.reservation = proposal.disposition
		result.disposition = admissionReservationHeld
		if proposal.disposition == reservationInvalid || proposal.disposition == reservationUnknown {
			return out, errAdmissionUnsafe
		}
		if proposal.disposition == reservationProposed {
			next.ledger = *proposal.next
			result.disposition = admissionProposed
		}
	}
	nextData, err := encodeAdmissionRecord(next)
	if err != nil {
		return out, err
	}
	result.stateDigest = sha256.Sum256(nextData)
	prepared, err := encodeAdmissionIntent(admissionIntent{version: admissionRecordVersion, phase: intentPrepared, boot: binding.boot, owner: tx.owner, predecessor: tx.digest, stateDigest: result.stateDigest, next: &next})
	if err != nil {
		return out, err
	}
	if err := tx.publishTransition(ctx, nextData, prepared, func() error {
		if now := time.Now(); now.Before(measurement.completed) || now.Sub(measurement.taken) > admissionSampleAge {
			return errAdmissionUnsafe
		}
		return nil
	}); err != nil {
		return out, err
	}
	return result, nil
}

// publishTransition makes the next state current in a fixed order, syncing
// the directory between steps: write the prepared intent and rename it over
// intent, write state-next and rename it over state, then write retiring and
// remove intent and then retiring. check, if not nil, runs before the first
// write and before the state rename. If a step fails after intent was
// removed, the intent is written back.
func (tx *admissionTransaction) publishTransition(ctx context.Context, nextData, prepared []byte, check func() error) (resultErr error) {
	removed := false
	defer func() {
		if resultErr != nil && removed {
			resultErr = errors.Join(resultErr, tx.restoreIntent(prepared))
		}
	}()
	if check != nil {
		if err := check(); err != nil {
			return err
		}
	}
	if err := tx.publishFile(ctx, "intent-prepared", "prepared", prepared); err != nil {
		return err
	}
	if err := tx.publishCheck(ctx, "prepared-rename"); err != nil {
		return err
	}
	if err := unix.Renameat(int(tx.directory.Fd()), "intent-prepared", int(tx.directory.Fd()), "intent"); err != nil {
		return err
	}
	if err := tx.publishCheck(ctx, "prepared-directory"); err != nil {
		return err
	}
	if err := unix.Fsync(int(tx.directory.Fd())); err != nil {
		return err
	}
	if err := tx.publishFile(ctx, "state-next", "state", nextData); err != nil {
		return err
	}
	if err := tx.publishCheck(ctx, "state-rename"); err != nil {
		return err
	}
	if check != nil {
		if err := check(); err != nil {
			return err
		}
	}
	if err := unix.Renameat(int(tx.directory.Fd()), "state-next", int(tx.directory.Fd()), "state"); err != nil {
		return err
	}
	if err := tx.publishCheck(ctx, "state-directory"); err != nil {
		return err
	}
	if err := unix.Fsync(int(tx.directory.Fd())); err != nil {
		return err
	}
	if err := tx.publishFile(ctx, "retiring", "retiring", prepared); err != nil {
		return err
	}
	if err := tx.publishCheck(ctx, "retiring-directory"); err != nil {
		return err
	}
	if err := unix.Fsync(int(tx.directory.Fd())); err != nil {
		return err
	}
	if err := tx.publishCheck(ctx, "intent-remove"); err != nil {
		return err
	}
	if err := unix.Unlinkat(int(tx.directory.Fd()), "intent", 0); err != nil {
		return err
	}
	removed = true
	if err := tx.publishCheck(ctx, "retired-directory"); err != nil {
		return err
	}
	if err := unix.Fsync(int(tx.directory.Fd())); err != nil {
		return err
	}
	if err := tx.publishCheck(ctx, "retiring-remove"); err != nil {
		return err
	}
	if err := unix.Unlinkat(int(tx.directory.Fd()), "retiring", 0); err != nil {
		return err
	}
	if err := tx.publishCheck(ctx, "complete-directory"); err != nil {
		return err
	}
	if err := unix.Fsync(int(tx.directory.Fd())); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return tx.stableLock()
}
