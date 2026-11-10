package research

import (
	"strings"
	"sync"
	"testing"
)

func TestValueReadsPublishedDataAndReportsBrokenEmbeddedData(t *testing.T) {
	if got := Value("redwood-needle-minimum"); got != 1 {
		t.Fatalf("redwood minimum: %g cm", got)
	}
	before, cached := raw, values
	t.Cleanup(func() { raw, values = before, cached })
	raw = []byte("numbers: [")
	values = sync.OnceValues(Read)
	if _, err := Read(); err == nil || !strings.Contains(err.Error(), fileName) {
		t.Fatalf("invalid embedded document: %v", err)
	}
	defer func() {
		err, ok := recover().(error)
		if !ok || !strings.Contains(err.Error(), fileName) {
			t.Fatalf("broken embedded data did not identify itself: %v", err)
		}
	}()
	Value("redwood-needle-minimum")
}
