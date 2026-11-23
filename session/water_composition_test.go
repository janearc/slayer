package session

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

func TestCameraInitialMediumRetainsWaterComposition(t *testing.T) {
	source := strings.Replace(cameraMediumSetText(t),
		"extinction-per-metre: [0.1, 0.2, 0.3]}",
		"extinction-per-metre: [0.1, 0.2, 0.3], "+
			"water-composition: {salinity: {value: 31, source: USGS, "+
			"at: '2026-11-18T18:00:00Z', qualifier: provisional}}}", 1)
	read, err := set.Read("camera-water-composition.yaml", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	staged := staging{given: asked{set: "camera-water-composition.yaml"}}
	if _, err := staged.eye(read); err != nil {
		t.Fatal(err)
	}
	if len(staged.given.initialMedia) != 1 {
		t.Fatal("camera has no initial medium")
	}
	medium := staged.given.initialMedia[0].ClearMedium
	water, opticalSource, has := medium.WaterComposition()
	if !has {
		t.Fatal("camera medium lost water composition")
	}
	salt, hasSalt := water.Salinity()
	if !hasSalt || salt.Value != 31 || salt.Source != "USGS" ||
		salt.At != "2026-11-18T18:00:00Z" || salt.Qualifier != "provisional" ||
		opticalSource != "synthetic validation coefficients; not measured water" {
		t.Fatalf("camera medium lost observation or optical source: %+v %q", salt, opticalSource)
	}
}

func TestCameraInitialMediumRetainsObservationsWithoutOpticalCoefficients(t *testing.T) {
	source := strings.Replace(cameraMediumSetText(t),
		", extinction-per-metre: [0.1, 0.2, 0.3]}",
		", water-composition: {salinity: {value: 31, source: USGS}}}", 1)
	source = strings.Replace(source,
		"source: synthetic validation coefficients; not measured water",
		"source: \"synthetic validation coefficients; not measured water\"", 1)
	read, err := set.Read("camera-water-without-optics.yaml", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	staged := staging{given: asked{set: "camera-water-without-optics.yaml"}}
	if _, err := staged.eye(read); err != nil {
		t.Fatal(err)
	}
	if len(staged.given.initialMedia) != 1 {
		t.Fatal("camera has no initial medium")
	}
	initial := staged.given.initialMedia[0]
	salt, hasSalt := initial.WaterComposition.Salinity()
	if !hasSalt || salt.Value != 31 || salt.Source != "USGS" ||
		initial.Source != "synthetic validation coefficients; not measured water" ||
		initial.ClearMedium != nil {
		t.Fatalf("camera observation or optics differ: %+v %+v", initial, salt)
	}
}
