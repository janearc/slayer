# Replacing one local snapshot

This private helper writes complete bytes to a temporary file beside the
destination, closes it, then renames it into place. A successful result
says publicationVisible. An empty result grants no permission to work.

A write or close error leaves the destination where it was. Cleanup
closes once and removes our temporary name, keeping every error cause.
A rename error says RenameUncertain: the caller must reconcile what
happened. We never delete the destination to undo a rename.

The Darwin adapter creates files with mode 0600 before umask. It follows
clean absolute linked parents. Choosing and protecting the root belongs
to its caller. Remove operates on one entry and can remove an empty
directory; the publisher gives it only its own temporary-file name.

There is no disk flush here. Jobs, results and shared admission need
their own recovery and durability policies.

The [design](DESIGN.md) and [adapter plan](LOCAL-DESIGN.md) record the
thirty-six passing tests and 100% production statement coverage.
Mutation and broader gates remain open.

The [named cleanup](CLEANUP.md) passed all thirty-six publication tests.
Replacement uses it and passes the same witnesses at eb026cb.
