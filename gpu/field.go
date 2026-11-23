//go:build darwin

package gpu

import (
	_ "embed"
	"fmt"
	"math"
	"strings"
	"time"

	kerr "github.com/janearc/celestial/hole"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
)

//go:embed field.metal
var fieldSource string

type fieldKernel interface {
	Grid(int, int, int, int, [][]byte) ([]float32, error)
	GPUTime() time.Duration
	Close()
}

type fieldOperations struct {
	seeds  func(*scene, fieldSeedInput) bool
	rows   int
	encode func(*thing.World, []grown, maths.Subspace, film.Sensor, time.Duration, int, int, ...int) (*scene, error)
	bake   func(*scene) (time.Duration, error)
	open   func(string, string) (fieldKernel, error)
}

type fieldSeedInput struct {
	source        *thing.Field
	black         *kerr.Hole
	space         maths.Subspace
	scale         float64
	eye           maths.Subspace
	lens          film.Sensor
	width, height int
}

// fieldDefaults binds transport stages to their production implementations.
func fieldDefaults() fieldOperations {
	return fieldOperations{seeds: fieldSeeds, rows: 16, encode: encodeGrown, bake: (*scene).bakeAll,
		open: func(source, name string) (fieldKernel, error) { return metal.Open(source, name) }}
}

// fieldSampled initially supports one independently placed Kerr field. Seeds
// use Celestial's observer construction; all integration runs on Metal.
func fieldSampled(world *thing.World, eye maths.Subspace, lens film.Sensor,
	at time.Duration, width, height, side int, cells ...int) (*frame.Frame, timing, error) {
	return fieldSampledUsing(world, eye, lens, at, width, height, side, fieldDefaults(), cells...)
}

// fieldSampledUsing exposes stage errors without requiring a failed GPU.
func fieldSampledUsing(world *thing.World, eye maths.Subspace, lens film.Sensor,
	at time.Duration, width, height, side int, ops fieldOperations, cells ...int) (*frame.Frame, timing, error) {
	var spent timing
	if len(world.Fields) != maxGravityFields {
		return nil, spent, unsupported{"multiple fields"}
	}
	field := world.Fields[0]
	if field == nil {
		return nil, spent, fmt.Errorf("gpu: invalid gravity field")
	}
	black, err := kerr.NewHole(field.Mass, field.Spin)
	if err != nil || !(field.Reach > 0) {
		return nil, spent, fmt.Errorf("gpu: invalid gravity field")
	}
	space := field.At(at)
	scale := space.Scale
	if scale == 0 {
		scale = 1
	}
	if !(scale > 0) {
		return nil, spent, unsupported{"a nonpositive field scale"}
	}
	if err := fieldPrecision(field, space, scale); err != nil {
		return nil, spent, err
	}
	side = max(1, side)
	w, h, err := checkedFineSize(width, height, side)
	if err != nil {
		return nil, spent, err
	}
	start := time.Now()
	// Field transport supplies its own ray tape below. Avoid generating a
	// custom lens's ordinary ray tape only to discard it immediately.
	encoded, err := ops.encode(world, nil, eye, nil, at, w, h, cells...)
	if err != nil {
		return nil, spent, err
	}
	if encoded.cleared {
		return nil, spent, unsupported{"refraction in a gravity field"}
	}
	if encoded.incident != nil {
		return nil, spent, unsupported{"cached volume-sky lighting in a gravity field"}
	}
	spent.seedCached = ops.seeds(encoded, fieldSeedInput{field, black, space, scale, eye, lens, w, h})
	var params packer
	// Integrate in mass units; the world transform carries physical scale.
	params.floats([]float32{1, float32(field.Spin / field.Mass), float32(field.Reach / field.Mass), float32(scale * field.Mass)})
	for _, v := range []maths.Vec{space.Origin, space.Orientation.Out(maths.XAxis), space.Orientation.Out(maths.YAxis), space.Orientation.Out(maths.ZAxis)} {
		params.floats([]float32{float32(v.X), float32(v.Y), float32(v.Z), 0})
	}
	transport := trace.FieldSettings()
	if field.LetGo > 0 {
		transport.Outward = field.LetGo
	}
	params.floats([]float32{metal.GeodesicTimeShare, metal.GeodesicHoverRatio, float32(transport.Steps), float32(transport.Core)})
	params.floats([]float32{float32(transport.Outward), float32(transport.ReachMargin), float32(transport.Strides), 0})
	spent.encode = time.Since(start)
	if spent.bake, err = ops.bake(encoded); err != nil {
		return nil, spent, err
	}
	start = time.Now()
	source := sourceWith(encoded.records(), encoded.counts.volumes)
	source = strings.Replace(source, "kernel void trace(", "kernel void straight_trace(", 1) + fieldSource
	kernel, err := ops.open(source, "trace")
	spent.compile = time.Since(start)
	if err != nil {
		return nil, spent, err
	}
	defer kernel.Close()
	inputs := append(encoded.inputs(), params.bytes)
	samples, err := kernel.Grid(w, h, ops.rows, encoded.perSample(), inputs)
	if err != nil {
		return nil, spent, err
	}
	spent.trace = kernel.GPUTime()
	start = time.Now()
	finish := finishing{world: world, paints: encoded.paints, records: encoded.records(), wide: wideOf(world, eye, lens, w, h), discs: film.Disced(world.Lamps)}
	picture := averaged(finish, samples, width, height, side)
	spent.average = time.Since(start)
	return picture, spent, nil
}

// fieldPrecision refuses values lost or made nonfinite in the shader format.
// Coordinates integrate in mass units, so astronomical masses do not enter
// the fourth powers used by the Kerr metric.
func fieldPrecision(field *thing.Field, space maths.Subspace, scale float64) error {
	spin := float32(field.Spin / field.Mass)
	if !(spin >= 0 && spin < 1) {
		return unsupported{"a spin unresolved in float32"}
	}
	for _, value := range []float64{field.Reach / field.Mass, scale * field.Mass} {
		packed := float32(value)
		if !(packed > 0) || math.IsInf(float64(packed), 0) {
			return unsupported{"a field scale unresolved in float32"}
		}
	}
	for _, value := range []float64{space.Origin.X, space.Origin.Y, space.Origin.Z} {
		if math.IsNaN(value) || math.IsInf(float64(float32(value)), 0) {
			return unsupported{"a nonfinite field origin"}
		}
	}
	return nil
}

// fieldSeeds constructs observer states once for each actual camera ray.
func fieldSeeds(encoded *scene, input fieldSeedInput) bool {
	field, black, space, scale := input.source, input.black, input.space, input.scale
	eye, lens, w, h := input.eye, input.lens, input.width, input.height
	encoded.rays.bytes = make([]byte, 0, w*h*fieldSeedWords*4)
	for y := range h {
		for x := range w {
			ray := film.Look(eye, lens, x, y, w, h, maths.Band{})
			local := space.RayIn(ray)
			distance := -1.0
			if local.At.Dot(local.At) < field.Reach*field.Reach {
				distance = 0
			} else {
				closing := local.At.Dot(local.Dir)
				discriminant := closing*closing - local.At.Dot(local.At) + field.Reach*field.Reach
				if discriminant >= 0 {
					entry := -closing - math.Sqrt(discriminant)
					if entry > 1e-9 {
						distance = entry * scale
					}
				}
			}
			rayAt := len(encoded.rays.bytes)
			encoded.rays.bytes = append(encoded.rays.bytes, make([]byte, rayStride)...)
			putRay(encoded.rays.bytes[rayAt:], ray)
			seed := kerr.NewRay(kerr.NewVec4(0, 0, 0, 0), kerr.NewVec4(0, 0, 0, 0))
			energy := 0.0
			if distance >= 0 {
				p := field.InHole(space.In(ray.Point(distance)))
				position := kerr.NewVec3(p[0], p[1], p[2])
				direction := field.InHole(space.Orientation.In(ray.Dir).Unit())
				obs := kerr.NewVec4(1, 0, 0, 0)
				still := black.Static(position)
				if distance == 0 {
					if field.EyeTime > 0 {
						v := field.InHole(field.Eye)
						still = kerr.NewVec4(field.EyeTime, v[0], v[1], v[2])
					}
					obs = still
				}
				seed = black.Arriving(position, black.FrameOf(position, still), kerr.NewVec3(direction[0], direction[1], direction[2]))
				energy = kerr.Dot(seed.Momentum(), obs)
			}
			for component, v := range []kerr.Vec4{seed.Position(), seed.Momentum()} {
				factor := 1.0
				if component == 0 {
					factor = 1 / field.Mass
				}
				encoded.rays.floats([]float32{float32(v.Time() * factor), float32(v.AlongX() * factor), float32(v.AlongY() * factor), float32(v.AlongZ() * factor)})
			}
			encoded.rays.floats([]float32{float32(distance), float32(energy)})
		}
	}
	return false
}
