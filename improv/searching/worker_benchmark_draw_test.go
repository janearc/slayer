//go:build draw && darwin

package searching

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/gpu"
	"github.com/janearc/slayer/maths"
)

type heardWorkerMeasurement struct {
	Cue                 string    `json:"cue"`
	At                  float64   `json:"at"`
	Recorded            string    `json:"recorded_at"`
	Stage               stagePlan `json:"stage"`
	StartProgress       float64   `json:"start_progress"`
	ProgressStep        float64   `json:"progress_step"`
	Workers             int       `json:"workers"`
	Case                string    `json:"case"`
	Width, Height, Rays int
	Build               time.Duration `json:"build_ns"`
	Cold                gpu.Profile   `json:"cold"`
	Frames              []gpu.Profile `json:"frames"`
	CPU                 time.Duration `json:"cpu_ns"`
	Wall                time.Duration `json:"wall_ns"`
	ProcessPeakRSS      int64         `json:"process_peak_rss_bytes"`
}

// BenchmarkHeardWorkers uses a selected Keck stage from the live screenplay.
// It draws no deliverable film: measured profiles come from Slayer's own
// retained renderer. Concurrent production work is deliberate and recorded.
func BenchmarkHeardWorkers(b *testing.B) {
	path := os.Getenv("HEARD_BENCH_PLAN")
	if path == "" {
		b.Fatal("HEARD_BENCH_PLAN must name the exact live screenplay")
	}
	edit, err := Read(path)
	if err != nil {
		b.Fatal(err)
	}
	cue := os.Getenv("HEARD_BENCH_CUE")
	if cue == "" {
		cue = "18.1"
	}
	var plan *stagePlan
	for _, shot := range edit.Plan.Shots {
		if shot.From == cue && shot.Stage != nil && shot.Stage.Kind == "kecks" {
			copy := *shot.Stage
			plan = &copy
			break
		}
	}
	if plan == nil {
		b.Fatal("selected Keck stage not found")
	}
	workersList := []int{2, 1, 4, 8, 12, 16}
	if text := os.Getenv("HEARD_BENCH_WORKERS"); text != "" {
		workersList = nil
		for _, word := range strings.Split(text, ",") {
			n, err := strconv.Atoi(word)
			if err != nil || n < 1 || n > 16 {
				b.Fatal("invalid worker count", word)
			}
			workersList = append(workersList, n)
		}
	}
	for _, workers := range workersList {
		b.Run(fmt.Sprintf("tracking/workers=%d", workers), func(b *testing.B) { benchmarkHeardStage(b, plan, workers, false) })
	}
	if os.Getenv("HEARD_BENCH_NO_CAMERA") == "1" {
		return
	}
	for _, workers := range []int{2, 8} {
		b.Run(fmt.Sprintf("camera-only/workers=%d", workers), func(b *testing.B) { benchmarkHeardStage(b, plan, workers, true) })
	}
}

func benchmarkHeardStage(b *testing.B, plan *stagePlan, workers int, cameraOnly bool) {
	previous := runtime.GOMAXPROCS(workers)
	defer runtime.GOMAXPROCS(previous)
	// Keep the benchmark inside active motion; updateGround settles at u=.62.
	// Explicit progress is separate from the absolute animation clock.
	atSeconds := 64.0
	if value := os.Getenv("HEARD_BENCH_AT"); value != "" {
		n, err := strconv.ParseFloat(value, 64)
		if err != nil {
			b.Fatal(err)
		}
		atSeconds = n
	}
	started := time.Now()
	stage, err := newHeardStage(plan, "")
	if err != nil {
		b.Fatal(err)
	}
	eye, err := stage.update(plan, State{At: atSeconds, Beat: atSeconds * 66 / 60}, .3)
	if err != nil {
		b.Fatal(err)
	}
	measured := heardWorkerMeasurement{Cue: os.Getenv("HEARD_BENCH_CUE"), At: atSeconds, Recorded: time.Now().UTC().Format(time.RFC3339Nano), Stage: *plan, StartProgress: .3, ProgressStep: .01, Workers: workers, Case: "tracking", Width: 480, Height: 270, Rays: 1, Build: time.Since(started)}
	if cameraOnly {
		measured.Case = "camera-only"
	}
	drawer, err := gpu.NewFrames(gpu.FasterLighting)
	if err != nil {
		b.Fatal(err)
	}
	defer drawer.Close()
	lens := film.Pinhole{Wide: plan.Wide * maths.Degree}
	_, measured.Cold, err = drawer.Draw(stage.world, 1, eye, lens, time.Duration(atSeconds*float64(time.Second)), 480, 270, 1)
	if err != nil {
		b.Fatal(err)
	}
	var cpuProfile *os.File
	if workers == 2 && !cameraOnly && os.Getenv("HEARD_BENCH_CPU_PROFILE") != "" {
		cpuProfile, err = os.Create(os.Getenv("HEARD_BENCH_CPU_PROFILE"))
		if err != nil {
			b.Fatal(err)
		}
		if err = pprof.StartCPUProfile(cpuProfile); err != nil {
			b.Fatal(err)
		}
	}
	before := heardBenchmarkCPU(b)
	started = time.Now()
	b.ResetTimer()
	for index := range b.N {
		at := time.Duration(atSeconds * float64(time.Second))
		generation := uint64(1)
		next := eye
		if cameraOnly {
			next.Origin.X += .01 * float64(index+1)
			next.Orientation = maths.LookAt(next.Origin, vector(plan.Look))
		} else {
			at += time.Duration(index+1) * time.Second / 6
			next, err = stage.update(plan, State{At: at.Seconds(), Beat: at.Seconds() * 66 / 60}, .3+float64(index+1)*.01)
			if err != nil {
				b.Fatal(err)
			}
			generation = uint64(index + 2)
		}
		_, profile, drawError := drawer.Draw(stage.world, generation, next, lens, at, 480, 270, 1)
		if drawError != nil {
			b.Fatal(drawError)
		}
		measured.Frames = append(measured.Frames, profile)
	}
	b.StopTimer()
	measured.Wall = time.Since(started)
	measured.CPU = heardBenchmarkCPU(b) - before
	if cpuProfile != nil {
		pprof.StopCPUProfile()
		if err = cpuProfile.Close(); err != nil {
			b.Fatal(err)
		}
	}
	var usage syscall.Rusage
	if err = syscall.Getrusage(syscall.RUSAGE_SELF, &usage); err != nil {
		b.Fatal(err)
	}
	measured.ProcessPeakRSS = usage.Maxrss
	var encode, acceleration, trace, finish, boxed, dispatch time.Duration
	for _, p := range measured.Frames {
		encode += p.Encode
		acceleration += p.AccelerationWall
		trace += p.TraceGPU
		finish += p.Finish
		boxed += p.BoxedCPU.Wall
		dispatch += p.DispatchWall
	}
	n := float64(len(measured.Frames))
	for name, value := range map[string]time.Duration{"encode-ms/frame": encode, "accel-ms/frame": acceleration, "trace-gpu-ms/frame": trace, "finish-ms/frame": finish, "boxed-cpu-ms/frame": boxed, "dispatch-ms/frame": dispatch, "cpu-ms/frame": measured.CPU} {
		b.ReportMetric(float64(value)/float64(time.Millisecond)/n, name)
	}
	b.ReportMetric(n/measured.Wall.Seconds(), "frames/s")
	directory := os.Getenv("HEARD_BENCH_OUT")
	if directory == "" {
		b.Fatal("HEARD_BENCH_OUT is required")
	}
	if err = os.MkdirAll(directory, 0755); err != nil {
		b.Fatal(err)
	}
	raw, err := json.MarshalIndent(measured, "", "  ")
	if err != nil {
		b.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(directory, fmt.Sprintf("%s-workers-%d.json", measured.Case, workers)), append(raw, '\n'), 0644); err != nil {
		b.Fatal(err)
	}
}

func heardBenchmarkCPU(b *testing.B) time.Duration {
	b.Helper()
	var usage syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &usage); err != nil {
		b.Fatal(err)
	}
	return time.Duration(usage.Utime.Nano() + usage.Stime.Nano())
}

// BenchmarkHeardMultitudeInventory records the unchanged finale's actual
// construction size. It does no GPU work and makes no deliverable images.
func BenchmarkHeardMultitudeInventory(b *testing.B) {
	for range b.N {
		stage, err := newHeardStage(&stagePlan{Kind: "multitude"}, "")
		if err != nil {
			b.Fatal(err)
		}
		shapes := map[string]int{}
		parts := 0
		for _, being := range stage.crowd.beings {
			parts += len(being.parts)
		}
		for _, part := range stage.world.Things {
			shapes[fmt.Sprintf("%T", part.Shape)]++
		}
		report := struct {
			Beings, Parts, Things, Volumes int
			Shapes                         map[string]int
		}{len(stage.crowd.beings), parts, len(stage.world.Things), len(stage.world.Volumes), shapes}
		raw, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			b.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(os.Getenv("HEARD_BENCH_OUT"), "multitude-inventory.json"), append(raw, '\n'), 0644); err != nil {
			b.Fatal(err)
		}
		b.ReportMetric(float64(report.Beings), "beings")
		b.ReportMetric(float64(report.Things), "things")
	}
}
