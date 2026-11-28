# Azazel

Azazel queues Slayer renders on a shared machine. submit a request
to the operator's queue, check its status, and collect the finished
files. the worker admits CPU and GPU work separately. Slayer still
reads the scene and job and writes the picture or film.

if you already have a queue path and an Azazel executable, start
with [submit work](#submit-work). to run your own queue, see
[build and discover](#build-and-discover) and
[create a validation queue](#create-a-validation-queue).

## Submit work

this CPU still writes `out/first-picture.png` and
`out/first-picture.pfm` in the checkout. save this request as
`request.yaml` and replace each `/ABS/SLAYER` with the checkout
path. use the versioned binary named by the operator for a shared queue.
request `lane: tests` for this CPU render or `lane: gpu` for a Metal render.

```yaml
agent: studio
key: first-picture-v1
command: /ABS/SLAYER/bin/slayer
args:
  - draw
  - doc/examples/jobs/first-picture.yaml
directory: /ABS/SLAYER
inputs:
  - /ABS/SLAYER/doc/examples/jobs/first-picture.yaml
lane: tests
timeout_seconds: 300
```

```sh
bin/azazel submit --queue /ABS/QUEUE request.yaml
bin/azazel status --queue /ABS/QUEUE
```

save the sequence number from the acknowledgement. find it in the
JSON returned by `status`. wait for `succeeded` before opening the
outputs. inspect the picture and raw file separately.

Use the same agent, key and request when retrying an uncertain submission.
Azazel returns the original acknowledgement. Reusing a key with different
content fails. A new key means another job, so do not change keys merely
because the response was lost.

The command and declared inputs are hashed at submission and checked before
execution. Keep those files unchanged until
execution. Include referenced sets and assets in `inputs` when their
identity matters; Azazel does not discover every transitive input.

Pin a submitted command to its versioned binary path. A shared launcher can
change while the job waits, causing its identity check to refuse execution.
For comparisons between branches or revisions, give each build a distinct
executable path and each picture a distinct output path. Preserve both
binaries and all declared inputs until their jobs finish. Submit one request
per foreground invocation, with a different key for each intended job.

Only `SLAYER_*` environment overrides are accepted.

Jobs use Slayer's caches with the worker's environment and request overrides.
Operators should set shared `SLAYER_CACHE`, `SLAYER_RENDERS` and source-shelf
paths explicitly. Persistent GPU trees live in the cache home, `SLAYER_CACHE`, in
`trees.bolt`; `SLAYER_TREE_STORE=files` keeps a file a tree instead and avoids
the one shared database lock.

Keep authored `--fresh` flags: comparisons may deliberately require a new
picture. That flag does not disable every lower-level cache. Process-local
caches remain local to each submitted foreground command.

Any foreground Slayer tool can be scheduled this way, including `slayer`,
`catalogue` and `searching`. Do not detach children or daemonize. Interactive
applications are not batch jobs. A tool refusal remains a failed result;
Azazel cannot add a rendering capability absent from the selected tool.

## Build and discover

The queue record format is version 3. Older Azazel clients reject new ledgers;
new clients reject version 1 and 2 ledgers until an operator migrates them. Preserve
the old database and its logs before migration. Status uses a read-only
transaction and does not rewrite the ledger. `work --serve` stays alive while
paused, so `resume` alone releases the latch. Renderer executables submitted
as jobs remain pinned separately from the deployed Azazel client.

Build through Game with the source shelf configured. Before starting a
worker that runs Slayer render jobs, export `SLAYER_CACHE` with the absolute
path to the shared cache, or set it in Game config if Game starts the worker.

```sh
game build azazel-binary
bin/azazel --help
bin/azazel --agent-help
bin/azazel schema
```

`--agent-help` includes the submission workflow, a complete request,
state meanings, retry rules, resource policy and recovery instructions.
`schema` prints the checked YAML/JSON contract. Arguments remain an array;
they are never passed through a shell.

Pzm's `--dry-run` prints the requested action without invoking the handler.
It is an intent preview: it does not validate input files or queue state,
submit work, create a queue or acquire a worker lock. Its JSON includes
`dry_run: true`; it is never a submission acknowledgement.

The custom build target does not establish release provenance. Check the
build workflow before distributing a binary.

## Create a validation queue

choose an absolute queue path whose parent exists.

```sh
bin/azazel init --queue /ABS/QUEUE
```

in the worker terminal, export the absolute cache path so the
renderer processes inherit it.

```sh
export SLAYER_CACHE=/ABS/CACHE
bin/azazel work --queue /ABS/QUEUE --serve
```

run the worker in that terminal. It stays in the foreground and starts
independent render supervisors. Azazel owns its CPU and GPU render admission;
it does not launch Game to execute a render. A second main worker cannot
dispatch from the same queue. Direct Slayer jobs and Game GPU tests do not
yet share this admission.
Without `--serve`, the worker exits when no pending jobs remain.

`status.scheduling` names the sampled counter (`Device Utilization %`), its
sample time, target, active counts and admission hold. Unknown GPU readings
hold GPU starts. Host pressure and disk checks still govern both resources.

## Worker scheduling

The scheduling target is 100% GPU utilization with a runnable backlog,
with CPU jobs eligible alongside GPU work and admission governed by system
pressure. The worker samples device utilization and independently admits CPU
work. By default, GPU starts are spaced five seconds apart, with up to four
GPU attempts and two CPU attempts. These bounds include claimed attempts.

Use `work --gpu-jobs 1` to begin with one additional GPU job when large
scenes need a measured ramp. Values from one through 64 are accepted;
the default is four. `status.scheduling.gpu_limit` records the limit.

Stored live tuning takes precedence over the startup setting. Clear all
overrides with `tune --reset`; existing attempts continue. Reset restores
default pressure stops and five-second admission spacing.
See [the utilization policy](../azazel/DESIGN.md#utilization-under-backlog).

change a running queue without restarting it.

```sh
bin/azazel tune --queue /ABS/QUEUE --gpu-jobs 4 --gpu-start-seconds 1 --pressure-policy observe
```

Controls persist in the ledger. Coordinators reread them every second;
supervisors apply pressure policy while running. Concurrency accepts 1..64.

Reducing it leaves active jobs running and holds later starts. `observe`
records memory pressure without automatically pausing or killing work;
`stop` restores automatic pressure stops. Disk stops and unknown observations
still apply. Tuning does not clear an existing pause; use `resume` explicitly.
An unresolved fenced attempt retains its slot until recovery resolves it.

## Results, logs and stops

| state | meaning |
|---|---|
| queued | accepted and waiting |
| running | claimed by the worker; it may still be waiting for admission |
| succeeded | the foreground process returned success |
| failed | the failure and exit status are retained |
| cancelled | cancellation was recorded |
| interrupted | the operator resolved an uncertain attempt |

Read `held` to explain an admission wait. Read `result` for the terminal
reason and exit code. Logs are named `SEQUENCE.ATTEMPT.log`, with 20-digit
sequences and four-digit attempts, inside the queue directory. Existing
logs are never replaced. A successful worker exit means it drained the
queue; individual jobs can still have failed.

`executing` is retained execution-claim history, not a current liveness flag.
It may remain true after success or failure; use `state` and `result` for
completion. `process_group` is also historical. Clients must not signal that
identifier or treat its continued presence in JSON as proof of a live process.

Memory policy follows system pressure. The worker records sampled peak
process-group residency and the most recent pressure category. It removes
an inherited `GOMEMLIMIT`, does not impose a small Go heap, and does not
stop solely because RSS crosses an estimate. An optional `memory_bytes`
field in an older request is retained as metadata, not an enforced ceiling.

Load average and accumulated swap alone do not hold work. Render processes
default to eight Go workers. Set `SLAYER_CPU_WORKERS` in a request's
environment to choose 1 through 1024 for that job without restarting the
worker. A running process keeps the setting it started with; build and test
limits are separate.

With the default `stop` policy, elevated memory pressure latches a stop.
The `observe` policy records it and continues. Free disk below 30 GiB always
latches a stop.
Unknown observations hold new work
and stop active work. The monitor samples process groups; it is not a
kernel memory cap or complete accounting of GPU allocations.

```sh
bin/azazel cancel --queue /ABS/QUEUE --agent studio 1
bin/azazel pause --queue /ABS/QUEUE --reason "inspect host pressure"
bin/azazel resume --queue /ABS/QUEUE
```

Pause stops active work and retains waiting jobs. Resume explicitly clears
the latch; admission checks still apply. Failed jobs are not retried until
they pass. Inspect the cause and deliberately submit a corrected request.

## Recovery and retained data

The queue commits submissions before acknowledging them. It does not prune
history; at 10,000 retained jobs it refuses new submissions. Preserve the
queue directory and logs. Do not delete the database or lock file to clear
a stuck worker, and do not initialize over an existing queue.

A restarted worker adopts surviving supervisors without starting their jobs
again. Each claim has a renewable deadline in the queue; its supervisor
renews it while running. Expiry alone does not authorize replay. A command
inherits an attempt lock, and its process group is retained in the ledger.
Recovery refuses a held attempt lock or a process group still present. A
claimed attempt that never executed returns to the queue after those checks.

If a supervisor died and its recorded group has stopped, the attempt becomes
interrupted, with outputs retained and no automatic replay. An unknown launch
remains fenced for inspection. After confirming quiescence, the operator uses:

```sh
bin/azazel recover --queue /ABS/QUEUE --worker RECORDED_TOKEN --stopped 1
```

Recovery requeues never-started work; executed work records interruption.
It never signals an old PID. `execute` and `work-gpu` are internal handoffs;
agents must not call them. Commands must keep their inherited attempt lock
open and must not daemonize or escape their process group.

Stopping the dispatcher leaves active supervisors running. Use `cancel` for
a job stop, or `pause` to stop active work and hold the queue. Restart the
dispatcher with the same queue; do not delete its database or lock files.

## Pictures and the gallery

set `pictures: N` to the expected final PNG count for Barkeep,
excluding `icon-*.png`. without that count Barkeep omits the render.
the folder page counts the PNGs it finds independently.

[`azazel/pages`](../azazel/pages/README.md) keeps a job's folder page, its
film so far, `latest.html` and the two indexes. The main worker
starts one publisher automatically when `--renders ROOT` or `SLAYER_RENDERS`
names the render root. It reads the ledger without changing job records:

```sh
bin/azazel work --serve --queue /ABS/QUEUE --renders /ABS/RENDERS
```

The operator can also run the publisher alone for recovery:

```sh
bin/azazel pages --serve --queue /ABS/QUEUE --renders /ABS/RENDERS
```

A second one for the same queue is refused. Its log is `QUEUE/pages.log`.

partial films use `ffmpeg` for encoding and `ffprobe` for verification.
they require a readable recorded soundtrack and verified audio in the
encoded movie. a digest, when recorded, must match before encoding.
if a check fails, the page explains why the partial stayed unpublished.

Jobs that finish between ten-second polls are published too. Failed final
publication retries on the next poll without rerunning the render. Clean
shutdown makes a final ledger refresh with a five-second deadline. After
a restart, automatic catch-up includes jobs finished in the last ten minutes;
older missed outputs require the operator's one-folder `publish` command.

For a render begun before, the operator can publish one folder once:

```sh
bin/azazel publish --renders /ABS/RENDERS --folder animations/X --pictures N
```

Use new output paths. The scheduler does not yet stop a tool from replacing
an existing output or verify a complete film before marking process success.
The job ledger does not guarantee either.

See [the design](../azazel/DESIGN.md) for the component boundaries.
`azazel render` remains the older direct session adapter and bypasses the
queue. `azazel composite` still refuses direct admission. Submit the actual
supported Slayer command through `submit` for queue evaluation.

## Frame diagnostics

Set `SLAYER_OBSERVE_FRAMES: "on"` in a queue request's `environment` to opt into frame observations
from a compatible Slayer executable.

The supervisor passes a private local
datagram descriptor and an attempt nonce; requests without the field keep
their ordinary command, arguments, environment and descriptors.

A binary
without support still runs normally but cannot produce frame observations. An
older queue worker preserves this environment setting but cannot record it.
Only the request's exact value `on` enables diagnostics; inherited settings,
`off` and other values do not enable it.

The optional receipt is `QUEUE/SEQUENCE.ATTEMPT.frames.jsonl` with the same
zero-padded sequence and attempt used by queue logs. Each versioned packet
records its attempt, frame index, scene time when known, kept/drawn state and
available stage timings. A terminal packet records attempted and dropped
events.

The final receipt line describes transport completeness; a missing
terminal, a drop, invalid packet, storage failure or the 64 MiB limit leaves
the receipt incomplete. The 64 MiB limit includes reserved trailer space.
Receipt completeness does not verify images, source identity, performance
profiles or job success. These observations do not control admission or retry.

Receipt setup and writes are best effort: a stalled consumer drops packets
rather than slowing a render, and setup failure can leave no receipt. Observing
frames creates no extra render, modifies no frame output, and creates no film.
The normal publisher continues to require the verified soundtrack for partials.

Receipt writes stop if queue-filesystem free space is unknown or less than the
queue's 30 GiB floor plus 64 MiB. A missing trailer means incomplete. Each
write rechecks free space; independent writers can still race those observations,
because this check does not reserve disk space for the queue.

## Worker metrics and profiling

Metrics are optional and do not control admission or rendering. Set
`SLAYER_METRICS_ADDR=127.0.0.1:9468` on `azazel work --serve` to expose
Prometheus text at `http://127.0.0.1:9468/metrics`. Choose a distinct port
for each worker. `SLAYER_PPROF_ADDR=127.0.0.1:6060` independently enables the
Go profiling endpoints under `/debug/pprof/` on loopback. Leave these unset
to disable them.

If a listener cannot start, the worker logs the error and
continues its queue; a scrape alone does not prove that a renderer is healthy.

The Go runtime metrics and profiles describe the Azazel worker process.

Renderer processes do not inherit these listener addresses: a job may request
its own `SLAYER_METRICS_ADDR` and `SLAYER_PPROF_ADDR` in its environment, at
different addresses. Do not profile active renders without coordination.

The collector reads the queue and native host counters without changing the
queue.

Concurrent scrapes share a one-second cached observation to bound
repeated ledger and process-group reads; the earliest active lease expiry
limits that cache.

`azazel_collection_age_seconds` reports age since the
collection began; `azazel_collection_duration_seconds` gives its duration.
`azazel_jobs_*`, oldest wait, paused state, active and limit gauges
describe its current snapshot.

`azazel_queue_snapshot_available` marks failed
reads. The direct `azazel_gpu_device_utilization_percent` sample is for the
whole Apple GPU, not an individual render, and has an availability gauge.

The `azazel_scheduler_*` GPU value is the worker's retained admission sample;
use its availability, known and age gauges before treating it as current.
Raw `azazel_host_swap_*` values are bytes, pressure level is the native macOS
value (1 normal, 2 warning, 4 critical), and queue-filesystem free disk is in
bytes. Each source has its own availability gauge, and missing values are
omitted instead of being reported as zero.

The current render group RSS is a
sum of each process's resident bytes, not unique physical memory; shared pages
can be counted more than once. `azazel_render_group_resident_known` is zero
if any running group could not be measured. The retained peak sum is the sum
of per-job peaks, which may have occurred at different times.

An unexpired lease is required before sampling a group, but a lease alone
cannot prove that an operating system has not reused a process group ID after
a crash; the live RSS reading is therefore an observation, not ownership proof.
`azazel_render_group_footprint_bytes` instead sums current physical
footprints across groups only when every running leader's recorded boot and
birth identity matches a stable member sample and its lease is unexpired.

Its known flag is zero for older attempts, failures, identity uncertainty or
overflow. This is an observed process-footprint sum, not unique system memory;
samples are not atomic and short-lived members can be missed.
`azazel_host_cpu_busy_percent` measures all processors from consecutive macOS
host tick samples; its known flag is zero until a comparable pair is available.

The sample window says how far apart those observations were. Windows shorter
than one second or with no tick change are reported as unknown while the
baseline continues to accumulate.
`azazel_host_physical_memory_total_*` and `azazel_host_vm_*` expose independent
availability and macOS physical-memory and VM page-state readings.

Speculative
pages are included in free pages. Compressor pages are resident storage for
compressed data, not uncompressed memory or swap. Free pages are not a measure
of available memory.

The `azazel_retained_jobs_*_seconds_sum` and matching count gauges summarize
queue waiting and execution time over the retained history; divide sum by count
for an average. They are ledger-derived gauges, not permanent counters, and
are not a distribution or a count of discarded attempts.
