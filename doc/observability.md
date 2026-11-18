# Observability and profiling

Slayer, Azazel's worker, Grissom and the native Ferruccio application use
the same in-process metrics and profiling support. Neither listener is
started by default. Set a different address for each process that needs to
be scraped:

```sh
SLAYER_METRICS_ADDR=127.0.0.1:9464 azazel work --queue /absolute/queue
curl -fsS http://127.0.0.1:9464/metrics
```

Use the normal arguments for the worker on your machine. A Prometheus scrape
job can target that address at a 15 second interval. A separate Slayer or
viewer process needs a separate port. Azazel does not automatically pass its
listener addresses to its render children: the worker's Go heap is not the
renderer heap. Explicit per-job environment settings can opt a renderer in.

Choose unique ports for concurrent processes.
Each process hosts at most one set of listeners. If a configured listener
cannot start, the application reports the error and continues its work.
The metrics endpoint has no authentication or TLS; bind it to loopback or
put it behind a trusted, access-controlled proxy.

```yaml
scrape_configs:
  - job_name: azazel
    static_configs:
      - targets: ["127.0.0.1:9464"]
```

Metrics use Prometheus text format, with histogram buckets for elapsed
seconds. Labels do not include job names, paths, commands or agent identity.

Counters reset when their process restarts; queue counts are current gauges,
not a durable event history. A sample that could not be measured is omitted;
its corresponding availability gauge reports zero. GPU utilization is the
device-wide sample and includes work from other processes.

A current process group RSS sum and summed per-job peak RSS describe different
quantities; neither includes every Metal allocation. Go heap statistics
describe only the
process hosting the endpoint and exclude native and GPU allocations.

`go_gomaxprocs` reports the effective Go worker limit in that process, which
may differ from a requested environment value.

On supported macOS builds, the same endpoint also reports current process RSS
and cumulative process CPU time with availability flags. Those readings do
not include child processes, and RSS does not account for every GPU allocation.

Current and lifetime-peak physical footprint are also reported where the
kernel provides them. Apple's footprint accounting can include compressed,
swapped and accessed Metal resources; it is distinct from resident bytes and
must not be added to RSS or treated as a separate GPU total.

Ferruccio's no-cgo sampler reads RSS and footprint separately, so they are
nearby observations rather than one atomic snapshot.

Slayer's GPU timing histograms record each shutter glimpse. Device timings
sum command-buffer execution durations; buffers can overlap, so those sums
can exceed the elapsed wall stage. They are not device utilization and must
not be added to wall times. Logical input bytes, retained-input traffic and
the last glimpse's Go sample buffer are not GPU resident memory.

Frame drawing, development, titles, kept-frame reads, PNG writing and movie
encoding are separate stages.

The titles stage includes the HUD; PNG writing includes hashing and ledger
recording. Movie encoding measures handing each frame to the writer, including
its initial open, but excludes the final close and flush. Grissom offscreen
timing includes submission, waiting and readback; Ferruccio preview
round-trip time includes work done by its separate render process.

The optional Metal device allocation gauge is sampled after each GPU glimpse;
check its availability gauge. It is not a transient peak, may miss resources
already released, and is not a verified per-process allocation measurement.

Metal's recommended working set is a performance recommendation, not a hard
memory limit. Apple documents
[device allocation](https://developer.apple.com/documentation/metal/mtldevice/currentallocatedsize)
and the [recommended working set](https://developer.apple.com/documentation/metal/mtldevice/recommendedmaxworkingsetsize).

Useful PromQL panels include:

```promql
azazel_gpu_device_utilization_percent
azazel_host_swap_used_bytes
azazel_host_memory_pressure_level
azazel_render_group_resident_bytes
azazel_jobs_queued
rate(slayer_frames_drawn_total[5m])
rate(slayer_gpu_acceleration_wall_seconds_sum[5m]) / rate(slayer_gpu_acceleration_wall_seconds_count[5m])
rate(slayer_gpu_trace_device_seconds_sum[5m]) / rate(slayer_gpu_trace_device_seconds_count[5m])
go_memstats_heap_alloc_bytes
```

Import [the example Grafana dashboard](observability-dashboard.json), select
your Prometheus data source, and configure each process as a scrape target.

Gate device and host graphs on their corresponding `*_available` gauges.
`azazel_scheduler_sample_age_seconds` reveals a stale scheduler sample. A
missing graph is different from a measured zero. The scheduler sample and
the independently sampled device gauge can be taken at different times.

For Go profiles, opt in separately on loopback only:

```sh
SLAYER_PPROF_ADDR=127.0.0.1:6060 slayer draw /absolute/job.json
go tool pprof http://127.0.0.1:6060/debug/pprof/heap
go tool pprof 'http://127.0.0.1:6060/debug/pprof/profile?seconds=30'
curl -fsS 'http://127.0.0.1:6060/debug/pprof/trace?seconds=5' -o /tmp/slayer.trace
go tool trace /tmp/slayer.trace
```

Profiling perturbs performance. Compare like-for-like output and frame
clocks before drawing conclusions.

On macOS, `sample`, `vmmap`, Instruments (`xctrace`) and Metal profiling can
supplement Go CPU, heap and execution traces; native memory and GPU work are
not visible in a Go heap profile. Do not attach invasive tools to another
person's active render. Profiling endpoints expose process details and accept
CPU/trace requests, so keep them on loopback and close them after use.
