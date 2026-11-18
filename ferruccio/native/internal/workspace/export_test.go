package workspace

import "testing"

// TestExportWithNoDeviceSaysCPU is the anti-test: an export drawn with no
// GPU device given to gg must not claim one in its receipt.
func TestExportWithNoDeviceSaysCPU(t *testing.T) {
	work := playWorkspace(t)
	work.ui.SetRoot(work.viewer)
	// The pane's usual size in the other export tests.
	const width, height = 800, 600
	work.ui.Window().HandleResize(width, height)
	if _, err := work.exportWith(nil); err != nil {
		t.Fatal(err)
	}
	if drawn := work.ExportDrawn(); drawn.Backend != "cpu" || drawn.Shared {
		t.Fatalf("export with no device says %+v; want cpu, not shared", drawn)
	}
}
