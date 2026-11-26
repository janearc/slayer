# Snapshotting explicitly named inputs

Snapshot copies a caller-enumerated list of regular files to a newly created
private directory. It returns each relative name, the bytes actually copied
and their SHA-256. The caller supplies a positive maximum for the total copied
bytes. Names are clean relative paths that stay below the new directory;
duplicates and file-versus-directory collisions are refused.

The source's final path component must be a regular file rather than a
symlink. A source changed or replaced while it is being opened is refused
where detected. The caller must keep sources stable during the copy and keep
other writers out of the snapshot directory; hashing a copy does not prove
that the original remained unchanged while read. The returned manifest
describes the copied bytes, not what a later renderer actually reads.

The caller is responsible for enumerating the complete input closure,
connecting the renderer to these copies, and preventing later mutation.
This package does not discover inputs, certify provenance or promise durable
storage. It creates the target only when no path already occupies that name.
On failure it attempts to remove only the directory it created and reports
cleanup failure. The target's parent must already exist.
