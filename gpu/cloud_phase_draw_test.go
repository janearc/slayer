//go:build draw && darwin

package gpu

import (
	"encoding/binary"
	"math"
	"testing"
	"time"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
)

type cloudPhaseChoice struct {
	g        float64
	explicit bool
}

func phaseCloudWorld(t *testing.T, waterG float64,
	choices []cloudPhaseChoice, blocker float64,
) (*thing.World, []trace.InitialMedium, maths.Subspace) {
	t.Helper()
	water, err := thing.NewClearMediumWithScattering("independent cloud phase",
		maths.RGB(.025, .015, .008), maths.RGB(.012, .018, .02),
		waterG, 1, 16, 20)
	if err != nil {
		t.Fatal(err)
	}
	field, err := shape.NewHeightField([]float64{0, 0, 0, 0}, 2, 200)
	if err != nil {
		t.Fatal(err)
	}
	moon, err := emit.NewDistant(maths.V(.6, .8, 0), maths.RGB(.03, .04, .05))
	if err != nil {
		t.Fatal(err)
	}
	world := thing.NewWorld()
	world.Sky = sky.Gradient{}
	boundary := world.Add(thing.Thing{Shape: field,
		Material: &thing.Material{Through: 1, Gloss: 1,
			Index: 1.333, ClearMedium: water}})
	world.Lamps = []thing.Lamp{moon}
	world.OceanSurface = &thing.OceanSurface{Boundary: boundary,
		Mean: maths.Vec{}, ExtentMetres: 200, MetresPerUnit: 1,
		DirectMoonMeanPlane: true, SunRoute: thing.OceanSunRouteMoon,
		SunLampIndex: 0, FillLampIndex: -1,
		ClearCloudWaterStepMode: "ray-spacing"}
	for i, choice := range choices {
		cloud, err := medium.NewClearCloud(propellerFixture(t, 5))
		if err != nil {
			t.Fatal(err)
		}
		if choice.explicit {
			cloud, err = medium.NewClearCloudWithHenyeyGreenstein(
				cloud.DensityField(), choice.g)
			if err != nil {
				t.Fatal(err)
			}
		}
		origin := maths.V(0, -6, -3+float64(i)*3)
		world.AddVolume(thing.Volume{Medium: cloud,
			Material: &thing.Material{Colour: maths.Grey(.9)},
			Motion:   thing.Still(maths.NewSubspace(origin, maths.Identity, 1))})
	}
	if blocker != 0 {
		world.Add(thing.Thing{Shape: shape.Plane{},
			Material: &thing.Material{Colour: maths.Grey(.2)},
			Motion: thing.Still(maths.NewSubspace(maths.V(0, blocker, 0),
				maths.Identity, 1))})
	}
	initial := []trace.InitialMedium{{Index: 1.333, ClearMedium: water}}
	eye := opticalEye(maths.V(.9, -6, -8), maths.V(0, 0, 1))
	return world, initial, eye
}

func TestMetalCloudPhaseIndependentMultipleVolumesAndBlockers(t *testing.T) {
	legacyBlue := float32(0)
	for _, row := range []struct {
		name    string
		water   float64
		choices []cloudPhaseChoice
		blocker float64
	}{
		{"omitted_rayleigh", 0, []cloudPhaseChoice{{0, false}, {0, false}}, 0},
		{"explicit_isotropic", 0, []cloudPhaseChoice{{0, true}, {0, true}}, 0},
		{"different_cloud_laws", 0, []cloudPhaseChoice{{.8, true}, {-.3, true}}, 0},
		{"omitted_HG_water", .8, []cloudPhaseChoice{{0, false}, {0, false}}, 0},
		{"water_leg_blocker", 0, []cloudPhaseChoice{{.8, true}, {0, true}}, -1},
		{"air_leg_blocker", 0, []cloudPhaseChoice{{.8, true}, {0, true}}, 2},
	} {
		t.Run(row.name, func(t *testing.T) {
			world, initial, eye := phaseCloudWorld(t, row.water,
				row.choices, row.blocker)
			cpu, err := trace.SampledWithInitialMedia(world, eye,
				film.Pinhole{}, 0, 1, 1, 1, initial)
			if err != nil {
				t.Fatal(err)
			}
			got, profile, err := SampledWithInitialMedia(Software, world,
				eye, film.Pinhole{}, 0, 1, 1, 1, initial)
			if err != nil {
				t.Fatal(err)
			}
			a, b := cpu.At(0, 0), got.At(0, 0)
			t.Logf("%q CPU%v Metal%v", profile.Device, a, b)
			pairs := [][2]float32{{a.R, b.R}, {a.G, b.G}, {a.B, b.B}}
			for channel, pair := range pairs {
				limit := math.Abs(float64(pair[0]))*.003 + 1e-9
				if math.Abs(float64(pair[0]-pair[1])) > limit {
					t.Fatalf("channel%d CPU%g Metal%g", channel,
						pair[0], pair[1])
				}
			}
			if row.blocker != 0 && (b.R != 0 || b.G != 0 || b.B != 0) {
				t.Fatal("cloud phase bypassed opaque source visibility")
			}
			if row.name == "omitted_rayleigh" {
				legacyBlue = b.B
			}
			if row.name == "explicit_isotropic" && b.B == legacyBlue {
				t.Fatal("explicit HG0 inherited water Rayleigh")
			}
			encoded, err := encodeKeptWithInitialMedia(world, nil, eye,
				film.Pinhole{}, 0, 1, 1, nil, initial)
			if err != nil {
				t.Fatal(err)
			}
			for i, choice := range row.choices {
				base := i * volumeStruct.size()
				air := base + fieldOffset(volumeStruct, "air") + 4
				scatter := base + fieldOffset(volumeStruct, "scatter") + 12
				flag := binary.LittleEndian.Uint32(encoded.volumes.bytes[air:])
				bits := binary.LittleEndian.Uint32(encoded.volumes.bytes[scatter:])
				g := math.Float32frombits(bits)
				if (flag != 0) != choice.explicit ||
					choice.explicit && g != float32(choice.g) {
					t.Fatalf("cloud%d packed override(%g,%d) want%+v",
						i, g, flag, choice)
				}
			}
		})
	}
}

func TestMetalCloudPhaseZeroDensityKeepsWaterExact(t *testing.T) {
	world, initial, eye := phaseCloudWorld(t, 0,
		[]cloudPhaseChoice{{.8, true}}, 0)
	config := propellerFixture(t, 5).Parameters()
	config.VisualOnset = false
	field, err := medium.NewPropellerCavitation(config)
	if err != nil {
		t.Fatal(err)
	}
	cloud, err := medium.NewClearCloudWithHenyeyGreenstein(field, .8)
	if err != nil {
		t.Fatal(err)
	}
	world.Volumes[0].Medium = cloud
	zero, _, err := SampledWithInitialMedia(Software, world, eye,
		film.Pinhole{}, 0, 1, 1, 1, initial)
	if err != nil {
		t.Fatal(err)
	}
	world.Volumes = nil
	bare, _, err := SampledWithInitialMedia(Software, world, eye,
		film.Pinhole{}, 0, 1, 1, 1, initial)
	if err != nil {
		t.Fatal(err)
	}
	if zero.At(0, 0) != bare.At(0, 0) {
		t.Fatal("explicit cloud phase duplicated or changed bare water source")
	}
}

func TestCloudPhaseEncoderRefusesRoundedEndpoints(t *testing.T) {
	for _, g := range []float64{math.Nextafter(1, 0), math.Nextafter(-1, 0)} {
		world, initial, eye := phaseCloudWorld(t, 0,
			[]cloudPhaseChoice{{g, true}}, 0)
		if _, err := encodeKeptWithInitialMedia(world, nil, eye,
			film.Pinhole{}, 0, 1, 1, nil, initial); err == nil {
			t.Fatal("double-valid cloud HG reached invalid GPU endpoint")
		}
	}
}

func TestMetalCloudPhaseWrappedFizzMatchesCPUWithoutBaking(t *testing.T) {
	world, initial, eye := phaseCloudWorld(t, 0,
		[]cloudPhaseChoice{{.8, true}}, 0)
	fizz := medium.Fizz{Radius: 1, Height: 4.5, Thick: .05,
		Rise: 1, Churn: .8, Seed: 705}
	cloud, err := medium.NewClearCloudWithHenyeyGreenstein(fizz, 0)
	if err != nil {
		t.Fatal(err)
	}
	world.AddVolume(thing.Volume{Medium: cloud,
		Material: &thing.Material{Colour: maths.Grey(.9)},
		Motion:   thing.Still(maths.Subspace{Origin: maths.V(0, -7, -3)})})
	for _, at := range []time.Duration{0, 700 * time.Millisecond} {
		cpu, err := trace.SampledWithInitialMedia(world, eye,
			film.Pinhole{}, at, 1, 1, 1, initial)
		if err != nil {
			t.Fatal(err)
		}
		got, profile, err := SampledWithInitialMedia(Software, world,
			eye, film.Pinhole{}, at, 1, 1, 1, initial)
		if err != nil {
			t.Fatal(err)
		}
		a, b := cpu.At(0, 0), got.At(0, 0)
		t.Logf("%q at%s wrapped Fizz CPU%v Metal%v", profile.Device, at, a, b)
		pairs := [][2]float32{{a.R, b.R}, {a.G, b.G}, {a.B, b.B}}
		for channel, pair := range pairs {
			if math.Abs(float64(pair[0]-pair[1])) >
				math.Abs(float64(pair[0]))*.003+1e-9 {
				t.Fatalf("Fizz channel%d CPU%g Metal%g",
					channel, pair[0], pair[1])
			}
		}
		encoded, err := encodeKeptWithInitialMedia(world, nil, eye,
			film.Pinhole{}, at, 1, 1, nil, initial)
		if err != nil {
			t.Fatal(err)
		}
		if encoded.boxedCPU.Cells != 0 ||
			len(encoded.boxes.bytes) != 4*(len(propellerFields)+len(fizzFields)) {
			t.Fatal("mixed propeller and wrapped Fizz acquired a density grid")
		}
	}
}
