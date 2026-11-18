package hillside

import (
	"encoding/binary"
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/airlight"
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
	"github.com/janearc/slayer/trace"
)

func groundlessOceanSunRead(t *testing.T) set.Set {
	t.Helper()
	text := strings.Replace(groundlessAirFixture,
		"      material: interface\n",
		"      material: interface\n      direct-sun: mean-plane\n", 1)
	text = strings.Replace(text,
		"clear: {through: 1, index: 1.333, gloss: 1, rank: 0}",
		"clear: {through: 1, index: 1.333, gloss: 1, rank: 0, extinction-per-metre: [0.1, 0.1, 0.1]}", 1)
	text = strings.Replace(text, "look_at: [0, 0, 0]",
		"look_at: [0, 0, -2]", 1)
	text = strings.Replace(text, "location: {at: [0, 12, 50]}",
		"location: {at: [0, -4, -2]}", 1)
	text = strings.Replace(text, "      title: view\n",
		"      title: view\n      initial-media: [interface]\n", 1)
	text = text[:strings.Index(text, "  set:\n")]
	return groundlessAirRead(t, text)
}

func tinyAiredFields(t *testing.T) *airlight.Fields {
	t.Helper()
	// FieldsFromBytes is the same checked ingress used by the cache. The
	// tiny table exercises the aired staging adapter; it is not a solved
	// atmospheric model.
	numbers := []float64{1, 1, 2, 1, 361, 20, 550, 1, 0,
		0.1, 0.1, 1, 1}
	for range 361 {
		numbers = append(numbers, 1)
	}
	raw := make([]byte, len(numbers)*8)
	for index, number := range numbers {
		binary.LittleEndian.PutUint64(raw[index*8:], math.Float64bits(number))
	}
	fields, err := airlight.FieldsFromBytes(raw)
	if err != nil {
		t.Fatal(err)
	}
	return fields
}

func injectedAiredFields(t *testing.T) *staging {
	t.Helper()
	stage := testStaging()
	stage.airFieldsProvider = func(string) (*airlight.Fields,
		airlight.Plan, cacheNote, error) {
		return tinyAiredFields(t), airlight.Plan{}, cacheNote{hit: true}, nil
	}
	return stage
}

func TestGroundlessAirOceanTagsAiredSunAndKeepsSkySeparate(t *testing.T) {
	read := groundlessOceanSunRead(t)
	stage := injectedAiredFields(t)
	stage.given.set = "groundless-ocean-air.yaml"
	if err := stage.place(read, set.Planet{}, false); err != nil {
		t.Fatal(err)
	}
	if len(stage.world.Lamps) != 1 {
		t.Fatalf("groundless aired lamps = %d, want exactly its one direct sun", len(stage.world.Lamps))
	}
	if _, ok := stage.world.Sky.(*airlight.Sky); !ok {
		t.Fatalf("aired sky was lost or treated as a lamp: %T", stage.world.Sky)
	}
	if err := stage.tagOceanDirectSun(read, 0); err != nil {
		t.Fatal(err)
	}
	if tag := stage.given.oceanSunTag; tag == nil || tag.sun != 0 || tag.fill != -1 || tag.route != thing.OceanSunRouteGroundlessAir {
		t.Fatalf("groundless-air sun tag = %+v, want lamp 0, no fill, groundless-air route", tag)
	}
	if err := stage.halved(read); err != nil {
		t.Fatal(err)
	}
	ocean := stage.world.OceanSurface
	if ocean == nil || ocean.SunRoute != thing.OceanSunRouteGroundlessAir || ocean.SunLampIndex != 0 || ocean.FillLampIndex != -1 {
		t.Fatalf("staged ocean source descriptor = %+v", ocean)
	}
	if stage.world.Sky != stage.airedSky {
		t.Fatal("world half replaced the exact air-computed sky instance")
	}
	eye, err := stage.eye(read)
	if err != nil {
		t.Fatal(err)
	}
	if eye.Origin != maths.V(0, -2, 4) ||
		eye.Forward() != maths.V(0, 0, -1) {
		t.Fatalf("staged underwater camera = origin %v forward %v; want (0,-2,4) toward (0,-2,0)",
			eye.Origin, eye.Forward())
	}
	sunToward := skyward(120, 35)
	if sunToward.Dot(maths.V(0, 0, 1)) <= 0 {
		t.Fatalf("sun direction %v does not illuminate the camera-facing sphere side", sunToward)
	}
	if len(stage.given.initialMedia) != 1 {
		t.Fatalf("groundless camera media = %+v, want the ocean medium", stage.given.initialMedia)
	}
	targetCenter, targetRadius := maths.V(0, -2, 2), 0.4
	towardTarget := targetCenter.Sub(eye.Origin)
	along := towardTarget.Dot(eye.Forward())
	miss := towardTarget.Sub(eye.Forward().Scale(along)).Len()
	if along <= 0 || miss >= targetRadius {
		t.Fatalf("camera ray misses submerged sphere: along %g, miss %g, radius %g",
			along, miss, targetRadius)
	}
	boundary := stage.world.OceanSurface.Boundary
	field, ok := boundary.Shape.(*shape.HeightField)
	if !ok {
		t.Fatalf("staged ocean boundary has type %T", boundary.Shape)
	}
	targetFront := targetCenter.Add(maths.V(0, 0, targetRadius))
	if depth := field.Distance(boundary.At(0).In(targetFront)); !(depth < 0) {
		t.Fatalf("camera-facing target surface is not inside staged water: signed depth %g", depth)
	}
	stage.world.Add(thing.Thing{Shape: shape.Sphere{Radius: targetRadius},
		Material: &thing.Material{Colour: maths.Grey(0.8)},
		Motion:   thing.Still(maths.Subspace{Origin: targetCenter})})
	source, ok := stage.world.Lamps[stage.world.OceanSurface.SunLampIndex].(emit.Distant)
	if !ok {
		t.Fatalf("staged source has type %T", stage.world.Lamps[stage.world.OceanSurface.SunLampIndex])
	}
	_, _, sourceLight := source.Light(maths.Vec{}, maths.Vec{})
	if !(sourceLight.G > 0) || math.IsNaN(sourceLight.G) || math.IsInf(sourceLight.G, 0) {
		t.Fatalf("synthetic 550 nm aired lamp has no finite green-channel power: %+v", sourceLight)
	}
	withSun, err := trace.RenderWithInitialMedia(stage.world, eye,
		film.Pinhole{Wide: 0.2}, 0, 1, 1, stage.given.initialMedia)
	if err != nil {
		t.Fatalf("CPU render of staged groundless-air ocean path: %v", err)
	}
	legacyWorld := *stage.world
	legacyOcean := *stage.world.OceanSurface
	legacyOcean.DirectSunMeanPlane = false
	legacyWorld.OceanSurface = &legacyOcean
	withoutOceanPath, err := trace.RenderWithInitialMedia(&legacyWorld, eye,
		film.Pinhole{Wide: 0.2}, 0, 1, 1, stage.given.initialMedia)
	if err != nil {
		t.Fatalf("CPU render with the ocean approximation disabled: %v", err)
	}
	with, without := withSun.At(0, 0), withoutOceanPath.At(0, 0)
	t.Logf("staged one-pixel render: transmitted green=%g; opaque-boundary legacy green=%g", with.G, without.G)
	if math.IsNaN(float64(with.R)) || math.IsInf(float64(with.R), 0) ||
		math.IsNaN(float64(without.R)) || math.IsInf(float64(without.R), 0) {
		t.Fatalf("non-finite staged render pixels: direct=%+v legacy=%+v", with, without)
	}
	if !(with.G > 0 && with.G > without.G) {
		t.Fatalf("transmitted sun should add positive green irradiance because the legacy shadow ray treats the ocean sheet as opaque: direct=%+v legacy=%+v", with, without)
	}
}

func TestGroundlessAirOceanRefusesAReplacedSky(t *testing.T) {
	read := groundlessOceanSunRead(t)
	stage := injectedAiredFields(t)
	stage.given.set = "groundless-ocean-air.yaml"
	if err := stage.place(read, set.Planet{}, false); err != nil {
		t.Fatal(err)
	}
	stage.world.Sky = sky.Gradient{}
	if err := stage.tagOceanDirectSun(read, 0); err == nil {
		t.Fatal("groundless-air path accepted a different sky after aired")
	}
}

func TestGroundlessAirOceanRejectsDuplicateOrWrongAiredSun(t *testing.T) {
	for _, name := range []string{"duplicate", "wrong"} {
		t.Run(name, func(t *testing.T) {
			read := groundlessOceanSunRead(t)
			stage := injectedAiredFields(t)
			stage.given.set = "groundless-ocean-air.yaml"
			if err := stage.place(read, set.Planet{}, false); err != nil {
				t.Fatal(err)
			}
			if name == "duplicate" {
				stage.world.Lamps = append(stage.world.Lamps, stage.world.Lamps[0])
			} else {
				sun := skyward(120, 35)
				wrong, err := emit.NewDistant(sun, maths.Grey(1e30))
				if err != nil {
					t.Fatal(err)
				}
				stage.world.Lamps[0] = wrong
			}
			if err := stage.tagOceanDirectSun(read, 0); err == nil {
				t.Fatalf("%s aired sun was accepted", name)
			}
		})
	}
}

func TestGroundlessAirWithoutOceanKeepsAiredLampAndSkyUnchanged(t *testing.T) {
	read := groundlessAirRead(t, groundlessAirFixture)
	stage := injectedAiredFields(t)
	stage.given.set = "groundless-air.yaml"
	if err := stage.place(read, set.Planet{}, false); err != nil {
		t.Fatal(err)
	}
	if len(stage.world.Lamps) != 1 || stage.world.Sky == nil {
		t.Fatalf("legacy aired stage changed: lamps=%d sky=%T", len(stage.world.Lamps), stage.world.Sky)
	}
	if err := stage.tagOceanDirectSun(read, 0); err != nil {
		t.Fatal(err)
	}
	if stage.given.oceanSunTag != nil {
		t.Fatalf("legacy stage gained ocean tag: %+v", stage.given.oceanSunTag)
	}
}
