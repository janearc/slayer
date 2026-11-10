# Keep the first result

This publisher writes a complete private file, closes it, and asks the
filesystem to install it only if its destination is absent. Another
attempt's result keeps its name and bytes.

An existing destination is a distinct refusal. It is not reported as this
attempt's success, and it is not assumed safe to reuse. A later verifier
must establish its content and provenance before reuse.

Failures clean up only this attempt's temporary. An installation error
may have an uncertain effect, so the destination is left for reconciliation.

Write, close, installation and cleanup errors retain their causes. A
failed close gets one attempt, and a successful installation leaves no
temporary to remove.

Generated mocks check the exact write/close/install order. Tiny native
tests connect those steps to real filesystem exclusion. The upstream
caller still owns result verification, root protection and durability.
This layer grants no permission to start work.

See [the design](IMMUTABLE-DESIGN.md) for the boundary and paired checks.
