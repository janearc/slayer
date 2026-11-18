# bookshelf review for the agent queue

Reviewed on 2026-11-15 against a fresh clone of
`the-vorrh/roots/bookshelf-road.git`, commit
`592acbfcacc815863659cdceba3ade1a2398a6c8`. The clone is in an agent's
`work/bookshelf-azazel-guidance-2026-11-15`. It has no local edits.

## choices

The queue schedules an executable, argument vector and working directory.
Slayer remains responsible for scenes, instruments, films and its other
operations. This follows `engineering/general/design.md`: compose the
existing primitives instead of adding a second renderer inside Azazel.
New Slayer verbs do not require a scheduler implementation of that verb.

The submitted YAML or JSON passes through Pzm's schema reader before it
can become a job. Unknown fields, duplicate keys, extra documents and
oversized requests are refusals. The job preserves arguments as an array;
it never interprets them as shell text. This follows
`engineering/general/contracts.md`.

An agent and its request key identify one submission. An identical retry
returns its original acknowledgement; reuse with different content fails.
The ledger is outside the worker, in a private bbolt database. Only one
worker holds the queue lock. These choices apply the idempotency and
external-state requirements in `engineering/general/services.md` to the
local command interface. There is no network service in this patch.

Unknown host observations hold unstarted work outside Game's lane. A
known disk stop latches the queue until explicit resume. Uncertain job
lifetime prevents another Azazel job from starting. Failures remain failed;
they are never retried to obtain a passing result. A cancellation racing
with a host observation must remain cancellation, not job failure.

Tests use generated mocks with checked expectations at the execution,
resource observation and filesystem boundaries. Bounded real subprocesses
exercise argument forwarding, failure status, cancellation and the Game
handoff. Handwritten canned executors and monitors were removed after
reading `engineering/general/testing.md`.

The existing two-CPU limit remains in force. The shelf's general four-CPU
example does not override the stricter Slayer and workspace instructions.
Checks and builds use Game with one heavy process at a time.

## renders

`briefs/the-render-brief.md` and `renders/AGENTS.md` govern render jobs:

- Stills go directly under `renders/frames/DATE-OWNER-WHAT`; films under
  `renders/animations/DATE-OWNER-WHAT`.
- The recipe and set accompany the pictures. The recipe identifies the
  command, revision, settings, start time and known defects.
- The team's notes identify active work before it starts. Gallery updates
  run while the producer lives and once more after it ends.
- A still expected to take over five minutes gets a quick preview in the
  same folder. Films become visible from their first frame.
- Barkeep's progress writer respects another active writer. Job acceptance
  is not render progress, and process success is not picture verification.

The generic scheduler must preserve those destinations and tool arguments.
It must not hide partial images in scratch until a job completes. The
gallery and Barkeep integrations still need implementation and smoke proof
before the render workflow is complete.

## integration inputs

The branch includes these inspected tips. A merge records available source;
it does not certify every imported feature as finished.

| tip | work informing the queue |
|---|---|
| an agent `e3d942c1` | imported assets, Oakland, catalogue orbits, Ferruccio history |
| an agent `937c5310` | searching plans, films, progress and render memory policy |
| an agent `b894a90d` | atmosphere, terrain and Venera jobs |
| Dikdik `ba6de66a` | catalogue maturity reporting and framing corrections |

The Ferruccio UI remains subject to its restart brief. Importing its
history does not resolve the rejected design or nested-module integration.

## outstanding acceptance work

The shelf's bar is unchanged. A passing focused suite is not acceptance.
The queue package initially measured 95.7% after the mock change;
it has not met the required 100%, mutation and independent review gates.

One focused mutation restored the cancellation bug through a compiler
overlay. The dedicated regression test failed on the incorrect failed
state. This establishes that regression's sensitivity, not a complete
mutation pass for the package.

The generic foreground runner does not yet provide the full design's
shared memory reservations with unrelated Game jobs, verified render
publication, source-revision receipts or autonomous descendant recovery.
Process groups do not contain descendants that deliberately leave them.
An operator's recovery attestation is not an operating-system proof.

The former host swap guard held the initial native render smoke. That
attempt produced no output. Later native render results under the approved
pressure policy are recorded in [validation](VALIDATION.md), with current
coverage and remaining migration blockers.

On 2026-11-15 the memory policy was explicitly changed to system pressure.
The queue records peak group RSS, removes the inherited Go heap limit, and
no longer stops or holds work solely for RSS or accumulated swap. Elevated
pressure and the existing disk floor remain latched stops.
