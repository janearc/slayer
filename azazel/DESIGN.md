# azazel

Azazel carries rendering work inside Slayer. It keeps the host usable,
records what ran, and resumes work from verified results after interruption.
This document describes the work being built, including the parts that do
not exist yet. A completed primitive does not establish a working queue.

## boundaries

Azazel lives under `azazel/` in the Slayer module, on its own branch.
Its resource accounting must not import the tracer, Metal, catalogue
readers, or physics. Renderer adapters compose those packages later.

Nilgai normalizes observations; Celestial owns physical quantities and
models; Celestial-data obtains archive products. Those responsibilities
remain with their packages.

Game's existing lanes and Azazel's renders must draw from one host budget.
The interface between them is durable records and locks on disk. Reading
records is observation; acquiring an admission grant is authority to start.

A Carrie pane discovers and presents that state through its configurable
provider interface. Carrie does not schedule or own jobs.

The common admission operation must cover both reading the available
budget and reserving it. Two callers must not each spend the same room.

Every start needs it, including the first Game lane. Game currently measures
the host only before opening extra lanes; that is not this requirement.
The Game integration is a companion change, not something an Azazel-only
test can claim to establish.

## what a job carries

A submitted job names its requester, source and executable identities,
arguments, working directory, input hashes, expected outputs, resource
limits and timeout. Render tasks also name the scene, exact frame times,
dimensions, sampling, device and output roles.

Changing an identity changes the result key. An output path alone is not
an input identity.

The engine protocol follows the original typed message design. Messages
are generated from the contract, bounded and checked before action.
The revised wire and file contracts need review before integration.

An unknown feature must receive an explicit refusal; it must never become
a picture with that feature quietly omitted.

## admission and stopping

Admission evaluates measured host conditions and outstanding reservations
under a shared lock. Unknown measurements hold work. A hold
records its reason and does not turn a queued job into a failed result.
The scheduler's observation records remain readable while work is held.

Thread limits apply to each process. Worker counts and concurrent jobs
also consume the shared budget. The development limit is two Go CPUs per
process. A queue slot or a `GOMAXPROCS` value alone is not a memory limit.

Each job declares a memory ceiling. Admission reserves ceilings against
the configured job budget. Host-pressure admission is an additional check;
it must account for outstanding reservations that have not become measured
usage, without subtracting the same resident bytes twice.

The running-job monitor measures the job's process group, not just its
first process. Exceeding the job's ceiling stops the group. A sampled
monitor can react only after a sample; it is not a kernel-enforced cap.

The Darwin measurement and termination behavior must be established by a
bounded platform check before it is relied upon.

The operational guard for tonight is one heavy job, load below eight,
swap below eight GiB, and at least thirty GiB free disk. Under the disk
floor, running work stops and waiting work stays stopped.

A host stop is latched and needs an explicit resume; a later favorable
sample does not silently release the queue. These are configurable host
policies, not physical constants or permission for a caller to raise them.

## lifetime and results

An attempt has a private output directory and an identified process group.
It cannot publish a frame merely by naming it. Azazel verifies the bytes,
hash, dimensions, frame identity and expected role before publication.

Publication must preserve an existing immutable result, including a race
with another publisher. An incomplete attempt is not a completed frame.

The engine must deliver exactly the expected frames before `done` can be
accepted. A responsive process that makes no progress is stalled; progress
within a frame counts. An honest long first frame must not be killed merely
because no previous frame has supplied a timing estimate.

Cancellation is idempotent. Closing the parent's input must end an engine
within its stated grace, including one stuck while releasing resources.
Timeout and stop handling reach child processes. A restarted runner must
establish process identity before signaling an old group.

A failed test is a stored failed result, not a reason to rerun it until
it passes. Retry rules distinguish unavailable infrastructure from invalid
input and refusals. Verification failures and exhausted retries remain
visible.

Health, metrics and the result gallery report missing checks as missing.

## implementation order

1. Pure resource quantities and admission decisions, with tests and
   anti-tests derived from this document.
2. Typed durable records, atomic publication, reservations and stop state.
3. Generated file and engine contracts, reviewed before their integration.
4. Process identity, group supervision and host measurements, with generated
   mocks and bounded real-process checks where the operating system matters.
5. Queue composition and the companion Game admission integration.
6. Pzm entry points, declared configuration, flags, status and service health.
7. Slayer engine adapter, verified render results and restart behavior.
8. Gallery, cache, baselines and the remaining original-plan features.

Each step follows tests, anti-tests, a readme, types and function signatures,
verification against those signatures, then one function at a time.
Focused checks establish red, green and resistance to deliberate faults.
Acceptance review and the full relevant gates follow a ready branch.

## first primitive: reserved memory

The first function answers one question: does a positive memory request fit
inside the configured memory ceiling after existing reservations?
It takes a budget and request as separate private structures and returns a
predicate. Counts are unsigned bytes, not physical mass or a percentage.

- A zero request is refused, so an omitted ceiling cannot become unbounded.
- A budget already reserved past its ceiling is refused.
- An exact fit is allowed.
- One byte above the unreserved capacity is refused.
- Arithmetic must work through the entire unsigned range. It must never
  grant work because adding two counts wrapped around.
- The predicate performs no allocation, I/O, host measurement or reservation.
  Its answer is used inside the later locked admission transaction.

Tests compare small authored byte ledgers, full-width boundaries, and an
independent arbitrary-precision sum. Anti-tests cover zero and broken
budgets, spending existing reservations, and arithmetic overflow.
This predicate alone cannot start, limit or stop a job.

## second primitive: measured host admission

A pure policy decision evaluates a supplied host sample. It returns one
private decision value, carrying a reason instead of granting authority.
It reads no clock, process, file, configuration or flag service. The later
locked admission transaction must also check outstanding reservations.

The caller supplies a positive finite load ceiling, a positive swap ceiling
in bytes, and a positive free-disk floor in bytes. There are no hidden
defaults.

Each measurement has its own known bit: a measured zero swap is
different from a failed swap read. Load must be finite and nonnegative.
A zero swap ceiling is an invalid omitted threshold. To allow measured
zero swap and hold any positive byte, supply a ceiling of one byte.

Decisions have this precedence:

1. Invalid policy refuses admission.
2. An already latched stop remains stopped, even with favorable samples.
3. Known free disk below its floor reports low disk, even when another
   measurement is unavailable. Missing load must not mask a known disk stop.
4. Any unknown measurement or invalid load holds admission as unknown.
5. Load at or above its ceiling holds admission.
6. Swap at or above its ceiling holds admission.
7. Otherwise the host conditions permit admission. Exact disk floor is
   permitted; exact load or swap ceiling is held.

The zero decision is unknown, so an omitted decision is never a grant.
The function does not write the stop latch or terminate a process. A later
state transition must persist the low-disk stop before any later sample can
release work. This predicate cannot establish the monitoring or stop path.

Tests pair safe cases with unknown, invalid, stopped and boundary cases,
including every combination of measurement availability. Rapid checks
that increasing free disk cannot withdraw a ready decision, while changing
numerical conditions cannot clear a supplied stop latch.

Those properties test policy invariants independently of its branch order.

## prerequisites to resolve

The October plan requires Pzm's share/idler and sealed configuration verbs.
Pzm main `804658c3` reserves `share`, `check-config`, `edit-config` and
`seal` but does not implement them. They must be provided by Pzm; Azazel
must not install another parser or declare its own versions.

Flipr is not required. Admission uses host measurements, outstanding
reservations and the latched stop; no external flag-service decision is
a prerequisite for starting work.

The original plans are in Vanguard `4d51ba4` and Lamb-debacle `f7e5ddc`.
The October review's root location is superseded by the current Slayer
location. Its frame-count, stall-clock and verification findings remain
requirements.

Historical descriptions of renderer APIs require checking against the
current Slayer base, `57510cb`.
