# Shared job queue

This package owns the bbolt ledger and Azazel's foreground scheduler.
[Scheduler design](../../DESIGN.md) describes request identity, admission
and restart behavior. [The operator guide](../../../doc/azazel.md) documents
submission, tuning and recovery.

Do not replace a live lock file. The dispatcher owns `worker.lock`. An
attempt's lock stays with its supervisor and is inherited by its command.
A process group or inherited lock can survive the supervisor. Recovery
signals no stored PID and keeps work fenced while either is live.

Renderer commands must remain in their foreground process group.
Attempt status records process outcomes. Image verification remains with
the renderer or the result consumer.
