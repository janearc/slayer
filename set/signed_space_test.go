package set_test

import (
	"errors"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

// signedPlacement is one native object in the same public space format as Ferruccio.
func signedPlacement(schema, setting, east, north string, settingLast bool) string {
	source := smallScene
	object := "  set:\n    things:\n      ship:\n        kind: mcrn-larson\n        seed: 1\n        east: " + east + "\n        north: " + north + "\n        height: {value: 0, unit: m}\n"
	if settingLast {
		source += object + setting
	} else {
		source += setting + object
	}
	if schema != "" {
		source = schema + "\n" + source
	}
	return source
}

// TestSpaceThingCoordinatesCrossTheOriginInEitherKeyOrder preserves signed metre values.
func TestSpaceThingCoordinatesCrossTheOriginInEitherKeyOrder(t *testing.T) {
	for _, last := range []bool{false, true} {
		for _, position := range []struct {
			east, north         string
			wantEast, wantNorth float64
		}{
			{"-0.5", "-2", -0.5, -2},
			{"-1000000000", "1000000000", -1e9, 1e9},
			{"1000000000", "-1000000000", 1e9, -1e9},
		} {
			source := signedPlacement(v2Label, "  setting: space\n", position.east, position.north, last)
			made, err := set.Read("signed-space.yaml", []byte(source))
			if err != nil {
				t.Fatalf("space position %s/%s settingLast=%t: %v", position.east, position.north, last, err)
			}
			east, north := made.Things()[0].At()
			if east != position.wantEast || north != position.wantNorth {
				t.Fatalf("coordinate changed: %v/%v", east, north)
			}
		}
	}
}

// TestGroundThingCoordinatesStayNonnegative rejects the same negative edit in every ground format.
func TestGroundThingCoordinatesStayNonnegative(t *testing.T) {
	for _, format := range []struct{ schema, setting string }{
		{"", ""}, {v1Label, ""}, {v2Label, ""}, {v2Label, "  setting: ground\n"},
	} {
		for _, position := range [][2]string{{"-0.5", "1"}, {"1", "-0.5"}} {
			source := signedPlacement(format.schema, format.setting, position[0], position[1], false)
			_, err := set.Read("negative-ground.yaml", []byte(source))
			if err == nil {
				t.Fatalf("ground format %q accepted %v", format, position)
			}
			if format.schema != "" {
				var invalid *jsonschema.ValidationError
				if !errors.As(err, &invalid) {
					t.Fatalf("ground coordinate escaped schema validation: %v", err)
				}
			}
		}
		source := signedPlacement(format.schema, format.setting, "0", "0", false)
		if _, err := set.Read("zero-ground.yaml", []byte(source)); err != nil {
			t.Fatalf("ground origin refused: %v", err)
		}
	}
}

// TestSpaceThingCoordinatesKeepFiniteCapsAndHeightFloor refuses unsafe transforms.
func TestSpaceThingCoordinatesKeepFiniteCapsAndHeightFloor(t *testing.T) {
	for _, bad := range []string{"-1000000001", "1000000001", ".nan", ".inf", "-.inf"} {
		for _, position := range [][2]string{{bad, "0"}, {"0", bad}} {
			source := signedPlacement(v2Label, "  setting: space\n", position[0], position[1], false)
			if _, err := set.Read("bad-space.yaml", []byte(source)); err == nil {
				t.Fatalf("space accepted %v", position)
			}
		}
	}
	source := signedPlacement(v2Label, "  setting: space\n", "-0.5", "-0.5", false)
	source = strings.Replace(source, "height: {value: 0,", "height: {value: -0.5,", 1)
	if _, err := set.Read("below-plane.yaml", []byte(source)); err == nil {
		t.Fatal("negative height accepted")
	}
}
