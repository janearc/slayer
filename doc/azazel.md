# Azazel

Azazel is Slayer's render-work and future queue subsystem. Today it offers
a direct set-rendering command and private components for resource decisions,
locking, observation and publication. Those components do not yet compose
into a runnable shared queue. Game lanes remain the current development
coordination mechanism.

## What works today

| Part | Current responsibility |
|---|---|
| `azazel render` | draw one set through the Hillside library |
| `azazel/engine` | adapter returning Hillside's result/exit status |
| private budget logic | checked host decisions, latched stops and reservation proposals |
| owned reservations | checked add/release with owner and incarnation identities |
| Darwin transactions | locked observation, fresh sampling and predecessor-bound publication |
| publication and locking | bounded filesystem operations under explicit ownership contracts |
| native process observations | checked process facts, not proof of complete group quiescence |

The longer [admission guide](render-admission.md) explains the implemented
transaction boundary. The [design](../azazel/DESIGN.md) describes the intended
queue, including features that are not available yet.

## Build and run the direct renderer

After [build setup](building.md), use the declared custom target for
Azazel's command:

```sh
game build azazel-binary
bin/azazel --age
game lane gpu -- game run azazel -- render doc/observatory-ecosystem.yaml -out out/azazel-observatory.png
```

The custom target uses raw compilation rather than the ordinary stamped
build. Its age can be empty; do not use it as release provenance. This
example set needs its declared source data and Metal backend. Pick a small
supported set and check its configuration before spending a large render.

`render FILE [settings]` passes the set and settings to `hillside.Run`,
the library used by `slayer render`. It does not submit the file to a
queue, reserve shared resources or supervise descendants. For ordinary
jobs with films/sound/titles, use `slayer draw JOB`.

The CLI returns usage status 2 for a malformed invocation. Render results
follow Hillside: successful output, input/setting refusals or staging/draw
failure. Preserve the actual error, not just the number, in a report.

## Composite admission deliberately refuses

`azazel composite SETTINGS` currently reports that shared admission,
resource reservations, locking and publication are unavailable as a live
work-start boundary. Its presence is a refusal interface, not an alternate
working compositor command. Experimental Omnipositor runs are a separate
explicit workflow with their own limitations.

## What a future queue must guarantee

A start must atomically observe capacity and reserve it, so two callers
cannot spend the same available memory. A job must bind source, executable,
arguments, input hashes, exact frame times and expected outputs. A held
job needs an observable reason; an unknown measurement cannot mean zero.

Stopping must cover the work's actual lifetime and descendants. Capacity
cannot be released merely because a parent exited or a caller says it
stopped. Restart/recovery must establish process ownership before signaling
anything. Verified output publication must distinguish complete results
from interrupted work and preserve existing immutable results.

The private reservation/transaction code checks supplied bindings; an owner
name is not authenticated process identity. The local publication helper's
rename is not a universal machine-crash durability guarantee. Completing
these primitives does not establish end-to-end supervision or recovery.

## Testing and extending it

Start with pure decision and refusal tests:

```sh
game test -- -count=1 ./azazel/internal/budget ./azazel/engine ./azazel/cmd/azazel
```

Platform filesystem/process checks need their actual supported host. The
storage probe uses an explicit `storageprobe` tag; it is not enabled by
the ordinary draw tag. Follow its package instructions before running it.
Keep tests bounded to controlled temporary files and processes, preserving
unrelated host work. Read [testing](testing.md) for lanes and evidence scope.

Resource logic must stay independent of the tracer, Metal and scene builders.
Renderer adapters belong at the engine boundary. Internal zero/unknown
results must remain non-authorizing; preserve causes on native failures.
Use the component guides for exact contracts:

- [Owned reservations](../azazel/OWNED-RESERVATIONS.md).
- [Transactions](../azazel/TRANSACTION.md).
- [Publication](../azazel/internal/publication/README.md).
- [Locking](../azazel/internal/locking/README.md).
- [Process observations](../azazel/internal/processinfo/README.md).

Full shared admission, engine/Game integration, monitoring and recovery
remain required before describing Azazel as a usable render queue.
