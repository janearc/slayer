# Storage probe

Before building the publisher, we need to know what the local filesystem
actually does. These six paired checks time small file flushes, compare
the two directory-sync paths, and examine a successful and failed rename.
They write only temporary test files.

All six checks passed at 2f91c81 under the coordinated queue.
The result log preserves the three file
timings and both directory observations; it does not identify which
internal path Go's successful Sync used.

The temporary volume was not identified in that log, so do not transfer
the result to a later root.

The [design](DESIGN.md) says exactly what each observation can establish.
The measurements are not a production cadence or a durability guarantee.
Run the `azazel-storage-probe` Game recipe through the coordinated queue.

Its explicit storageprobe build tag keeps drive flushes out of ordinary
test suites. A comment correction narrows the failed-rename check to its
actual absent-source refusal. No further flush run is needed for wording.
