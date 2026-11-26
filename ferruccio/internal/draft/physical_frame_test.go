package draft

import (
	"math"
	"strings"
	"testing"
)

func TestPhysicalFrameEditPreservesRigAndSourceHistory(t *testing.T) {
	source := strings.Replace(poseSource, "film: {width: 640, height: 480, rays: 1, exposure: 2000, wide: 45}", "film: {width: 640, rays: 1, frame: {width: 36, height: 24}, lens: {focal: 50, stop: 8, focus: craft}, shutter: 0.5, glimpses: 2} # Keep rig", 1)
	document := poseDraft(t, source)
	if size, ok := document.PhysicalFrame(); !ok || size != [2]float64{36, 24} {
		t.Fatalf("frame %v %v", size, ok)
	}
	for _, bad := range [][2]float64{{0, 24}, {math.NaN(), 24}, {36, 1001}} {
		if err := document.SetPhysicalFrame(bad); err == nil || document.Text() != source {
			t.Fatal("invalid frame changed source")
		}
	}
	if err := document.SetPhysicalFrame([2]float64{70.41, 52.63}); err != nil {
		t.Fatal(err)
	}
	if size, ok := document.PhysicalFrame(); !ok || size != [2]float64{70.41, 52.63} || !strings.Contains(document.Text(), "Keep rig") || !strings.Contains(document.Text(), "focal: 50") {
		t.Fatalf("frame %v %v", size, ok)
	}
}
