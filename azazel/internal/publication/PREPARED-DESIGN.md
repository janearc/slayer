# Completing a prepared publication

This private byte-layer operation composes the existing snapshot publisher.
It replaces the intent with complete prepared bytes, replaces the state
with complete proposed bytes, then removes the intent. Completion is
reported only after all three operations succeed, in that order.

The caller must already hold the stable transaction lock and must keep it
through this operation.

Before calling, it must validate the observing
intent's ownership, exact predecessor, permitted transition and encoded
records. This helper neither validates those records nor grants admission.
Its result is evidence about publication steps, not a launch token.

The candidate private basenames are `intent` and `state` in one selected
clean absolute parent. Distinct fixed names avoid accepting two arbitrary
spellings that might identify the same entry on a normalizing filesystem.

They are not an installed shared file contract. Integration must settle
the root, codec, names and cross-participant locking together.

## Outcomes

The zero result is unknown. Prepared-intent failure prevents state
replacement and intent removal. State failure prevents intent removal.

Intent-removal failure reports uncertainty, even for an absent entry;
it does not acknowledge completion. The original cause remains available
through errors.Is, including any temporary cleanup causes.

The existing publisher owns each transferred temporary and its one close.
This helper never retries, removes the state, or deletes the intent as
rollback after either publication fails. A failed rename may have changed
the namespace, so the recovery reader must inspect the actual records.

Even successful completion does not prove a disk flush or power-loss
survival. Empty and binary payloads are valid at this opaque byte layer;
the future codec must separately reject invalid authority records.

The caller must not clear an observing intent on error or sleep under
the transaction lock. Known disk loss must still stop the relevant work
if publishing its proposed stop fails. Neither process signaling nor the
earlier measurement and stop transition is implemented by this helper.

## Tests before the body

Generated filesystem and temporary mocks verify the exact two full
write/close/rename sequences, then the single intent removal. Paired
refusals cover invalid parents and a missing filesystem, failed intent
create/write/close/rename, and failed state create/write/close/rename.

Failed intent removal must not acknowledge completion.
No unexpected later step or destination deletion
is permitted. Joined cleanup errors must retain every cause.

Two Rapid properties check exact independently generated prepared/state
bytes and failure at each publication phase. A successful-looking zero
result is separately refused.

Real temporary-directory witnesses inspect
the final state and absent intent, and retain the prepared intent when
state replacement is obstructed. They test the complete local adapter,
without selecting a production root or claiming transaction exclusion.

Compile all types and tests against an empty function first. Read the
actual assertion failures before implementing this one function. Follow
with the focused green check and small scoped mutation checks; wider
acceptance remains at the branch review boundary.

The stub compiled after a test argument-type repair. Thirteen of fourteen
tests then rejected the empty body through assertions; the zero-result
anti-test passed. Both exact Rapid repros were read and kept unchanged.
The first body passed all 76 publication tests and 100% statements.
