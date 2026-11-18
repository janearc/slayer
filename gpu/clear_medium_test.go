package gpu

import (
	"encoding/binary"
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
)

func TestGPUOpticalPackingCarriesNamedAndInitialState(t *testing.T) {
	medium, err := thing.NewClearMedium("test water", maths.Grey(0.1), 1)
	if err != nil {
		t.Fatal(err)
	}
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Plane{}, Material: &thing.Material{
		Through: 1, Index: 1.33, Rank: 2, ClearMedium: medium,
	}})
	world.Add(thing.Thing{Shape: shape.Plane{}, Material: &thing.Material{
		Through: 1, Index: 1.33, Rank: 2, ClearMedium: medium,
	}})
	media := []trace.InitialMedium{{Index: 1.33, Rank: 2,
		ClearMedium: medium}}
	encoded, err := encodeKeptWithInitialMedia(world, nil, maths.Subspace{},
		film.Pinhole{}, 0, 1, 1, nil, media)
	if err != nil {
		t.Fatal(err)
	}
	if !encoded.opticalState || len(encoded.inputs()) != 19 {
		t.Fatalf("optical state = %v, input buffers = %d; want enabled and 19",
			encoded.opticalState, len(encoded.inputs()))
	}
	if got := int32(binary.LittleEndian.Uint32(encoded.materials.bytes[60:])); got != 0 {
		t.Fatalf("Material.info.w medium ID = %d, want 0", got)
	}
	if got := int32(binary.LittleEndian.Uint32(encoded.materials.bytes[materialStruct.size()+60:])); got != 0 {
		t.Fatalf("second boundary medium ID = %d, want shared ID 0", got)
	}
	if len(encoded.clearMedia.bytes) != clearMediumStruct.size() ||
		math.Float32frombits(binary.LittleEndian.Uint32(encoded.clearMedia.bytes[0:])) != 0.1 ||
		math.Float32frombits(binary.LittleEndian.Uint32(encoded.clearMedia.bytes[12:])) != 1 {
		t.Fatalf("packed optical table = %v, want extinction/scatter/profile rows",
			encoded.clearMedia.bytes)
	}
	if len(encoded.initialState.bytes) != 16+trace.MaxInitialMedia*16 {
		t.Fatalf("initial state bytes = %d, want %d", len(encoded.initialState.bytes),
			16+trace.MaxInitialMedia*16)
	}
	if count := binary.LittleEndian.Uint32(encoded.initialState.bytes); count != 1 {
		t.Fatalf("initial stack count = %d, want 1", count)
	}
	if got := int32(binary.LittleEndian.Uint32(encoded.initialState.bytes[20:])); got != 2 {
		t.Fatalf("initial rank = %d, want 2", got)
	}
	if got := int32(binary.LittleEndian.Uint32(encoded.initialState.bytes[24:])); got != 0 {
		t.Fatalf("initial medium ID = %d, want 0", got)
	}
	if got := math.Float32frombits(binary.LittleEndian.Uint32(encoded.initialState.bytes[16:])); got != 1.33 {
		t.Fatalf("initial index = %g, want float32(1.33)", got)
	}
}

func TestGPUOpticalPackingCarriesExplicitOceanSpotProfile(t *testing.T) {
	medium, err := thing.NewClearMediumWithScattering("spot water",
		maths.RGB(.02, .01, .01), maths.RGB(.01, .005, .002),
		.7, 1, 12, 8)
	if err != nil {
		t.Fatal(err)
	}
	field, err := shape.NewHeightField([]float64{0, 0, 0, 0}, 2, 20)
	if err != nil {
		t.Fatal(err)
	}
	boundary := &thing.Thing{Shape: field, Material: &thing.Material{
		Through: 1, Gloss: 1, Index: 1.333, ClearMedium: medium}}
	world := thing.NewWorld()
	world.Things = append(world.Things, boundary)
	world.Lamps = []thing.Lamp{film.Spot{At: maths.V(0, -2, 0),
		Aim: maths.V(0, -1, 0), Cone: 60 * maths.Degree,
		Strength: maths.Grey(20)}}
	world.OceanSurface = &thing.OceanSurface{Boundary: boundary,
		Mean: maths.Vec{}, ExtentMetres: 20, MetresPerUnit: 1,
		SpotInscatter: true}
	initial := []trace.InitialMedium{{Index: 1.333, ClearMedium: medium}}
	encoded, err := encodeKeptWithInitialMedia(world, nil, maths.Subspace{},
		film.Pinhole{}, 0, 1, 1, nil, initial)
	if err != nil {
		t.Fatal(err)
	}
	want := []float32{.03, .015, .012, 1, .01, .005, .002, .7, 12, 8}
	for index, value := range want {
		got := math.Float32frombits(binary.LittleEndian.Uint32(
			encoded.clearMedia.bytes[index*4:]))
		if got != value {
			t.Errorf("packed scattering optical value %d = %g, want %g",
				index, got, value)
		}
	}
}

func TestGPUOrdinaryPackingKeepsTheLegacyInputLayout(t *testing.T) {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Plane{}, Material: &thing.Material{
		Colour: maths.Grey(0.5),
	}})
	encoded, err := encode(world, maths.Subspace{}, film.Pinhole{}, 0, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if encoded.opticalState || len(encoded.inputs()) != 17 ||
		len(encoded.clearMedia.bytes) != 0 || len(encoded.initialState.bytes) != 0 {
		t.Fatalf("ordinary input layout changed: optical=%v inputs=%d media=%d initial=%d",
			encoded.opticalState, len(encoded.inputs()), len(encoded.clearMedia.bytes),
			len(encoded.initialState.bytes))
	}
	if got := binary.LittleEndian.Uint32(encoded.materials.bytes[60:]); got != 0 {
		t.Fatalf("legacy Material.info.w changed to %d", got)
	}
}

func TestGPURejectsClearCoefficientsLostInFloat32(t *testing.T) {
	medium, err := thing.NewClearMedium("unresolvable water",
		maths.RGB(math.MaxFloat64, 0, 0), 1)
	if err != nil {
		t.Fatal(err)
	}
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Plane{}, Material: &thing.Material{
		Through: 1, Index: 1.33, ClearMedium: medium,
	}})
	if _, err := encode(world, maths.Subspace{}, film.Pinhole{}, 0, 1, 1); err == nil ||
		!strings.Contains(err.Error(), "outside Metal float32 range") {
		t.Fatalf("unrepresentable optical coefficient was not refused: %v", err)
	}
}

func TestGPURefusesParticipatingClearMediaInsteadOfDroppingScattering(t *testing.T) {
	medium, err := thing.NewClearMediumWithScattering("scattering water",
		maths.Grey(0), maths.Grey(0.02), 0, 1, 4, 10)
	if err != nil {
		t.Fatal(err)
	}
	world := thing.NewWorld()
	media := []trace.InitialMedium{{Index: 1.33, ClearMedium: medium}}
	_, err = encodeKeptWithInitialMedia(world, nil, maths.Subspace{},
		film.Pinhole{}, 0, 1, 1, nil, media)
	if err == nil || !strings.Contains(err.Error(), "in-scattering") {
		t.Fatalf("GPU silently reduced an in-scattering initial medium to extinction: %v", err)
	}

	boundaryWorld := thing.NewWorld()
	boundaryWorld.Add(thing.Thing{Shape: shape.Plane{}, Material: &thing.Material{
		Through: 1, Index: 1.33, ClearMedium: medium,
	}})
	_, err = encodeKeptWithInitialMedia(boundaryWorld, nil, maths.Subspace{},
		film.Pinhole{}, 0, 1, 1, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "in-scattering") {
		t.Fatalf("GPU silently reduced a scattering boundary material to extinction: %v", err)
	}
}

func TestGPURefusesCPUOnlyClearCloudComposition(t *testing.T) {
	water, err := thing.NewClearMedium("test water", maths.Grey(.1), 1)
	if err != nil {
		t.Fatal(err)
	}
	cloud, err := medium.NewClearCloud(medium.Puff{Radius: 2, Thick: .5})
	if err != nil {
		t.Fatal(err)
	}
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Plane{}, Material: &thing.Material{
		Through: 1, Index: 1.33, ClearMedium: water,
	}})
	world.AddVolume(thing.NewVolume(cloud, nil))
	_, err = encodeKeptWithInitialMedia(world, nil, maths.Subspace{},
		film.Pinhole{}, 0, 1, 1, nil,
		[]trace.InitialMedium{{Index: 1.33, ClearMedium: water}})
	if err == nil || !strings.Contains(err.Error(), "local cloud volumes") {
		t.Fatalf("GPU did not refuse CPU-only mixed transport: %v", err)
	}
}

func TestGPUHiddenVolumesDoNotActivateClearCloudGuard(t *testing.T) {
	for _, named := range []bool{false, true} {
		world := thing.NewWorld()
		material := &thing.Material{Through: 1, Index: 1.333}
		if named {
			var err error
			material.ClearMedium, err = thing.NewClearMedium(
				"hidden-only water", maths.Grey(.1), 1)
			if err != nil {
				t.Fatal(err)
			}
		}
		world.Add(thing.Thing{Shape: shape.Plane{}, Material: material})
		world.AddVolume(thing.Volume{Medium: medium.Puff{Radius: 1},
			Attrs: thing.Attrs{Hidden: true}})
		encoded, err := encode(world, maths.Subspace{}, film.Pinhole{},
			0, 1, 1)
		if err != nil {
			t.Fatalf("named=%v: hidden cloud refused sea: %v", named, err)
		}
		if encoded.counts.volumes != 0 || encoded.opticalState != named {
			t.Fatalf("named=%v: volumes=%d optical=%v", named,
				encoded.counts.volumes, encoded.opticalState)
		}
	}
}

func TestGPUMixedVisibleCloudsDoNotDowngradeNamedWater(t *testing.T) {
	water, err := thing.NewClearMedium("mixed water", maths.Grey(.1), 1)
	if err != nil {
		t.Fatal(err)
	}
	cloud, err := medium.NewClearCloud(medium.Puff{Radius: 1})
	if err != nil {
		t.Fatal(err)
	}
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Plane{}, Material: &thing.Material{
		Through: 1, Index: 1.333, ClearMedium: water,
	}})
	world.AddVolume(thing.Volume{Medium: cloud})
	world.AddVolume(thing.Volume{Medium: medium.Puff{Radius: 1}})
	_, err = encode(world, maths.Subspace{}, film.Pinhole{}, 0, 1, 1)
	if err == nil || !strings.Contains(err.Error(), "explicitly opted-in") {
		t.Fatalf("mixed visible clouds bypassed optical validation: %v", err)
	}
}

func TestGPURefusesOceanMeanPlaneSunWithoutDrawing(t *testing.T) {
	world := thing.NewWorld()
	world.OceanSurface = &thing.OceanSurface{DirectSunMeanPlane: true}
	err := refuse(world)
	if err == nil || !strings.Contains(err.Error(),
		"CPU-only ocean mean-plane direct-sun transport") {
		t.Fatalf("GPU did not refuse the CPU-only authored ocean source path: %v", err)
	}
}
