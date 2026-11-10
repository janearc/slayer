# Borrow a file, hold a lock

This is a small private piece of Azazel's future admission transaction.
Give it a file handle you already own; it will try to hold an exclusive
lock without waiting. You get held, busy or failed, with the refusal cause
kept when it cannot hold the lock.

The zero result is unknown. It never grants permission to start work.
An already-held handle can report held again; there is no nesting count
that makes it safe to close that handle early.

The caller keeps the file open and its inode stable. The release method
releases that handle's lock while leaving the file open. A failed
release keeps its cause; it cannot establish that ownership has ended.

The [acquisition design](DESIGN.md) and [release plan](RELEASE.md) explain
the boundaries. Tests use tiny real files: another open handle must
observe the kernel's ownership. A successful-looking return is not enough.

At 65aff04, all sixteen acquisition/release tests passed, including both
Rapid properties with 100 cases each and both retained counterexamples.
Production statement coverage was 100%. Mutations and the remaining
shared admission work are still open.
