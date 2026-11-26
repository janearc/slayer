package main

import (
	"encoding/json"
	"fmt"
	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/scene/thing"
	"runtime"
	"time"
)

type resourceRow struct {
	Producer                *producer             `json:"producer,omitempty"`
	Kind                    string                `json:"kind"`
	Detail                  string                `json:"detail"`
	Seed                    uint64                `json:"seed"`
	Construction            time.Duration         `json:"construction_ns"`
	Preparation             time.Duration         `json:"preparation_ns"`
	Allocated               uint64                `json:"allocated_bytes"`
	Parts                   int                   `json:"parts"`
	Geometry                thing.GeometryStorage `json:"geometry"`
	RetainLocalGeometry     bool                  `json:"retain_local_geometry"`
	IndependentPlacement    bool                  `json:"independent_placement"`
	NeedsNativeMeasurement  bool                  `json:"needs_native_measurement"`
	SharedMaterials         bool                  `json:"shared_materials"`
	NeedsFrameStateSnapshot bool                  `json:"needs_frame_state_snapshot"`
	Error                   string                `json:"error,omitempty"`
}

// resources visits every selected constructor at every advertised detail.
// JSON lines are emitted immediately, so one failed model cannot hide others.
func resources(call pzm.Call) error {
	menu, err := menuForCall(call)
	if err != nil {
		return err
	}
	entries, err := menu.selectEntries(call.Args())
	if err != nil {
		return err
	}
	writer := json.NewEncoder(call.Out())
	by, err := compiledProducer()
	if err != nil {
		return err
	}
	failed := 0
	for _, entry := range entries {
		for _, detail := range entry.Details {
			// Release the preceding specimen before constructing the next.
			runtime.GC()
			row := measureEntryResources(entry, detail)
			row.Producer = by
			if row.Error != "" {
				failed++
			}
			if err := writer.Encode(row); err != nil {
				return err
			}
		}
	}
	if failed > 0 {
		return fmt.Errorf("catalogue resources: %d representations failed; all rows retained", failed)
	}
	return nil
}

// measureResources builds one kind at the given detail and records the
// construction time and the memory it allocated.
func measureResources(name, detail string) resourceRow {
	return measureEntryResources(entry{Kind: name}, detail)
}

func measureEntryResources(entry entry, detail string) resourceRow {
	row := resourceRow{Kind: entry.Kind, Detail: detail, Seed: 1}
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	start := time.Now()
	kind, err := entry.kind()
	if err != nil {
		row.Error = err.Error()
		return row
	}
	made, err := manufacture.MakeWithComplexity(kind, manufacture.Sow(row.Seed), detail)
	row.Construction = time.Since(start)
	if err != nil {
		row.Error = err.Error()
		return row
	}
	row.Parts = made.PartCount()
	start = time.Now()
	ready, err := made.Prepare()
	row.Preparation = time.Since(start)
	if err != nil {
		row.Error = err.Error()
		return row
	}
	runtime.ReadMemStats(&after)
	row.Allocated = after.TotalAlloc - before.TotalAlloc
	profile := ready.Resources()
	row.Geometry = profile.Geometry
	row.RetainLocalGeometry = profile.RetainLocalGeometry
	row.IndependentPlacement = profile.IndependentPlacement
	row.NeedsNativeMeasurement = profile.NeedsNativeMeasurement
	row.SharedMaterials = profile.SharedMaterials
	row.NeedsFrameStateSnapshot = profile.NeedsFrameStateSnapshot
	return row
}
