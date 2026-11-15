//go:build draw

package hillside

import (
	_ "embed"
	"testing"
	"time"

	"github.com/janearc/puzzlemouth/numbers"
)

// sequencePair is defined as the count of frames in the two-frame smoke test.
const sequencePair = 2

//go:embed testdata/camera-numbers.yaml
var cameraNumbers []byte

// sequenceNumber reads a named, authored camera expectation through Pzm.
func sequenceNumber(test *testing.T, name string) float64 {
	test.Helper()
	choices, err := numbers.Read("testdata/camera-numbers.yaml", cameraNumbers)
	if err != nil {
		test.Fatal(err)
	}
	value, err := choices.Get(name)
	if err != nil {
		test.Fatal(err)
	}
	return value.Value
}

// sequenceTime puts a named fixture time on the renderer clock.
func sequenceTime(test *testing.T, name string) time.Duration {
	test.Helper()
	seconds := sequenceNumber(test, name)
	return time.Duration(seconds * float64(time.Second))
}
