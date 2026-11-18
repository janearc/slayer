// Package telemetry collects read-only observations of an Azazel queue.
package telemetry

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/janearc/slayer/azazel/internal/footprint"
	"github.com/janearc/slayer/azazel/internal/queue"
	"github.com/janearc/slayer/internal/observability"
)

// Ledger supplies the same checked snapshot used by azazel status.
type Ledger interface {
	Status(context.Context) (queue.Snapshot, error)
}

// Collector takes an independent snapshot on each scrape. It never changes
// queue state or controls admission.
type Collector struct {
	Ledger    Ledger
	Directory string
	cpu       *cpuMeter
	cache     *collectorCache
}

type collectorCache struct {
	sync.Mutex
	taken      time.Time
	finished   time.Time
	until      time.Time
	leaseUntil time.Time
	samples    []observability.Sample
}

// read returns the cached samples while the cache window is open. The bool is
// false once the window has closed.
func (cache *collectorCache) read(now time.Time) ([]observability.Sample, bool) {
	if cache.taken.IsZero() || !now.Before(cache.until) {
		return nil, false
	}
	return cache.withAge(now), true
}

// withAge returns the cached samples with their ages added. Render group
// readings are dropped, and their known flags set to zero, once the lease has
// ended.
func (cache *collectorCache) withAge(now time.Time) []observability.Sample {
	out := make([]observability.Sample, 0, len(cache.samples)+1)
	for _, sample := range cache.samples {
		if !cache.leaseUntil.IsZero() && !now.Before(cache.leaseUntil) {
			if sample.Name == "azazel_render_group_footprint_bytes" || sample.Name == "azazel_render_group_resident_bytes" {
				continue
			}
			if sample.Name == "azazel_render_group_footprint_known" || sample.Name == "azazel_render_group_resident_known" {
				sample.Value = 0
			}
		}
		out = append(out, sample)
	}
	return append(out, observability.Sample{Name: "azazel_collection_age_seconds", Value: max(0, now.Sub(cache.taken).Seconds())}, observability.Sample{Name: "azazel_collection_duration_seconds", Value: max(0, cache.finished.Sub(cache.taken).Seconds())})
}

// NewCollector makes one independent CPU sampling history for this worker.
func NewCollector(ledger Ledger, directory string) Collector {
	return Collector{Ledger: ledger, Directory: directory, cpu: &cpuMeter{}, cache: &collectorCache{}}
}

// Collect returns only readings that succeeded, alongside availability flags.
func (c Collector) Collect() []observability.Sample {
	if c.cache == nil {
		samples, _ := c.collect()
		return samples
	}
	if !c.cache.TryLock() {
		return []observability.Sample{{Name: "azazel_collection_available", Value: 0}, {Name: "azazel_queue_snapshot_available", Value: 0}}
	}
	defer c.cache.Unlock()
	if cached, ok := c.cache.read(time.Now()); ok {
		return cached
	}
	started := time.Now()
	samples, lease := c.collect()
	finished := time.Now()
	c.cache.taken, c.cache.finished, c.cache.until, c.cache.leaseUntil, c.cache.samples = started, finished, started.Add(time.Second), lease, samples
	if !lease.IsZero() && lease.Before(c.cache.until) {
		c.cache.until = lease
	}
	return c.cache.withAge(time.Now())
}

// collect takes one set of readings from the host, the CPU meter, the GPU
// counter and the queue. It also returns when the earliest lease ends.
func (c Collector) collect() ([]observability.Sample, time.Time) {
	out := hostSamples(nativeHost{}, c.Directory)
	out = append(out, observability.Sample{Name: "azazel_collection_available", Value: 1})
	if c.cpu != nil {
		out = append(out, c.cpu.sample(time.Now(), nativeCPUTicks)...)
	}
	gpu, gpuKnown := queue.MetricsGPUDeviceUtilization()
	out = append(out, observability.Sample{Name: "azazel_gpu_device_utilization_available", Value: flag(gpuKnown)})
	if gpuKnown {
		out = append(out, observability.Sample{Name: "azazel_gpu_device_utilization_percent", Value: float64(gpu)})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if c.Ledger == nil {
		return append(out, observability.Sample{Name: "azazel_queue_snapshot_available", Value: 0}), time.Time{}
	}
	snapshot, err := c.Ledger.Status(ctx)
	if err != nil {
		return append(out, observability.Sample{Name: "azazel_queue_snapshot_available", Value: 0}), time.Time{}
	}
	out = append(out, observability.Sample{Name: "azazel_queue_snapshot_available", Value: 1})
	sampledAt := time.Now()
	var earliest time.Time
	for _, job := range snapshot.Jobs {
		if job.State == queue.Running && job.LeaseUntil.After(sampledAt) && (earliest.IsZero() || job.LeaseUntil.Before(earliest)) {
			earliest = job.LeaseUntil
		}
	}
	return append(out, queueSamplesWithFootprintClock(snapshot, sampledAt, queue.MetricsGroupResident, footprint.BootID, func(group int) footprint.Observation { return footprint.Sample(group, footprint.Native()) }, time.Now)...), earliest
}

// queueSamples turns a queue snapshot into samples, without footprint or boot
// identity readings.
func queueSamples(snapshot queue.Snapshot, now time.Time, resident func(int) (uint64, error)) []observability.Sample {
	return queueSamplesWithFootprint(snapshot, now, resident, nil, nil)
}

// queueSamplesWithFootprint is queueSamples with footprint readings from
// sampleFootprint and the boot identity from boot.
func queueSamplesWithFootprint(snapshot queue.Snapshot, now time.Time, resident func(int) (uint64, error), boot func() (string, error), sampleFootprint func(int) footprint.Observation) []observability.Sample {
	return queueSamplesWithFootprintClock(snapshot, now, resident, boot, sampleFootprint, func() time.Time { return now })
}

// queueSamplesWithFootprintClock is queueSamplesWithFootprint with the time
// for lease checks taken from clock.
func queueSamplesWithFootprintClock(snapshot queue.Snapshot, now time.Time, resident func(int) (uint64, error), boot func() (string, error), sampleFootprint func(int) footprint.Observation, clock func() time.Time) []observability.Sample {
	counts := map[queue.State]float64{}
	var oldest, peak, current, waitSum, waitCount, runSum, runCount float64
	known := true
	footKnown := true
	var footBytes uint64
	seenGroups := map[int]bool{}
	for _, job := range snapshot.Jobs {
		counts[job.State]++
		if !job.Submitted.IsZero() && !job.Started.IsZero() && !job.Started.Before(job.Submitted) {
			waitSum += job.Started.Sub(job.Submitted).Seconds()
			waitCount++
		}
		if !job.Started.IsZero() && !job.Finished.IsZero() && !job.Finished.Before(job.Started) {
			runSum += job.Finished.Sub(job.Started).Seconds()
			runCount++
		}
		if job.State == queue.Queued && !job.Submitted.IsZero() {
			oldest = max(oldest, now.Sub(job.Submitted).Seconds())
		}
		if job.State != queue.Running {
			continue
		}
		peak += float64(job.PeakResidentBytes)
		if !job.Executing || job.ProcessGroup <= 0 || job.LeaseUntil.IsZero() || !now.Before(job.LeaseUntil) {
			known = false
			footKnown = false
			continue
		}
		if seenGroups[job.ProcessGroup] {
			known = false
			footKnown = false
			continue
		}
		seenGroups[job.ProcessGroup] = true
		if boot == nil || sampleFootprint == nil || job.ProcessStartSeconds == 0 || !footprint.ValidBootID(job.ProcessBootID) {
			footKnown = false
		} else {
			first, err := boot()
			if err != nil || first != job.ProcessBootID {
				footKnown = false
			} else {
				observed := sampleFootprint(job.ProcessGroup)
				last, err := boot()
				leader := footprint.Leader{BootID: job.ProcessBootID, Identity: footprint.Identity{PID: job.ProcessGroup, Group: job.ProcessGroup, StartSeconds: job.ProcessStartSeconds, StartMicroseconds: job.ProcessStartMicroseconds}}
				if err != nil || last != first || observed.Finished.IsZero() || !observed.Finished.Before(job.LeaseUntil) || !footprint.LeaderMatches(leader, last, observed) || observed.GroupCurrentBytes > math.MaxUint64-footBytes {
					footKnown = false
				} else {
					footBytes += observed.GroupCurrentBytes
				}
			}
		}
		bytes, err := resident(job.ProcessGroup)
		if err != nil || !clock().Before(job.LeaseUntil) {
			known = false
			continue
		}
		current += float64(bytes)
	}
	out := make([]observability.Sample, 0, 20)
	put := func(name string, value float64) { out = append(out, observability.Sample{Name: name, Value: value}) }
	for _, state := range []queue.State{queue.Queued, queue.Running, queue.Succeeded, queue.Failed, queue.Cancelled, queue.Interrupted} {
		put("azazel_jobs_"+string(state), counts[state])
	}
	put("azazel_queue_oldest_wait_seconds", oldest)
	put("azazel_retained_jobs_queue_wait_seconds_sum", waitSum)
	put("azazel_retained_jobs_queue_wait_count", waitCount)
	put("azazel_retained_jobs_run_seconds_sum", runSum)
	put("azazel_retained_jobs_run_count", runCount)
	put("azazel_queue_paused", flag(snapshot.Paused != ""))
	put("azazel_running_peak_resident_bytes_sum", peak)
	put("azazel_render_group_resident_known", flag(known))
	if known {
		put("azazel_render_group_resident_bytes", current)
	}
	put("azazel_render_group_footprint_known", flag(footKnown))
	if footKnown {
		put("azazel_render_group_footprint_bytes", float64(footBytes))
	}
	scheduling := snapshot.Scheduling
	available := scheduling != nil && !scheduling.Observed.IsZero()
	put("azazel_scheduler_sample_available", flag(available))
	put("azazel_scheduler_gpu_sample_known", flag(available && scheduling.GPUKnown))
	if available {
		put("azazel_scheduler_sample_age_seconds", max(0, now.Sub(scheduling.Observed).Seconds()))
		put("azazel_scheduler_gpu_active", float64(scheduling.GPUActive))
		put("azazel_scheduler_gpu_limit", float64(scheduling.GPULimit))
		put("azazel_scheduler_cpu_active", float64(scheduling.CPUActive))
		put("azazel_scheduler_held", flag(scheduling.Held != ""))
		if scheduling.GPUKnown {
			put("azazel_scheduler_gpu_device_utilization_percent", float64(scheduling.GPUPercent))
		}
	}
	return out
}

// flag converts a boolean to a gauge value of 1 or 0.
func flag(value bool) float64 {
	if value {
		return 1
	}
	return 0
}
