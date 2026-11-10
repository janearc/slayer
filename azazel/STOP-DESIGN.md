# Host stop value and durable latch draft

The private pure value transition is implemented and tested. The durable
stop record, resume command and process termination are not implemented.
This extends the two admission decisions in `DESIGN.md`. The durable record
contract remains a draft and needs review before Game or Carrie shares it.

## The state and the event

A host stop is separate from an ordinary admission hold. High load,
unavailable measurements or unavailable flags can hold waiting work.
Known disk below its configured floor must also stop running work and
keep the host stopped until explicitly resumed.

The private state is a stop latch and a generation number. The
generation identifies the state an operator inspected; it is not a PID,
wall-clock timestamp or process birth identity.

The event carries a fresh host decision, an explicit resume
request if there is one, and the generation that request expects. A
periodic favorable observation is not a resume request.

The pure transition returns one result: the proposed next state and a
disposition explaining whether it changed, held, or refused the event.
It does not write the state or signal a process. A returned proposal is
not an acknowledgement that the host has resumed.

## Transition rules

- Low disk latches the stop, regardless of a simultaneous resume request.
- Repeating low disk on an already stopped host is idempotent.
- Ordinary observations never clear a stop, including a ready decision.
- An explicit resume can clear a stop only with a fresh ready decision
  and the expected current generation.
- A stale resume cannot clear a newer stop.
- Unknown, invalid-policy, high-load and high-swap decisions do not
  authorize a resume. An unknown enum value must not authorize one either.
- An already clear host remains clear without manufacturing a transition.
- State changes advance the generation without wrapping. An exhausted
  generation must hold the host stopped and refuse resumption. Exhaustion
  is checked first and forces a stopped result even for a supplied clear
  state at that generation; it cannot advance or wrap the generation.
- The last generation is reserved for stopping: a resume that would
  consume it instead advances to that exhausted, stopped state. Every
  successful resume therefore leaves a generation available for a new stop.

The zero state is clear at generation zero, not permission to start. A
ready admission decision, a reservation and the flag check are still
needed. An event with no explicit resume leaves the latch alone unless
it reports low disk. A resume on an already clear state is idempotent.

For a stopped state, a ready explicit resume with a stale generation is
refused as stale; other ineligible resume events hold it stopped.

Repeated low disk within an existing stop deliberately keeps its generation.
A resume inspected earlier within that same stop can still apply once a
fresh ready decision is supplied.

Only a new stop after a completed resume invalidates that earlier token.

Tests should pair each permitted transition with the refusal on the other
side of its boundary. Rapid should exercise event sequences and assert
that a stop never clears without an eligible explicit resume. A separate
authored sequence must catch a resume that was prepared before a new stop.

## The transaction around it

All participants use the same shared admission lock. Inside it the owner
reads the current stop record, validates its version and integrity,
obtains the measurements needed for its decision, computes the transition,
and publishes the record atomically before acknowledging a change.

Fresh host conditions must be evaluated without short-circuiting on the
old latch; the old latch is the transition's state input. The ordinary
admission path continues to refuse a stopped host. This distinction must
be explicit in the composing code.

Evaluate a resume without clearing the persisted latch first.

A failed read, unknown record version or failed publication grants no
authority to start or resume work. After a failed stop write, the runner
must still stop running work and keep admission held.

Failure to save a stop cannot become permission to continue.
Crash recovery must not infer a successful resume from a missing temporary
file or a favorable sample.

The filesystem boundary needs generated mocks and bounded real-filesystem
tests for locking, atomic publication and crash leftovers. Those checks
are separate from proving a pure transition. PID identity and descendant
termination remain the process supervisor's responsibility.

## Recovery question to resolve before the record is implemented

Holding the lock after a failed stop write protects only a live owner.
If that owner dies and the disk later recovers, an old clear record cannot
tell another participant that a stop was observed but never saved. Atomic
rename alone cannot repair that missing history.

The admission contract therefore needs a durable in-progress intent or an
unclean-owner rule that holds recovery until explicitly cleared. It must
not copy Game's current dead-ticket cleanup as permission to resume.

Game currently tolerates incomplete ticket lines for observation and clears
unlocked dead tickets.

A shared stop record needs a stricter contract.
This draft does not yet choose the journal/owner protocol. A pure latch
transition must not be presented as proof that crash recovery is solved.
