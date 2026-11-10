# Storage capability design

These are bounded Darwin checks for the filesystem operations the future
publisher needs. They write only small files in the test's temporary
directory. They create no admission root and change no host settings.

The checks measure three small `File.Sync` calls, report both Go's directory
sync and the direct directory `fsync` result, and check replacement behavior
with a reader that already has the old file open. Paired refusals check a
closed descriptor and a rename of an absent source, refused before change.

That specific refusal keeps the original destination; an arbitrary rename
error can have an uncertain effect and needs higher-level reconciliation.

Go 1.27.3's Darwin `File.Sync` uses `F_FULLFSYNC`, with an `fsync` fallback
on `ENOTSUP`. The direct directory check is reported separately. A Go
directory-sync refusal is recorded as an unsupported variant, never called
a successful sync.

Direct directory `fsync` must succeed for this candidate publication plan
to proceed.
A separate one-byte child is created before each directory operation, so
the direct path is not timed solely on the directory just flushed by Go.

Run these through the coordinated Game queue. Passing establishes the
observed operations on this filesystem, not a durable transaction, a
hardware power-loss result or a measured production observation cadence.

The storageprobe build tag keeps these explicit flush requests out of an
ordinary suite. The Game recipe opts in. Go's successful Sync call does
not reveal whether its full-flush path or fallback supplied the success.
