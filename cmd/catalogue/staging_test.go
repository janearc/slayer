package main

import (
	_ "embed"
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
)

//go:embed testdata/staging.yaml
var stagingRaw []byte

type stagingContract struct {
	Kind, Detail string
	Seed         uint64
	Staging      staging
	Refused      string
	Webb         struct {
		Kind       string
		Suns       int
		Sight      maths.Vec `yaml:"line-of-sight"`
		Shield     maths.Vec `yaml:"shield-side"`
		FromSight  float64   `yaml:"degrees-from-sight"`
		FromShield float64   `yaml:"degrees-from-shield-side"`
		Within     float64   `yaml:"within-degrees"`
	}
}

func readStaging(test *testing.T) stagingContract {
	test.Helper()
	var data stagingContract
	if err := yamlfile.Read("staging.yaml", stagingRaw, &data); err != nil {
		test.Fatal(err)
	}
	return data
}

// degreesBetween is the angle between two ways, in degrees.
func degreesBetween(one, other maths.Vec) float64 {
	alignment := one.Unit().Dot(other.Unit())
	return math.Acos(alignment) / maths.Degree
}

func TestAStagedEntryIsLitByItsStagingAndNotTheStudio(test *testing.T) {
	data := readStaging(test)
	plan := defaultPlan(test)
	plan.Seed = data.Seed
	studio, err := plan.construct(entry{Kind: data.Kind}, data.Detail)
	if err != nil {
		test.Fatal(err)
	}
	staged, err := plan.construct(entry{Kind: data.Kind,
		Staging: &data.Staging}, data.Detail)
	if err != nil {
		test.Fatal(err)
	}
	if studio.world.Sky != plan.Sky || staged.world.Sky != data.Staging.Sky {
		test.Fatal("the wrong sky behind a specimen", studio.world.Sky,
			staged.world.Sky)
	}
	more := len(staged.world.Lamps) - len(studio.world.Lamps)
	if more != len(data.Staging.Sources)-len(plan.Sources) {
		test.Fatal("the studio's lamps on a staged specimen",
			len(staged.world.Lamps), len(studio.world.Lamps))
	}
}

// A plan that asks for its studio lights a staged entry by the studio:
// the plan's sky behind it and the plan's lamps on it, as an entry with
// no staging has. The anti-test is the one above, where the plan does
// not ask and the staging wins.
func TestAPlanThatAsksForItsStudioLightsAStagedEntryByIt(test *testing.T) {
	data := readStaging(test)
	plan := defaultPlan(test)
	plan.Seed = data.Seed
	bare, err := plan.construct(entry{Kind: data.Kind}, data.Detail)
	if err != nil {
		test.Fatal(err)
	}
	plan.Studio = true
	asked, err := plan.construct(entry{Kind: data.Kind,
		Staging: &data.Staging}, data.Detail)
	if err != nil {
		test.Fatal(err)
	}
	if asked.world.Sky != plan.Sky {
		test.Fatal("the staging's sky behind a specimen whose plan asked "+
			"for the studio", asked.world.Sky)
	}
	if len(asked.world.Lamps) != len(bare.world.Lamps) {
		test.Fatal("the staging's lamps on a specimen whose plan asked for "+
			"the studio", len(asked.world.Lamps), len(bare.world.Lamps))
	}
}

func TestTheMenuRefusesAStagedLampThatLiesNoWay(test *testing.T) {
	data := readStaging(test)
	made, err := parseMenu("refused.yaml", []byte(data.Refused))
	if made != nil || !errors.Is(err, emit.ErrDistant) ||
		!strings.Contains(err.Error(), data.Webb.Kind) {
		test.Fatal("took a staged lamp that lies no way", made, err)
	}
}

func TestWebbIsStagedWithTheSunAtTheEdgeOfItsFieldOfRegard(test *testing.T) {
	data := readStaging(test)
	menu, err := readMenu()
	if err != nil {
		test.Fatal(err)
	}
	var webb *staging
	for _, each := range menu.Entries {
		if each.Kind == data.Webb.Kind {
			webb = each.Staging
		}
	}
	if webb == nil || len(webb.Sources) != data.Webb.Suns {
		test.Fatal("webb is not staged with one sun", webb)
	}
	sun := webb.Sources[0].Toward
	fromSight := degreesBetween(sun, data.Webb.Sight)
	fromShield := degreesBetween(sun, data.Webb.Shield)
	if math.Abs(fromSight-data.Webb.FromSight) > data.Webb.Within ||
		math.Abs(fromShield-data.Webb.FromShield) > data.Webb.Within {
		test.Fatal("webb's sun is not at the edge, on the shield's side",
			fromSight, fromShield)
	}
}
