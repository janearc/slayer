package beamstep

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"testing"
)

// kernelPath is the Metal kernel's text, from this package's directory.
const kernelPath = "../../gpu/trace.metal"

// kernelNumber is the number the kernel's text gives a named constexpr
// constant of int or float type, with any f suffix left off.
func kernelNumber(text, name string) (float64, error) {
	pattern := regexp.MustCompile(`(?m)^constexpr constant (?:int|float) ` +
		regexp.QuoteMeta(name) + `\s*=\s*([-+0-9.eE]+)f?;`)
	found := pattern.FindStringSubmatch(text)
	if found == nil {
		return 0, fmt.Errorf("the kernel's text has no constant %s", name)
	}
	return strconv.ParseFloat(found[1], 64)
}

// parallelFault is what is wrong with the kernel's beam_parallel against
// the package's: nothing is the empty string.
func parallelFault(text string) string {
	got, err := kernelNumber(text, "beam_parallel")
	if err != nil {
		return err.Error()
	}
	if got != parallel {
		return fmt.Sprintf("the kernel's beam_parallel is %g, the "+
			"package's is %g", got, parallel)
	}
	return ""
}

// The kernel's beam_parallel is the number the package takes a ray as
// along a beam's axis below: the two are typed apart, and a quadratic
// divided by a different small number would disagree with the other.
func TestTheKernelsParallelIsThePackages(t *testing.T) {
	source, err := os.ReadFile(kernelPath)
	if err != nil {
		t.Fatal(err)
	}
	if fault := parallelFault(string(source)); fault != "" {
		t.Error(fault)
	}
}

// Anti: a kernel text with another number for beam_parallel is reported.
func TestAKernelWithAnotherParallelIsReported(t *testing.T) {
	text := "constexpr constant float beam_parallel = 1e-5f;\n"
	if parallelFault(text) == "" {
		t.Error("a kernel with a parallel of 1e-5 was not reported")
	}
	if parallelFault("nothing here\n") == "" {
		t.Error("a kernel with no beam_parallel was not reported")
	}
}
