//go:build draw && darwin

package gpu

import (
	"encoding/binary"
	"errors"
	"math"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/scene/gas"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// the numbers of the test gas, per metre at unit density: it absorbs the
// same at every wavelength, and scatters by rayleigh's fourth power from
// 550 nm, so its extinction is not its scatter and neither is flat.
const (
	duskAbsorb  = 0.08
	duskScatter = 0.25
	duskPivot   = 550.0
)

// dusk is a gas of the test's numbers with a phase asymmetry of its own.
func dusk(asymmetry float64) gas.Gas {
	return gas.Gas{Name: "dusk", Phase: asymmetry,
		Visible: func(nm float64) (absorb, scatter float64) {
			ratio := duskPivot / nm
			return duskAbsorb, duskScatter * math.Pow(ratio, 4)
		}}
}

// sunlit is a lamp far away along a way, of a light.
func sunlit(toward maths.Vec, light maths.Light) film.Sun {
	return film.Sun{Toward: toward, Strength: light}
}

// puffOf is a gas over a puff of a radius at the world's origin, thick
// at its middle, in the gas's own units of one metre.
func puffOf(air gas.Gas, radius, thick float64) thing.Volume {
	return thing.Volume{Medium: gas.Of{Shape: medium.Puff{Radius: radius,
		Thick: thick}, In: air, Unit: 1}}
}

// gasEye looks at the origin from twelve units along z.
func gasEye() maths.Subspace {
	from := maths.V(0, 0, 12)
	return maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.Vec{})}
}

// gridded is the tolerance a grid of cells is held to, the one the baked
// hall and the fizzing glass are held to: the kernel reads between cells
// where the tracer asks the medium itself.
var gridded = tolerance{mean: 3e-4, largest: 1e-3}

// a rayleigh gas lit from the side: the phase function's even lobe and
// the blue gas's greater extinction both show.
func TestTheGPUDrawsARayleighGasAsSlayerDoes(t *testing.T) {
	world := thing.NewWorld()
	world.AddVolume(puffOf(dusk(0), 3, 1))
	world.Lamp(sunlit(maths.V(1, 0.2, 0.1), maths.RGB(5, 4.5, 4)))
	against(t, world, gasEye())
}

// a gas that scatters forward, lit from behind: the light turns hardly at
// all on its way to the eye.
func TestTheGPUDrawsAForwardScatteringGasAsSlayerDoes(t *testing.T) {
	world := thing.NewWorld()
	world.AddVolume(puffOf(dusk(0.6), 3, 1))
	world.Lamp(sunlit(maths.V(0.15, 0.1, -1), maths.RGB(5, 4.5, 4)))
	against(t, world, gasEye())
}

// a gas lit only by a lamp from everywhere, taken evenly.
func TestTheGPUDrawsAGasUnderAFloodAsSlayerDoes(t *testing.T) {
	world := thing.NewWorld()
	world.AddVolume(puffOf(dusk(0), 3, 1))
	world.Lamp(film.Flood{Strength: maths.RGB(2, 1.8, 1.5)})
	against(t, world, gasEye())
}

// ground is a large dull ball, its top a few units under the origin.
func ground() thing.Thing {
	return thing.Thing{Shape: shape.Sphere{Radius: 60},
		Material: &thing.Material{Name: "ground", Colour: maths.Grey(0.7)},
		Motion:   at(0, -63, 0)}
}

// a gas in front of a lit surface: the surface is seen through the gas,
// and the light that reaches it comes through the gas too.
func TestTheGPUDrawsAGasBeforeASurfaceAsSlayerDoes(t *testing.T) {
	world := thing.NewWorld()
	world.Add(ground())
	world.AddVolume(puffOf(dusk(0), 4, 1))
	world.Lamp(sunlit(maths.V(0.4, 1, 0.3), maths.RGB(5, 4.5, 4)))
	from := maths.V(0, 2, 12)
	eye := maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.V(0, -1, 0))}
	against(t, world, eye)
}

// a gas and an ordinary puff that overlap on a ray, each marched in turn
// by how far along it each step is.
func TestTheGPUDrawsAGasAndAPuffTogetherAsSlayerDoes(t *testing.T) {
	world := thing.NewWorld()
	world.AddVolume(puffOf(dusk(0.3), 3, 1))
	haze := &thing.Material{Name: "haze", Colour: maths.RGB(0.9, 0.8, 0.7)}
	world.AddVolume(thing.Volume{Medium: medium.Puff{Radius: 3, Thick: 0.4},
		Material: haze, Motion: at(1.5, 0, 1)})
	world.Lamp(sunlit(maths.V(1, 0.4, 0.5), maths.RGB(5, 4.5, 4)))
	against(t, world, gasEye())
}

// a gas over a wispy puff, which the kernel asks itself as the tracer does.
func TestTheGPUDrawsAWispyGasAsSlayerDoes(t *testing.T) {
	world := thing.NewWorld()
	wisps := medium.Wispy{Of: medium.Puff{Radius: 3, Thick: 1}, Seed: 3,
		Scale: 1.5, Amount: 0.5}
	world.AddVolume(thing.Volume{Medium: gas.Of{Shape: wisps,
		In: dusk(0), Unit: 1}})
	world.Lamp(sunlit(maths.V(1, 0.2, 0.1), maths.RGB(5, 4.5, 4)))
	against(t, world, gasEye())
}

// ramp is a medium the kernel cannot ask, a block of three units a side
// whose density rises in a straight line along x from a tenth at one face
// to nine tenths at the other. A grid of cells reads between its cells in
// straight lines, so a straight ramp is what it draws exactly; the
// tracer's own error is in the edge alone.
type ramp struct{}

// rampHalf is half the side of the ramp's block.
const rampHalf = 1.5

// Density is a tenth at the face on minus x, rising to nine tenths at the
// face on plus x, and nought outside the block.
func (ramp) Density(at maths.Vec) float64 {
	if math.Abs(at.X) > rampHalf || math.Abs(at.Y) > rampHalf ||
		math.Abs(at.Z) > rampHalf {
		return 0
	}
	return 0.5 + 0.4*at.X/rampHalf
}

// Bounds is the block.
func (ramp) Bounds() maths.Box {
	return maths.Box{Min: maths.V(-rampHalf, -rampHalf, -rampHalf),
		Max: maths.V(rampHalf, rampHalf, rampHalf)}
}

// a gas the kernel does not ask itself is sampled into a box, as any
// medium is, and read between its cells.
func TestTheGPUDrawsABoxedGasAsSlayerDoes(t *testing.T) {
	world := thing.NewWorld()
	world.AddVolume(thing.Volume{Medium: gas.Of{Shape: ramp{},
		In: dusk(0), Unit: 1}})
	world.Lamp(sunlit(maths.V(1, 0.2, 0.1), maths.RGB(5, 4.5, 4)))
	within(t, world, gasEye(), 96, 64, 2, gridded)
}

// a glowing surface seen through the middle of a gas puff of radius 2,
// thick 1 at its middle, is dimmed by e to the minus the gas's extinction
// times the puff's optical depth along the chord, in each channel.
//
// The density falls from thick at the middle as one less smoothstep of
// the share of the radius, so along a chord through the middle it is
// thick * (1 - (3u^2 - 2u^3)) with u the share, whose integral over u
// from 0 to 1 is 1 - (1 - 1/2) = 1/2. The chord is two radii, of two
// halves, so the optical depth is radius * thick * extinction = 2 *
// extinction. The gas's extinction at 610, 550 and 465 nm is its absorb
// of 0.08 plus its scatter of 0.25 * (550/nm)^4: 0.24522, 0.33 and
// 0.56930, so the surface keeps exp(-0.49045) = 0.61235 of its red,
// exp(-0.66) = 0.51685 of its green and exp(-1.13861) = 0.32026 of its
// blue. No lamp lights the gas, so it adds nothing of its own.
func TestAGlowSeenThroughAGasIsDimmedByBeersLaw(t *testing.T) {
	world := thing.NewWorld()
	world.AddVolume(puffOf(dusk(0), 2, 1))
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 30},
		Material: &thing.Material{Name: "glow", Glow: maths.Grey(1)},
		Motion:   at(0, 0, -40)})
	lens := film.Pinhole{Wide: 50 * maths.Degree}
	drawn, err := Sampled(world, gasEye(), lens, 0, 1, 1, 1)
	if errors.Is(err, metal.ErrNoDevice) {
		t.Skip("no gpu here")
	}
	if err != nil {
		t.Fatal(err)
	}
	got := drawn.At(0, 0)
	want := [3]float64{0.61235, 0.51685, 0.32026}
	for channel, light := range []float32{got.R, got.G, got.B} {
		if !(math.Abs(float64(light)-want[channel]) <= 1e-3*want[channel]) {
			t.Errorf("channel %d keeps %v of the glow, want %v", channel,
				light, want[channel])
		}
	}
}

// volumeRow is the numbers of a field of the first volume packed in a
// scene, four to a field.
func volumeRow(encoded *scene, name string) [4]float32 {
	var row [4]float32
	start := fieldOffset(volumeStruct, name)
	for part := range row {
		bits := binary.LittleEndian.Uint32(encoded.volumes.bytes[start+4*part:])
		row[part] = math.Float32frombits(bits)
	}
	return row
}

// a gas volume carries the gas's extinction at the eye's three bands in
// its tint, its scatter at them and its asymmetry beside, as the tracer's
// stage asks them: 610, 550 and 465 nm, read here from the gas by hand.
func TestAGasVolumeCarriesTheTracersNumbers(t *testing.T) {
	bands := []float64{610, 550, 465}
	for name, air := range map[string]gas.Gas{"air": gas.Air(),
		"haze": gas.Haze(), "dusk": dusk(0.6)} {
		world := thing.NewWorld()
		world.AddVolume(thing.Volume{Medium: gas.Of{
			Shape: medium.Puff{Radius: 1, Thick: 1}, In: air, Unit: 1000}})
		encoded := newScene()
		if err := encoded.volumesOf(world, 0); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		per := gas.Of{In: air, Unit: 1000}.Gas()
		tint, scatter := volumeRow(encoded, "tint"), volumeRow(encoded, "scatter")
		for index, nm := range bands {
			band := maths.Band{Nanometres: nm}
			wantTint := float32(per.Extinction(band))
			wantScatter := float32(per.Scatter(band))
			if !(tint[index] == wantTint) || !(scatter[index] == wantScatter) ||
				!(wantTint > 0) {
				t.Errorf("%s at %v nm: tint %v scatter %v, want %v and %v",
					name, nm, tint[index], scatter[index], wantTint,
					wantScatter)
			}
		}
		if got := scatter[3]; !(got == float32(air.Phase)) {
			t.Errorf("%s: asymmetry %v, want %v", name, got, air.Phase)
		}
		if flag := volumeRow(encoded, "air"); flag[0] == 0 {
			t.Errorf("%s: a gas volume is not flagged as one", name)
		}
	}
}

// a volume that is not a gas is not flagged as one, and keeps its tint.
func TestAPuffIsNotAGas(t *testing.T) {
	world := thing.NewWorld()
	world.AddVolume(thing.Volume{Medium: medium.Puff{Radius: 1, Thick: 1}})
	encoded := newScene()
	if err := encoded.volumesOf(world, 0); err != nil {
		t.Fatal(err)
	}
	if flag := volumeRow(encoded, "air"); flag[0] != 0 {
		t.Errorf("a puff is flagged as a gas: %v", flag)
	}
}
