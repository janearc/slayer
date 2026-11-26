package manufacture

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"testing"

	"github.com/janearc/celestial/measure"
)

func TestKelpPrimitiveGeometryCapture(t *testing.T) {
	want := map[string]string{"blade": "54623b3c0e7b7da0ff0a2ba7e9fddd793c3c45f3fcc20d6f60bd6f6b4235e8fd", "bubble-0.5": "14aad5d9bee42d6f7db684c5aad5219daa1159ea9a474bf767db4837fa77d22a", "bubble-0.75": "7e1371d03fc7bcdda7f57a6867047b6740678adc3151a667bd77dacbf970ddce", "cone": "5c7bfcf19751b16b827aea6aad408497258e3d3081571bfbd8cfa561354239a2", "round": "283b74fc2a354800eeadcd1b6670352a3734c8e68ff070810e8e5d8b3e0547c7"}
	for _, tc := range []struct {
		name              string
		flat              bool
		length, foot, tip float64
		sides             int
	}{
		{"round", false, 1.7, .4, .15, 6},
		{"cone", false, .8, .3, 0, 8},
		{"blade", true, 1.2, .15, .03, 4},
	} {
		makeTube := newTube
		if tc.flat {
			makeTube = newBlade
		}
		made, err := makeTube(measure.Metres(tc.length), measure.Metres(tc.foot), measure.Metres(tc.tip))
		if err != nil {
			t.Fatal(err)
		}
		h := sha256.New()
		for _, tri := range made.skin(cut{around: tc.sides}) {
			for _, p := range tri {
				for _, n := range []float64{p.X, p.Y, p.Z} {
					if err := binary.Write(h, binary.LittleEndian, math.Float64bits(n)); err != nil {
						t.Fatal(err)
					}
				}
			}
		}
		if got := fmt.Sprintf("%x", h.Sum(nil)); got != want[tc.name] {
			t.Errorf("%s: got %s, want %s", tc.name, got, want[tc.name])
		}
	}
	for _, aspect := range []float64{.5, .75} {
		bubble, err := bubbleMesh(aspect)
		if err != nil {
			t.Fatal(err)
		}
		h := sha256.New()
		for _, p := range bubble.Points() {
			for _, n := range []float64{p.X, p.Y, p.Z} {
				if err := binary.Write(h, binary.LittleEndian, math.Float64bits(n)); err != nil {
					t.Fatal(err)
				}
			}
		}
		for _, f := range bubble.Faces() {
			for _, n := range f {
				if err := binary.Write(h, binary.LittleEndian, int64(n)); err != nil {
					t.Fatal(err)
				}
			}
		}
		name := "bubble-" + fmt.Sprint(aspect)
		if got := fmt.Sprintf("%x", h.Sum(nil)); got != want[name] {
			t.Errorf("%s: got %s, want %s", name, got, want[name])
		}
	}
}
