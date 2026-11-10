package embers

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/celestial/measure"
)

func TestEmitterLimitsAcceptTheirEndpointsAndRefuseTheNextValue(t *testing.T) {
	for _, example := range []struct {
		name   string
		rate   float64
		life   time.Duration
		accept bool
	}{
		{"sub-unit rate", .5, time.Second, true},
		{"maximum rate", 1e6, time.Millisecond, true},
		{"past maximum rate", math.Nextafter(1e6, math.Inf(1)), time.Millisecond, false},
		{"maximum live budget", 8190, time.Second, true},
		{"one past live budget", 8190.5, time.Second, false},
	} {
		t.Run(example.name, func(t *testing.T) {
			spec := fixture()
			spec.Rate, spec.Life = example.rate, example.life
			_, err := New(spec)
			if (err == nil) != example.accept {
				t.Fatalf("rate %.18g life %s: %v", spec.Rate, spec.Life, err)
			}
		})
	}
	spec := fixture()
	spec.Hot = spec.Cool
	if _, err := New(spec); err != nil {
		t.Fatalf("constant-temperature particle refused: %v", err)
	}
}

func TestFirebrandsRetainTheirIdentityRadiusAndAuthoredCoolingLine(t *testing.T) {
	source, err := New(fixture())
	if err != nil {
		t.Fatal(err)
	}
	for _, sample := range []struct {
		age    time.Duration
		kelvin float64
	}{
		{0, 1400}, {750 * time.Millisecond, 1225}, {1500 * time.Millisecond, 1050}, {2250 * time.Millisecond, 875},
	} {
		particle := source.particle(5, sample.age)
		if particle.ID() != 5 || particle.Age() != sample.age || particle.Radius() != measure.Metres(.02) {
			t.Fatal("particle lost identity, time or physical radius")
		}
		if got := particle.Heat().InKelvin(); math.Abs(got-sample.kelvin) > 1e-10 {
			t.Fatalf("at %s heat %g K, authored linear value %g K", sample.age, got, sample.kelvin)
		}
	}
}

func TestReleasePositionsFillTheDiskWithoutAnAngularBias(t *testing.T) {
	spec := fixture()
	source, err := New(spec)
	if err != nil {
		t.Fatal(err)
	}
	const count = 2048
	radius := spec.Disk.InMetres()
	meanX, meanZ, meanSquared, meanFourth := 0.0, 0.0, 0.0, 0.0
	for id := range int64(count) {
		p := source.particle(id, 0).Position()
		if p.Y != spec.Height.InMetres() {
			t.Fatal("birth height changed")
		}
		squared := (p.X*p.X + p.Z*p.Z) / (radius * radius)
		if !(squared >= 0 && squared <= 1) {
			t.Fatalf("birth outside disk: %v", p)
		}
		meanX += p.X / radius
		meanZ += p.Z / radius
		meanSquared += squared
		meanFourth += squared * squared
	}
	// Uniform disk area makes squared radius uniform on [0,1].
	if math.Abs(meanX/count) > .05 || math.Abs(meanZ/count) > .05 ||
		math.Abs(meanSquared/count-.5) > .03 || math.Abs(meanFourth/count-1.0/3) > .03 {
		t.Fatalf("release disk bias: means %g,%g radial moments %g,%g", meanX/count, meanZ/count, meanSquared/count, meanFourth/count)
	}
}
