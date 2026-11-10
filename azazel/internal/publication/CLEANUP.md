# Keep ownership until publication

The publisher gives deferred cleanup a named resource owner without
changing its filesystem contract. The helper first passed independent
red and green witnesses. Replacement now delegates to it and passes the
same thirty-six publication witnesses at eb026cb.

Create its owner only after successful temporary creation. It retains
the trusted filesystem, opened temporary and whether close was attempted.
It never owns the destination. Failed creation transfers nothing, so the
caller must retain its existing no-close, no-remove witness.

The deferred method changes the caller's final result in place because
cleanup failures occur after the primary result is selected. Returning
an ignored value from a deferred method would discard those failures.
This side effect belongs to the operation that already owns that result.

Use a pointer receiver: deferring a value receiver would capture the old
close state and permit a second close after the normal path closes it.
The caller must mark close attempted before calling Close, even if that
call fails. Register cleanup once, after successful creation only.

The method's only further side effects are closing the owned handle if
necessary and removing its private temporary name. On confirmed visible
publication it does neither. It joins every cleanup cause with the primary
cause, preserving the stage.

It does not rename, publish or remove a destination. No concurrency,
retry or repeated-cleanup contract is offered.

The caller supplies a nonnil final-result pointer and a trusted opened
temporary. A rename refusal remains uncertain even after cleanup succeeds.
This helper selects no root, grants no work and establishes no durability.

## Paired witnesses

Three examples establish close-before-remove, removal after a prior close
attempt, and preservation of all primary and cleanup causes. A named Rapid
property varies prior close, cleanup refusals and the primary failure stage.

Four anti-tests forbid touching a visible publication, retrying a failed
close, converting rename uncertainty to success, or converting a removal
failure to visibility. Generated mocks permit only the owned temporary
path, rejecting destination removal, rename and unexpected calls.

Write these tests first, then the friendly readme, then the owner type and
empty method stub. Read actual compile and assertion-red evidence before
implementing this one method.

Passing helper tests alone do not prove replacement delegates to it;
that later edit needs the existing publisher
witnesses, including failed creation and native composition.

The types compiled at cf41374. Seven witnesses rejected the empty stub
through assertions; the visible no-op anti-test passed. The generated
all-false counterexample is retained unchanged.

The method was written after those actual results were read. Both source
reviewers accepted b0b999b before its passing run: thirty-six publication
tests, both Rapid properties with 100 cases each, and full production
statement coverage.

Only after those full results were read did replacement delegate its
deferred cleanup to this owner. Its failed-create path still registers
nothing; its normal close marks this same owner before attempting Close.
Test source and both retained counterexamples are unchanged.
