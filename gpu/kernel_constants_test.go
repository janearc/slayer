package gpu

import (
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"github.com/janearc/slayer/render/beamstep"
)

// kernelPattern finds a named constexpr constant of int or float type in
// the kernel's text: its lead, the number, and any f suffix.
func kernelPattern(name string) *regexp.Regexp {
	return regexp.MustCompile(`(?m)^(constexpr constant (?:int|float) ` +
		regexp.QuoteMeta(name) + `\s*=\s*)([-+0-9.eE]+)(f?;)`)
}

// kernelValue is the number the kernel's text gives a named constant.
func kernelValue(text, name string) (float64, error) {
	found := kernelPattern(name).FindStringSubmatch(text)
	if found == nil {
		return 0, fmt.Errorf("the kernel's text has no constant %s", name)
	}
	return strconv.ParseFloat(found[2], 64)
}

// intervalsFault is what is wrong with the kernel's most_beam_intervals
// against beamstep's MostIntervals: nothing is the empty string.
func intervalsFault(text string) string {
	got, err := kernelValue(text, "most_beam_intervals")
	if err != nil {
		return err.Error()
	}
	if got != float64(beamstep.MostIntervals) {
		return fmt.Sprintf("the kernel keeps %g stretches, beamstep %d",
			got, beamstep.MostIntervals)
	}
	return ""
}

// The kernel keeps as many stretches of a ray apart as the host does: the
// two numbers are typed apart, and a kernel with fewer would merge beams
// the host keeps apart, or the reverse. beam_parallel is beamstep's own
// test, as the host's is not exported.
func TestTheKernelKeepsAsManyStretchesAsBeamstep(t *testing.T) {
	if fault := intervalsFault(kernelText(t)); fault != "" {
		t.Error(fault)
	}
}

// Anti: a kernel text that keeps one fewer stretch than beamstep is
// reported, and so is one with no such constant.
func TestAKernelWithFewerStretchesIsReported(t *testing.T) {
	fewer := kernelPattern("most_beam_intervals").ReplaceAllString(
		kernelText(t), fmt.Sprintf("${1}%d${3}", beamstep.MostIntervals-1))
	if intervalsFault(fewer) == "" {
		t.Error("a kernel keeping one fewer stretch was not reported")
	}
	if intervalsFault("nothing here\n") == "" {
		t.Error("a kernel with no most_beam_intervals was not reported")
	}
}
