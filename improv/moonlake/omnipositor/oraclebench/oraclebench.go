package oraclebench

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/big"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Duration time.Duration

// MarshalJSON writes the duration as a JSON string in Go duration syntax,
// such as "1.5s", not as a number of nanoseconds.
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

// UnmarshalJSON reads a JSON string in Go duration syntax. A JSON null gives
// zero. Any other kind of value, or text time.ParseDuration refuses, is an
// error.
func (d *Duration) UnmarshalJSON(data []byte) error {
	text := strings.TrimSpace(string(data))
	if text == "null" {
		*d = 0
		return nil
	}
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return err
	}
	*d = Duration(parsed)
	return nil
}

type Command struct {
	Binary string   `json:"binary"`
	Args   []string `json:"args"`
	Dir    *string  `json:"dir"`
	Env    []string `json:"env"`
}

type Provenance struct {
	Command Command `json:"command"`
}

type ScenarioIdentity struct {
	FrameIndex     *int64  `json:"frame_index"`
	Width          int     `json:"width"`
	Height         int     `json:"height"`
	FPS            float64 `json:"fps"`
	BinarySHA256   string  `json:"binary_sha256"`
	SourceRevision string  `json:"source_revision"`
	InputDigest    string  `json:"input_digest"`
}

type TimeReport struct {
	Wall         *Duration `json:"wall"`
	User         *Duration `json:"user"`
	System       *Duration `json:"system"`
	PeakRSSBytes *int64    `json:"peak_rss_bytes"`
}

type NamedByteCount struct {
	Name  string `json:"name"`
	Bytes *int64 `json:"bytes"`
}

type StageBytes struct {
	ReadBytes         *int64           `json:"read_bytes"`
	WrittenBytes      *int64           `json:"written_bytes"`
	IntermediateBytes []NamedByteCount `json:"intermediate_bytes"`
}

type StageSample struct {
	Name    string     `json:"name"`
	Command Command    `json:"command"`
	Timing  TimeReport `json:"timing"`
	Bytes   StageBytes `json:"bytes"`
}

type RunSample struct {
	Provenance    Provenance       `json:"provenance"`
	Scenario      ScenarioIdentity `json:"scenario"`
	Comparability Comparability    `json:"comparability"`
	Stages        []StageSample    `json:"stages"`
}

type Summary struct {
	Observed int       `json:"observed"`
	Unknown  int       `json:"unknown"`
	Min      *Duration `json:"min"`
	Median   *Duration `json:"median"`
	Max      *Duration `json:"max"`
}

type IntSummary struct {
	Observed int    `json:"observed"`
	Unknown  int    `json:"unknown"`
	Min      *int64 `json:"min"`
	Median   *int64 `json:"median"`
	Max      *int64 `json:"max"`
}

type NamedByteSummary struct {
	Name  string     `json:"name"`
	Bytes IntSummary `json:"bytes"`
}

type StageAggregate struct {
	Name              string             `json:"name"`
	Command           Command            `json:"command"`
	Wall              Summary            `json:"wall"`
	User              Summary            `json:"user"`
	System            Summary            `json:"system"`
	PeakRSS           IntSummary         `json:"peak_rss_bytes"`
	ReadBytes         IntSummary         `json:"read_bytes"`
	WrittenBytes      IntSummary         `json:"written_bytes"`
	IntermediateBytes []NamedByteSummary `json:"intermediate_bytes"`
}

type AggregateReport struct {
	Runs           []RunSample      `json:"runs"`
	Provenance     Provenance       `json:"provenance"`
	Scenario       ScenarioIdentity `json:"scenario"`
	ComparableRuns int              `json:"comparable_runs"`
	Comparability  Comparability    `json:"comparability"`
	Stages         []StageAggregate `json:"stages"`
}

type CheckState string

const (
	CheckUnknown CheckState = "unknown"
	CheckPass    CheckState = "pass"
	CheckFail    CheckState = "fail"
)

type ComparabilityStatus string

const (
	Comparable    ComparabilityStatus = "comparable"
	NonComparable ComparabilityStatus = "noncomparable"
)

type PowerState string

const (
	PowerMains   PowerState = "mains"
	PowerBattery PowerState = "battery"
)

type MemoryPressureState string

const (
	MemoryNormal   MemoryPressureState = "normal"
	MemoryWarning  MemoryPressureState = "warning"
	MemoryCritical MemoryPressureState = "critical"
)

type RenderActivity string

const (
	RenderIdle   RenderActivity = "idle"
	RenderActive RenderActivity = "active"
)

type EnvironmentSnapshot struct {
	Load1          *float64             `json:"load_1"`
	Load5          *float64             `json:"load_5"`
	Load15         *float64             `json:"load_15"`
	Power          *PowerState          `json:"power"`
	MemoryPressure *MemoryPressureState `json:"memory_pressure"`
	Render         *RenderActivity      `json:"render"`
}

type Comparability struct {
	Status         ComparabilityStatus `json:"status"`
	Load           CheckState          `json:"load"`
	Power          CheckState          `json:"power"`
	MemoryPressure CheckState          `json:"memory_pressure"`
	Render         CheckState          `json:"render"`
	Observed       EnvironmentSnapshot `json:"observed"`
	Reasons        []string            `json:"reasons"`
}

// ParseBSDTime reads the output of /usr/bin/time -l and returns the wall, user
// and system times and the peak resident set size. Lines it does not know are
// skipped. It refuses a nil reader, a malformed or repeated figure, and output
// that lacks any of the four.
func ParseBSDTime(r io.Reader) (TimeReport, error) {
	if r == nil {
		return TimeReport{}, fmt.Errorf("oraclebench: nil reader")
	}
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 4096), 1<<20)
	var report TimeReport
	seen := map[string]bool{}
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 0 {
			continue
		}
		if err := parseBSDTimeLine(fields, &report, seen); err != nil {
			return TimeReport{}, err
		}
	}
	if err := scanner.Err(); err != nil {
		return TimeReport{}, fmt.Errorf("oraclebench: read time output: %w", err)
	}
	if report.Wall == nil || report.User == nil || report.System == nil || report.PeakRSSBytes == nil {
		return TimeReport{}, fmt.Errorf("oraclebench: incomplete time output")
	}
	return report, nil
}

// parseBSDTimeLine sends one split line of time output to the resident set
// size parser or to the time parser, by which words it holds. A line with
// neither is ignored.
func parseBSDTimeLine(fields []string, report *TimeReport, seen map[string]bool) error {
	if hasPhrase(fields, []string{"maximum", "resident", "set", "size"}) {
		return parseRSSLine(fields, report, seen)
	}
	if !containsTimeLabel(fields) {
		return nil
	}
	return parseTimeLine(fields, report, seen)
}

// parseRSSLine reads a line of exactly the form "N maximum resident set size"
// into PeakRSSBytes, taking N as given. It refuses any other shape, a second
// such line, and a value int64 cannot hold.
func parseRSSLine(fields []string, report *TimeReport, seen map[string]bool) error {
	if len(fields) != 5 || fields[1] != "maximum" || fields[2] != "resident" || fields[3] != "set" || fields[4] != "size" {
		return fmt.Errorf("oraclebench: malformed maximum resident set size")
	}
	if seen["rss"] {
		return fmt.Errorf("oraclebench: duplicate maximum resident set size")
	}
	if !looksLikeUnsignedInteger(fields[0]) {
		return fmt.Errorf("oraclebench: malformed maximum resident set size")
	}
	value, err := strconv.ParseInt(fields[0], 10, 64)
	if err != nil {
		return fmt.Errorf("oraclebench: malformed maximum resident set size")
	}
	if value < 0 {
		return fmt.Errorf("oraclebench: negative maximum resident set size")
	}
	report.PeakRSSBytes = int64Ptr(value)
	seen["rss"] = true
	return nil
}

// parseTimeLine reads a line of value and label pairs, such as "1.5 real 1.0
// user 0.2 sys", into Wall, User and System. It refuses an odd field count, a
// label other than real, user or sys, a repeated label and a bad number.
func parseTimeLine(fields []string, report *TimeReport, seen map[string]bool) error {
	if len(fields)%2 != 0 {
		return fmt.Errorf("oraclebench: malformed time line")
	}
	for i := 0; i < len(fields); i += 2 {
		valueText := fields[i]
		label := fields[i+1]
		if !isTimeLabel(label) {
			return fmt.Errorf("oraclebench: malformed time line")
		}
		if seen[label] {
			return fmt.Errorf("oraclebench: duplicate %s", label)
		}
		duration, err := parseSeconds(valueText)
		if err != nil {
			return fmt.Errorf("oraclebench: %s: %w", label, err)
		}
		switch label {
		case "real":
			report.Wall = durationPtr(time.Duration(duration))
		case "user":
			report.User = durationPtr(time.Duration(duration))
		case "sys":
			report.System = durationPtr(time.Duration(duration))
		}
		seen[label] = true
	}
	return nil
}

// isTimeLabel reports whether text is one of the labels real, user or sys.
func isTimeLabel(text string) bool {
	switch text {
	case "real", "user", "sys":
		return true
	default:
		return false
	}
}

// containsTimeLabel reports whether any field is real, user or sys.
func containsTimeLabel(fields []string) bool {
	for _, field := range fields {
		if isTimeLabel(field) {
			return true
		}
	}
	return false
}

// hasPhrase reports whether phrase appears in fields as consecutive entries,
// in order, starting anywhere.
func hasPhrase(fields, phrase []string) bool {
	for i := 0; i+len(phrase) <= len(fields); i++ {
		match := true
		for j := range phrase {
			if fields[i+j] != phrase[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// looksLikeUnsignedInteger reports whether text is one or more ASCII digits
// and nothing else, so no sign and no spaces.
func looksLikeUnsignedInteger(text string) bool {
	if text == "" {
		return false
	}
	for _, r := range text {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// looksLikeLowerHexDigest reports whether text is exactly size characters,
// each a digit or a lowercase letter from a to f.
func looksLikeLowerHexDigest(text string, size int) bool {
	if len(text) != size {
		return false
	}
	for _, r := range text {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'a' && r <= 'f':
		default:
			return false
		}
	}
	return true
}

// looksLikeGitRevision reports whether text is a full revision hash in
// lowercase hex, either 40 characters or 64.
func looksLikeGitRevision(text string) bool {
	return looksLikeLowerHexDigest(text, 40) || looksLikeLowerHexDigest(text, 64)
}

// looksLikeDecimal reports whether text is a plain decimal number: digits with
// at most one point, and a digit after the point if there is one. No sign and
// no exponent. A leading point, as in ".5", is allowed.
func looksLikeDecimal(text string) bool {
	if text == "" {
		return false
	}
	digits := 0
	dot := false
	fracDigits := 0
	for _, r := range text {
		switch {
		case r >= '0' && r <= '9':
			digits++
			if dot {
				fracDigits++
			}
		case r == '.':
			if dot {
				return false
			}
			dot = true
		default:
			return false
		}
	}
	if digits == 0 {
		return false
	}
	if dot && fracDigits == 0 {
		return false
	}
	return true
}

// parseSeconds turns decimal seconds text into a Duration. It refuses text
// looksLikeDecimal refuses, a fraction finer than one nanosecond, and a value
// too large for int64 nanoseconds.
func parseSeconds(text string) (Duration, error) {
	if !looksLikeDecimal(text) {
		return 0, fmt.Errorf("invalid duration %q", text)
	}
	value := new(big.Rat)
	if _, ok := value.SetString(text); !ok || value.Sign() < 0 {
		return 0, fmt.Errorf("invalid duration %q", text)
	}
	value.Mul(value, big.NewRat(int64(time.Second), 1))
	if !value.IsInt() {
		return 0, fmt.Errorf("invalid duration %q", text)
	}
	nanos := value.Num()
	if !nanos.IsInt64() {
		return 0, fmt.Errorf("invalid duration %q", text)
	}
	return Duration(time.Duration(nanos.Int64())), nil
}

// AssessComparability checks a snapshot of the machine against four limits:
// one-minute load of 4 or less, mains power, normal memory pressure and no
// render running. A reading that is missing counts as unknown, not as a pass.
// The result is Comparable only if all four pass. Otherwise it is
// NonComparable with a reason for each check that did not pass.
func AssessComparability(snapshot EnvironmentSnapshot) Comparability {
	result := Comparability{Observed: snapshot}
	result.Load = assessLoad(snapshot.Load1)
	result.Power = assessPower(snapshot.Power)
	result.MemoryPressure = assessMemoryPressure(snapshot.MemoryPressure)
	result.Render = assessRender(snapshot.Render)
	if result.Load == CheckPass && result.Power == CheckPass && result.MemoryPressure == CheckPass && result.Render == CheckPass {
		result.Status = Comparable
		return result
	}
	result.Status = NonComparable
	if result.Load != CheckPass {
		result.Reasons = append(result.Reasons, reasonForLoad(snapshot.Load1))
	}
	if result.Power != CheckPass {
		result.Reasons = append(result.Reasons, reasonForPower(snapshot.Power))
	}
	if result.MemoryPressure != CheckPass {
		result.Reasons = append(result.Reasons, reasonForMemoryPressure(snapshot.MemoryPressure))
	}
	if result.Render != CheckPass {
		result.Reasons = append(result.Reasons, reasonForRender(snapshot.Render))
	}
	return result
}

// Validate recomputes the comparability from c.Observed and refuses c if its
// status, its four checks or its reasons differ from that.
func (c Comparability) Validate() error {
	expected := AssessComparability(c.Observed)
	if c.Status != expected.Status {
		return fmt.Errorf("comparability status does not match observed state")
	}
	if c.Load != expected.Load || c.Power != expected.Power || c.MemoryPressure != expected.MemoryPressure || c.Render != expected.Render {
		return fmt.Errorf("comparability checks do not match observed state")
	}
	if !reflect.DeepEqual(c.Reasons, expected.Reasons) {
		return fmt.Errorf("comparability reasons do not match observed state")
	}
	return nil
}

// assessLoad is unknown for a missing, NaN or infinite load, a fail for a load
// above 4, and a pass otherwise, so a load of exactly 4 passes.
func assessLoad(load *float64) CheckState {
	if load == nil || math.IsNaN(*load) || math.IsInf(*load, 0) {
		return CheckUnknown
	}
	if *load > 4 {
		return CheckFail
	}
	return CheckPass
}

// assessPower is unknown if power is missing, a fail for any state but mains,
// and a pass for mains.
func assessPower(power *PowerState) CheckState {
	if power == nil {
		return CheckUnknown
	}
	if *power != PowerMains {
		return CheckFail
	}
	return CheckPass
}

// assessMemoryPressure is unknown if the state is missing, a fail for any
// state but normal, and a pass for normal.
func assessMemoryPressure(memoryPressure *MemoryPressureState) CheckState {
	if memoryPressure == nil {
		return CheckUnknown
	}
	if *memoryPressure != MemoryNormal {
		return CheckFail
	}
	return CheckPass
}

// assessRender is unknown if the activity is missing, a fail for any activity
// but idle, and a pass for idle.
func assessRender(render *RenderActivity) CheckState {
	if render == nil {
		return CheckUnknown
	}
	if *render != RenderIdle {
		return CheckFail
	}
	return CheckPass
}

// reasonForLoad words why the load check did not pass: unavailable if missing,
// invalid if NaN or infinite, and otherwise that the load exceeds the limit.
func reasonForLoad(load *float64) string {
	if load == nil {
		return "load unavailable"
	}
	if math.IsNaN(*load) || math.IsInf(*load, 0) {
		return "load invalid"
	}
	return fmt.Sprintf("load %.3g exceeds limit", *load)
}

// reasonForPower words why the power check did not pass: unavailable if
// missing, otherwise the state it was in.
func reasonForPower(power *PowerState) string {
	if power == nil {
		return "power unavailable"
	}
	return fmt.Sprintf("power %s", *power)
}

// reasonForMemoryPressure words why the memory pressure check did not pass:
// unavailable if missing, otherwise the state it was in.
func reasonForMemoryPressure(memoryPressure *MemoryPressureState) string {
	if memoryPressure == nil {
		return "memory pressure unavailable"
	}
	return fmt.Sprintf("memory pressure %s", *memoryPressure)
}

// reasonForRender words why the render check did not pass: unavailable if
// missing, otherwise the activity it was in.
func reasonForRender(render *RenderActivity) string {
	if render == nil {
		return "render activity unavailable"
	}
	return fmt.Sprintf("render %s", *render)
}

// AggregateRuns summarises runs into one report. It refuses an empty list, any
// run that fails Validate, and runs that differ in scenario, command or stage
// layout. Only the comparable runs are summarised, and the report is
// Comparable only if there are at least 3 of them. With none, Stages is left
// empty. The input slice is copied, not changed.
func AggregateRuns(runs []RunSample) (AggregateReport, error) {
	if len(runs) == 0 {
		return AggregateReport{}, fmt.Errorf("oraclebench: no runs")
	}
	copied := append([]RunSample(nil), runs...)
	for i := range copied {
		if err := copied[i].Validate(); err != nil {
			return AggregateReport{}, fmt.Errorf("oraclebench: run %d: %w", i, err)
		}
	}
	if err := validateScenarioIdentity(copied); err != nil {
		return AggregateReport{}, err
	}
	comparable := make([]RunSample, 0, len(copied))
	for _, run := range copied {
		if run.Comparability.Status == Comparable {
			comparable = append(comparable, run)
		}
	}
	report := AggregateReport{Runs: copied, Provenance: copied[0].Provenance, Scenario: copied[0].Scenario, ComparableRuns: len(comparable), Comparability: compareRuns(comparable)}
	if len(comparable) == 0 {
		return report, nil
	}
	stages, err := aggregateStages(comparable)
	if err != nil {
		return AggregateReport{}, err
	}
	report.Stages = stages
	return report, nil
}

// validateScenarioIdentity refuses the first run that differs from runs[0] in
// scenario, provenance command, stage count, stage names, stage commands or
// intermediate byte names. The caller must pass at least one run.
func validateScenarioIdentity(runs []RunSample) error {
	baseline := runs[0]
	for i := 1; i < len(runs); i++ {
		run := runs[i]
		if !reflect.DeepEqual(run.Scenario, baseline.Scenario) {
			return fmt.Errorf("oraclebench: run %d: scenario mismatch", i)
		}
		if run.Provenance.Command.Binary != baseline.Provenance.Command.Binary {
			return fmt.Errorf("oraclebench: run %d: provenance command binary mismatch", i)
		}
		if !reflect.DeepEqual(run.Provenance.Command.Args, baseline.Provenance.Command.Args) {
			return fmt.Errorf("oraclebench: run %d: provenance command args mismatch", i)
		}
		if !reflect.DeepEqual(run.Provenance.Command.Dir, baseline.Provenance.Command.Dir) {
			return fmt.Errorf("oraclebench: run %d: provenance command dir mismatch", i)
		}
		if !reflect.DeepEqual(run.Provenance.Command.Env, baseline.Provenance.Command.Env) {
			return fmt.Errorf("oraclebench: run %d: provenance command env mismatch", i)
		}
		if len(run.Stages) != len(baseline.Stages) {
			return fmt.Errorf("oraclebench: run %d: stage count differs", i)
		}
		for stageIndex, stage := range baseline.Stages {
			current := run.Stages[stageIndex]
			if current.Name != stage.Name {
				return fmt.Errorf("oraclebench: run %d stage %d: name mismatch", i, stageIndex)
			}
			if !reflect.DeepEqual(current.Command, stage.Command) {
				return fmt.Errorf("oraclebench: run %d stage %d: command mismatch", i, stageIndex)
			}
			if len(current.Bytes.IntermediateBytes) != len(stage.Bytes.IntermediateBytes) {
				return fmt.Errorf("oraclebench: run %d stage %d: intermediate count mismatch", i, stageIndex)
			}
			for intermediateIndex, entry := range stage.Bytes.IntermediateBytes {
				if current.Bytes.IntermediateBytes[intermediateIndex].Name != entry.Name {
					return fmt.Errorf("oraclebench: run %d stage %d: intermediate name mismatch", i, stageIndex)
				}
			}
		}
	}
	return nil
}

// compareRuns judges a set of runs that were already found comparable. It
// is NonComparable with a reason if there are fewer than 3, and Comparable
// otherwise.
func compareRuns(runs []RunSample) Comparability {
	if len(runs) == 0 {
		return Comparability{Status: NonComparable, Reasons: []string{"no comparable runs"}}
	}
	if len(runs) < 3 {
		return Comparability{Status: NonComparable, Reasons: []string{fmt.Sprintf("need at least 3 comparable runs, got %d", len(runs))}}
	}
	return Comparability{Status: Comparable}
}

// aggregateStages summarises each stage of runs[0] across all the runs: wall,
// user and system time, peak size, bytes read and written, and each
// intermediate byte count. A value a run did not report is counted as
// unknown. It refuses runs whose stages do not line up. At least one run is
// needed.
func aggregateStages(runs []RunSample) ([]StageAggregate, error) {
	baseline := runs[0]
	stages := make([]StageAggregate, len(baseline.Stages))
	for index, stage := range baseline.Stages {
		stages[index].Name = stage.Name
		stages[index].Command = stage.Command
		wallValues := make([]time.Duration, 0, len(runs))
		userValues := make([]time.Duration, 0, len(runs))
		systemValues := make([]time.Duration, 0, len(runs))
		peakValues := make([]int64, 0, len(runs))
		readValues := make([]int64, 0, len(runs))
		writtenValues := make([]int64, 0, len(runs))
		intermediateValues := make([][]int64, len(stage.Bytes.IntermediateBytes))
		for i := range intermediateValues {
			intermediateValues[i] = make([]int64, 0, len(runs))
			stages[index].IntermediateBytes = append(stages[index].IntermediateBytes, NamedByteSummary{Name: stage.Bytes.IntermediateBytes[i].Name})
		}
		for runIndex, run := range runs {
			if len(run.Stages) != len(baseline.Stages) {
				return nil, fmt.Errorf("oraclebench: run %d: stage count differs", runIndex)
			}
			current := run.Stages[index]
			if current.Name != stage.Name {
				return nil, fmt.Errorf("oraclebench: run %d stage %d: name mismatch", runIndex, index)
			}
			if !reflect.DeepEqual(current.Command, stage.Command) {
				return nil, fmt.Errorf("oraclebench: run %d stage %d: command mismatch", runIndex, index)
			}
			if len(current.Bytes.IntermediateBytes) != len(stage.Bytes.IntermediateBytes) {
				return nil, fmt.Errorf("oraclebench: run %d stage %d: intermediate count mismatch", runIndex, index)
			}
			for i, entry := range current.Bytes.IntermediateBytes {
				if entry.Name != stage.Bytes.IntermediateBytes[i].Name {
					return nil, fmt.Errorf("oraclebench: run %d stage %d: intermediate name mismatch", runIndex, index)
				}
				if entry.Bytes != nil {
					intermediateValues[i] = append(intermediateValues[i], *entry.Bytes)
				}
			}
			if current.Timing.Wall != nil {
				wallValues = append(wallValues, time.Duration(*current.Timing.Wall))
			}
			if current.Timing.User != nil {
				userValues = append(userValues, time.Duration(*current.Timing.User))
			}
			if current.Timing.System != nil {
				systemValues = append(systemValues, time.Duration(*current.Timing.System))
			}
			if current.Timing.PeakRSSBytes != nil {
				peakValues = append(peakValues, *current.Timing.PeakRSSBytes)
			}
			if current.Bytes.ReadBytes != nil {
				readValues = append(readValues, *current.Bytes.ReadBytes)
			}
			if current.Bytes.WrittenBytes != nil {
				writtenValues = append(writtenValues, *current.Bytes.WrittenBytes)
			}
		}
		stages[index].Wall = durationSummary(wallValues, len(runs))
		stages[index].User = durationSummary(userValues, len(runs))
		stages[index].System = durationSummary(systemValues, len(runs))
		stages[index].PeakRSS = intSummary(peakValues, len(runs))
		stages[index].ReadBytes = intSummary(readValues, len(runs))
		stages[index].WrittenBytes = intSummary(writtenValues, len(runs))
		for i := range stages[index].IntermediateBytes {
			stages[index].IntermediateBytes[i].Bytes = intSummary(intermediateValues[i], len(runs))
		}
	}
	return stages, nil
}

// durationSummary counts the values seen and the unknown ones, sampleCount
// minus those seen, and gives their min, median and max. It sorts values in
// place. With no values the three are left unset.
func durationSummary(values []time.Duration, sampleCount int) Summary {
	result := Summary{Observed: len(values), Unknown: sampleCount - len(values)}
	if len(values) == 0 {
		return result
	}
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	result.Min = durationPtr(values[0])
	result.Max = durationPtr(values[len(values)-1])
	result.Median = durationPtr(durationMedian(values))
	return result
}

// intSummary counts the values seen and the unknown ones, sampleCount minus
// those seen, and gives their min, median and max. It sorts values in place.
// With no values the three are left unset.
func intSummary(values []int64, sampleCount int) IntSummary {
	result := IntSummary{Observed: len(values), Unknown: sampleCount - len(values)}
	if len(values) == 0 {
		return result
	}
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	result.Min = int64Ptr(values[0])
	result.Max = int64Ptr(values[len(values)-1])
	result.Median = int64Ptr(intMedian(values))
	return result
}

// durationMedian is the middle of values, which must already be sorted
// ascending. For an even count it is the mean of the middle two, rounded up
// to a whole nanosecond. It is 0 for an empty list.
func durationMedian(values []time.Duration) time.Duration {
	if len(values) == 0 {
		return 0
	}
	if len(values) == 1 {
		return values[0]
	}
	mid := len(values) / 2
	if len(values)%2 == 1 {
		return values[mid]
	}
	lower := values[mid-1]
	upper := values[mid]
	delta := upper - lower
	return lower + delta/2 + delta%2
}

// intMedian is the middle of values, which must already be sorted ascending.
// For an even count it is the mean of the middle two, rounded up to a whole
// number. It is 0 for an empty list.
func intMedian(values []int64) int64 {
	if len(values) == 0 {
		return 0
	}
	if len(values) == 1 {
		return values[0]
	}
	mid := len(values) / 2
	if len(values)%2 == 1 {
		return values[mid]
	}
	lower := values[mid-1]
	upper := values[mid]
	delta := upper - lower
	return lower + delta/2 + delta%2
}

// durationPtr returns a pointer to a Duration holding v.
func durationPtr(v time.Duration) *Duration {
	d := Duration(v)
	return &d
}

// int64Ptr returns a pointer to a copy of v.
func int64Ptr(v int64) *int64 {
	return &v
}

// Validate refuses a command with no binary, a NUL in the binary, directory,
// any argument or any environment entry, a directory that is set but empty,
// an empty environment entry, an entry that is not KEY=value with a key, and
// a key that appears twice.
func (c Command) Validate() error {
	if c.Binary == "" {
		return fmt.Errorf("command binary is required")
	}
	if strings.ContainsRune(c.Binary, '\x00') {
		return fmt.Errorf("command binary contains NUL")
	}
	if c.Dir != nil && *c.Dir == "" {
		return fmt.Errorf("command dir is empty")
	}
	if c.Dir != nil && strings.ContainsRune(*c.Dir, '\x00') {
		return fmt.Errorf("command dir contains NUL")
	}
	for i, arg := range c.Args {
		if strings.ContainsRune(arg, '\x00') {
			return fmt.Errorf("command arg %d contains NUL", i)
		}
	}
	seenEnv := make(map[string]struct{}, len(c.Env))
	for i, env := range c.Env {
		if env == "" {
			return fmt.Errorf("command env %d is empty", i)
		}
		if strings.ContainsRune(env, '\x00') {
			return fmt.Errorf("command env %d contains NUL", i)
		}
		key, _, ok := strings.Cut(env, "=")
		if !ok || key == "" {
			return fmt.Errorf("command env %d has invalid key", i)
		}
		if _, ok := seenEnv[key]; ok {
			return fmt.Errorf("command env %d duplicates key %q", i, key)
		}
		seenEnv[key] = struct{}{}
	}
	return nil
}

// Validate refuses a scenario with no frame index or a negative one, a width
// or height that is not positive, an fps that is not finite and positive, a
// binary hash that is not 64 lowercase hex characters, a source revision
// that is not 40 or 64, and an input digest that is not 64.
func (s ScenarioIdentity) Validate() error {
	if s.FrameIndex == nil {
		return fmt.Errorf("frame index is required")
	}
	if *s.FrameIndex < 0 {
		return fmt.Errorf("frame index is negative")
	}
	if s.Width <= 0 {
		return fmt.Errorf("width must be positive")
	}
	if s.Height <= 0 {
		return fmt.Errorf("height must be positive")
	}
	if math.IsNaN(s.FPS) || math.IsInf(s.FPS, 0) || s.FPS <= 0 {
		return fmt.Errorf("fps must be finite and positive")
	}
	if !looksLikeLowerHexDigest(s.BinarySHA256, 64) {
		return fmt.Errorf("binary sha256 is invalid")
	}
	if !looksLikeGitRevision(s.SourceRevision) {
		return fmt.Errorf("source revision is invalid")
	}
	if !looksLikeLowerHexDigest(s.InputDigest, 64) {
		return fmt.Errorf("input digest is invalid")
	}
	return nil
}

// Validate refuses a time report with any figure that is present and negative.
// A missing figure is allowed.
func (t TimeReport) Validate() error {
	return validateTimeReport(t)
}

// validateTimeReport refuses a wall, user, system or peak size that is
// present and negative. It is shared by TimeReport.Validate and
// StageSample.Validate.
func validateTimeReport(t TimeReport) error {
	if t.Wall != nil && *t.Wall < 0 {
		return fmt.Errorf("wall is negative")
	}
	if t.User != nil && *t.User < 0 {
		return fmt.Errorf("user is negative")
	}
	if t.System != nil && *t.System < 0 {
		return fmt.Errorf("system is negative")
	}
	if t.PeakRSSBytes != nil && *t.PeakRSSBytes < 0 {
		return fmt.Errorf("peak rss is negative")
	}
	return nil
}

// Validate refuses a negative read or written count, an intermediate entry
// with an empty name, and an intermediate entry with a negative count.
// Missing counts are allowed.
func (s StageBytes) Validate() error {
	if s.ReadBytes != nil && *s.ReadBytes < 0 {
		return fmt.Errorf("read bytes is negative")
	}
	if s.WrittenBytes != nil && *s.WrittenBytes < 0 {
		return fmt.Errorf("written bytes is negative")
	}
	for i, entry := range s.IntermediateBytes {
		if entry.Name == "" {
			return fmt.Errorf("intermediate %d name is empty", i)
		}
		if entry.Bytes != nil && *entry.Bytes < 0 {
			return fmt.Errorf("intermediate %d bytes is negative", i)
		}
	}
	return nil
}

// Validate refuses a stage sample with no name, or with a command, timing or
// byte counts that fail their own Validate. The error says which.
func (s StageSample) Validate() error {
	if s.Name == "" {
		return fmt.Errorf("stage name is required")
	}
	if err := s.Command.Validate(); err != nil {
		return fmt.Errorf("command: %w", err)
	}
	if err := validateTimeReport(s.Timing); err != nil {
		return fmt.Errorf("timing: %w", err)
	}
	if err := s.Bytes.Validate(); err != nil {
		return fmt.Errorf("bytes: %w", err)
	}
	return nil
}

// Validate refuses a run whose command, scenario or comparability fails its
// own check, that has no stages, or that has a stage that fails Validate.
// The error says which part failed.
func (r RunSample) Validate() error {
	if err := r.Provenance.Command.Validate(); err != nil {
		return fmt.Errorf("provenance: %w", err)
	}
	if err := r.Scenario.Validate(); err != nil {
		return fmt.Errorf("scenario: %w", err)
	}
	if err := r.Comparability.Validate(); err != nil {
		return fmt.Errorf("comparability: %w", err)
	}
	if len(r.Stages) == 0 {
		return fmt.Errorf("stages are required")
	}
	for i, stage := range r.Stages {
		if err := stage.Validate(); err != nil {
			return fmt.Errorf("stage %d: %w", i, err)
		}
	}
	return nil
}
