// Command cavitation writes bounded bubble and propeller-field scene blocks.
package main

import (
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/puzzlemouth/verb"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	yaml "gopkg.in/yaml.v3"
)

var build, built, who, sessions string

// main runs the tool and exits with the status it returns.
func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

// run registers the verbs and dispatches the arguments to one of them.
func run(args []string, out, errs io.Writer) int {
	tool := pzm.New("cavitation", "bounded authored Alfa gas-sphere placements",
		pzm.Build(build), pzm.Built(built), pzm.Who(who), pzm.Sessions(sessions))
	facts := verb.NewFacts(verb.WritesNothing, verb.LeavesAlone,
		verb.ThisMachine, verb.NoOne, verb.Light)
	tool.Verb("cloud", verb.NoShort, "write a seeded scene sphere block",
		"Emit map-space spheres using a deterministic native constructor. These are authored gas inclusions, not solved cavitation or water transport.",
		"Example_cloud", facts, cloud,
		pzm.Value("count", verb.NoShort, "sphere count, 0..512; default 32"),
		pzm.Value("seed", verb.NoShort, "deterministic seed; default 7052026"),
		pzm.Value("speed-mps", verb.NoShort, "reference boat speed; default 20"),
		pzm.Value("trail-seconds", verb.NoShort, "authored trail duration; default 1.4"),
		pzm.Value("rise-mps", verb.NoShort, "chosen rise rate; default 3"),
		pzm.Value("length-m", verb.NoShort, "craft length; default 81.4"),
		pzm.Value("screw-aft-m", verb.NoShort, "screw distance aft of bow; default 79.6"),
		pzm.Value("screw-offset-m", verb.NoShort, "gap after screw; default 0.8"),
		pzm.Value("screw-up-m", verb.NoShort, "screw axis above keel; default 4.75"),
		pzm.Value("spread-m", verb.NoShort, "maximum lateral spread; default 3.4"),
		pzm.Value("radius-min-m", verb.NoShort, "minimum sphere radius; default 0.18"),
		pzm.Value("radius-max-m", verb.NoShort, "maximum sphere radius; default 0.6"),
		pzm.Value("east-m", verb.NoShort, "craft map east coordinate; default -1500"),
		pzm.Value("north-m", verb.NoShort, "craft map north coordinate; default 0"),
		pzm.Value("keel-up-m", verb.NoShort, "keel map up coordinate; default -18.24"),
		pzm.Value("heading-deg", verb.NoShort, "clockwise from north; default 340"))
	tool.Verb("field", verb.NoShort, "write an opt-in propeller-tip vapor field",
		"Emit one bounded analytic density field. RPM, pressure deficit and recovery are chosen shot controls; they do not establish Alfa cavitation onset or solve fluid motion.",
		"Example_field", facts, field,
		pzm.Value("east-m", verb.NoShort, "propeller-axis map east; Alfa preset -1486.6954"),
		pzm.Value("north-m", verb.NoShort, "propeller-axis map north; Alfa preset -36.554"),
		pzm.Value("up-m", verb.NoShort, "propeller-axis map up; Alfa preset -55.25"),
		pzm.Value("axis-east", verb.NoShort, "downstream map east component; default 0.342"),
		pzm.Value("axis-north", verb.NoShort, "downstream map north component; default -0.94"),
		pzm.Value("axis-up", verb.NoShort, "downstream map up component; default 0"),
		pzm.Value("tip-radius-m", verb.NoShort, "chosen Alfa model radius; default 2.1"),
		pzm.Value("blades", verb.NoShort, "propeller blade count, 1..16; Alfa default 5"),
		pzm.Value("rpm", verb.NoShort, "chosen phase rate; default 300"),
		pzm.Value("slipstream-mps", verb.NoShort, "chosen convection speed; default 20"),
		pzm.Value("length-m", verb.NoShort, "maximum downstream field length; default 40"),
		pzm.Value("core-radius-m", verb.NoShort, "chosen filament core; default 0.2"),
		pzm.Value("density-per-m", verb.NoShort, "chosen density coefficient; default 0.8"),
		pzm.Value("fragment-scale-m", verb.NoShort, "advected cavity breakup scale; default 0.5"),
		pzm.Value("fragment-contrast", verb.NoShort, "density breakup0..1; default0 leaves continuous cores"),
		pzm.Value("fragment-seed", verb.NoShort, "deterministic breakup seed; default705"),
		pzm.Value("pressure-deficit-pa", verb.NoShort, "chosen visual pressure margin; default 12000"),
		pzm.Value("recovery-pa-per-m", verb.NoShort, "chosen recovery gradient; default 600"))
	return tool.RunWith(args, os.Stdin, out, errs)
}

type vaporCavityRow struct {
	Blades           int     `yaml:"blades"`
	FragmentScale    float64 `yaml:"fragment-scale-m"`
	FragmentContrast float64 `yaml:"fragment-contrast"`
	FragmentSeed     uint64  `yaml:"fragment-seed"`
	Colour           struct {
		SrgbLinear [3]float64 `yaml:"srgb_linear"`
	} `yaml:"colour"`
	Origin          [3]float64 `yaml:"origin"`
	Axis            [3]float64 `yaml:"axis"`
	TipRadius       float64    `yaml:"tip-radius"`
	RPM             float64    `yaml:"rpm"`
	Slipstream      float64    `yaml:"slipstream-mps"`
	Length          float64    `yaml:"length-m"`
	CoreRadius      float64    `yaml:"core-radius-m"`
	Density         float64    `yaml:"density-per-m"`
	PressureDeficit float64    `yaml:"pressure-deficit-pa"`
	Recovery        float64    `yaml:"recovery-pa-per-m"`
	VisualOnset     bool       `yaml:"visual-onset"`
	ClearWater      bool       `yaml:"clear-water"`
	Source          string     `yaml:"source"`
}

type vaporCavityDocument struct {
	Fields map[string]vaporCavityRow `yaml:"vapor-cavities"`
}

// field writes the vapor-cavity block, with each value defaulted.
func field(call pzm.Call) error {
	if len(call.Args()) != 0 {
		return errors.New("field takes values only; positional arguments are not accepted")
	}
	defaults := map[string]float64{
		"blades":           5,
		"fragment-scale-m": .5, "fragment-contrast": 0, "fragment-seed": 705,
		"east-m": -1486.6954, "north-m": -36.554, "up-m": -55.25,
		"axis-east": 0.342, "axis-north": -0.94, "axis-up": 0,
		"tip-radius-m": 2.1, "rpm": 300, "slipstream-mps": 20,
		"length-m": 40, "core-radius-m": 0.2, "density-per-m": 0.8,
		"pressure-deficit-pa": 12000, "recovery-pa-per-m": 600,
	}
	values := make(map[string]float64, len(defaults))
	for name, fallback := range defaults {
		parsed, err := value(call, name, fallback)
		if err != nil {
			return err
		}
		values[name] = parsed
	}
	if math.Trunc(values["blades"]) != values["blades"] {
		return errors.New("blades takes a whole number from 1 to 16")
	}
	if values["fragment-seed"] < 0 || values["fragment-seed"] > 1<<53 ||
		math.Trunc(values["fragment-seed"]) != values["fragment-seed"] {
		return errors.New("fragment-seed takes a whole number from 0 to 2^53")
	}
	if values["fragment-scale-m"] < .001 {
		return errors.New("fragment-scale-m takes a number from 0.001 to 20 m")
	}
	_, err := medium.NewPropellerCavitation(medium.PropellerCavitationConfig{
		Blades:           int(values["blades"]),
		FragmentScaleM:   values["fragment-scale-m"],
		FragmentContrast: values["fragment-contrast"],
		FragmentSeed:     uint64(values["fragment-seed"]),
		Origin:           maths.V(values["east-m"], values["up-m"], -values["north-m"]),
		Axis:             maths.V(values["axis-east"], values["axis-up"], -values["axis-north"]),
		TipRadius:        values["tip-radius-m"], RPM: values["rpm"],
		SlipstreamMPS: values["slipstream-mps"], LengthM: values["length-m"],
		CoreRadiusM: values["core-radius-m"], DensityPerM: values["density-per-m"],
		PressureDeficitPa: values["pressure-deficit-pa"],
		RecoveryPaPerM:    values["recovery-pa-per-m"], VisualOnset: true,
	})
	if err != nil {
		return err
	}
	row := vaporCavityRow{Colour: struct {
		SrgbLinear [3]float64 `yaml:"srgb_linear"`
	}{SrgbLinear: [3]float64{0.82, 0.84, 0.86}},
		Blades:           int(values["blades"]),
		FragmentScale:    values["fragment-scale-m"],
		FragmentContrast: values["fragment-contrast"],
		FragmentSeed:     uint64(values["fragment-seed"]),
		Origin:           [3]float64{values["east-m"], values["north-m"], values["up-m"]},
		Axis:             [3]float64{values["axis-east"], values["axis-north"], values["axis-up"]},
		TipRadius:        values["tip-radius-m"], RPM: values["rpm"],
		Slipstream: values["slipstream-mps"], Length: values["length-m"],
		CoreRadius: values["core-radius-m"], Density: values["density-per-m"],
		PressureDeficit: values["pressure-deficit-pa"],
		Recovery:        values["recovery-pa-per-m"], VisualOnset: true,
		ClearWater: true, Source: "eye"}
	raw, err := yaml.Marshal(vaporCavityDocument{Fields: map[string]vaporCavityRow{"main-screw": row}})
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(call.Out(), "# Propeller-tip density field; authored visual onset, not a cavitation prediction.\n", string(raw))
	return err
}

type sphereRow struct {
	Material string     `yaml:"material"`
	Middle   [3]float64 `yaml:"middle"`
	Radius   float64    `yaml:"radius"`
	Source   string     `yaml:"source"`
}

type sphereDocument struct {
	Spheres map[string]sphereRow `yaml:"spheres"`
}

// cloud writes a seeded block of authored gas spheres.
func cloud(call pzm.Call) error {
	if len(call.Args()) != 0 {
		return errors.New("cloud takes values only; positional arguments are not accepted")
	}
	config := manufacture.CavitationConfig{Count: 32, Seed: 7052026,
		SpeedMPS: 20, TrailSeconds: 1.4, RiseMPS: 3, LengthM: 81.4,
		ScrewAftM: 79.6, ScrewOffsetM: 0.8, SpreadM: 3.4,
		ScrewUpM: 4.75, RadiusMinM: 0.18, RadiusMaxM: 0.6}
	pose := manufacture.CavitationPose{EastM: -1500, KeelUpM: -18.24,
		HeadingDegrees: 340}
	var err error
	for _, item := range []struct {
		name string
		to   *float64
		base float64
	}{
		{"speed-mps", &config.SpeedMPS, config.SpeedMPS},
		{"trail-seconds", &config.TrailSeconds, config.TrailSeconds},
		{"rise-mps", &config.RiseMPS, config.RiseMPS},
		{"length-m", &config.LengthM, config.LengthM},
		{"screw-aft-m", &config.ScrewAftM, config.ScrewAftM},
		{"screw-offset-m", &config.ScrewOffsetM, config.ScrewOffsetM},
		{"screw-up-m", &config.ScrewUpM, config.ScrewUpM},
		{"spread-m", &config.SpreadM, config.SpreadM},
		{"radius-min-m", &config.RadiusMinM, config.RadiusMinM},
		{"radius-max-m", &config.RadiusMaxM, config.RadiusMaxM},
		{"east-m", &pose.EastM, pose.EastM},
		{"north-m", &pose.NorthM, pose.NorthM},
		{"keel-up-m", &pose.KeelUpM, pose.KeelUpM},
		{"heading-deg", &pose.HeadingDegrees, pose.HeadingDegrees},
	} {
		*item.to, err = value(call, item.name, item.base)
		if err != nil {
			return err
		}
	}
	if call.Given("count") {
		config.Count, err = strconv.Atoi(call.Value("count"))
		if err != nil {
			return fmt.Errorf("--count needs a whole number: %w", err)
		}
	}
	if call.Given("seed") {
		config.Seed, err = strconv.ParseInt(call.Value("seed"), 10, 64)
		if err != nil {
			return fmt.Errorf("--seed needs a signed integer: %w", err)
		}
	}
	particles, err := manufacture.NewCavitationCloud(config, pose)
	if err != nil {
		return err
	}
	document := sphereDocument{Spheres: make(map[string]sphereRow, len(particles))}
	for index, particle := range particles {
		document.Spheres[fmt.Sprintf("cavitation-%02d", index+1)] = sphereRow{
			Material: "air-bubble",
			Middle:   [3]float64{round3(particle.EastM), round3(particle.NorthM), round3(particle.UpM)},
			Radius:   round3(particle.RadiusM),
			Source:   "eye",
		}
	}
	raw, err := yaml.Marshal(document)
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(call.Out(), "# Authored scene geometry; this distribution is not a fluid simulation.\n", string(raw))
	return err
}

// value reads a finite number from the named flag, or the fallback when
// the flag is not given.
func value(call pzm.Call, name string, fallback float64) (float64, error) {
	if !call.Given(name) {
		return fallback, nil
	}
	parsed, err := strconv.ParseFloat(call.Value(name), 64)
	if err != nil || math.IsNaN(parsed) || math.IsInf(parsed, 0) {
		return 0, fmt.Errorf("--%s needs a finite number", name)
	}
	return parsed, nil
}

// round3 rounds value to three decimal places.
func round3(value float64) float64 { return math.Round(value*1000) / 1000 }
