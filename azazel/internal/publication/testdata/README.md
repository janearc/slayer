# Keep each byte witness exact

`payloads.yaml` preserves the original write, rename and publication
examples. Their byte orders differ, so each keeps its own named data.
The replacement examples also keep a nil slice and an allocated empty
slice as separate inputs, followed by binary bytes and the text payload.

Pzm decodes the bytes directly. Setup checks the input roles and the
generated zero-length boundary. Partial-write counts are checked against
the actual fixture body, including the one-byte shortfall.

The generated payloads still range from zero through 256 bytes, with
the same Rapid label, trial reporter and fresh mock controller. Expected
outcomes and ordered calls remain in Go. Native readers, sibling files,
inode identity and permissions keep their original checks.

These are tiny test fixtures. The data grants no filesystem authority,
changes no production defaults and establishes no disk-flush guarantee.
Retained counterexample bytes stay unchanged.
