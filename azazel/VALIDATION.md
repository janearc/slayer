# queue validation, 2026-11-15

## heavy film admission

The original three GPU jobs completed. Transferring twelve remaining film
chunks exposed a staging problem: four scene builds started within seventeen
seconds and system memory pressure became elevated. The queue latched a stop,
retaining four terminal failures and eight pending requests. Inspection found
no PNG outputs from those four attempts and no surviving process groups.
New keys and fresh destinations are required for their deliberate replacements.

The worker now accepts `--gpu-jobs 1` through `4` so an operator can measure
one additional heavy scene before increasing overlap. The bound is forwarded
to the actual Game GPU coordinator and appears in status. Load average alone
no longer blocks admission: a regression with load 24 and normal pressure
failed before that repair. Elevated pressure and low-disk protections remain.
This run does not establish sustained full GPU utilization.

The one-extra-job run also triggered pressure alongside the existing direct
film chunk. Its recorded peak group residency was 10.53 GB before stopping;
that excludes GPU allocation accounting. Further overlap was held pending
the existing chunk's completion. No failed request was silently replayed.

Independent review accepted the unstarted-work and live-lease recovery fixes.
It found an outdated agent-help assertion, missing restart regression coverage,
and an unresolved attempt that made the whole dispatcher exit. The assertion
now matches the shipped help.

Regression cases reproduce unknown-launch and live-group refusals, then
verify both remain fenced while CPU work can proceed; unstarted and
surviving-lease cases are also covered. Focused race checks pass. The
review's broader orchestration coverage request remains open.

## concurrent dispatch update

The worker now samples the named device-utilization counter, admits GPU
work within a four-attempt bound and CPU work within a two-attempt bound,
and spaces GPU starts by five seconds. GPU work shares an actual Game lane
under a batch coordinator; CPU work does not wait for that batch to finish.

The retry-identity regression failed for three explicit-empty input cases
before normalization, then passed through the CLI and durable ledger.
Independent review accepted that repair. Restart regressions reproduced
lost unstarted work and recovery during live execution before their fixes.

New checks require unstarted work to return to the queue and refuse recovery
while a supervisor or inherited command lease is live. A killed supervisor
test confirms its command retains the lease; only after that process group
ends can recovery record interruption. Old parent signals no longer cancel
work on the submitting agent's behalf.

The native built-worker proof started two GPU-lane test commands and one
CPU-lane command concurrently, killed the main dispatcher with SIGKILL,
and restarted it against the same ledger. All three survived and succeeded
with attempt one and exactly one start/end marker each.

The complete queue/CLI race run found one timing-dependent fixture that
cancelled before its child had started. Replacing that timeout with an
explicit recorded-process barrier made the focused restart/concurrency race
checks pass. The other full-suite checks passed; repeat full acceptance
after the remaining integration work.

Before shared dispatch, an exclusive database-file lock protected a backup.
Reopening that backup reproduced all eighteen queued requests, fingerprints
and the pause exactly. All eighteen requested picture destinations were
absent before dispatch. The first six shared jobs have since succeeded,
including the three GPU requests and the packaged GUI preview.

This establishes bounded dispatcher restart and command execution, not
arbitrary power-loss immunity, daemon containment or completed publication
acceptance. The historical serial evidence below remains separate.

This is evidence for the current branch, not approval for shared migration.
The queue is inside Slayer on `feat/azazel-agent-queue`. Merge `aa248a92`
includes the `937c5310` head. The queue changes were uncommitted when
the native smoke binaries were built; their submitted executable hashes
are retained in the validation queue.

## process and storage checks

Game ran the queue, budget and command suites with the race detector.
The queue suite includes independent concurrent submitting processes,
immediate client termination after acknowledgement, termination inside an
uncommitted transaction, and termination after claiming a job.

Acknowledged jobs remained present. The incomplete transaction did not
erase the predecessor. An abandoned claim blocked a replacement attempt.
The property test compares submission retries, conflicting reuse and
cancellation against an independent agent/key ledger.

Initialization now requires successful directory synchronization for the
new database and its parent directory. A generated mock established the
missing operation before the fix and verifies that synchronization failures
prevent acknowledgement. This is not proof against arbitrary device or
power failure.

The approved memory policy has positive and negative tests: an eight-GiB
resident observation with normal pressure does not kill the job or impose
a Go heap limit; warning or critical pressure stops admission; accumulated
swap with normal pressure does not reject admission. Peak measured group
residency and the pressure category are retained in the job record.

The CLI suite verifies every mutating verb's dry-run against unchanged
database bytes and absence of created queues or worker locks. The original
handlers failed these checks; intent previews now bypass those handlers.
Aliases advertised by Pzm reach the queue, and stray operator arguments
are refused. The shipped `--agent-help` explains these boundaries.

Last measured statement coverage was 94.8% for the queue, 96.7% for the
budget package, and 82.2% for the command package. These do not meet the 100% gate.
Focused vet passed. Full-project acceptance, the complete mutation gate
and independent reviews remain outstanding.

Full-tree Game lint also reports inherited prose and comment findings in
the merged scene and Ferruccio work. Queue-guide findings were repaired;
this is not a clean full-tree lint result.

Game sweep at `119a1c0d` reported 52 findings, including queue-facing
files as well as merged work: machine paths and private word-policy hits.
These remain unresolved. No distribution release or gate exemption was
created, and the branch has not been pushed.

## native jobs through Azazel and Game

The isolated queue is in an agent's
`reports/azazel-queue-v2/queue`. No production queue was installed, and no
migration or render pause was announced.

| job | observed result |
|---|---|
| CPU smoke still | PNG and raw output; PNG is 64 by 36 |
| CPU smoke film | three 64 by 36 PNG frames; H.264 is 64 by 48 with padded rows, 2 fps, 1.5 seconds |
| Heard individual 41 | 160 by 90 PNG through standard `slayer draw` |
| Oakland imported set | assessment and 64 by 36 PNG |
| Venera 13 terrain/air set | assessment and 64 by 36 PNG |
| unsupported Slayer verb | retained failure, exit 2 |
| schema after the refusal | succeeded; earlier failure did not stop dispatch |

The first still and film fixtures omitted the required Slayer `$schema`.
Slayer refused them. Both failed requests and their original inputs were
retained; explicitly corrected submissions received new keys and passed.
The ledger contains eleven jobs: eight successes and three retained failures.
No automatic retry converted a failed job into success.

The movie was independently inspected with ffprobe through Game. PNG
headers and dimensions were checked, and the heard image was inspected.
These tiny functional smokes establish execution paths, not scene maturity,
film completeness for arbitrary inputs, or rendering quality acceptance.

The worker waited behind other agents' live GPU jobs and resumed through
Game normally. Outputs went directly to the shared gallery folders:

- `renders/frames/2026-11-15-<owner>-azazel-queue-v2`
- `renders/animations/2026-11-15-<owner>-azazel-queue-v2`

The page watcher ran during production and rebuilt the indexes afterward.
Both folders appear in `renders/latest.html`. The progress watcher respected
an existing Barkeep writer. Those watchers were explicitly launched by the
validation run; automatic queue integration remains unfinished.
Recipes and measured limitations accompany the images. Reusable smoke
inputs are in [examples/validation](examples/validation/README.md).

## remaining migration blockers

The queue does not yet prevent a tool from overwriting prior outputs,
verify arbitrary expected outputs before publication, or provide complete
automatic descendant recovery. Backup/restore acceptance, gallery/progress
integration and resource coordination with unrelated Game jobs remain.
The remaining current workloads also need a maintained compatibility
matrix. A generic executable interface does not prove every workload.

Do not announce the render pause or migrate agents on this evidence alone.
