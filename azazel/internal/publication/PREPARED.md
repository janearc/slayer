# Finishing a prepared update

Once you have checked an operation and still hold its transaction lock,
this helper can publish the last three steps for you. It writes the
prepared intent first, writes the new state next, then removes the intent.
Only that complete sequence produces a completion result.

If either write fails, the helper leaves the intent for recovery. If
removal fails, it reports uncertainty instead of guessing what happened.
You can still inspect each original error with errors.Is. The existing
snapshot publisher takes care of each private temporary file.

You own the lock, the checked records and the selected parent directory.
Keep the lock until this call returns. A completion result is about file
publication; the admission layer must still decide whether it can issue
a grant.

This byte helper chooses no live root, parses no record and
makes no disk-flush promise.

The [design](PREPARED-DESIGN.md) describes the failure boundaries and
paired tests. The first body passed all 76 publication tests, including
the two new Rapid properties at 100 trials each, with 100% production
statements. Wider acceptance remains open.
