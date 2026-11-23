package searching

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/janearc/slayer/job"
	"os"
	"slices"
	"testing"
	"time"
)

func TestResourceProbesUseActualCutAndJobTimes(t *testing.T) {
	edit := &Edit{Plan: Plan{Shots: []ShotPlan{{Scene: "heard", Stage: &stagePlan{Kind: "kecks"}}, {Scene: "orbital"}, {Scene: "heard", Stage: &stagePlan{Kind: "multitude"}}}}, ShotSpans: [][2]float64{{0, 2}, {2, 4}, {4, 5}}}
	times := []time.Duration{1900 * time.Millisecond, 2 * time.Second, 2250 * time.Millisecond, 2500 * time.Millisecond, 3 * time.Second, 3999 * time.Millisecond, 4 * time.Second}
	probes, err := edit.ResourceProbePlan(times)
	if err != nil {
		t.Fatal(err)
	}
	if len(probes) != 3 || probes[0].Kind != "kecks" || len(probes[0].Frames()) != 1 || probes[0].Advancing {
		t.Fatal("short clipped shot changed", probes)
	}
	if probes[1].Entry.Index != 1 || probes[1].Adjacent[0].Index != 3 || probes[1].Adjacent[1].Index != 4 || probes[1].Late.Index != 5 || !probes[1].Advancing {
		t.Fatal("lost early motion or cut", probes[1])
	}
	for _, probe := range probes {
		for _, sample := range probe.Frames() {
			if sample.At != times[sample.Index] || edit.shot(sample.At.Seconds()) != probe.Shot {
				t.Fatal("invented frame clock or crossed cut", sample)
			}
		}
	}
}

// The opt-in diagnostic writes the actual production sampling plan without
// opening a world or a device. Normal unit tests do not depend on local jobs.
func TestResourceProbePlanForJob(t *testing.T) {
	path := os.Getenv("HEARD_PROBE_JOB")
	if path == "" {
		t.Skip("HEARD_PROBE_JOB not set")
	}
	wanted, err := job.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	plan, ok := wanted.Scene().Parameters()["plan"].(string)
	if !ok {
		t.Fatal("job does not name a heard screenplay")
	}
	edit, err := Read(plan)
	if err != nil {
		t.Fatal(err)
	}
	probes, err := edit.ResourceProbePlan(wanted.Times())
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(plan)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(raw)
	report := struct {
		JobKey, ScreenplaySHA256 string
		RequestedFrames          int
		Probes                   []SceneProbe
	}{wanted.PictureKey(), hex.EncodeToString(digest[:]), len(wanted.Times()), probes}
	raw, err = json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(os.Getenv("HEARD_PROBE_OUT"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
	t.Logf("%d scenes represented", len(probes))
}

func TestResourceProbesKeepFrozenClocksAndRefuseUncoveredFrames(t *testing.T) {
	edit := &Edit{Plan: Plan{Shots: []ShotPlan{{Scene: "heard"}}}, ShotSpans: [][2]float64{{0, 4}}}
	probes, err := edit.ResourceProbePlan([]time.Duration{time.Second, time.Second})
	if err != nil || probes[0].Advancing {
		t.Fatal("a hold was called motion", probes, err)
	}
	for _, times := range [][]time.Duration{nil, {4 * time.Second}, {2 * time.Second, time.Second}} {
		if _, err := edit.ResourceProbePlan(times); err == nil {
			t.Fatal("invalid requested clock accepted", times)
		}
	}
}

func TestResourceProbesWorkloadAdapterRetainsAllRoles(t *testing.T) {
	edit := &Edit{Plan: Plan{Shots: []ShotPlan{{Scene: "heard", Stage: &stagePlan{Kind: "kecks"}}, {Scene: "orbital"}}}, ShotSpans: [][2]float64{{0, 2}, {2, 4}}}
	times := []time.Duration{1900000001, 2000000003, 2250000007, 2500000011, 3000000013, 3999000017}
	probes, err := edit.WorkloadProbePlan(times)
	if err != nil || len(probes) != 2 {
		t.Fatalf("adapter failed: %+v %v", probes, err)
	}
	if len(probes[0].Frames) != 1 || !slices.Equal(probes[0].Frames[0].Roles, []string{"entry", "adjacent-0", "late"}) {
		t.Fatalf("lost coincident roles: %+v", probes[0])
	}
	for _, probe := range probes {
		for _, frame := range probe.Frames {
			if frame.At != times[frame.Index] {
				t.Fatal("rounded requested clock", frame)
			}
		}
	}
	if _, err := edit.WorkloadProbePlan([]time.Duration{4 * time.Second}); err == nil {
		t.Fatal("adapter accepted uncovered clock")
	}
}
