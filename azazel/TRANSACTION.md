# Locked admission publication

The Darwin-only budget transaction is private and is not connected to render,
composite, or Game admission. Its successful internal outcome records a durable
value transition, never a live grant. Owner membership is not authenticated
process identity. No caller may treat `admissionProposed` as permission to work.

## Entry and observation

The caller provisions a stable trusted directory and an empty permanent `lock`
file. The transaction creates neither and never replaces or removes the lock.

Directory traversal uses no-follow descriptor-relative opens. The final directory
must belong to the effective UID with mode 0700. Entries must be owner-only
regular files without hard links; record reads are bounded. The trust boundary
excludes malicious same-UID writers and administrators. Participants must not
rename the root or lock, or replace ancestors during a transaction.

Nonblocking flock is held through observation, sampling, publication, and outcome
return, until explicit close. The opened lock inode is checked against its name.
Entry validates canonical state and exact supplied boot, policy, and byte-limit
bindings. Any intent or publication scratch entry causes refusal. No recovery or
scratch cleanup is implemented. Process death releases only the kernel lock.

Observation requires exact membership in the validated owners. It exclusively
creates `intent`, writes an observing record linked to the predecessor digest,
syncs the file, closes it, and syncs the directory. A handle becomes ready only
when observation completes successfully. Failed creation, writing, flushing, or
cancellation cannot be upgraded to a successful observation by calling admit.

## Internal admission

A ready handle permits one admission attempt. It rereads and validates state and
the observing intent under the held lock, checking the exact predecessor digest,
boot, and owner recorded by this handle. Request bindings and reservation owner
must match.

It invokes the supplied measurement function while locked. The sample
must carry matching boot/policy/limit bindings, all three available measurements,
a finite nonnegative load, normal memory pressure, and ordered start/completion
timestamps within this invocation. Sample age from its start must not exceed
five seconds. Freshness is checked before publication and again before replacing
state.

A private native Darwin sampler is available for this measurement callback; it
is not wired to engine admission. It reads kern.bootsessionuuid before and after
sampling and requires an exact match to the supplied boot binding.

Native sysctl
reads supply the one-minute vm.loadavg with unsigned fixed-point load fields,
exact vm.swapusage used bytes, and kern.memorystatus_vm_pressure_level. Only
normal pressure (1) passes; warning (2) and critical (4) hold with an error, and
unknown levels fail closed.

Malformed layouts and inconsistent counters are
rejected. Disk space is Bavail times Bsize from Fstatfs on the supplied open
output-directory descriptor, pinned during the call; counter consistency and
multiplication overflow are checked. Renaming the directory does not redirect
the sample. No external sampling processes are launched.

The sampler records start/completion timestamps, rejects durations over five
seconds and backward time, and checks cancellation around native reads. Native
syscalls cannot be interrupted mid-call. Reads are sequential, not an atomic
kernel snapshot, and the descriptor does not prove later output uses that same
directory. These transaction checks still validate supplied facts, not their
provenance: a dishonest callback can lie about timestamps and measurements.

Native boot-session identity is not authenticated process identity. A blocked
filesystem syscall can outlast the age bound after the final check. Cancellation
checks and syscalls are not atomic.

The existing host and latched-stop transitions are evaluated before the owned
reservation ledger. Known low disk persists the stop, favorable measurements do
not clear an existing stop, high load/swap hold admission, and duplicate or
capacity-limited reservations remain holds. A proposed reservation copies the
ledger. Holds also publish the next predecessor-linked state. Unknown, invalid,
failed, stale, or mismatched inputs return a zero outcome and leave the fence.

No successful result is returned before the full publication sequence completes.
The request and predecessor values are not mutated.

## Internal release

A ready handle permits one admission or release attempt, never both. Release is
private and has no grant, engine, render, composite, or Game wiring. The release
request supplies the exact reservation ID and owner name plus incarnation. It
must match the observed owner, the locked predecessor, and current bindings.
The current state and observing intent are reread and validated before proposing
a change; matching a cached owner or a reservation ID alone is insufficient.

Release additionally requires a typed `reservationQuiescenceProof` whose state
explicitly asserts that work and all descendants are quiescent. The proof binds
the exact release request, boot/policy/limit, and predecessor digest. Its zero
value, another request's proof, and stale predecessor proofs fail closed.

This is
an explicit trusted-caller attestation, not independently authenticated evidence:
there is no PID-death inference, native process verification, or quiescence
collector. A caller must establish and maintain quiescence before constructing
this proof. Owner membership alone is not sufficient release authority.

An exact match copies the ledger and removes only that reservation. An absent ID
publishes an unchanged ledger with `releaseNotFound`; repeating release on a new
observed handle therefore releases no additional bytes. A same-handle retry is
refused. Wrong owners, stale incarnations, invalid proofs, changed predecessor
bytes, or invalid intents return zero outcomes and retain the fence. Stop, policy,
owners, and byte limit are preserved, including when the ledger is full.

Release and admission share the same publication implementation below. A failure
before state replacement preserves the exact predecessor bytes. A failure after
replacement can leave the proposed release in state, but retained intent or
retirement fences prevent any subsequent participant from using that capacity.

There is deliberately no rollback. The final-unlink acknowledgement boundary
also applies to release: death there can leave an unacknowledged committed
release. No stronger all-errors-preserve-state or exactly-once claim is made.

## Publication order and interruption

1. Exclusively create `intent-prepared`, write the canonical prepared intent with
   the complete next state and its digest, sync and close it, rename it over
   `intent`, then sync the directory.
2. Exclusively create `state-next`, write, sync, and close it, atomically rename it
   over `state`, then sync the directory.
3. Exclusively create `retiring` containing the prepared intent, sync and close
   it, then sync the directory. This redundant fence protects interruption after
   removing the primary intent.
4. Remove `intent` and sync the directory. Remove `retiring` and sync the directory
   before returning an internal outcome.

Errors leave the current primary intent and scratch entries untouched. If an
error is observed after intent removal, the implementation attempts to recreate
and sync the prepared intent without honoring the cancelled context. Failure of
that restoration is joined into the returned error; successful restoration is
not guaranteed on a failing filesystem.

A surviving retirement fence also blocks
entry. There is no rollback of an already replaced state and no retry on the
same handle. Partial scratch files are not silently discarded.

There is necessarily an acknowledgement boundary: death after the final fence
unlink can leave a complete new state with no intent and no returned outcome.
The killed-process test explicitly covers this boundary; subsequent entry can
read that committed state. This is not a live or silently returned grant. An
ordinary returned error at that boundary attempts to restore the prepared fence.

Exactly-once acknowledgement across process death, power loss, and permanent
storage failures is not claimed. Successful fsync calls and SIGKILL tests are not
power-loss durability tests; no stronger hardware flush guarantee is asserted.

## Verification and remaining authority gaps

New transition tests were written before the transition implementation. They cover
successful publication, holds and disk-stop persistence, input preservation,
repeated attempts, invalid or uncertain samples and bindings, mismatched owner
and intent, and changed predecessor bytes. Per-handle fault hooks exercise every
publication operation boundary and assert zero outcomes plus retained intent.

Subprocesses are killed at every named publication boundary; another process
contends while publication is paused, and reopening refuses surviving fences.
Existing tests cover simultaneous observing processes, killed observing owners,
unsafe entry types, cancellation, and handle lifetime. Hooks inject boundary
errors, not kernel-level short writes or real fsync failures.

Release tests were written before implementation and cover exact owner and
incarnation refusal, absent IDs, repeated requests, full entry/byte ledgers,
proof binding, changed locked predecessor, cancellation, replaced locks, pending
entries, and exact unchanged capacity before state replacement.

Fault hooks and
SIGKILL cover every publication boundary, with concurrent lock contention and
pending-fence refusal after interruption. After state replacement, tests check
the exact proposed byte reduction and that no new transaction can consume it
while fenced. These remain boundary simulations, not physical power-loss tests.

Sampler tests use a mockgen-generated adapter to cover malformed and unknown
fields, unsigned high-bit load values, zero load scale, pressure holds,
overflow, cancellation, stale/future timestamps, and renamed output directories.
A small read-only Darwin integration exercises native boot, resource, and disk
sampling without rendering. The focused TestHost/TestSample/TestAdmission/
TestTransaction suite passed after the unsigned-load correction.

The focused TestTransaction/TestAdmission/TestRecord/TestRelease suite passed
through Game.
Game also reported that it could not determine whether its version met .game's
0.5.0 requirement. No scene renders were run.

Still absent: authenticated process PID/start/boot identity, registration and
descendant quiescence verification, recovery and abandoned-intent reconciliation,
explicit resume, physical-memory and processor accounting, and public
render/composite/Game admission wiring. Native host sampling is available only
as the private Darwin callback described above; it does not confer work authority.

A committed internal reservation can remain stranded after death until a future
identity-aware recovery protocol exists. This implementation deliberately does
not reclaim it or authorize work on its basis.
