# azazel

Start with [the Azazel guide](../doc/azazel.md) for commands, implemented
components, test entry points and the boundary of the unfinished queue.

Azazel is Slayer's planned queue for frames, films and checks.
The [design](DESIGN.md) describes it; no runnable queue exists yet.

You should be able to ask for a render, leave it waiting, and come back to
a verified result while the machine stays usable. Its record will identify
the executable and inputs, resource use and checks. Interrupted work will
resume from verified outputs.

The private pieces cover memory and host decisions, stop transitions,
[snapshot publication](internal/publication/README.md) and
[nonblocking locks](internal/locking/README.md), plus
[native process observations](internal/processinfo/README.md).

The private pieces also include [owned reservations](OWNED-RESERVATIONS.md)
and [locked admission/publication transactions](TRANSACTION.md). Their
successful results are checked internal state transitions, not live grants.
Read [render admission](../doc/render-admission.md) for the current user-facing
boundary between Game lanes, direct rendering and the unfinished queue.

The [local design](internal/publication/LOCAL-DESIGN.md) records publication
witnesses and source reviews. Selected comments have a reviewed Qwen pass.
Final independent acceptance and private test-boundary design remain open.

[azazel/engine](engine/README.md) draws one set file through the library
`slayer render` uses, and `azazel render` is the command over it. It is not
the queue, and the pieces above are not wired to it.

Shared admission, recovery, supervision and engine integration remain.
The command and configuration will use Pzm. Flipr is not required.
