package telemetry

import (
	"context"
	"errors"
	"math"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/janearc/slayer/azazel/internal/footprint"
	"github.com/janearc/slayer/azazel/internal/queue"
	"github.com/janearc/slayer/internal/observability"
)

// bootTail is the part of a boot identifier after its first eight digits.
const bootTail = "-89ab-cdef-0123-456789abcdef"

func values(samples []observability.Sample) map[string]float64 {
	out := make(map[string]float64, len(samples))
	for _, sample := range samples {
		out[sample.Name] = sample.Value
	}
	return out
}

func TestQueueSamplesShowStateAndSchedulingAge(t *testing.T) {
	now := time.Unix(1000, 0)
	snapshot := queue.Snapshot{Paused: "paused", Jobs: []queue.Job{
		{State: queue.Queued, Submitted: now.Add(-20 * time.Second)},
		{State: queue.Running, Executing: true, ProcessGroup: 7, LeaseUntil: now.Add(30 * time.Second), PeakResidentBytes: 200},
		{State: queue.Failed, Submitted: now.Add(-50 * time.Second), Started: now.Add(-40 * time.Second), Finished: now.Add(-10 * time.Second)},
	}, Scheduling: &queue.Scheduling{Observed: now.Add(-8 * time.Second), GPUKnown: true, GPUPercent: 53, GPUActive: 1, GPULimit: 4, Held: "wait"}}
	got := values(queueSamples(snapshot, now, func(group int) (uint64, error) {
		if group != 7 {
			t.Fatal(group)
		}
		return 100, nil
	}))
	for name, want := range map[string]float64{
		"azazel_jobs_queued": 1, "azazel_jobs_running": 1, "azazel_jobs_failed": 1,
		"azazel_queue_paused": 1, "azazel_queue_oldest_wait_seconds": 20,
		"azazel_retained_jobs_queue_wait_seconds_sum": 10, "azazel_retained_jobs_queue_wait_count": 1,
		"azazel_retained_jobs_run_seconds_sum": 30, "azazel_retained_jobs_run_count": 1,
		"azazel_scheduler_gpu_device_utilization_percent": 53,
		"azazel_scheduler_sample_age_seconds":             8, "azazel_scheduler_gpu_sample_known": 1,
		"azazel_scheduler_held": 1, "azazel_render_group_resident_bytes": 100,
		"azazel_render_group_resident_known":     1,
		"azazel_running_peak_resident_bytes_sum": 200,
	} {
		if got[name] != want {
			t.Errorf("%s = %v, want %v", name, got[name], want)
		}
	}
}

func TestQueueSamplesDoNotInventMissingMeasurements(t *testing.T) {
	now := time.Unix(1000, 0)
	snapshot := queue.Snapshot{Jobs: []queue.Job{{State: queue.Running, Executing: true, ProcessGroup: 7}}}
	got := values(queueSamples(snapshot, now, func(int) (uint64, error) { return 0, errors.New("unavailable") }))
	if got["azazel_scheduler_gpu_sample_known"] != 0 || got["azazel_render_group_resident_known"] != 0 {
		t.Fatal(got)
	}
	if _, ok := got["azazel_scheduler_gpu_device_utilization_percent"]; ok {
		t.Fatal("invented GPU", got)
	}
	if _, ok := got["azazel_render_group_resident_bytes"]; ok {
		t.Fatal("invented residency", got)
	}
}

func TestInvalidLeaseDoesNotAttributeAReusedProcessGroup(t *testing.T) {
	now := time.Unix(1000, 0)
	for _, lease := range []time.Time{{}, now.Add(-time.Second)} {
		snapshot := queue.Snapshot{Jobs: []queue.Job{{State: queue.Running, Executing: true, ProcessGroup: 7, LeaseUntil: lease}}}
		got := values(queueSamples(snapshot, now, func(int) (uint64, error) { t.Fatal("sampled an invalid owner"); return 0, nil }))
		if got["azazel_render_group_resident_known"] != 0 {
			t.Fatal(got)
		}
	}
}

type ledgerFunc func(context.Context) (queue.Snapshot, error)

func (f ledgerFunc) Status(ctx context.Context) (queue.Snapshot, error) { return f(ctx) }

func TestCollectorReportsFailedLedger(t *testing.T) {
	collector := Collector{Ledger: ledgerFunc(func(context.Context) (queue.Snapshot, error) { return queue.Snapshot{}, errors.New("cannot read") })}
	got := values(collector.Collect())
	if got["azazel_queue_snapshot_available"] != 0 {
		t.Fatal(got)
	}
	if _, ok := got["azazel_jobs_queued"]; ok {
		t.Fatal("invented queue", got)
	}
}

func TestConcurrentScrapesShareOneHostAndLedgerRead(t *testing.T) {
	var calls atomic.Int64
	collector := NewCollector(ledgerFunc(func(context.Context) (queue.Snapshot, error) {
		calls.Add(1)
		return queue.Snapshot{}, nil
	}), t.TempDir())
	var done sync.WaitGroup
	for range 8 {
		done.Add(1)
		go func() { defer done.Done(); collector.Collect() }()
	}
	done.Wait()
	if got := calls.Load(); got != 1 {
		t.Fatal("scrapes read ledger", got, "times")
	}
}

func TestScrapeDoesNotWaitForStuckCollection(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	collector := NewCollector(ledgerFunc(func(context.Context) (queue.Snapshot, error) { close(entered); <-release; return queue.Snapshot{}, nil }), t.TempDir())
	done := make(chan struct{})
	go func() { collector.Collect(); close(done) }()
	<-entered
	started := time.Now()
	got := values(collector.Collect())
	if time.Since(started) > time.Second || got["azazel_collection_available"] != 0 || got["azazel_queue_snapshot_available"] != 0 {
		t.Fatal(got)
	}
	if _, ok := got["azazel_gpu_device_utilization_percent"]; ok {
		t.Fatal("invented stale metrics", got)
	}
	close(release)
	<-done
}

func TestCacheExpiresAtFirstLeaseAndReportsAge(t *testing.T) {
	now := time.Unix(100, 0)
	cache := collectorCache{taken: now, finished: now.Add(100 * time.Millisecond), until: now.Add(500 * time.Millisecond), leaseUntil: now.Add(500 * time.Millisecond), samples: []observability.Sample{{Name: "azazel_render_group_footprint_known", Value: 1}, {Name: "azazel_render_group_footprint_bytes", Value: 10}, {Name: "azazel_render_group_resident_known", Value: 1}, {Name: "azazel_render_group_resident_bytes", Value: 8}}}
	got, ok := cache.read(now.Add(400 * time.Millisecond))
	if !ok || values(got)["azazel_collection_age_seconds"] != 0.4 {
		t.Fatal(got, ok)
	}
	if _, ok = cache.read(now.Add(500 * time.Millisecond)); ok {
		t.Fatal("cached beyond lease")
	}
	lateCache := cache.withAge(now.Add(600 * time.Millisecond))
	late := values(lateCache)
	if late["azazel_render_group_footprint_known"] != 0 {
		t.Fatal(late)
	}
	if _, ok := late["azazel_render_group_footprint_bytes"]; ok {
		t.Fatal(late)
	}
	if late["azazel_render_group_resident_known"] != 0 {
		t.Fatal(late)
	}
	if _, ok := late["azazel_render_group_resident_bytes"]; ok {
		t.Fatal(late)
	}
}

func TestCollectionAgeIncludesSlowObservation(t *testing.T) {
	start := time.Unix(100, 0)
	cache := collectorCache{taken: start, finished: start.Add(2 * time.Second), until: start.Add(time.Second), leaseUntil: start.Add(1500 * time.Millisecond), samples: []observability.Sample{{Name: "azazel_render_group_resident_known", Value: 1}, {Name: "azazel_render_group_resident_bytes", Value: 8}}}
	aged := cache.withAge(start.Add(2 * time.Second))
	got := values(aged)
	if got["azazel_collection_age_seconds"] != 2 || got["azazel_collection_duration_seconds"] != 2 || got["azazel_render_group_resident_known"] != 0 {
		t.Fatal(got)
	}
	if _, ok := got["azazel_render_group_resident_bytes"]; ok {
		t.Fatal(got)
	}
	if _, ok := cache.read(start.Add(2 * time.Second)); ok {
		t.Fatal("reused stale collection")
	}
}

func TestLeaseExpiresDuringResidentReadIsUnknown(t *testing.T) {
	now := time.Unix(1000, 0)
	job := queue.Job{State: queue.Running, Executing: true, ProcessGroup: 7, LeaseUntil: now.Add(time.Second)}
	got := values(queueSamplesWithFootprintClock(queue.Snapshot{Jobs: []queue.Job{job}}, now, func(int) (uint64, error) { return 10, nil }, nil, nil, func() time.Time { return now.Add(2 * time.Second) }))
	if got["azazel_render_group_resident_known"] != 0 {
		t.Fatal(got)
	}
	if _, ok := got["azazel_render_group_resident_bytes"]; ok {
		t.Fatal(got)
	}
}

type footprintFixture struct {
	values  map[int]uint64
	missing int
}

func (f footprintFixture) Group(group int) ([]int, error) {
	if f.missing == group {
		return nil, errors.New("gone")
	}
	return []int{group}, nil
}
func (f footprintFixture) Identity(pid int) (footprint.Identity, error) {
	return footprint.Identity{PID: pid, Group: pid, StartSeconds: uint64(pid * 10)}, nil
}
func (f footprintFixture) Footprint(pid int) (footprint.Reading, error) {
	return footprint.Reading{Current: f.values[pid]}, nil
}

func TestGroupFootprintRequiresEveryLiveIdentityAndMatchingBoot(t *testing.T) {
	const boot = "01234567" + bootTail
	now := time.Now()
	job := func(pid int) queue.Job {
		return queue.Job{State: queue.Running, Executing: true, ProcessGroup: pid, ProcessStartSeconds: uint64(pid * 10), ProcessBootID: boot, LeaseUntil: now.Add(time.Minute)}
	}
	snapshot := queue.Snapshot{Jobs: []queue.Job{job(7), job(9)}}
	for _, row := range []struct {
		name   string
		source footprintFixture
		boot   string
		mutate func(*queue.Snapshot)
		known  bool
		want   float64
	}{
		{"valid", footprintFixture{values: map[int]uint64{7: 10, 9: 20}}, boot, nil, true, 30},
		{"reboot", footprintFixture{values: map[int]uint64{7: 10, 9: 20}}, "11234567" + bootTail, nil, false, 0},
		{"missing group", footprintFixture{values: map[int]uint64{7: 10, 9: 20}, missing: 9}, boot, nil, false, 0},
		{"overflow", footprintFixture{values: map[int]uint64{7: math.MaxUint64, 9: 1}}, boot, nil, false, 0},
		{"reused leader", footprintFixture{values: map[int]uint64{7: 10, 9: 20}}, boot, func(s *queue.Snapshot) { s.Jobs[0].ProcessStartSeconds++ }, false, 0},
		{"old metadata", footprintFixture{values: map[int]uint64{7: 10, 9: 20}}, boot, func(s *queue.Snapshot) { s.Jobs[0].ProcessBootID = "" }, false, 0},
		{"expired lease", footprintFixture{values: map[int]uint64{7: 10, 9: 20}}, boot, func(s *queue.Snapshot) { s.Jobs[0].LeaseUntil = now.Add(-time.Second) }, false, 0},
	} {
		t.Run(row.name, func(t *testing.T) {
			s := snapshot
			s.Jobs = append([]queue.Job(nil), snapshot.Jobs...)
			if row.mutate != nil {
				row.mutate(&s)
			}
			got := values(queueSamplesWithFootprint(s, now, func(int) (uint64, error) { return 1, nil }, func() (string, error) { return row.boot, nil }, func(group int) footprint.Observation { return footprint.Sample(group, row.source) }))
			if got["azazel_render_group_footprint_known"] != flag(row.known) {
				t.Fatal(got)
			}
			value, exists := got["azazel_render_group_footprint_bytes"]
			if exists != row.known || exists && value != row.want {
				t.Fatal(got)
			}
		})
	}
}

func TestDuplicateGroupIsUnknownForRSSAndFootprint(t *testing.T) {
	const boot = "01234567" + bootTail
	now := time.Now()
	job := queue.Job{State: queue.Running, Executing: true, ProcessGroup: 7, ProcessStartSeconds: 70, ProcessBootID: boot, LeaseUntil: now.Add(time.Minute)}
	snapshot := queue.Snapshot{Jobs: []queue.Job{job, job}}
	source := footprintFixture{values: map[int]uint64{7: 10}}
	got := values(queueSamplesWithFootprint(snapshot, now, func(int) (uint64, error) { return 10, nil }, func() (string, error) { return boot, nil }, func(group int) footprint.Observation { return footprint.Sample(group, source) }))
	if got["azazel_render_group_resident_known"] != 0 || got["azazel_render_group_footprint_known"] != 0 {
		t.Fatal(got)
	}
	if _, ok := got["azazel_render_group_resident_bytes"]; ok {
		t.Fatal(got)
	}
	if _, ok := got["azazel_render_group_footprint_bytes"]; ok {
		t.Fatal(got)
	}
}

func TestBootChangeDuringFootprintReadIsUnknown(t *testing.T) {
	const boot = "01234567" + bootTail
	now := time.Now()
	job := queue.Job{State: queue.Running, Executing: true, ProcessGroup: 7, ProcessStartSeconds: 70, ProcessBootID: boot, LeaseUntil: now.Add(time.Minute)}
	reads := 0
	bootRead := func() (string, error) {
		reads++
		if reads == 1 {
			return boot, nil
		}
		return "11234567" + bootTail, nil
	}
	source := footprintFixture{values: map[int]uint64{7: 10}}
	got := values(queueSamplesWithFootprint(queue.Snapshot{Jobs: []queue.Job{job}}, now, func(int) (uint64, error) { return 10, nil }, bootRead, func(group int) footprint.Observation { return footprint.Sample(group, source) }))
	if got["azazel_render_group_footprint_known"] != 0 {
		t.Fatal(got)
	}
	if _, ok := got["azazel_render_group_footprint_bytes"]; ok {
		t.Fatal(got)
	}
}

func TestLeaseExpiresDuringFootprintReadIsUnknown(t *testing.T) {
	const boot = "01234567" + bootTail
	now := time.Now()
	job := queue.Job{State: queue.Running, Executing: true, ProcessGroup: 7, ProcessStartSeconds: 70, ProcessBootID: boot, LeaseUntil: now.Add(time.Second)}
	source := footprintFixture{values: map[int]uint64{7: 10}}
	got := values(queueSamplesWithFootprint(queue.Snapshot{Jobs: []queue.Job{job}}, now, func(int) (uint64, error) { return 10, nil }, func() (string, error) { return boot, nil }, func(group int) footprint.Observation {
		observed := footprint.Sample(group, source)
		observed.Finished = now.Add(2 * time.Second)
		return observed
	}))
	if got["azazel_render_group_footprint_known"] != 0 {
		t.Fatal(got)
	}
	if _, ok := got["azazel_render_group_footprint_bytes"]; ok {
		t.Fatal(got)
	}
}
