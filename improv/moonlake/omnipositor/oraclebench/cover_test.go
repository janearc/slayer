package oraclebench

import (
	"encoding/json"
	"math"
	"reflect"
	"strings"
	"testing"
	"testing/iotest"
	"time"
)

// TestDurationUnmarshalJSON checks a duration string, a null, a number and a bad string.
func TestDurationUnmarshalJSON(t *testing.T) {
	d := Duration(time.Hour)
	if err := json.Unmarshal([]byte(`null`), &d); err != nil || d != 0 {
		t.Errorf("null: d=%v err=%v", d, err)
	}
	if err := json.Unmarshal([]byte(`"1.5s"`), &d); err != nil || time.Duration(d) != 1500*time.Millisecond {
		t.Errorf("string: d=%v err=%v", d, err)
	}
	if err := json.Unmarshal([]byte(`5`), &d); err == nil {
		t.Error("a number was accepted")
	}
	if err := json.Unmarshal([]byte(`"soon"`), &d); err == nil {
		t.Error("an unparsable duration was accepted")
	}
}

// TestParseBSDTimeRefusals feeds ParseBSDTime input that each breaks one rule, and one input with blank lines it must skip.
func TestParseBSDTimeRefusals(t *testing.T) {
	if _, err := ParseBSDTime(nil); err == nil || !strings.Contains(err.Error(), "nil reader") {
		t.Errorf("nil reader: %v", err)
	}
	if _, err := ParseBSDTime(iotest.ErrReader(iotest.ErrTimeout)); err == nil || !strings.Contains(err.Error(), "read time output") {
		t.Errorf("read failure: %v", err)
	}
	if _, err := ParseBSDTime(strings.NewReader(strings.Repeat("a", 2<<20))); err == nil || !strings.Contains(err.Error(), "read time output") {
		t.Errorf("line over the limit: %v", err)
	}
	cases := []struct{ name, text, want string }{
		{"duplicate rss", "1 maximum resident set size\n2 maximum resident set size\n", "duplicate maximum resident set size"},
		{"rss not digits", "12x maximum resident set size\n", "malformed maximum resident set size"},
		{"rss too large", "99999999999999999999 maximum resident set size\n", "malformed maximum resident set size"},
		{"rss shape", "1 2 maximum resident set size\n", "malformed maximum resident set size"},
	}
	for _, tc := range cases {
		if _, err := ParseBSDTime(strings.NewReader(tc.text)); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: %v", tc.name, err)
		}
	}
	got, err := ParseBSDTime(strings.NewReader("\n  \n0.50 real 0.20 user 0.30 sys\n\n7 maximum resident set size\n"))
	if err != nil || *got.PeakRSSBytes != 7 || time.Duration(*got.Wall) != 500*time.Millisecond {
		t.Errorf("blank lines: %+v, %v", got, err)
	}
}

// TestLooksLikeHelpers checks the text tests on empty, signed, split and plain inputs.
func TestLooksLikeHelpers(t *testing.T) {
	for text, want := range map[string]bool{"": false, "12": true, "-1": false, "1 2": false, "1.5": false} {
		if got := looksLikeUnsignedInteger(text); got != want {
			t.Errorf("looksLikeUnsignedInteger(%q) = %v, want %v", text, got, want)
		}
	}
	for text, want := range map[string]bool{
		"": false, "7": true, ".5": true, "1.25": true, "1.2.3": false, "5.": false, ".": false, "-1": false, "1e3": false, "a": false,
	} {
		if got := looksLikeDecimal(text); got != want {
			t.Errorf("looksLikeDecimal(%q) = %v, want %v", text, got, want)
		}
	}
}

// TestMediansRoundUp checks the medians of no values, one value and an even count whose mean is not whole.
func TestMediansRoundUp(t *testing.T) {
	if durationMedian(nil) != 0 || intMedian(nil) != 0 {
		t.Error("empty median is not zero")
	}
	if durationMedian([]time.Duration{5}) != 5 || intMedian([]int64{5}) != 5 {
		t.Error("single median is not the value")
	}
	if got := durationMedian([]time.Duration{1, 4}); got != 3 {
		t.Errorf("duration median of 1 and 4 = %d, want 3", got)
	}
	if got := intMedian([]int64{1, 4}); got != 3 {
		t.Errorf("int median of 1 and 4 = %d, want 3", got)
	}
}

// TestComparabilityValidateRefusesMismatches checks a status, a check and a reason list that do not follow from what was observed.
func TestComparabilityValidateRefusesMismatches(t *testing.T) {
	load := 9.0
	good := AssessComparability(EnvironmentSnapshot{Load1: &load})
	if err := good.Validate(); err != nil {
		t.Fatalf("a computed comparability was refused: %v", err)
	}
	wrongStatus := good
	wrongStatus.Status = Comparable
	wrongCheck := good
	wrongCheck.Load = CheckPass
	wrongReasons := good
	wrongReasons.Reasons = []string{"because"}
	for name, c := range map[string]Comparability{"status": wrongStatus, "check": wrongCheck, "reasons": wrongReasons} {
		if err := c.Validate(); err == nil || !strings.Contains(err.Error(), name) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// TestReasonsNameWhatWentWrong checks the wording for a load that is not a number and for each missing reading.
func TestReasonsNameWhatWentWrong(t *testing.T) {
	nan := math.NaN()
	got := AssessComparability(EnvironmentSnapshot{Load1: &nan})
	want := []string{"load invalid", "power unavailable", "memory pressure unavailable", "render activity unavailable"}
	if !reflect.DeepEqual(got.Reasons, want) || got.Status != NonComparable {
		t.Errorf("reasons=%v status=%v", got.Reasons, got.Status)
	}
	high := 6.0
	battery, warning, active := PowerBattery, MemoryWarning, RenderActive
	got = AssessComparability(EnvironmentSnapshot{Load1: &high, Power: &battery, MemoryPressure: &warning, Render: &active})
	want = []string{"load 6 exceeds limit", "power battery", "memory pressure warning", "render active"}
	if !reflect.DeepEqual(got.Reasons, want) {
		t.Errorf("reasons=%v, want %v", got.Reasons, want)
	}
}

// TestAggregateRunsRefusals checks no runs, an invalid run and each kind of layout mismatch are refused.
func TestAggregateRunsRefusals(t *testing.T) {
	if _, err := AggregateRuns(nil); err == nil || !strings.Contains(err.Error(), "no runs") {
		t.Errorf("no runs: %v", err)
	}
	cases := []struct {
		name   string
		mutate func(*RunSample)
		want   string
	}{
		{"invalid run", func(r *RunSample) { r.Stages = nil }, "run 1: stages are required"},
		{"stage count", func(r *RunSample) { r.Stages = r.Stages[:1] }, "run 1: stage count differs"},
		{"stage command", func(r *RunSample) { r.Stages[1].Command.Args = []string{"other"} }, "run 1 stage 1: command mismatch"},
		{"intermediate count", func(r *RunSample) { r.Stages[1].Bytes.IntermediateBytes = nil }, "run 1 stage 1: intermediate count mismatch"},
		{"intermediate name", func(r *RunSample) { r.Stages[1].Bytes.IntermediateBytes[0].Name = "jpg" }, "run 1 stage 1: intermediate name mismatch"},
	}
	for _, tc := range cases {
		first := runSample(t, "run", 100*time.Millisecond, 10, 7, true)
		second := runSample(t, "run", 100*time.Millisecond, 10, 7, true)
		tc.mutate(&second)
		if _, err := AggregateRuns([]RunSample{first, second}); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: %v", tc.name, err)
		}
	}
}

// TestAggregateRunsWithNothingComparable checks a report built from runs that are all noncomparable has no stages and says why.
func TestAggregateRunsWithNothingComparable(t *testing.T) {
	runs := []RunSample{
		runSample(t, "run", 100*time.Millisecond, 10, 7, false),
		runSample(t, "run", 200*time.Millisecond, 20, 7, false),
	}
	got, err := AggregateRuns(runs)
	if err != nil {
		t.Fatal(err)
	}
	if got.ComparableRuns != 0 || len(got.Stages) != 0 || got.Comparability.Status != NonComparable ||
		!reflect.DeepEqual(got.Comparability.Reasons, []string{"no comparable runs"}) {
		t.Errorf("report=%+v", got)
	}
	if len(got.Runs) != 2 {
		t.Errorf("runs kept: %d, want 2", len(got.Runs))
	}
}

// TestAggregateRunsCountsMissingFiguresAsUnknown checks a figure no run reported is unknown in every sample and has no min, median or max.
func TestAggregateRunsCountsMissingFiguresAsUnknown(t *testing.T) {
	var runs []RunSample
	for range 3 {
		r := runSample(t, "run", 100*time.Millisecond, 10, 7, true)
		r.Stages[0].Timing = TimeReport{}
		r.Stages[0].Bytes.ReadBytes = nil
		runs = append(runs, r)
	}
	got, err := AggregateRuns(runs)
	if err != nil {
		t.Fatal(err)
	}
	if got.Comparability.Status != Comparable {
		t.Fatalf("status %v", got.Comparability.Status)
	}
	stage := got.Stages[0]
	if stage.Wall.Observed != 0 || stage.Wall.Unknown != 3 || stage.Wall.Min != nil || stage.Wall.Median != nil || stage.Wall.Max != nil {
		t.Errorf("wall %+v", stage.Wall)
	}
	if stage.PeakRSS.Observed != 0 || stage.PeakRSS.Unknown != 3 || stage.PeakRSS.Median != nil {
		t.Errorf("peak %+v", stage.PeakRSS)
	}
	if stage.ReadBytes.Unknown != 3 || stage.WrittenBytes.Observed != 3 {
		t.Errorf("read %+v written %+v", stage.ReadBytes, stage.WrittenBytes)
	}
}

// TestTimeReportValidate checks each figure that is negative is named, and that missing figures pass.
func TestTimeReportValidate(t *testing.T) {
	neg := Duration(-1)
	negInt := int64(-1)
	cases := []struct {
		name string
		t    TimeReport
		want string
	}{
		{"wall", TimeReport{Wall: &neg}, "wall is negative"},
		{"user", TimeReport{User: &neg}, "user is negative"},
		{"system", TimeReport{System: &neg}, "system is negative"},
		{"peak", TimeReport{PeakRSSBytes: &negInt}, "peak rss is negative"},
	}
	for _, tc := range cases {
		if err := tc.t.Validate(); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: %v", tc.name, err)
		}
	}
	if err := (TimeReport{}).Validate(); err != nil {
		t.Errorf("empty report: %v", err)
	}
}

// TestStageBytesValidate checks each count or name that is bad is named.
func TestStageBytesValidate(t *testing.T) {
	neg := int64(-1)
	cases := []struct {
		name string
		b    StageBytes
		want string
	}{
		{"read", StageBytes{ReadBytes: &neg}, "read bytes is negative"},
		{"written", StageBytes{WrittenBytes: &neg}, "written bytes is negative"},
		{"unnamed", StageBytes{IntermediateBytes: []NamedByteCount{{}}}, "intermediate 0 name is empty"},
		{"negative intermediate", StageBytes{IntermediateBytes: []NamedByteCount{{Name: "a", Bytes: &neg}}}, "intermediate 0 bytes is negative"},
	}
	for _, tc := range cases {
		if err := tc.b.Validate(); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: %v", tc.name, err)
		}
	}
}

// TestStageAndRunValidateNameTheFailingPart checks the stage and run checks say which part failed.
func TestStageAndRunValidateNameTheFailingPart(t *testing.T) {
	neg := Duration(-1)
	negInt := int64(-1)
	good := runSample(t, "run", 100*time.Millisecond, 10, 7, true)
	stageCases := []struct {
		name   string
		mutate func(*StageSample)
		want   string
	}{
		{"name", func(s *StageSample) { s.Name = "" }, "stage name is required"},
		{"command", func(s *StageSample) { s.Command.Binary = "" }, "command: command binary is required"},
		{"timing", func(s *StageSample) { s.Timing.Wall = &neg }, "timing: wall is negative"},
		{"bytes", func(s *StageSample) { s.Bytes.ReadBytes = &negInt }, "bytes: read bytes is negative"},
	}
	for _, tc := range stageCases {
		stage := good.Stages[0]
		stage.Bytes.IntermediateBytes = append([]NamedByteCount(nil), stage.Bytes.IntermediateBytes...)
		tc.mutate(&stage)
		if err := stage.Validate(); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("stage %s: %v", tc.name, err)
		}
	}
	runCases := []struct {
		name   string
		mutate func(*RunSample)
		want   string
	}{
		{"provenance", func(r *RunSample) { r.Provenance.Command.Binary = "" }, "provenance: command binary is required"},
		{"stages", func(r *RunSample) { r.Stages = nil }, "stages are required"},
		{"second stage", func(r *RunSample) {
			r.Stages = append([]StageSample(nil), r.Stages...)
			r.Stages[1].Name = ""
		}, "stage 1: stage name is required"},
	}
	for _, tc := range runCases {
		run := good
		tc.mutate(&run)
		if err := run.Validate(); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("run %s: %v", tc.name, err)
		}
	}
}
