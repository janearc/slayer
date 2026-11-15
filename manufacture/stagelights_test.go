package manufacture

import (
	"fmt"
	"math"
	"testing"
)

// the stage lights' contracts: every kind is catalogued, a laser's
// lines are the row's lines through the row's opening, the mixed
// heads split one power by their shares, the strobe is a cold wide
// cone, and no stage light has any representation but quick.

// stageLaserLines is each laser kind's expected lines, in nanometres,
// with its share of the head's power.
func stageLaserLines() map[string][]ray {
	one := func(nano float64) []ray {
		return []ray{{wave: nmLine(nano), share: 1}}
	}
	return map[string][]ray{
		"laser-green": one(greenLine), "laser-red": one(redLine),
		"laser-blue": one(blueLine), "laser-cyan": one(cyanLine),
		"laser-amber": one(amberLine),
		"laser-magenta": {{wave: nmLine(redLine), share: 0.5},
			{wave: nmLine(blueLine), share: 0.5}},
		"laser-white": {{wave: nmLine(redLine), share: third},
			{wave: nmLine(greenLine), share: third},
			{wave: nmLine(blueLine), share: third}},
	}
}

func TestStageKindsAreCatalogued(t *testing.T) {
	for name := range stageRows() {
		if _, err := KindNamed(name); err != nil {
			t.Fatal(err)
		}
	}
}

func TestStageLaserLinesAndOpening(t *testing.T) {
	bore := laserRow(nil).bore
	for name, lines := range stageLaserLines() {
		kind, err := KindNamed(name)
		if err != nil {
			t.Fatal(err)
		}
		made, err := Make(kind, Sow(5))
		if err != nil {
			t.Fatal(err)
		}
		if len(made.lights) != len(lines) {
			t.Fatalf("%s: %d lights, want %d lines", name,
				len(made.lights), len(lines))
		}
		for index, expected := range lines {
			shining := made.lights[index].made
			if !shining.colour.lined() ||
				shining.colour.wave != expected.wave {
				t.Fatalf("%s line %d: wavelength %v, want %v", name,
					index, shining.colour.wave, expected.wave)
			}
			want := firstDark * inScene(expected.wave) / inScene(bore)
			if got := shining.half.InRadians(); !(math.Abs(got-want) <= want*1e-9) {
				t.Fatalf("%s line %d: half angle %g, want %g", name,
					index, got, want)
			}
			if soft := shining.soft.InRadians(); soft != 0 {
				t.Fatalf("%s line %d: an aperture has no soft edge, "+
					"got %g", name, index, soft)
			}
		}
	}
}

func TestStageLaserSharesOnePower(t *testing.T) {
	for _, name := range []string{"laser-magenta", "laser-white"} {
		kind, err := KindNamed(name)
		if err != nil {
			t.Fatal(err)
		}
		made, err := Make(kind, Sow(11))
		if err != nil {
			t.Fatal(err)
		}
		row := laserRow(nil)
		total := 0.0
		for _, one := range made.lights {
			total += one.made.power.InWatts()
		}
		if total < row.power.least-1e-9 || total > row.power.most+1e-9 {
			t.Fatalf("%s: lines sum to %g W, outside %g..%g", name,
				total, row.power.least, row.power.most)
		}
		lines := stageLaserLines()[name]
		for index, one := range made.lights {
			want := total * lines[index].share
			if got := one.made.power.InWatts(); !(math.Abs(got-want) <= total*1e-9) {
				t.Fatalf("%s line %d: %g W, want %g", name, index,
					got, want)
			}
		}
	}
}

func TestStageLasersShineFromOneMouth(t *testing.T) {
	kind, err := KindNamed("laser-white")
	if err != nil {
		t.Fatal(err)
	}
	made, err := Make(kind, Sow(5))
	if err != nil {
		t.Fatal(err)
	}
	first := made.lights[0]
	for _, one := range made.lights[1:] {
		if one.at != first.at || one.toward != first.toward {
			t.Fatal("laser-white: lines from different mouths")
		}
	}
	up := first.toward.Y
	most := math.Sin(laserRow(nil).lift.most * math.Pi / 180)
	if up < 0 || up > most+1e-9 {
		t.Fatalf("laser-white: tipped %g up as a sine, outside 0..%g",
			up, most)
	}
}

func TestStrobeIsAColdWideCone(t *testing.T) {
	kind, err := KindNamed("strobe")
	if err != nil {
		t.Fatal(err)
	}
	made, err := Make(kind, Sow(5))
	if err != nil {
		t.Fatal(err)
	}
	if len(made.lights) != 1 {
		t.Fatalf("strobe: %d lights, want 1", len(made.lights))
	}
	shining := made.lights[0].made
	row := strobeRow()
	if !shining.colour.glows() ||
		shining.colour.heat.InKelvin() != row.heat.InKelvin() {
		t.Fatalf("strobe: heat %v, want %v", shining.colour.heat,
			row.heat)
	}
	half := shining.half.InRadians() * 180 / math.Pi
	if half < row.half.least-1e-9 || half > row.half.most+1e-9 {
		t.Fatalf("strobe: half angle %g deg, outside %g..%g", half,
			row.half.least, row.half.most)
	}
	if soft := shining.soft.InRadians() * 180 / math.Pi; !(math.Abs(
		soft-row.soft.InRadians()*180/math.Pi) <= 1e-9) {
		t.Fatalf("strobe: soft edge %g deg, want %v", soft, row.soft)
	}
}

func TestStageLightsAreQuickOnly(t *testing.T) {
	for name := range stageRows() {
		kind, err := KindNamed(name)
		if err != nil {
			t.Fatal(err)
		}
		for _, level := range []string{"hull", "clumps", "demo"} {
			want := fmt.Sprintf(
				"manufacture: %s has no %s representation", name,
				level)
			if _, err := MakeWithComplexity(kind, Sow(5),
				level); err == nil || err.Error() != want {
				t.Fatalf("%s at %s: got %v, want %s", name, level,
					err, want)
			}
		}
		if _, err := MakeWithComplexity(kind, Sow(5),
			"quick"); err != nil {
			t.Fatal(err)
		}
	}
}
