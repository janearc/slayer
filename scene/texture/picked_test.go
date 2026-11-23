package texture_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/numbers"
)

// pickedFile is the file the package's chosen numbers are in.
const pickedFile = "picked-numbers.yaml"

// everyMatch is defined by regexp: a count below nought asks for every
// match.
const everyMatch = -1

// nameGroup is defined: the first group of the pattern below, the name.
const nameGroup = 1

// askedFor matches a number the code asks the picked file for, by name.
var askedFor = regexp.MustCompile(`pickedNumber\("([a-z0-9-]+)"\)`)

// namesAsked is every name the package's own code asks the picked file
// for. The tests' own files are not the package's code.
func namesAsked(test *testing.T) map[string]bool {
	test.Helper()
	files, err := filepath.Glob("*.go")
	if err != nil {
		test.Fatal(err)
	}
	asked := map[string]bool{}
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		body, err := os.ReadFile(file)
		if err != nil {
			test.Fatal(err)
		}
		for _, found := range askedFor.FindAllSubmatch(body, everyMatch) {
			asked[string(found[nameGroup])] = true
		}
	}
	return asked
}

// pickedRead is the picked file, read and checked as the package reads
// it.
func pickedRead(test *testing.T) numbers.Numbers {
	test.Helper()
	raw, err := os.ReadFile(pickedFile)
	if err != nil {
		test.Fatal(err)
	}
	read, err := numbers.Read(pickedFile, raw)
	if err != nil {
		test.Fatal(err)
	}
	return read
}

// Every name the code asks for is in the picked file, and the file reads
// whole: a name that is missing would stop the program where it starts.
func TestEveryPickedNumberTheCodeAsksForIsInTheFile(test *testing.T) {
	asked := namesAsked(test)
	if len(asked) == nothing {
		test.Fatal("the code asks for no picked numbers, so nothing was checked")
	}
	read := pickedRead(test)
	for name := range asked {
		if _, err := read.Get(name); err != nil {
			test.Errorf("the code asks for %q: %v", name, err)
		}
	}
}

// The anti-test of the one above: no number is in the file that nothing
// asks for, so a number the code stopped using does not stay to mislead.
func TestNoPickedNumberIsInTheFileThatNothingAsksFor(test *testing.T) {
	asked := namesAsked(test)
	for _, number := range pickedRead(test) {
		if !asked[number.Name] {
			test.Errorf("%q is in the picked file and nothing asks for it",
				number.Name)
		}
	}
}
