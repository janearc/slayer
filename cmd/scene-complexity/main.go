// Command scene-complexity reports constructed scene geometry from checked
// Slayer jobs without drawing a frame.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/puzzlemouth/verb"
	"github.com/janearc/slayer/improv/searching"
	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/session"
)

var build, built, who, sessions string
var authoredScenes sync.Once
var authoredScenesError error

// main runs the scene-complexity tool and exits with its status.
func main() {
	tool := pzm.New("scene-complexity", "count geometry constructed by Slayer jobs",
		pzm.Build(build), pzm.Built(built), pzm.Who(who), pzm.Sessions(sessions))
	reads := verb.NewFacts(verb.WritesNothing, verb.LeavesAlone,
		verb.ThisMachine, verb.NoOne, verb.Light)
	tool.Verb("report", verb.NoShort, "inspect actual constructed scene geometry without drawing",
		"Emits JSON lines for each selected frame time. Defaults to frame 0; --frames accepts comma-separated indexes, or --all inspects the complete job. Counts are current scene geometry, not a working-set or GPU-memory estimate.",
		"Example_report", reads, report,
		pzm.Value("frames", verb.NoShort, "comma-separated indexes or first/last; default 0"),
		pzm.Switch("all", verb.NoShort, "inspect every frame time in each job"))
	os.Exit(tool.Run(os.Args[1:], os.Stdout, os.Stderr))
}

type row struct {
	Job                  string                  `json:"job"`
	Scene                string                  `json:"scene"`
	JobSHA256            string                  `json:"job_sha256,omitempty"`
	JobKey               string                  `json:"job_key,omitempty"`
	InputPaths           map[string]string       `json:"input_paths,omitempty"`
	Build                buildInfo               `json:"build"`
	Frame                int                     `json:"frame"`
	FrameCount           int                     `json:"frame_count"`
	AtNanoseconds        int64                   `json:"scene_time_nanoseconds"`
	OpenNanoseconds      int64                   `json:"open_nanoseconds"`
	SampleNanoseconds    int64                   `json:"sample_nanoseconds"`
	OpenAllocatedBytes   uint64                  `json:"open_allocated_bytes"`
	SampleAllocatedBytes uint64                  `json:"sample_allocated_bytes"`
	Geometry             session.SceneComplexity `json:"complexity"`
	Error                string                  `json:"error,omitempty"`
}

type buildInfo struct {
	Version     string `json:"version"`
	BuiltAt     string `json:"built_at"`
	Builder     string `json:"builder"`
	Sessions    string `json:"sessions"`
	GoVersion   string `json:"go_version"`
	VCSRevision string `json:"vcs_revision"`
	VCSModified string `json:"vcs_modified"`
}

// stampedBuild returns the build stamp for a report row. A field the
// producer did not embed reads as unknown.
func stampedBuild() buildInfo {
	stamp := buildInfo{Version: knownOrUnknown(build), BuiltAt: knownOrUnknown(built),
		Builder: knownOrUnknown(who), Sessions: knownOrUnknown(sessions),
		GoVersion: "unknown", VCSRevision: "unknown", VCSModified: "unknown"}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return stamp
	}
	stamp.GoVersion = knownOrUnknown(info.GoVersion)
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			stamp.VCSRevision = knownOrUnknown(setting.Value)
		case "vcs.modified":
			stamp.VCSModified = knownOrUnknown(setting.Value)
		}
	}
	return stamp
}

// knownOrUnknown returns value, or unknown when value is empty.
func knownOrUnknown(value string) string {
	if value == "" {
		return "unknown"
	}
	return value
}

// report checks the report flags, then writes a row for each selected frame
// of each job.
func report(call pzm.Call) error {
	if len(call.Args()) == 0 {
		return fmt.Errorf("scene-complexity report needs one or more job files")
	}
	if call.On("all") && call.Given("frames") {
		return fmt.Errorf("use either --all or --frames")
	}
	return writeReports(context.Background(), call.Args(), call.Value("frames"), call.On("all"), call.Out())
}

// writeReports writes a report row for each selected frame of each job to
// output. A job or sample that fails writes an error row.
func writeReports(ctx context.Context, paths []string, frameSpec string, all bool, output io.Writer) error {
	authoredScenes.Do(func() { authoredScenesError = searching.RegisterSlayerScenes() })
	if authoredScenesError != nil {
		return fmt.Errorf("register authored scenes: %w", authoredScenesError)
	}
	encoder := json.NewEncoder(output)
	failed := 0
	for _, path := range paths {
		source, readErr := os.ReadFile(path)
		jobSHA := ""
		if readErr == nil {
			sum := sha256.Sum256(source)
			jobSHA = fmt.Sprintf("%x", sum)
		}
		wanted, err := job.Read(path)
		if err != nil {
			if readErr != nil {
				err = readErr
			}
			if encodeErr := encoder.Encode(row{Job: path, JobSHA256: jobSHA, Build: stampedBuild(), Error: err.Error()}); encodeErr != nil {
				return encodeErr
			}
			failed++
			continue
		}
		confirmed, confirmErr := os.ReadFile(path)
		if confirmErr != nil || !bytes.Equal(source, confirmed) {
			err = fmt.Errorf("job source changed while it was being read")
			if confirmErr != nil {
				err = fmt.Errorf("job source could not be confirmed after read: %w", confirmErr)
			}
			if encodeErr := encoder.Encode(row{Job: path, Scene: sceneLabel(wanted), JobSHA256: jobSHA,
				JobKey: wanted.Key(), InputPaths: wanted.Inputs(), Build: stampedBuild(), Error: err.Error()}); encodeErr != nil {
				return encodeErr
			}
			failed++
			continue
		}
		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		started := time.Now()
		session, err := session.Open(ctx, wanted, session.Stamped(build, built, who, sessions))
		opened := time.Since(started)
		runtime.ReadMemStats(&after)
		openAllocated := after.TotalAlloc - before.TotalAlloc
		if err != nil {
			if encodeErr := encoder.Encode(row{Job: path, Scene: sceneLabel(wanted),
				JobSHA256: jobSHA, JobKey: wanted.Key(), InputPaths: wanted.Inputs(), Build: stampedBuild(),
				OpenNanoseconds: opened.Nanoseconds(), OpenAllocatedBytes: openAllocated, Error: err.Error()}); encodeErr != nil {
				return encodeErr
			}
			failed++
			continue
		}
		times := session.Times()
		indices, err := frameIndices(frameSpec, all, len(times))
		if err != nil {
			_ = session.Close()
			if encodeErr := encoder.Encode(row{Job: path, Scene: sceneLabel(wanted),
				JobSHA256: jobSHA, JobKey: wanted.Key(), InputPaths: wanted.Inputs(), Build: stampedBuild(),
				FrameCount: len(times), OpenNanoseconds: opened.Nanoseconds(), OpenAllocatedBytes: openAllocated, Error: err.Error()}); encodeErr != nil {
				return encodeErr
			}
			failed++
			continue
		}
		for _, index := range indices {
			runtime.ReadMemStats(&before)
			started = time.Now()
			complexity, inspectErr := session.Inspect(index)
			sampled := time.Since(started)
			runtime.ReadMemStats(&after)
			row := row{Job: path, Scene: sceneLabel(wanted), Frame: index,
				JobSHA256: jobSHA, JobKey: wanted.Key(), InputPaths: wanted.Inputs(), Build: stampedBuild(),
				FrameCount: len(times), AtNanoseconds: int64(times[index]),
				OpenNanoseconds: opened.Nanoseconds(), SampleNanoseconds: sampled.Nanoseconds(),
				OpenAllocatedBytes: openAllocated, SampleAllocatedBytes: after.TotalAlloc - before.TotalAlloc,
				Geometry: complexity}
			if inspectErr != nil {
				row.Error = inspectErr.Error()
				failed++
			}
			if err := encoder.Encode(row); err != nil {
				_ = session.Close()
				return err
			}
		}
		if err := session.Close(); err != nil {
			if encodeErr := encoder.Encode(row{Job: path, Scene: sceneLabel(wanted),
				JobSHA256: jobSHA, JobKey: wanted.Key(), InputPaths: wanted.Inputs(), Build: stampedBuild(),
				FrameCount: len(times), OpenNanoseconds: opened.Nanoseconds(), OpenAllocatedBytes: openAllocated,
				Error: "close: " + err.Error()}); encodeErr != nil {
				return encodeErr
			}
			failed++
		}
	}
	if failed > 0 {
		return fmt.Errorf("scene-complexity: %d job or frame inspections failed; result rows were retained", failed)
	}
	return nil
}

// sceneLabel names the scene a job selects: its named scene, smoke, set,
// frames path, or form.
func sceneLabel(wanted job.Job) string {
	scene := wanted.Scene()
	if name, ok := scene.Named(); ok {
		return "named:" + name
	}
	if number, ok := scene.Smoke(); ok {
		return fmt.Sprintf("smoke:%d", number)
	}
	if path, ok := scene.Set(); ok {
		return "set:" + path
	}
	if path, ok := scene.Frames(); ok {
		return "frames:" + path
	}
	return scene.Form()
}

// frameIndices returns the frame indexes to report. It returns every index
// when all is set, frame zero when spec is empty, and the indexes in spec
// otherwise.
func frameIndices(spec string, all bool, count int) ([]int, error) {
	if count < 1 {
		return nil, fmt.Errorf("job has no frame times")
	}
	if all {
		indices := make([]int, count)
		for index := range count {
			indices[index] = index
		}
		return indices, nil
	}
	if spec == "" {
		return []int{0}, nil
	}
	seen := map[int]bool{}
	indices := []int{}
	for _, word := range strings.Split(spec, ",") {
		index := -1
		if word == "first" {
			index = 0
		} else if word == "last" {
			index = count - 1
		} else {
			var err error
			index, err = strconv.Atoi(word)
			if err != nil {
				return nil, fmt.Errorf("frame index %q is outside 0..%d", word, count-1)
			}
		}
		if index < 0 || index >= count {
			return nil, fmt.Errorf("frame index %q is outside 0..%d", word, count-1)
		}
		if seen[index] {
			return nil, fmt.Errorf("frame index %d is repeated", index)
		}
		seen[index] = true
		indices = append(indices, index)
	}
	return indices, nil
}
