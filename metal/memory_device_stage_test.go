//go:build darwin && cgo

package metal

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestDeviceMemoryStageDefaultOff(t *testing.T) {
	called := false
	var out bytes.Buffer
	reportDeviceMemoryStage(&out, false, "stage", func() (uint64, uint64, bool) {
		called = true
		return 1, 2, true
	})
	if called || out.Len() != 0 {
		t.Fatalf("disabled stage sampled=%v output=%q", called, out.String())
	}
}

func TestDeviceMemoryStageAvailability(t *testing.T) {
	for _, available := range []bool{false, true} {
		var out bytes.Buffer
		reportDeviceMemoryStage(&out, true, "example", func() (uint64, uint64, bool) {
			return 123, 456, available
		})
		var record struct {
			Event       string `json:"event"`
			Stage       string `json:"stage"`
			Scope       string `json:"scope"`
			Available   bool   `json:"available"`
			Allocated   uint64 `json:"allocated_bytes"`
			Recommended uint64 `json:"recommended_working_set_bytes"`
		}
		if err := json.Unmarshal(out.Bytes(), &record); err != nil {
			t.Fatal(err)
		}
		if record.Event != "device_memory_stage" || record.Stage != "example" || record.Scope != "default_metal_device" || record.Available != available || record.Allocated != 123 || record.Recommended != 456 {
			t.Fatalf("unexpected device sample: %+v", record)
		}
	}
}
