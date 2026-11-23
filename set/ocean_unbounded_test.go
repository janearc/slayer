package set_test

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

// unboundedText is the ocean set with its square taken out and `unbounded`
// put in its place.
func unboundedText(t *testing.T) string {
	t.Helper()
	text := strings.Replace(oceanSetText(t), "      extent: {value: 64, unit: m}\n"+
		"      samples: 9\n", "      unbounded: true\n", 1)
	if text == oceanSetText(t) {
		t.Fatal("the fixture's square was not found")
	}
	return text
}

func TestUnboundedOceanSaysNoSquare(t *testing.T) {
	source := unboundedText(t)
	read, err := set.Read("unbounded.yaml", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	world, _ := read.World()
	ocean, has := world.OceanSurface()
	if !has || !ocean.Unbounded() {
		t.Fatal("the ocean is not unbounded")
	}
	if ocean.Extent() != 0 || ocean.Samples() != 0 {
		t.Fatalf("an unbounded ocean has a square: %g, %d", ocean.Extent(),
			ocean.Samples())
	}
}

func TestUnboundedOceanRefusesWhatBelongsToASquare(t *testing.T) {
	for _, line := range []string{
		"      extent: {value: 64, unit: m}\n",
		"      samples: 9\n",
		"      shading-normal: spectral-slopes\n",
		"      interface-events: 2\n",
		"      flat-mean:\n        mean-level: {value: 0, unit: m}\n" +
			"        collar-width: {value: 16, unit: m}\n",
	} {
		broken := strings.Replace(unboundedText(t), "      unbounded: true\n",
			"      unbounded: true\n"+line, 1)
		if _, err := set.Read("unbounded-square.yaml", []byte(broken)); err == nil {
			t.Errorf("accepted an unbounded ocean that says %q", line)
		}
	}
}

// edgelessText is the ocean set with its square taken out and nothing put
// in its place.
func edgelessText(t *testing.T) string {
	t.Helper()
	return strings.Replace(unboundedText(t), "      unbounded: true\n", "", 1)
}

// An ocean that says no extent and no samples is the sea with no edge,
// with no `unbounded` key needed.
func TestOceanWithNoExtentAndNoSamplesIsTheSeaWithNoEdge(t *testing.T) {
	source := edgelessText(t)
	read, err := set.Read("edgeless.yaml", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	world, _ := read.World()
	ocean, has := world.OceanSurface()
	if !has || !ocean.Unbounded() {
		t.Fatal("an ocean with no square is not the sea with no edge")
	}
}

// Anti-test: extent without samples, and samples without extent, are
// refused, and the refusal names the key that is missing.
func TestOceanSaysItsExtentAndSamplesBothOrNeither(t *testing.T) {
	for line, missing := range map[string]string{
		"      extent: {value: 64, unit: m}\n": "no samples",
		"      samples: 9\n":                   "no extent",
	} {
		edited := strings.Replace(edgelessText(t), "      seed: 7\n",
			"      seed: 7\n"+line, 1)
		_, err := set.Read("half-square.yaml", []byte(edited))
		if err == nil || !strings.Contains(err.Error(), missing) {
			t.Errorf("%q: error %v, want one naming %q", line, err, missing)
		}
	}
}

// Anti-test: `unbounded: false` with no square asks for a square and
// says none, and is refused.
func TestOceanUnboundedFalseWithNoSquareIsRefused(t *testing.T) {
	text := strings.Replace(edgelessText(t), "      seed: 7\n",
		"      seed: 7\n      unbounded: false\n", 1)
	if _, err := set.Read("false-edgeless.yaml", []byte(text)); err == nil {
		t.Error("accepted unbounded: false with no extent and no samples")
	}
}

// swellLines is a swell block, which a sea with no edge or a finite one
// may say.
const swellLines = "      swell:\n" +
	"        height: {value: 2.5, unit: m}\n" +
	"        period: {value: 12, unit: s}\n" +
	"        toward: {value: 1.5, unit: rad}\n"

// A swell says its height, period and direction, and the ocean hands
// them back.
func TestOceanReadsItsSwell(t *testing.T) {
	for _, text := range []string{edgelessText(t), oceanSetText(t)} {
		edited := strings.Replace(text,
			"      seed: 7\n", "      seed: 7\n"+swellLines, 1)
		read, err := set.Read("swell.yaml", []byte(edited))
		if err != nil {
			t.Fatal(err)
		}
		world, _ := read.World()
		ocean, _ := world.OceanSurface()
		swell, has := ocean.Swell()
		if !has || swell.HeightMetres() != 2.5 || swell.PeriodSeconds() != 12 ||
			swell.TowardRadians() != 1.5 {
			t.Errorf("swell %+v, present %t", swell, has)
		}
	}
	source := edgelessText(t)
	read, err := set.Read("no-swell.yaml", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	world, _ := read.World()
	ocean, _ := world.OceanSurface()
	if _, has := ocean.Swell(); has {
		t.Error("an ocean that says no swell has one")
	}
}

// Anti-test: a swell with any of its three left out is refused.
func TestOceanSwellSaysAllThreeOrNone(t *testing.T) {
	for _, drop := range []string{"        height: {value: 2.5, unit: m}\n",
		"        period: {value: 12, unit: s}\n",
		"        toward: {value: 1.5, unit: rad}\n"} {
		lines := strings.Replace(swellLines, drop, "", 1)
		edited := strings.Replace(
			edgelessText(t), "      seed: 7\n", "      seed: 7\n"+lines, 1)
		_, err := set.Read("half-swell.yaml", []byte(edited))
		if err == nil {
			t.Errorf("accepted a swell without %q", drop)
		}
	}
}

func TestOceanWithoutUnboundedStillNeedsItsSquare(t *testing.T) {
	for _, line := range []string{"      extent: {value: 64, unit: m}\n",
		"      samples: 9\n"} {
		broken := strings.Replace(oceanSetText(t), line, "", 1)
		if _, err := set.Read("no-square.yaml", []byte(broken)); err == nil {
			t.Errorf("accepted a finite ocean without %q", line)
		}
	}
	explicitly := strings.Replace(oceanSetText(t), "      samples: 9\n",
		"      samples: 9\n      unbounded: false\n", 1)
	read, err := set.Read("finite.yaml", []byte(explicitly))
	if err != nil {
		t.Fatal(err)
	}
	world, _ := read.World()
	if ocean, _ := world.OceanSurface(); ocean.Unbounded() {
		t.Fatal("unbounded: false read as unbounded")
	}
}
