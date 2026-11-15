package processinfo

import (
	"context"
	"errors"
	"math"
	"os"
	"strconv"
	"strings"
	"syscall"
)

var ErrOwnerUnknown = errors.New("process owner identity is unknown")
var ErrDescendantsUnknown = errors.New("complete descendant quiescence cannot be proven by process snapshots")

const quiescenceMemberLimit = 4096

type OwnerIdentity struct {
	boot  string
	pid   int
	start uint64
}

// Boot is the id of the system boot in which the owner ran.
func (id OwnerIdentity) Boot() string { return id.boot }

// PID is the process id of the owner.
func (id OwnerIdentity) PID() int { return id.pid }

// Start is the process start time as the system reported it.
func (id OwnerIdentity) Start() uint64 { return id.start }

// valid is true when the boot id and pid are valid and the start is nonzero.
func (id OwnerIdentity) valid() bool {
	return validBoot(id.boot) && validOwnerPID(id.pid) && id.start != 0
}

// Incarnation is the identity as text, "process-v1.boot.pid.start", or the
// empty string if the identity is not valid.
func (id OwnerIdentity) Incarnation() string {
	if !id.valid() {
		return ""
	}
	return "process-v1." + id.boot + "." + strconv.Itoa(id.pid) + "." + strconv.FormatUint(id.start, 10)
}

// ParseOwnerIdentity reads back a token made by Incarnation. It returns
// os.ErrInvalid for any token that is too long, malformed, or would not
// print back to the same text.
func ParseOwnerIdentity(token string) (OwnerIdentity, error) {
	if len(token) > 80 {
		return OwnerIdentity{}, os.ErrInvalid
	}
	parts := strings.Split(token, ".")
	if len(parts) != 4 || parts[0] != "process-v1" {
		return OwnerIdentity{}, os.ErrInvalid
	}
	pid, err := strconv.ParseInt(parts[2], 10, 32)
	if err != nil {
		return OwnerIdentity{}, os.ErrInvalid
	}
	start, err := strconv.ParseUint(parts[3], 10, 64)
	if err != nil {
		return OwnerIdentity{}, os.ErrInvalid
	}
	id := OwnerIdentity{boot: parts[1], pid: int(pid), start: start}
	if !id.valid() || id.Incarnation() != token {
		return OwnerIdentity{}, os.ErrInvalid
	}
	return id, nil
}

// validOwnerPID is true for a pid from 1 up to the largest 32-bit integer.
func validOwnerPID(pid int) bool { return pid > 0 && pid <= math.MaxInt32 }

// validBoot is true for a boot id spelt as a lowercase hex uuid in the
// 8-4-4-4-12 form that is not all zeros.
func validBoot(boot string) bool {
	if len(boot) != 36 {
		return false
	}
	zeros := true
	for i, c := range []byte(boot) {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
		} else if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		} else if c != '0' {
			zeros = false
		}
	}
	return !zeros
}

type OwnerState uint8

const (
	OwnerUnknown OwnerState = iota
	OwnerPresent
	OwnerDead
)

type OwnerObservation struct {
	State OwnerState
	Err   error
}

type QuiescenceState uint8

const QuiescenceUnknown QuiescenceState = 0

type QuiescenceObservation struct {
	State    QuiescenceState
	Owner    OwnerObservation
	Children []int
	Group    []int
	Err      error
}

type processRelations struct{ pid, parent, group int }

type ownerReader interface {
	ReadBoot() (string, error)
	ReadUsage(usageQuery) usageObservation
	ReadRelations(int) (processRelations, error)
	ListChildren(int, int) ([]int, error)
	ListGroup(int, int) ([]int, error)
}

// CaptureOwner is captureOwner run against the native darwin process reader.
func CaptureOwner(ctx context.Context, pid int) (OwnerIdentity, error) {
	return captureOwner(ctx, darwinOwnerReader{}, pid)
}

// ObserveOwner is observeOwner run against the native darwin process reader.
func ObserveOwner(ctx context.Context, id OwnerIdentity) OwnerObservation {
	return observeOwner(ctx, darwinOwnerReader{}, id)
}

// ObserveQuiescence is observeQuiescence run against the native darwin
// process reader.
func ObserveQuiescence(ctx context.Context, id OwnerIdentity) QuiescenceObservation {
	return observeQuiescence(ctx, darwinOwnerReader{}, id)
}

// ownerContext returns the error of ctx, or os.ErrInvalid if ctx is nil.
func ownerContext(ctx context.Context) error {
	if ctx == nil {
		return os.ErrInvalid
	}
	return ctx.Err()
}

// ownerBoot reads the boot id through reader and refuses one that is not a
// valid lowercase uuid. A cancelled ctx wins over a read error.
func ownerBoot(ctx context.Context, reader ownerReader) (string, error) {
	if err := ownerContext(ctx); err != nil {
		return "", err
	}
	boot, err := reader.ReadBoot()
	if canceled := ctx.Err(); canceled != nil {
		return "", canceled
	}
	if err != nil {
		return "", err
	}
	if !validBoot(boot) {
		return "", ErrOwnerUnknown
	}
	return boot, nil
}

// ownerUsage reads one usage observation for pid and refuses one that is
// not for that pid, has no start time, or exits before it started. A bare
// read failure that carries no other fields is returned as it came.
func ownerUsage(ctx context.Context, reader ownerReader, pid int) (usageObservation, error) {
	if err := ownerContext(ctx); err != nil {
		return usageObservation{}, err
	}
	got := reader.ReadUsage(usageQuery{pid: pid})
	if err := ctx.Err(); err != nil {
		return usageObservation{}, err
	}
	if got.state == usageFailed && got.err != nil && got.pid == 0 && got.born == 0 && got.exited == 0 {
		return usageObservation{}, got.err
	}
	if got.state != usageObserved || got.err != nil || got.pid != pid || got.born == 0 || got.exited != 0 && got.exited < got.born {
		return usageObservation{}, errors.Join(ErrOwnerUnknown, got.err)
	}
	return got, nil
}

// captureOwner builds the identity of the process pid. It reads the boot id,
// then the usage twice, then the boot id again. It refuses with
// ErrOwnerUnknown if the boot id or start time moved, or if the process has
// already exited.
func captureOwner(ctx context.Context, reader ownerReader, pid int) (OwnerIdentity, error) {
	if !validOwnerPID(pid) || reader == nil {
		return OwnerIdentity{}, os.ErrInvalid
	}
	boot, err := ownerBoot(ctx, reader)
	if err != nil {
		return OwnerIdentity{}, err
	}
	first, err := ownerUsage(ctx, reader, pid)
	if err != nil {
		return OwnerIdentity{}, err
	}
	second, err := ownerUsage(ctx, reader, pid)
	if err != nil {
		return OwnerIdentity{}, err
	}
	lastBoot, err := ownerBoot(ctx, reader)
	if err != nil {
		return OwnerIdentity{}, err
	}
	if boot != lastBoot || first.born != second.born || first.exited != 0 || second.exited != 0 {
		return OwnerIdentity{}, ErrOwnerUnknown
	}
	return OwnerIdentity{boot: boot, pid: pid, start: first.born}, nil
}

// observeOwner reports whether the process id names is present or dead. It
// reads the boot id before and after the usage read and gives up as unknown
// if either differs from the id's boot. A pid the system no longer knows, a
// start time that differs from the id's, or an exit time, counts as dead.
func observeOwner(ctx context.Context, reader ownerReader, id OwnerIdentity) OwnerObservation {
	unknown := func(err error) OwnerObservation { return OwnerObservation{Err: errors.Join(ErrOwnerUnknown, err)} }
	if !id.valid() || reader == nil {
		return unknown(os.ErrInvalid)
	}
	boot, err := ownerBoot(ctx, reader)
	if err != nil {
		return unknown(err)
	}
	if boot != id.boot {
		return unknown(ErrOwnerUnknown)
	}
	got, readErr := ownerUsage(ctx, reader, id.pid)
	lastBoot, err := ownerBoot(ctx, reader)
	if err != nil {
		return unknown(err)
	}
	if boot != lastBoot {
		return unknown(ErrOwnerUnknown)
	}
	if readErr != nil {
		if errors.Is(readErr, syscall.ESRCH) && !errors.Is(readErr, ErrOwnerUnknown) {
			return OwnerObservation{State: OwnerDead}
		}
		return unknown(readErr)
	}
	if got.born != id.start || got.exited != 0 {
		return OwnerObservation{State: OwnerDead}
	}
	return OwnerObservation{State: OwnerPresent}
}

// observeQuiescence lists the children of the owner and the members of its
// process group, so long as the owner is present before and after the reads.
// ctx must have a deadline. Err always wraps ErrDescendantsUnknown, because
// snapshots cannot prove that no descendant exists; State stays unknown.
func observeQuiescence(ctx context.Context, reader ownerReader, id OwnerIdentity) QuiescenceObservation {
	out := QuiescenceObservation{Err: ErrDescendantsUnknown}
	fail := func(err error) QuiescenceObservation { out.Err = errors.Join(out.Err, err); return out }
	if err := ownerContext(ctx); err != nil {
		return fail(err)
	}
	if _, bounded := ctx.Deadline(); !bounded {
		return fail(os.ErrInvalid)
	}
	out.Owner = observeOwner(ctx, reader, id)
	if out.Owner.State == OwnerUnknown {
		return fail(out.Owner.Err)
	}
	if out.Owner.State == OwnerDead {
		return out
	}
	if err := ctx.Err(); err != nil {
		return fail(err)
	}
	relations, err := reader.ReadRelations(id.pid)
	if canceled := ctx.Err(); canceled != nil {
		return fail(canceled)
	}
	if err != nil {
		return fail(err)
	}
	if relations.pid != id.pid || !validOwnerPID(relations.group) {
		return fail(ErrOwnerUnknown)
	}
	children, err := reader.ListChildren(id.pid, quiescenceMemberLimit)
	if canceled := ctx.Err(); canceled != nil {
		return fail(canceled)
	}
	if err != nil {
		return fail(err)
	}
	if len(children) >= quiescenceMemberLimit {
		return fail(ErrOwnerUnknown)
	}
	for _, pid := range children {
		if !validOwnerPID(pid) {
			return fail(ErrOwnerUnknown)
		}
	}
	out.Children = append([]int(nil), children...)
	group, err := reader.ListGroup(relations.group, quiescenceMemberLimit)
	if canceled := ctx.Err(); canceled != nil {
		return fail(canceled)
	}
	if err != nil {
		return fail(err)
	}
	if len(group) >= quiescenceMemberLimit {
		return fail(ErrOwnerUnknown)
	}
	for _, pid := range group {
		if !validOwnerPID(pid) {
			return fail(ErrOwnerUnknown)
		}
	}
	out.Group = append([]int(nil), group...)
	after := observeOwner(ctx, reader, id)
	if after.State != OwnerPresent {
		out.Children = nil
		out.Group = nil
		return fail(errors.Join(ErrOwnerUnknown, after.Err))
	}
	return out
}
