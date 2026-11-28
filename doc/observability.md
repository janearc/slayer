# Observability and profiling

Slayer, Azazel, Grissom and Ferruccio expose optional metrics and Go
profiles. set a different loopback port for each running process.
for an Azazel worker, set the metrics address when starting it:

```sh
SLAYER_METRICS_ADDR=127.0.0.1:9464 azazel work --queue /absolute/queue --serve
```

from another terminal, read the metrics with:

```sh
curl -fsS http://127.0.0.1:9464/metrics
```

set a renderer’s address separately in the job request’s
`environment`. Azazel keeps its own listener settings out of child
processes. when a listener fails to start, the application reports the
error and continues.

```yaml
scrape_configs:
  - job_name: azazel
    static_configs:
      - targets: ["127.0.0.1:9464"]
```

## Read the measurements

- counters reset when the process restarts. queue gauges report current
  state. an unavailable host or GPU observation omits the measurement
  and sets its availability gauge to zero.
- device utilization covers the whole GPU. the scheduler's retained
  utilization can be older than the current sample. check
  `azazel_scheduler_sample_age_seconds`.
- Go heap metrics cover the process hosting the endpoint. RSS, process
  footprint, group RSS and device allocations measure different quantities.
  shared pages can appear in more than one process’s RSS. the Metal gauge
  is sampled after each glimpse and can miss transient peaks.
- GPU timings sum command-buffer durations for each glimpse. buffers
  can overlap. compare these separately from wall times.
- the frame stages distinguish tracing, development, titles, kept-frame
  reads, PNG writing and movie encoding. titles include the HUD. PNG
  writing includes hashing and recording. the per-frame encoding time
  excludes the final movie close and flush.

## Dashboard queries

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

## Go profiles

set `SLAYER_PPROF_ADDR=127.0.0.1:6060` on the process to profile.
for a queued render, put it in the request’s `environment`.
while that process runs, collect a profile from another terminal:

```sh
go tool pprof http://127.0.0.1:6060/debug/pprof/heap
go tool pprof 'http://127.0.0.1:6060/debug/pprof/profile?seconds=30'
curl -fsS 'http://127.0.0.1:6060/debug/pprof/trace?seconds=5' -o /tmp/slayer.trace
go tool trace /tmp/slayer.trace
```

profiling changes timings. on macOS, `sample`, `vmmap` and
Instruments can supplement Go profiles when investigating native memory
or GPU work. the profiling listener accepts only loopback addresses.
