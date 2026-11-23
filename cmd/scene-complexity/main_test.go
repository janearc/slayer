package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime/debug"
	"strings"
	"testing"
	"time"
)

func TestBuildStampIncludesGoVersionAndAvailableVCSData(t *testing.T) {
	got := stampedBuild()
	info, ok := debug.ReadBuildInfo()
	if !ok {
		if got.GoVersion != "unknown" || got.VCSRevision != "unknown" || got.VCSModified != "unknown" {
			t.Fatalf("unavailable build info was not explicit: %+v", got)
		}
		return
	}
	wantVersion, wantRevision, wantModified := knownOrUnknown(info.GoVersion), "unknown", "unknown"
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			wantRevision = knownOrUnknown(setting.Value)
		case "vcs.modified":
			wantModified = knownOrUnknown(setting.Value)
		}
	}
	if got.GoVersion != wantVersion || got.VCSRevision != wantRevision || got.VCSModified != wantModified {
		t.Fatalf("build stamp %+v; want Go=%q revision=%q modified=%q",
			got, wantVersion, wantRevision, wantModified)
	}
	if got.Version != knownOrUnknown(build) || got.BuiltAt != knownOrUnknown(built) ||
		got.Builder != knownOrUnknown(who) || got.Sessions != knownOrUnknown(sessions) {
		t.Fatalf("Go metadata changed the independent Game stamp fields: %+v", got)
	}
}

func TestFrameIndicesAreExplicitAndBounded(t *testing.T) {
	for _, test := range []struct {
		spec string
		all  bool
		want []int
		bad  bool
	}{
		{"", false, []int{0}, false},
		{"1,3", false, []int{1, 3}, false},
		{"first,last", false, []int{0, 3}, false},
		{"", true, []int{0, 1, 2, 3}, false},
		{"1,1", false, nil, true},
		{"4", false, nil, true},
		{"-1", false, nil, true},
		{"1,,2", false, nil, true},
	} {
		got, err := frameIndices(test.spec, test.all, 4)
		if test.bad {
			if err == nil {
				t.Errorf("frameIndices(%q, %v) accepted an invalid selection", test.spec, test.all)
			}
			continue
		}
		if err != nil || !reflect.DeepEqual(got, test.want) {
			t.Errorf("frameIndices(%q, %v) = %v, %v; want %v", test.spec, test.all, got, err, test.want)
		}
	}
}

func TestWriteReportsNamedJobAtRequestedExactTimes(t *testing.T) {
	t.Setenv("SLAYER_CACHE", t.TempDir())
	path := writeJob(t, "scene: {named: orbital, with: {far: true}}\nrenderer: cpu\n"+
		"film: {width: 24, height: 14, rays: 1}\n"+
		"fps: 2\nstart: {value: 0.5, unit: s}\n"+
		"duration: {value: 1, unit: s}\n")
	var output bytes.Buffer
	if err := writeReports(t.Context(), []string{path}, "1", false, &output); err != nil {
		t.Fatalf("%v; output: %s", err, output.String())
	}
	var got row
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	jobDigest := sha256.Sum256(source)
	if got.Frame != 1 || got.FrameCount != 2 || got.AtNanoseconds != int64(time.Second) ||
		got.Geometry.Geometry.Things == 0 || got.Scene != "named:orbital" || got.Error != "" ||
		got.JobSHA256 != fmt.Sprintf("%x", jobDigest) || got.JobKey == "" ||
		got.Build.Version != "unknown" || got.OpenAllocatedBytes == 0 {
		t.Fatalf("unexpected exact-time report: %+v", got)
	}
}

func TestWriteReportsSetJobAndRetainsFailureRows(t *testing.T) {
	t.Setenv("SLAYER_CACHE", t.TempDir())
	setPath, err := filepath.Abs("../../sets/widow.yaml")
	if err != nil {
		t.Fatal(err)
	}
	path := writeJob(t, "scene: {set: "+setPath+"}\nrenderer: cpu\n")
	var output bytes.Buffer
	missing := filepath.Join(t.TempDir(), "missing.yaml")
	err = writeReports(t.Context(), []string{path, missing}, "", false, &output)
	if err == nil || !strings.Contains(err.Error(), "failed") {
		t.Fatalf("expected a reported failure: %v", err)
	}
	trimmed := strings.TrimSpace(output.String())
	lines := strings.Split(trimmed, "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d JSON rows: %s", len(lines), output.String())
	}
	var setRow row
	if err := json.Unmarshal([]byte(lines[0]), &setRow); err != nil {
		t.Fatal(err)
	}
	if setRow.Scene != "set:"+setPath || setRow.FrameCount != 1 || setRow.Geometry.Geometry.Things == 0 ||
		setRow.Error != "" || setRow.JobSHA256 == "" || setRow.JobKey == "" ||
		setRow.InputPaths["set"] != setPath {
		t.Fatalf("set row is incomplete: %+v", setRow)
	}
	var failure row
	if err := json.Unmarshal([]byte(lines[1]), &failure); err != nil {
		t.Fatal(err)
	}
	if failure.Error == "" || failure.Job == "" {
		t.Fatalf("failed job did not retain an error row: %+v", failure)
	}
}

func TestWriteReportsOrdinarySmokeJob(t *testing.T) {
	t.Setenv("SLAYER_CACHE", t.TempDir())
	path := writeJob(t, "scene: {smoke: 1}\nrenderer: cpu\n")
	var output bytes.Buffer
	if err := writeReports(t.Context(), []string{path}, "", false, &output); err != nil {
		t.Fatalf("%v; output: %s", err, output.String())
	}
	var got row
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Scene != "smoke:1" || got.FrameCount != 1 || got.Geometry.Geometry.Things == 0 || got.Error != "" {
		t.Fatalf("unexpected smoke report: %+v", got)
	}
}

func TestWriteReportsAShippedNamedSceneJob(t *testing.T) {
	t.Setenv("SLAYER_CACHE", t.TempDir())
	path, err := filepath.Abs("../../doc/examples/jobs/lantern-portrait.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := writeReports(t.Context(), []string{path}, "", false, &output); err != nil {
		t.Fatalf("%v; output: %s", err, output.String())
	}
	var got row
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Scene != "named:swarm" || got.FrameCount != 1 || got.Geometry.Geometry.Things == 0 || got.Error != "" {
		t.Fatalf("unexpected named-scene report: %+v", got)
	}
}

func writeJob(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "job.yaml")
	data := []byte("$schema: https://github.com/janearc/slayer/schemas/job-v1.json\n" + body)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
