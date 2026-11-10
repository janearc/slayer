# Let the first result keep its name

An attempt may finish after another attempt has already published the
same result. Installing its file must leave the first result alone.
The mutable snapshot publisher deliberately replaces old state, so it
cannot make this promise for results.

The private Darwin Install operation moves a completed temporary into an
absent name. An existing file, directory or symlink makes it refuse.
The caller keeps the native cause and reconciles any uncertain outcome.
There is no fallback that replaces the destination.

Its tests use tiny local files, including two writers released together.
Exactly one may install; the other must retain its own temporary. When a
result is already present, both writers must leave it intact. These
checks need actual filesystem behavior and do not render anything.

The first implementation passed all 44 publication tests, including the
eight new tests and both new 100-case byte properties. Every production
statement was covered. Independent final acceptance remains open.

This operation is one part of a future immutable publisher. It does not
check that bytes are a valid result or that an existing result can be
reused. Root protection, storage durability and admission remain separate
work. See [the design](EXCLUSIVE-DESIGN.md) for the exact boundary.
