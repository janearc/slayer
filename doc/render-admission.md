# Render admission and Azazel

Game lanes coordinate work on the shared development machine. Azazel has
a foreground command queue, plus the older direct rendering adapter. The
queue enters Game for each selected job. See [the human guide](azazel.md).

## What you can run

`azazel submit` records a request, and `azazel work` dispatches requests in
order through their Game lanes. The worker checks host conditions before
entering Game and again before execution. Unknown measurements hold work.
Waiting for host conditions does not retain a Game lane.

The current memory policy measures system pressure and records sampled peak
group RSS. It does not stop solely for an RSS estimate or accumulated swap,
and removes inherited `GOMEMLIMIT`. Elevated pressure and low disk latch a
stop. Group sampling does not account for every GPU allocation or contain
descendants that deliberately leave the group.

`azazel render` wraps the set-rendering engine also used by the older
`slayer render` path. It does not connect that render to the private
admission machinery. Ordinary jobs use `slayer draw JOB`; coordinate
expensive work with Game and the machine's queue policy.

The Azazel composite admission entry point deliberately refuses direct
starts while shared admission, reservations, locking and publication are
unavailable as a complete work-lifetime boundary. This is separate from
the explicitly run Omnipositor experiment. Having lower-level helpers or
a successful internal transaction must not be treated as a bypass.

## Memory limits

For large films, `GOMEMLIMIT` controls the Go heap softly, payload
counters cover selected buffers, and neither caps all Metal memory. See [building](building.md) for the current Game workflow
and [optimizations](optimizations.md) for renderer memory accounting.
