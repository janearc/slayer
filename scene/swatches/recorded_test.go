package swatches

import (
	"bufio"
	"flag"
	"fmt"
	"maps"
	"math"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/janearc/libtheme-css/primitives/swatch"
)

// update rewrites testdata/recorded.txt from the colours as they are now.
// no gate passes it: the recording moves by hand, in a change that means
// to move every colour, as a new libtheme does.
var update = flag.Bool("update", false,
	"rewrite testdata/recorded.txt from the colours as they are now")

// every named colour here is as recorded, bit for bit, as testdata/recorded.txt
// has it: none lost, none added, none changed. slayer paints with these
// colours until they are changed on purpose, with -update.
//
// this holds the colours to the recording, not the recording to the truth: a
// recording is never evidence. that each paint is a share of the light is
// TestEveryPaintIsAShareOfTheLight's, and is physics.
func TestEveryColourIsRecorded(t *testing.T) {
	ours := painted()
	if *update {
		record(t, ours)
	}
	file, err := os.Open("testdata/recorded.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	check(t, file, ours)
}

// painted is every named colour here, by its name.
func painted() map[string]swatch.Swatch {
	return expanded(map[string]any{
		"AuroraBlue":      AuroraBlue,
		"AuroraGreen":     AuroraGreen,
		"AuroraMauve":     AuroraMauve,
		"AuroraPurple":    AuroraPurple,
		"AuroraRed":       AuroraRed,
		"AuroraViolet":    AuroraViolet,
		"Blue":            Blue,
		"Chalk":           Chalk,
		"Charon":          Charon,
		"City":            City,
		"Clay":            Clay,
		"ComaDust":        ComaDust,
		"ComaDustCool":    ComaDustCool,
		"ComaGreen":       ComaGreen,
		"ComaGreenBright": ComaGreenBright,
		"Cream":           Cream,
		"Daylight":        Daylight,
		"DustTail":        DustTail,
		"DustTailPale":    DustTailPale,
		"Earth":           Earth,
		"EarthAir":        EarthAir,
		"Europa":          Europa,
		"FlareWhite":      FlareWhite,
		"Ganymede":        Ganymede,
		"Green":           Green,
		"Horizon":         Horizon,
		"Ice":             Ice,
		"Io":              Io,
		"IonGlow":         IonGlow,
		"IonTail":         IonTail,
		"IoPlume":         IoPlume,
		"JetBlue":         JetBlue,
		"JupiterOval":     JupiterOval,
		"MagnetarBlue":    MagnetarBlue,
		"Mars":            Mars,
		"MarsAir":         MarsAir,
		"Mercury":         Mercury,
		"Moon":            Moon,
		"Moss":            Moss,
		"NeptuneBlue":     NeptuneBlue,
		"NeptuneCloud":    NeptuneCloud,
		"NeptuneDark":     NeptuneDark,
		"NeptuneDeep":     NeptuneDeep,
		"NeptunePale":     NeptunePale,
		"NeptuneSpot":     NeptuneSpot,
		"Ochre":           Ochre,
		"Paint":           Paint,
		"Pluto":           Pluto,
		"PulsarBeam":      PulsarBeam,
		"PulsarWhite":     PulsarWhite,
		"Red":             Red,
		"Rust":            Rust,
		"SaturnBelt":      SaturnBelt,
		"SaturnCream":     SaturnCream,
		"SaturnOchre":     SaturnOchre,
		"SaturnPale":      SaturnPale,
		"Sea":             Sea,
		"Slate":           Slate,
		"Storm":           Storm,
		"SunWhite":        SunWhite,
		"TailGlow":        TailGlow,
		"Titan":           Titan,
		"TitanAir":        TitanAir,
		"Tungsten":        Tungsten,
		"UranusCyan":      UranusCyan,
		"UranusDeep":      UranusDeep,
		"UranusPale":      UranusPale,
		"UranusRing":      UranusRing,
		"Venus":           Venus,
		"VenusAir":        VenusAir,
		"Zenith":          Zenith,
		"Terran":          Terran,
	})
}

// check holds the colours to the recording read from file.
func check(t *testing.T, file *os.File, ours map[string]swatch.Swatch) {
	t.Helper()
	seen := 0
	lines := bufio.NewScanner(file)
	for lines.Scan() {
		line := lines.Text()
		if strings.HasPrefix(line, "#") {
			continue
		}
		var name string
		var bits [3]uint64
		if _, err := fmt.Sscanf(line, "%s %x %x %x", &name, &bits[0],
			&bits[1], &bits[2]); err != nil {
			t.Fatalf("%q: %v", line, err)
		}
		seen++
		s, ok := ours[name]
		if !ok {
			t.Errorf("%s is recorded and not here", name)
			continue
		}
		l := Light(s)
		if math.Float64bits(l.R) != bits[0] ||
			math.Float64bits(l.G) != bits[1] ||
			math.Float64bits(l.B) != bits[2] {
			t.Errorf("%s is %v, not recorded", name, l)
		}
	}
	if err := lines.Err(); err != nil {
		t.Fatal(err)
	}
	if seen != len(ours) {
		t.Errorf("%d colours here, %d recorded", len(ours), seen)
	}
}

// record writes the recording from the colours as they are now: each
// name in order, then the bits of its red, green and blue.
func record(t *testing.T, ours map[string]swatch.Swatch) {
	t.Helper()
	var written strings.Builder
	written.WriteString("# every named colour of scene/swatches, as its " +
		"Light: name, then the\n# bits of its red, green and blue. " +
		"written by TestEveryColourIsRecorded\n# with -update, in a " +
		"change meant to move them; the test holds slayer's\n# colours " +
		"to it.\n")
	for _, name := range slices.Sorted(maps.Keys(ours)) {
		l := Light(ours[name])
		fmt.Fprintf(&written, "%s %x %x %x\n", name, math.Float64bits(l.R),
			math.Float64bits(l.G), math.Float64bits(l.B))
	}
	if err := os.WriteFile("testdata/recorded.txt",
		[]byte(written.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}

// expanded is every colour in a map of named things: a swatch as itself,
// a palette or a pair field by field, and an array colour by colour.
func expanded(named map[string]any) map[string]swatch.Swatch {
	out := map[string]swatch.Swatch{}
	for name, v := range named {
		walk(out, name, reflect.ValueOf(v))
	}
	return out
}

// walk puts every colour in v into out, named from name.
func walk(out map[string]swatch.Swatch, name string, v reflect.Value) {
	if s, ok := v.Interface().(swatch.Swatch); ok {
		out[name] = s
		return
	}
	switch v.Kind() {
	case reflect.Struct:
		for i := range v.NumField() {
			walk(out, name+"."+v.Type().Field(i).Name, v.Field(i))
		}
	case reflect.Array:
		for i := range v.Len() {
			walk(out, fmt.Sprintf("%s[%d]", name, i), v.Index(i))
		}
	}
}
