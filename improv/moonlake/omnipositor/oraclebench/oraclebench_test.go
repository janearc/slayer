package oraclebench

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestParseBSDTime(t *testing.T) {
	got, err := ParseBSDTime(strings.NewReader("42 page reclaims\n0.20 user 0.50 real 0.30 sys\n123456 maximum resident set size\n"))
	if err != nil {
		t.Fatalf("ParseBSDTime() error = %v", err)
	}
	if got.Wall == nil || time.Duration(*got.Wall) != 500*time.Millisecond {
		t.Fatalf("wall = %v", got.Wall)
	}
	if got.User == nil || time.Duration(*got.User) != 200*time.Millisecond {
		t.Fatalf("user = %v", got.User)
	}
	if got.System == nil || time.Duration(*got.System) != 300*time.Millisecond {
		t.Fatalf("system = %v", got.System)
	}
	if got.PeakRSSBytes == nil || *got.PeakRSSBytes != 123456 {
		t.Fatalf("peak rss = %v", got.PeakRSSBytes)
	}
}

func TestParseSecondsExactAndBounded(t *testing.T) {
	tests := []struct {
		name string
		text string
		want time.Duration
	}{
		{name: "bsd-two-decimals", text: "0.20", want: 200 * time.Millisecond},
		{name: "trimmed-trailing-zeros", text: "1.230000000", want: 1230 * time.Millisecond},
		{name: "max-duration", text: "9223372036.854775807", want: time.Duration(1<<63 - 1)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseSeconds(tc.text)
			if err != nil {
				t.Fatalf("parseSeconds(%q) error = %v", tc.text, err)
			}
			if time.Duration(got) != tc.want {
				t.Fatalf("parseSeconds(%q) = %v, want %v", tc.text, got, tc.want)
			}
		})
	}

	for _, tc := range []struct {
		name string
		text string
	}{
		{name: "nonexact-fraction", text: "0.1234567891"},
		{name: "out-of-range", text: "9223372036.854775808"},
		{name: "negative-time", text: "-0.10"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := parseSeconds(tc.text); err == nil {
				t.Fatalf("parseSeconds(%q) error = nil", tc.text)
			}
		})
	}
}

func TestParseBSDTimeRejectsInvalidInput(t *testing.T) {
	for _, tc := range []struct {
		name string
		text string
	}{
		{name: "duplicate-real", text: "0.10 real\n0.20 real\n0.10 user\n0.10 sys\n1 maximum resident set size\n"},
		{name: "missing-system", text: "0.10 real\n0.10 user\n1 maximum resident set size\n"},
		{name: "malformed-rss", text: "0.10 real\n0.10 user\n0.10 sys\nmaximum resident set size nope\n"},
		{name: "negative-time", text: "-0.10 real\n0.10 user\n0.10 sys\n1 maximum resident set size\n"},
		{name: "loose-time-format", text: "user 0.10\n0.10 real\n0.10 sys\n1 maximum resident set size\n"},
		{name: "trailing-time-junk", text: "0.10 user extra\n0.10 real\n0.10 sys\n1 maximum resident set size\n"},
		{name: "trailing-rss-junk", text: "0.10 real\n0.10 user\n0.10 sys\n1 maximum resident set size extra\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ParseBSDTime(strings.NewReader(tc.text)); err == nil {
				t.Fatalf("ParseBSDTime() error = nil")
			}
		})
	}
}

func TestAssessComparability(t *testing.T) {
	load1 := 3.5
	load5 := 2.5
	load15 := 1.5
	power := PowerMains
	memory := MemoryNormal
	render := RenderIdle
	got := AssessComparability(EnvironmentSnapshot{
		Load1:          &load1,
		Load5:          &load5,
		Load15:         &load15,
		Power:          &power,
		MemoryPressure: &memory,
		Render:         &render,
	})
	if got.Status != Comparable {
		t.Fatalf("Status = %q", got.Status)
	}
	if got.Load != CheckPass || got.Power != CheckPass || got.MemoryPressure != CheckPass || got.Render != CheckPass {
		t.Fatalf("checks = %+v", got)
	}
	if got.Observed.Load1 == nil || *got.Observed.Load1 != load1 {
		t.Fatalf("observed load1 = %v", got.Observed.Load1)
	}
}

func TestAssessComparabilityFailsClosed(t *testing.T) {
	power := PowerBattery
	memory := MemoryWarning
	render := RenderActive
	got := AssessComparability(EnvironmentSnapshot{
		Power:          &power,
		MemoryPressure: &memory,
		Render:         &render,
	})
	if got.Status != NonComparable {
		t.Fatalf("Status = %q", got.Status)
	}
	if got.Load != CheckUnknown || got.Power != CheckFail || got.MemoryPressure != CheckFail || got.Render != CheckFail {
		t.Fatalf("checks = %+v", got)
	}
	if got.Observed.Load1 != nil || got.Observed.Load5 != nil || got.Observed.Load15 != nil {
		t.Fatalf("observed load values = %+v", got.Observed)
	}
	if len(got.Reasons) == 0 {
		t.Fatalf("reasons = %v", got.Reasons)
	}
}

func TestCommandValidate(t *testing.T) {
	tests := []struct {
		name    string
		command Command
		want    string
	}{
		{name: "valid-empty-arg", command: Command{Binary: "/bin/echo", Args: []string{""}}, want: ""},
		{name: "env-empty-value", command: Command{Binary: "/bin/echo", Env: []string{"A="}}, want: ""},
		{name: "binary-empty", command: Command{}, want: "command binary is required"},
		{name: "binary-nul", command: Command{Binary: "bin\x00ary"}, want: "command binary contains NUL"},
		{name: "dir-empty", command: Command{Binary: "/bin/echo", Dir: stringPtr("")}, want: "command dir is empty"},
		{name: "dir-nul", command: Command{Binary: "/bin/echo", Dir: stringPtr("/tmp\x00dir")}, want: "command dir contains NUL"},
		{name: "arg-nul", command: Command{Binary: "/bin/echo", Args: []string{"ok", "bad\x00"}}, want: "command arg 1 contains NUL"},
		{name: "env-empty", command: Command{Binary: "/bin/echo", Env: []string{""}}, want: "command env 0 is empty"},
		{name: "env-nul", command: Command{Binary: "/bin/echo", Env: []string{"A=B\x00"}}, want: "command env 0 contains NUL"},
		{name: "env-missing-key", command: Command{Binary: "/bin/echo", Env: []string{"=value"}}, want: "command env 0 has invalid key"},
		{name: "env-duplicate-key", command: Command{Binary: "/bin/echo", Env: []string{"A=B", "A=C"}}, want: "command env 1 duplicates key \"A\""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.command.Validate()
			if tc.want == "" {
				if err != nil {
					t.Fatalf("Command.Validate() error = %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("Command.Validate() error = nil")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Command.Validate() error = %v, want substring %q", err, tc.want)
			}
		})
	}
}

func TestScenarioIdentityValidateAcceptsLowercaseDigests(t *testing.T) {
	frameIndex := int64(1)
	scenario := ScenarioIdentity{
		FrameIndex:     &frameIndex,
		Width:          1,
		Height:         1,
		FPS:            1,
		BinarySHA256:   strings.Repeat("a", 64),
		SourceRevision: strings.Repeat("b", 64),
		InputDigest:    strings.Repeat("c", 64),
	}
	if err := scenario.Validate(); err != nil {
		t.Fatalf("ScenarioIdentity.Validate() error = %v", err)
	}
}

func TestRunSampleValidateRejectsComparabilityMismatch(t *testing.T) {
	got := runSample(t, "run", 100*time.Millisecond, 10, 7, true)
	got.Comparability.Status = NonComparable
	if err := got.Validate(); err == nil {
		t.Fatalf("RunSample.Validate() error = nil")
	}
}

func TestRunSampleValidateRejectsScenarioIdentityProblems(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*RunSample)
	}{
		{name: "missing-frame-index", mutate: func(sample *RunSample) { sample.Scenario.FrameIndex = nil }},
		{name: "negative-frame-index", mutate: func(sample *RunSample) { value := int64(-1); sample.Scenario.FrameIndex = &value }},
		{name: "nonpositive-width", mutate: func(sample *RunSample) { sample.Scenario.Width = 0 }},
		{name: "nonpositive-height", mutate: func(sample *RunSample) { sample.Scenario.Height = 0 }},
		{name: "nonpositive-fps", mutate: func(sample *RunSample) { sample.Scenario.FPS = 0 }},
		{name: "invalid-binary-sha256", mutate: func(sample *RunSample) { sample.Scenario.BinarySHA256 = "not-hex" }},
		{name: "upper-binary-sha256", mutate: func(sample *RunSample) { sample.Scenario.BinarySHA256 = strings.ToUpper(sample.Scenario.BinarySHA256) }},
		{name: "upper-source-revision", mutate: func(sample *RunSample) {
			sample.Scenario.SourceRevision = strings.ToUpper(sample.Scenario.SourceRevision)
		}},
		{name: "short-source-revision", mutate: func(sample *RunSample) { sample.Scenario.SourceRevision = "abcdef1234567890abcdef1234567890abcdef1" }},
		{name: "long-source-revision", mutate: func(sample *RunSample) { sample.Scenario.SourceRevision = "abcdef1234567890abcdef1234567890abcdef123" }},
		{name: "upper-input-digest", mutate: func(sample *RunSample) { sample.Scenario.InputDigest = strings.ToUpper(sample.Scenario.InputDigest) }},
		{name: "short-input-digest", mutate: func(sample *RunSample) {
			sample.Scenario.InputDigest = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcde"
		}},
		{name: "long-input-digest", mutate: func(sample *RunSample) {
			sample.Scenario.InputDigest = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0"
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sample := runSample(t, "run", 100*time.Millisecond, 10, 7, true)
			tc.mutate(&sample)
			if err := sample.Validate(); err == nil {
				t.Fatalf("RunSample.Validate() error = nil")
			}
		})
	}
}

func TestAggregateRuns(t *testing.T) {
	runs := []RunSample{
		runSample(t, "run", 100*time.Millisecond, 10, 7, true),
		runSample(t, "run", 300*time.Millisecond, 30, 21, true),
		runSample(t, "run", 900*time.Millisecond, 90, 63, false),
	}
	got, err := AggregateRuns(runs)
	if err != nil {
		t.Fatalf("AggregateRuns() error = %v", err)
	}
	if got.ComparableRuns != 2 {
		t.Fatalf("ComparableRuns = %d", got.ComparableRuns)
	}
	if len(got.Stages) != 2 {
		t.Fatalf("stages = %d", len(got.Stages))
	}
	if got.Stages[0].Name != "lightshow" || got.Stages[1].Name != "develop" {
		t.Fatalf("stage names = %+v", got.Stages)
	}
	if got.Stages[0].Wall.Observed != 2 || got.Stages[0].Wall.Unknown != 0 {
		t.Fatalf("stage 0 wall counts = %+v", got.Stages[0].Wall)
	}
	if got.Stages[0].Wall.Median == nil || time.Duration(*got.Stages[0].Wall.Median) != 200*time.Millisecond {
		t.Fatalf("stage 0 wall median = %v", got.Stages[0].Wall.Median)
	}
	if got.Stages[0].Wall.Min == nil || time.Duration(*got.Stages[0].Wall.Min) != 100*time.Millisecond {
		t.Fatalf("stage 0 wall min = %v", got.Stages[0].Wall.Min)
	}
	if got.Stages[0].Wall.Max == nil || time.Duration(*got.Stages[0].Wall.Max) != 300*time.Millisecond {
		t.Fatalf("stage 0 wall max = %v", got.Stages[0].Wall.Max)
	}
	if got.Stages[0].PeakRSS.Observed != 2 || got.Stages[0].PeakRSS.Unknown != 0 {
		t.Fatalf("stage 0 peak rss counts = %+v", got.Stages[0].PeakRSS)
	}
	if got.Stages[0].PeakRSS.Median == nil || *got.Stages[0].PeakRSS.Median != 20 {
		t.Fatalf("stage 0 peak rss median = %v", got.Stages[0].PeakRSS.Median)
	}
	if got.Stages[0].IntermediateBytes[0].Name != "pfm" {
		t.Fatalf("stage 0 intermediate name = %+v", got.Stages[0].IntermediateBytes)
	}
	if got.Stages[0].IntermediateBytes[0].Bytes.Median == nil || *got.Stages[0].IntermediateBytes[0].Bytes.Median != 14 {
		t.Fatalf("stage 0 intermediate median = %v", got.Stages[0].IntermediateBytes[0].Bytes.Median)
	}
}

func TestAggregateRunsRequiresThreeComparableRuns(t *testing.T) {
	runs := []RunSample{
		runSample(t, "run", 100*time.Millisecond, 10, 7, true),
		runSample(t, "run", 300*time.Millisecond, 30, 21, true),
	}
	got, err := AggregateRuns(runs)
	if err != nil {
		t.Fatalf("AggregateRuns() error = %v", err)
	}
	if got.Comparability.Status != NonComparable {
		t.Fatalf("Comparability.Status = %q", got.Comparability.Status)
	}
	if got.ComparableRuns != 2 {
		t.Fatalf("ComparableRuns = %d", got.ComparableRuns)
	}
}

func TestAggregateRunsRejectsStageMismatch(t *testing.T) {
	good := runSample(t, "run", 100*time.Millisecond, 10, 7, true)
	bad := runSample(t, "run", 300*time.Millisecond, 30, 21, false)
	bad.Stages[1].Name = "unexpected"
	if _, err := AggregateRuns([]RunSample{good, bad}); err == nil {
		t.Fatalf("AggregateRuns() error = nil")
	}
}

func TestAggregateRunsRejectsProvenanceMismatch(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*RunSample)
		want   string
	}{
		{name: "binary", mutate: func(sample *RunSample) { sample.Provenance.Command.Binary = "/opt/other" }, want: "provenance command binary mismatch"},
		{name: "args", mutate: func(sample *RunSample) { sample.Provenance.Command.Args = []string{"-l", "changed"} }, want: "provenance command args mismatch"},
		{name: "dir", mutate: func(sample *RunSample) { sample.Provenance.Command.Dir = stringPtr("/tmp/other") }, want: "provenance command dir mismatch"},
		{name: "env", mutate: func(sample *RunSample) { sample.Provenance.Command.Env = []string{"A=B", "C=DIFF"} }, want: "provenance command env mismatch"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			good := runSample(t, "run", 100*time.Millisecond, 10, 7, true)
			bad := runSample(t, "run", 300*time.Millisecond, 30, 21, false)
			tc.mutate(&bad)
			_, err := AggregateRuns([]RunSample{good, bad})
			if err == nil {
				t.Fatalf("AggregateRuns() error = nil")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("AggregateRuns() error = %v, want substring %q", err, tc.want)
			}
		})
	}
}

func TestAggregateRunsRejectsScenarioMismatch(t *testing.T) {
	good := runSample(t, "run", 100*time.Millisecond, 10, 7, true)
	bad := runSample(t, "run", 300*time.Millisecond, 30, 21, true)
	bad.Scenario.FrameIndex = testInt64Ptr(9)
	if _, err := AggregateRuns([]RunSample{good, bad}); err == nil {
		t.Fatalf("AggregateRuns() error = nil")
	}
}

func TestAggregateRunsRejectsScenarioSizeMismatch(t *testing.T) {
	good := runSample(t, "run", 100*time.Millisecond, 10, 7, true)
	bad := runSample(t, "run", 300*time.Millisecond, 30, 21, true)
	bad.Scenario.Width++
	if _, err := AggregateRuns([]RunSample{good, bad}); err == nil {
		t.Fatalf("AggregateRuns() error = nil")
	}
}

func TestAggregateRunsRejectsScenarioSettingsMismatch(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*RunSample)
		want   string
	}{
		{name: "input-digest", mutate: func(sample *RunSample) {
			sample.Scenario.InputDigest = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
		}, want: "scenario mismatch"},
		{name: "source-revision", mutate: func(sample *RunSample) {
			sample.Scenario.SourceRevision = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
		}, want: "scenario mismatch"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			good := runSample(t, "run", 100*time.Millisecond, 10, 7, true)
			bad := runSample(t, "run", 300*time.Millisecond, 30, 21, true)
			tc.mutate(&bad)
			_, err := AggregateRuns([]RunSample{good, bad})
			if err == nil {
				t.Fatalf("AggregateRuns() error = nil")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("AggregateRuns() error = %v, want substring %q", err, tc.want)
			}
		})
	}
}

func TestAggregateRunsTracksUnknownSamplesAndPreservesPrecision(t *testing.T) {
	wall0 := time.Duration(1<<53 + 3)
	wall2 := time.Duration(1<<53 + 7)
	peak0 := int64(1<<53 + 5)
	peak1 := peak0 + 2
	peak2 := peak0 + 4
	runs := []RunSample{
		runSample(t, "run", wall0, peak0, 7, true),
		runSample(t, "run", wall0+2, peak1, 21, true),
		runSample(t, "run", wall2, peak2, 63, true),
	}
	runs[1].Stages[0].Timing.Wall = nil
	got, err := AggregateRuns(runs)
	if err != nil {
		t.Fatalf("AggregateRuns() error = %v", err)
	}
	if got.Comparability.Status != Comparable {
		t.Fatalf("Comparability.Status = %q", got.Comparability.Status)
	}
	if got.Stages[0].Wall.Observed != 2 || got.Stages[0].Wall.Unknown != 1 {
		t.Fatalf("stage 0 wall counts = %+v", got.Stages[0].Wall)
	}
	if got.Stages[0].User.Observed != 3 || got.Stages[0].User.Unknown != 0 {
		t.Fatalf("stage 0 user counts = %+v", got.Stages[0].User)
	}
	if got.Stages[0].System.Observed != 3 || got.Stages[0].System.Unknown != 0 {
		t.Fatalf("stage 0 system counts = %+v", got.Stages[0].System)
	}
	if got.Stages[0].PeakRSS.Observed != 3 || got.Stages[0].PeakRSS.Unknown != 0 {
		t.Fatalf("stage 0 peak rss counts = %+v", got.Stages[0].PeakRSS)
	}
	if got.Stages[0].ReadBytes.Observed != 3 || got.Stages[0].ReadBytes.Unknown != 0 {
		t.Fatalf("stage 0 read counts = %+v", got.Stages[0].ReadBytes)
	}
	if got.Stages[0].WrittenBytes.Observed != 3 || got.Stages[0].WrittenBytes.Unknown != 0 {
		t.Fatalf("stage 0 written counts = %+v", got.Stages[0].WrittenBytes)
	}
	if got.Stages[0].Wall.Min == nil || time.Duration(*got.Stages[0].Wall.Min) != wall0 {
		t.Fatalf("stage 0 wall min = %v", got.Stages[0].Wall.Min)
	}
	if got.Stages[0].Wall.Median == nil || time.Duration(*got.Stages[0].Wall.Median) != wall0+2 {
		t.Fatalf("stage 0 wall median = %v", got.Stages[0].Wall.Median)
	}
	if got.Stages[0].Wall.Max == nil || time.Duration(*got.Stages[0].Wall.Max) != wall2 {
		t.Fatalf("stage 0 wall max = %v", got.Stages[0].Wall.Max)
	}
	if got.Stages[0].PeakRSS.Min == nil || *got.Stages[0].PeakRSS.Min != peak0 {
		t.Fatalf("stage 0 peak rss min = %v", got.Stages[0].PeakRSS.Min)
	}
	if got.Stages[0].PeakRSS.Median == nil || *got.Stages[0].PeakRSS.Median != peak1 {
		t.Fatalf("stage 0 peak rss median = %v", got.Stages[0].PeakRSS.Median)
	}
	if got.Stages[0].PeakRSS.Max == nil || *got.Stages[0].PeakRSS.Max != peak2 {
		t.Fatalf("stage 0 peak rss max = %v", got.Stages[0].PeakRSS.Max)
	}
}

func TestRunSampleJSONPreservesUnknowns(t *testing.T) {
	got := runSample(t, "run-a", 100*time.Millisecond, 10, 7, true)
	got.Comparability = AssessComparability(EnvironmentSnapshot{})
	text, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if !strings.Contains(string(text), `"load_1":null`) {
		t.Fatalf("json output = %s", text)
	}
	if !strings.Contains(string(text), `"wall":"100ms"`) {
		t.Fatalf("json output = %s", text)
	}
	var round RunSample
	if err := json.Unmarshal(text, &round); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if round.Stages[0].Timing.Wall == nil || time.Duration(*round.Stages[0].Timing.Wall) != 100*time.Millisecond {
		t.Fatalf("round trip wall = %v", round.Stages[0].Timing.Wall)
	}
	if round.Comparability.Observed.Load1 != nil {
		t.Fatalf("round trip load1 = %v", round.Comparability.Observed.Load1)
	}
}

func runSample(t *testing.T, name string, wall time.Duration, peakRSS int64, intermediate int64, comparable bool) RunSample {
	t.Helper()
	load1 := 2.0
	if !comparable {
		load1 = 8.0
	}
	load5 := 1.5
	load15 := 1.0
	power := PowerMains
	memory := MemoryNormal
	render := RenderIdle
	comparability := AssessComparability(EnvironmentSnapshot{
		Load1:          &load1,
		Load5:          &load5,
		Load15:         &load15,
		Power:          &power,
		MemoryPressure: &memory,
		Render:         &render,
	})
	lightWall := wall
	lightUser := wall / 2
	lightSystem := wall / 4
	developWall := wall + 50*time.Millisecond
	developUser := developWall / 2
	developSystem := developWall / 4
	frameIndex := int64(17)
	return RunSample{
		Provenance: Provenance{
			Command: Command{Binary: "/usr/bin/time", Args: []string{"-l", "oraclebench"}, Dir: stringPtr("/tmp"), Env: []string{"A=B", "C=D"}},
		},
		Scenario: ScenarioIdentity{
			FrameIndex:     &frameIndex,
			Width:          1920,
			Height:         1080,
			FPS:            24,
			BinarySHA256:   strings.Repeat("a", 64),
			SourceRevision: "abcdef1234567890abcdef1234567890abcdef12",
			InputDigest:    "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		},
		Comparability: comparability,
		Stages: []StageSample{
			{
				Name:    "lightshow",
				Command: Command{Binary: "/usr/bin/time", Args: []string{"-l", "lightshow"}},
				Timing: TimeReport{
					Wall:         testDurationPtr(lightWall),
					User:         testDurationPtr(lightUser),
					System:       testDurationPtr(lightSystem),
					PeakRSSBytes: testInt64Ptr(peakRSS),
				},
				Bytes: StageBytes{
					ReadBytes:         testInt64Ptr(100),
					WrittenBytes:      testInt64Ptr(200),
					IntermediateBytes: []NamedByteCount{{Name: "pfm", Bytes: testInt64Ptr(intermediate)}},
				},
			},
			{
				Name:    "develop",
				Command: Command{Binary: "/usr/bin/time", Args: []string{"-l", "develop"}},
				Timing: TimeReport{
					Wall:         testDurationPtr(developWall),
					User:         testDurationPtr(developUser),
					System:       testDurationPtr(developSystem),
					PeakRSSBytes: testInt64Ptr(peakRSS + 5),
				},
				Bytes: StageBytes{
					ReadBytes:         testInt64Ptr(300),
					WrittenBytes:      testInt64Ptr(400),
					IntermediateBytes: []NamedByteCount{{Name: "png", Bytes: testInt64Ptr(intermediate + 1)}},
				},
			},
		},
	}
}

func testDurationPtr(v time.Duration) *Duration {
	d := Duration(v)
	return &d
}

func testInt64Ptr(v int64) *int64 {
	return &v
}

func stringPtr(v string) *string {
	return &v
}
