# Reading native process start and exit observations

This Darwin primitive reads one positive, natively representable PID with
the public proc_pid_rusage API and RUSAGE_INFO_V0. It reports the queried
PID and the native start/exit Mach tick fields. It does not interpret
them as a live grant, complete process-tree closure or permission to signal.

The installed SDK declares this API for live processes and zombies, with
zero for success and minus one with errno for failure. A successful call
is an observation of the selected process during that call. The process
may exit before the caller acts on the observation.

## ABI and source

Use the compiler's libproc.h and sys/resource.h declarations and named
RUSAGE_INFO_V0 constant. Let C compile the structure layout; do not copy
its offsets, size or a numeric flavor into Go. V0 already has the needed
fields; the additional resource accounting of later versions is unused.

The request supplies a Go integer PID. Refuse nonpositive values and a
value that cannot round-trip through the API's C int before calling it.
A refusal returns a failed disposition and os.ErrInvalid, with no observed
PID or timestamps. No narrowed value may accidentally select another PID.

On native failure, retain the native cause with errors.Is and report a
failed disposition without an observation. On success, ignore stale errno
and return the native fields without caching or inventing a missing value.
The zero result is unknown. Failed and unknown results carry no identity.

The raw observation may contain zero timestamps. A separate composing
validator must decide whether it can use those fields in a boot-scoped
identity. This primitive neither obtains the boot identity nor interprets
an exit-field zero as a promise that a process remains alive.

The executable UUID in the same native structure is not a process birth
identifier and is not returned. The private unique-ID flavor found in an
older source tree is absent from these public SDK declarations; it is not
part of this adapter's contract.

The current published kernel source supplies the start field from the
process statistics and preserves it across exec. Those sources explain
the interface; they are not established as the exact running kernel build.
Bounded native witnesses must establish the behavior actually observed here.

## Native witnesses and anti-tests

Read this test process and require a matching queried PID, positive start
stamp and no reported exit stamp. Repeated reads of the same process keep
the start stamp while Go code continues running between them.

Pair those with nonpositive and unrepresentable PIDs. Every refusal must
return the exact argument error and no observed PID or timestamps. Rapid
varies invalid inputs without spawning processes or reading arbitrary PIDs.

Launch one test-owned child, wait for its ready handshake, then observe
both processes. The child has a distinct PID and a later start stamp.
Closing its input ends it; an independent deadline and cleanup reap it.
Tests never generate hundreds of children or choose arbitrary unrelated PIDs.

After reaping that child, a query of its former PID must not return the
old observation. Native failure retains its cause. If the PID has already
been reused, a new observation must have a different start stamp; the test
does not assume that a reaped PID stays absent.

The child handshake and deadline are test fixtures, not a product protocol.
Their picked bounds live in checked local data. Real libproc calls are
needed to test the actual SDK, native error and process behavior; a mock
of the same native call would not establish those properties.

## Development order and limits

Write the tests and anti-tests first, then the friendly guide, types and
empty method. Require compile and actual assertion-red evidence before
implementing this single native-reading method. Its cgo preamble includes
headers and linkage only, with no second hand-written C function body.

The checks run offline through Game with CPU, aggregate sampled RSS and
outer time guards. A missing C compiler or unsupported native call fails
the check. It does not skip and report success.

No signal, process-group stop, orphan reclamation, PID-reuse-free action
or reparented-descendant proof is implemented. Those need a separately
reviewed ownership and lifetime strategy; a sampled PID check alone cannot
close a race between reading identity and signaling a reused PID.

## Composed owner identity and bounded quiescence observation

`CaptureOwner` composes the native reader with `kern.bootsessionuuid`. Two
positive, matching start observations with no exit stamp are fenced by two
matching boot reads. A missing, malformed, all-zero or changing boot UUID,
zero birth, PID mismatch, reuse during capture, native failure or cancellation
returns no identity. UUIDs are normalized to lowercase at the native boundary.

The public identity's fields are private; `ParseOwnerIdentity` accepts only
its exact canonical serialization, including positive native-width PID and
nonzero unsigned start. Parsing a durable identity is not authenticating it.

`Incarnation` returns `process-v1.<boot-uuid>.<pid>.<mach-start>` using only
transaction-owner token characters. It is suitable for the existing owner
incarnation field, but no transaction, reservation, release proof or rendering
wiring is changed here.

The supervisor must bind the captured identity to its
actual launch and durable owner record, not capture a possibly reused PID
later and silently replace the stored identity. Start is an opaque native
Mach timestamp, not a wall-clock time or executable UUID.

`ObserveOwner` checks the exact boot first and fences the usage observation
with another boot read. Present means that exact PID/start was observed with
no exit stamp during the call, not a continuing liveness guarantee. Dead
means the stored incarnation is gone: the kernel returned ESRCH, a valid
observation has a different start, or that incarnation has an exit stamp.

A replacement process is never reported as the original owner. Boot mismatch
is unknown rather than evidence about the current boot's PID. Permission,
I/O, malformed and omitted observations remain unknown and retain causes.
Owner death says nothing about children or reserved capacity.

`ObserveQuiescence` requires a caller deadline and performs at most four
usage reads, one BSD-info read and two fixed-capacity list reads. It observes
the owner's direct children and current process group using the public SDK
`proc_listchildpids`, `proc_listpgrppids` and `PROC_PIDTBSDINFO` declarations.

C determines native layouts and sizes. Each list has a 4096-PID ceiling;
a full buffer is ambiguous truncation and is refused, not retried indefinitely.
Returned PID lists are diagnostic samples only, not owner identities or a
complete descendant set. Identity is rechecked after sampling; a changed or
unknown owner invalidates the lists. No cached PID result supplies authority.

The public wrapper source divides list results by sizeof(int), so these two
list calls return PID counts, not bytes. Its lower `proc_listpids` wrapper
maps syscall failure to zero, making zero ambiguous with an empty list.
A zero result is therefore unknown even if errno is zero; a positive result
ignores stale errno. BSD-info success must return exactly the SDK structure
size. Native failure causes remain inspectable with errors.Is.

There is deliberately no successful whole-tree quiescence state. Every
quiescence observation returns `QuiescenceUnknown` and an error including
`ErrDescendantsUnknown`.

An empty process group, a dead/reused leader, a
zombie or repeated stable snapshots cannot prove historical descendants are
finished: children can fork, change group/session and be reparented between
observations. Recursing over sampled parent PIDs would add races, not proof,
so this implementation does not pretend that it establishes transitive
closure.

A future success primitive needs an independently enforced lifetime
containment strategy and complete launch-time tracking. Do not construct a
transaction release attestation from these diagnostic samples.

All loops and allocations are bounded. Context cancellation is checked before
and after native calls. These synchronous read-only syscalls cannot be forcibly
interrupted by context; deadlines bound further observation work, not kernel
scheduling latency. No goroutine or retry loop hides a blocked syscall.

Tests were written first: the requested Game command failed compilation with
missing types, then compiled against refusal stubs and failed actual assertions
before implementation. Rapid generates identity round trips and replacement
start stamps against mock readers. Deterministic mocks exercise boot changes,
exit/capture races, permission/I/O failures, omitted data, malformed identities,
list errors and truncation, and cancellation.

Bounded native children establish
stable own identity, distinct child identity, post-reap death, real child/group
sampling and refusal to infer descendant closure. The existing usage witnesses
are also exposed under `TestUsageNativeWitnesses` so the requested selection
actually runs them. Existing fixture deadlines and cleanup remain unchanged.

## Sources read

- Installed MacOSX.sdk: libproc.h declarations and return contract;
  sys/resource.h V0 and V4 layouts. The observed kernel is Darwin 27.2.0.
- [Apple resource collection](https://raw.githubusercontent.com/apple-oss-distributions/xnu/main/bsd/kern/kern_resource.c),
  gather_rusage_info and proc_get_rusage: start, executable UUID and zombies.
- [Apple fork path](https://raw.githubusercontent.com/apple-oss-distributions/xnu/main/bsd/kern/kern_fork.c),
  the start-statistics assignment and exec preservation.
- [Apple public wrapper](https://raw.githubusercontent.com/apple-oss-distributions/xnu/main/libsyscall/wrappers/libproc/libproc.c),
  proc_pid_rusage's direct native delegation and list count/error conversion.
- Installed MacOSX.sdk libproc.h and sys/proc_info.h: child/group list
  declarations, PROC_PIDTBSDINFO, proc_bsdinfo and native structure size.
- [Apple process information syscall](https://raw.githubusercontent.com/apple-oss-distributions/xnu/main/bsd/kern/proc_info.c),
  proc_listpids, proc_pidbsdinfo, proc_pidinfo and proc_pid_rusage: list
  truncation, current parent/group fields, permissions and ESRCH lookup.
