package manufacture

import (
	"math"
	"reflect"
	"testing"
)

func TestCavitationCloudIsSeededBoundedAndSpeedScaled(t *testing.T) {
	config := CavitationConfig{
		Count: 32, Seed: 7052026, SpeedMPS: 20, TrailSeconds: 1.4,
		RiseMPS: 3, LengthM: 81.4, ScrewAftM: 79.6, ScrewOffsetM: 0.8,
		ScrewUpM: 4.75,
		SpreadM:  3.4, RadiusMinM: 0.18, RadiusMaxM: 0.6,
	}
	pose := CavitationPose{EastM: -1500, KeelUpM: -18.24, HeadingDegrees: 340}
	first, err := NewCavitationCloud(config, pose)
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewCavitationCloud(config, pose)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("same seed and settings did not reproduce the cloud")
	}
	if len(first) != config.Count {
		t.Fatalf("made %d bubbles, want %d", len(first), config.Count)
	}
	minEast, maxEast := math.Inf(1), math.Inf(-1)
	for _, bubble := range first {
		if bubble.RadiusM < config.RadiusMinM || bubble.RadiusM > config.RadiusMaxM {
			t.Errorf("sphere radius outside authored range: %+v", bubble)
		}
		if !cavitationFinite(bubble.EastM, bubble.NorthM, bubble.UpM, bubble.RadiusM) {
			t.Fatalf("nonfinite bubble placement: %+v", bubble)
		}
		minEast = math.Min(minEast, bubble.EastM)
		maxEast = math.Max(maxEast, bubble.EastM)
	}
	if maxEast-minEast > config.SpeedMPS*config.TrailSeconds+2*config.SpreadM {
		t.Errorf("map-space trail extent %.3fm exceeds authored bound", maxEast-minEast)
	}
	fasterConfig := config
	fasterConfig.SpeedMPS *= 2
	faster, err := NewCavitationCloud(fasterConfig, pose)
	if err != nil {
		t.Fatal(err)
	}
	if faster[len(faster)-1].EastM <= first[len(first)-1].EastM {
		t.Fatal("increasing speed did not extend eastward trail for this pose")
	}
}

func TestCavitationCloudRefusesUnboundedAndOverflowingSettings(t *testing.T) {
	base := CavitationConfig{Count: 32, SpeedMPS: 20, TrailSeconds: 1.4,
		RiseMPS: 3, LengthM: 81.4, ScrewAftM: 79.6, ScrewOffsetM: 0.8,
		ScrewUpM: 4.75,
		SpreadM:  3.4, RadiusMinM: 0.18, RadiusMaxM: 0.6}
	pose := CavitationPose{}
	for name, change := range map[string]func(*CavitationConfig){
		"too many":        func(c *CavitationConfig) { c.Count = 513 },
		"negative speed":  func(c *CavitationConfig) { c.SpeedMPS = -1 },
		"speed cap":       func(c *CavitationConfig) { c.SpeedMPS = 101 },
		"trail length":    func(c *CavitationConfig) { c.SpeedMPS = 100; c.TrailSeconds = 6 },
		"rise extent":     func(c *CavitationConfig) { c.RiseMPS = 10; c.TrailSeconds = 10.1 },
		"spread cap":      func(c *CavitationConfig) { c.SpreadM = 51 },
		"radius cap":      func(c *CavitationConfig) { c.RadiusMaxM = 5.1 },
		"bad radius":      func(c *CavitationConfig) { c.RadiusMaxM = 0.1 },
		"screw offset":    func(c *CavitationConfig) { c.ScrewOffsetM = 5.1 },
		"finite overflow": func(c *CavitationConfig) { c.SpeedMPS = math.MaxFloat64; c.TrailSeconds = math.MaxFloat64 },
		"not finite":      func(c *CavitationConfig) { c.SpreadM = math.NaN() },
	} {
		t.Run(name, func(t *testing.T) {
			config := base
			change(&config)
			if _, err := NewCavitationCloud(config, pose); err == nil {
				t.Fatal("invalid cloud config was accepted")
			}
		})
	}
}

func TestCavitationCloudRefusesUnboundedPose(t *testing.T) {
	config := CavitationConfig{Count: 1, SpeedMPS: 20, TrailSeconds: 1,
		LengthM: 81.4, ScrewAftM: 79.6, ScrewOffsetM: 0.8, ScrewUpM: 4.75,
		RadiusMinM: 0.18, RadiusMaxM: 0.6}
	if _, err := NewCavitationCloud(config,
		CavitationPose{EastM: math.MaxFloat64}); err == nil {
		t.Fatal("unbounded world coordinate was accepted")
	}
	if _, err := NewCavitationCloud(config,
		CavitationPose{EastM: cavitationMaxWorldHorizontal - 1}); err == nil {
		t.Fatal("pose without room for the full trail envelope was accepted")
	}
}

func TestCavitationCloudAllowsAnEmptyCloud(t *testing.T) {
	cloud, err := NewCavitationCloud(CavitationConfig{
		LengthM: 81.4, ScrewAftM: 79.6, ScrewOffsetM: 0.8, ScrewUpM: 4.75,
		RadiusMinM: 0.1, RadiusMaxM: 0.2,
	}, CavitationPose{})
	if err != nil {
		t.Fatal(err)
	}
	if cloud == nil || len(cloud) != 0 {
		t.Fatalf("empty cloud = %#v, want non-nil empty slice", cloud)
	}
}
