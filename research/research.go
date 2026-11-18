// Package research exposes named values from research-derived-data.yml. Callers
// read the recorded data rather than keeping copies of published numbers in
// code. Before a build, sources verify checks the values against their pinned
// sources.
package research

import (
	_ "embed"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/janearc/puzzlemouth/numbers"
)

// fileName is the data's file, as a refusal names it.
const fileName = "research-derived-data.yml"

// raw is the data as the file holds it.
//
//go:embed research-derived-data.yml
var raw []byte

// Read is the research data, every number by its name.
func Read() (numbers.Numbers, error) {
	return numbers.Read(fileName, raw)
}

// values retains only the private lookup view of immutable embedded data.
// Read still parses a fresh caller-owned catalogue, including source records.
var values = sync.OnceValues(Read)

// Value is a measured number by its name. A name the data does not hold
// is a mistake in slayer's own code, which its tests find, so asking for
// one stops the program and says which.
func Value(name string) float64 {
	read, err := values()
	if err != nil {
		panic(err)
	}
	number, err := read.Get(name)
	if err != nil {
		panic(fmt.Errorf("research: %w", err))
	}
	if see := observer.Load(); see != nil {
		(*see)(*number)
	}
	return number.Value
}

// observer, when set, is told of every number Value hands out.
var observer atomic.Pointer[func(numbers.Number)]

// Observe has see told of each number Value reads from now on, until the
// function it returns is called. It is for a report of which measured
// numbers a piece of code reads; one observer is kept at a time, and the
// function returned puts back whichever was there before.
func Observe(see func(numbers.Number)) (stop func()) {
	before := observer.Swap(&see)
	return func() { observer.Store(before) }
}
