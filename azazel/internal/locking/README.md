# Borrow a file, hold a lock

This private helper provides a nonblocking file lock.
Give it a file handle you already own; it will try to hold an exclusive
lock without waiting. You get held, busy or failed, with the refusal cause
kept when it cannot hold the lock.

The zero result is unknown. It never grants permission to start work.
An already-held handle can report held again; there is no nesting count
that makes it safe to close that handle early.

The caller keeps the file open and its inode stable. The release method
releases that handle's lock while leaving the file open. A failed
release keeps its cause; it cannot establish that ownership has ended.

Tests use real files: another open handle must observe the kernel's
ownership. A successful-looking return is not enough.
