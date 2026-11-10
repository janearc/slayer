package set_test

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

func TestGroundObservatoryNamesWorkThroughBothSceneReaders(test *testing.T) {
	for _, kind := range []string{"palomar", "keck-i", "keck-ii", "subaru", "rubin", "elt", "griffith", "dobsonian-36", "apo-refractor-5", "sct-24", "mobile-crane", "mirror-gantry", "scaffold-tower", "service-truck", "utility-pickup", "equipment-trailer", "shipping-crate", "equipment-cabinet", "generator", "cooling-unit", "access-stairs", "work-platform", "handrail", "cable-tray", "weather-mast", "cfht", "gemini-north", "irtf", "uh-88", "ukirt", "jcmt", "sma-antenna", "sma-array", "vlba-maunakea", "panstarrs-1", "panstarrs-2", "faulkes-north", "dkist", "atlas-haleakala", "tlrs-4", "solarc", "mees-solar", "aeos", "mssc-1-6m", "mssc-twin-1-2m", "geodss", "geodss-site"} {
		for _, raw := range laserSources() {
			source := strings.ReplaceAll(raw, "recipe: ridge telescope", "kind: "+kind)
			if source == raw {
				test.Fatal("ground template was not installed in scene")
			}
			scene, err := set.Read("ground-observatory.yaml", []byte(source))
			if err != nil {
				test.Fatal(kind, err)
			}
			found := false
			for _, object := range scene.Things() {
				found = found || object.Kind() == kind
			}
			if !found {
				test.Fatal("scene lost its ground observatory", kind)
			}
		}
	}
}
