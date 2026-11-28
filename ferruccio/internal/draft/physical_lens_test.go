package draft

import (
	"math"
	"strings"
	"testing"
)

const physicalLensRig = "film: {width: 640, rays: 1, frame: {width: 36, height: 24}, lens: {focal: 50, stop: 8, focus: craft}, shutter: 0.5, glimpses: 2} # Keep rig"

func TestPhysicalLensChangesBothOpticsAndKeepsTheRestOfTheRig(t *testing.T) {
	source := strings.Replace(poseSource, "film: {width: 640, height: 480, rays: 1, exposure: 2000, wide: 45}", physicalLensRig, 1)
	for _, format := range []string{"set", "legacy-set"} {
		t.Run(format, func(t *testing.T) {
			current := source
			if format == "legacy-set" {
				current = strings.Replace(current, "$schema: https://github.com/janearc/slayer/schemas/ecosystem-v2.json\n", "", 1)
			}
			document := poseDraft(t, current)
			if lens, ok := document.PhysicalLens(); !ok || lens != (PhysicalLens{FocalMM: 50, FStop: 8}) {
				t.Fatalf("authored lens %+v, available %t", lens, ok)
			}
			if err := document.SetPhysicalLens(PhysicalLens{FocalMM: 50, FStop: 8}); err != nil || document.Text() != current {
				t.Fatalf("unchanged lens rewrote the source: %v", err)
			}
			want := PhysicalLens{FocalMM: 85, FStop: 2.8}
			if err := document.SetPhysicalLens(want); err != nil {
				t.Fatal(err)
			}
			if lens, ok := document.PhysicalLens(); !ok || lens != want {
				t.Fatalf("edited lens %+v, available %t", lens, ok)
			}
			for _, retained := range []string{"frame: {width: 36, height: 24}", "focus: craft", "shutter: 0.5", "glimpses: 2", "Keep rig", "Keep the scene note."} {
				if !strings.Contains(document.Text(), retained) {
					t.Errorf("edit lost %q", retained)
				}
			}
		})
	}
}

func TestPhysicalLensRefusesInvalidOpticsAndPinholeFilm(t *testing.T) {
	source := strings.Replace(poseSource, "film: {width: 640, height: 480, rays: 1, exposure: 2000, wide: 45}", physicalLensRig, 1)
	source = strings.Replace(source, "$schema: https://github.com/janearc/slayer/schemas/ecosystem-v2.json\n", "", 1)
	document := poseDraft(t, source)
	bad := []PhysicalLens{{FocalMM: math.NaN(), FStop: 8}, {FocalMM: 0.5, FStop: 8}, {FocalMM: 100001, FStop: 8}, {FocalMM: 50, FStop: 0.25}, {FocalMM: 50, FStop: 1001}, {FocalMM: 50, FStop: math.Inf(1)}}
	for _, lens := range bad {
		if err := document.SetPhysicalLens(lens); err == nil || document.Text() != source {
			t.Fatalf("invalid lens %+v changed source: %v", lens, err)
		}
	}
	pinhole := poseDraft(t, poseSource)
	if _, ok := pinhole.PhysicalLens(); ok {
		t.Fatal("pinhole film claims a physical lens")
	}
	if err := pinhole.SetPhysicalLens(PhysicalLens{FocalMM: 85, FStop: 2.8}); err == nil || pinhole.Text() != poseSource {
		t.Fatalf("physical lens edit converted a pinhole film: %v", err)
	}
}
