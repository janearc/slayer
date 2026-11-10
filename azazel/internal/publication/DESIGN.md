# Atomic replacement of a local snapshot

This private helper replaces one complete file through a temporary file
in the same directory. It is an implementation boundary, not the shared
admission wire contract. The shared lock, intent, boot identity, validation
of record contents and permission to grant work remain its caller's job.

The guarantee here is namespace visibility within the running operating
system: readers opening the destination see a complete replacement after
rename. It does not call a flush operation or claim survival of a machine
crash. Durable task/result publication needs a separate reviewed policy.

It is not an immutable-result publisher: replacement is intentional.

## The operation

The function receives a publisher holding a filesystem boundary, and a
request containing an absolute parent directory, one plain basename and
the already validated bytes. An empty byte slice is a valid file payload;
it is not necessarily a valid admission record.

Before I/O it refuses a missing filesystem, a nonabsolute or unclean
parent, an empty name, dot/dot-dot, or a name containing a path separator
or NUL.

The filesystem creates a private temporary file in that parent and returns
its handle and name, or an error with no handle to transfer.

The private filesystem adapter is trusted to return a valid handle and a
temporary name in that directory. This helper is not a sandbox against
malicious adapters or concurrent changes to ancestor directory symlinks.

Clean linked parents are followed by the local adapter and are accepted
here. The admission caller's future root policy is a separate boundary.

The parent must already equal filepath.Clean(parent). Cleaning only the
destination through filepath.Join could otherwise disagree with the raw
parent passed to CreateTemp when a symlink and dot-dot occur together.

Refuse that spelling rather than silently changing its meaning. A clean
absolute root directory remains a valid parent.

The publisher writes the whole payload, closes the temporary file, then
renames it over the destination. A short write with no error is still a
refusal. Rename cannot precede successful writing and closing.

The result is one value carrying a stage and an error. The zero stage
never means published, even if its error is nil. Only a successful rename
produces the visible stage. This result grants no admission or resume.

The rename-uncertain stage requires reconciliation: its error cannot
establish whether the replacement became visible.

## Failure and cleanup

Every successful create transfers ownership of the temporary handle. On
a later failure the publisher closes it if needed and removes only that
temporary name. It never deletes or rolls back the destination. Close is
attempted once, including when it fails.

Cleanup failures join the original failure so errors.Is can still identify
each cause.

A successful rename consumes the temporary name; no cleanup remove is
performed on that path or on the destination. A failed rename reports a
failure and attempts cleanup of the temporary name. It never removes or
rolls back the destination.

A real rename error may have an uncertain effect, so higher-level recovery
must reconcile the namespace. The mock tests check our call ordering and
cleanup targets. They cannot establish the filesystem state after a real
rename error.

Tests use generated filesystem and file mocks. Ordered expectations catch
publication before write/close, missing writes and duplicate cleanup.
Paired failures cover create, write, short write, close, rename and cleanup.
Authored payloads and a bounded Rapid byte property check exact writes.

The real adapter and native tests come after the private algorithm. Mock
success does not establish filesystem behavior. The separately queued
Darwin storage probe measures relevant platform operations without treating
its flush timings as proof of a production cadence or power-loss behavior.

The types compiled and all twelve original tests rejected the empty body
before implementation. All twelve then passed at 39a47d0 with 100 percent
statement coverage. The generated empty-payload counterexample is kept
unchanged.

Review added paired clean-root and unclean-parent witnesses.
The clean root passed while all four unclean spellings reached unexpected
I/O. The guard now refuses those spellings before any filesystem call;
all fourteen tests then passed at 8a75454 with 100 percent statement coverage.
