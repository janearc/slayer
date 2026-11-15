# Render admission and Azazel

Game lanes coordinate work on the shared development machine today.
Azazel contains the pieces for a future accountable render queue, plus
a direct rendering adapter. Its internal admission results are not live
permission to start a job. A job's `limits` block does not establish an
enforced process-wide ceiling on its own.

## What the integrated code provides

The private budget code validates resource decisions and stop transitions.
Owned reservations carry a unique reservation ID, owner and incarnation,
with checked byte arithmetic, independent next-state storage and exact
release matching. Stale owners or duplicate IDs are refused. Supplying
an owner string does not authenticate a process.

The Darwin transaction holds a stable nonblocking lock through observation,
fresh host sampling and publication. It binds records to predecessor,
boot, policy and limits, writes intent before transition, and returns no
successful internal outcome before publication completes. A known low-disk
stop stays latched; favorable later measurements do not silently clear it.

Native sampling reads load, swap, memory pressure and available filesystem
space without launching external sampler processes. Native process
observations and locking/publication helpers have their own checked
interfaces. These are observations and transactions under explicit trust
boundaries, not proof that every descendant has stopped.

See [owned reservations](../azazel/OWNED-RESERVATIONS.md),
[locked transactions](../azazel/TRANSACTION.md),
[publication](../azazel/internal/publication/README.md),
[locking](../azazel/internal/locking/README.md) and
[process observations](../azazel/internal/processinfo/README.md) for the
exact ownership, failure and platform contracts.

## What you can run

`azazel render` wraps the set-rendering engine also used by the older
`slayer render` path. It does not connect that render to the private
admission machinery. Ordinary jobs use `slayer draw JOB`; coordinate
expensive work with Game and the machine's queue policy.

The Azazel composite admission entry point deliberately refuses direct
starts while shared admission, reservations, locking and publication are
unavailable as a complete work-lifetime boundary. This is separate from
the explicitly run Omnipositor experiment. Having lower-level helpers or
a successful internal transaction must not be treated as a bypass.

## What remains to make a queue

Engine integration, authoritative ownership, complete resource accounting,
supervision, descendant recovery and crash/restart reconciliation remain.
The publication helpers' filesystem guarantees are narrower than arbitrary
machine-crash durability. A process snapshot or caller assertion cannot
by itself justify releasing capacity still used by running work.

These boundaries matter for large films: `GOMEMLIMIT` controls the Go
heap softly, payload counters cover selected buffers, and neither caps
all Metal memory. See [building](building.md) for the current Game workflow
and [optimizations](optimizations.md) for renderer memory accounting.
