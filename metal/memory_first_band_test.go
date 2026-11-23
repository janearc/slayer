//go:build darwin && cgo

package metal

import (
	"bytes"
	"reflect"
	"testing"
)

func TestFirstBandMemoryMarksOnlyOnce(t *testing.T) {
	var got []string
	marks := firstBandMemory{emit: func(stage string) { got = append(got, stage) }}
	for range 3 {
		marks.mark(bandMemoryCommit)
		marks.mark(bandMemoryWait)
		marks.mark(bandMemoryCopy)
		marks.mark(bandMemoryConsume)
	}
	want := []string{"first-band-committed", "first-band-waited", "first-band-copied", "first-band-consumed"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("first band marks = %v, want %v", got, want)
	}
}

func TestFirstBandMemoryEnabledEmitsOncePerStage(t *testing.T) {
	var output bytes.Buffer
	marks := newFirstBandMemoryWith(&output, true, func() (uint64, uint64, bool) { return 123, 456, true })
	for range 2 {
		marks.mark(bandMemoryCommit)
		marks.mark(bandMemoryWait)
		marks.mark(bandMemoryCopy)
		marks.mark(bandMemoryConsume)
	}
	for _, name := range firstBandMemoryNames {
		if got := bytes.Count(output.Bytes(), []byte(name)); got != 2 {
			t.Errorf("stage %s count = %d, want process and device records: %s", name, got, output.String())
		}
	}
}

func TestFirstBandMemoryIsOffByDefault(t *testing.T) {
	t.Setenv("SLAYER_MEMORY_STAGES", "")
	marks := newFirstBandMemory()
	if marks.emit != nil {
		t.Fatal("disabled diagnostic installed a reporter")
	}
	marks.mark(bandMemoryCommit)
	if marks.seen != 0 {
		t.Fatal("disabled diagnostic retained state")
	}
}
