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

## Sources read

- Installed MacOSX.sdk: libproc.h declarations and return contract;
  sys/resource.h V0 and V4 layouts. The observed kernel is Darwin 27.2.0.
- [Apple resource collection](https://raw.githubusercontent.com/apple-oss-distributions/xnu/main/bsd/kern/kern_resource.c),
  gather_rusage_info and proc_get_rusage: start, executable UUID and zombies.
- [Apple fork path](https://raw.githubusercontent.com/apple-oss-distributions/xnu/main/bsd/kern/kern_fork.c),
  the start-statistics assignment and exec preservation.
- [Apple public wrapper](https://raw.githubusercontent.com/apple-oss-distributions/xnu/main/libsyscall/wrappers/libproc/libproc.c),
  proc_pid_rusage's direct native delegation.
