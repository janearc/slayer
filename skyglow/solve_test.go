package skyglow

import (
	"math"
	"reflect"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
)

// testNodes is the grid the tests solve on: coarse, so a solve is quick,
// and not so coarse that a source sits in the same cell as its probes.
var testNodes = [3]int{12, 6, 12}

// testOptions are those the tests solve with: the series to its end.
func testOptions() Options {
	return Options{Nodes: testNodes, Orders: 6, Tolerance: 0.01, Workers: 2}
}

// testRegion is a block of air twelve hundred metres across and as deep
// as the city's air.
func testRegion() Region {
	return Region{Low: maths.V(-600, 0, -600), High: maths.V(600, 400, 600)}
}

// aLamp is a point source at a place, with an intensity, grey.
func aLamp(at maths.Vec, intensity float64, traced bool) Source {
	return Source{At: at, Power: maths.Grey(intensity), Traced: traced}
}

// thinExtinction is the extinction of air thin enough that a path of a
// kilometre loses a ten thousandth of its light: the inverse square law
// is then the whole of the first order.
const thinExtinction = 1e-7

// inverseSquareTolerance is how far the first order may be from the
// inverse square law in that air: the loss over the path, and rounding.
const inverseSquareTolerance = 1e-3

// One source in thin air gives, at each place, its intensity over four
// pi times the square of the distance: the first order, which is all the
// scattering there is when the air is thin.
func TestOneSourceInThinAirGivesTheInverseSquare(t *testing.T) {
	air := Air{Extinction: thinExtinction, Depth: 400, Albedo: 0.9}
	at := maths.V(0, 200, 0)
	options := testOptions()
	options.Orders = 1
	field, err := Solve(air, testRegion(), []Source{aLamp(at, 1e6, false)},
		options)
	if err != nil {
		t.Fatal(err)
	}
	made := field.grid
	for _, index := range []int{
		field.at(10, 3, 6), field.at(2, 1, 9), field.at(7, 5, 0)} {
		place := made.nodes[index]
		want := 1e6 / (fourPi * place.Sub(at).Dot(place.Sub(at)))
		got := field.values[index].G
		if math.Abs(got-want) > inverseSquareTolerance*want {
			t.Errorf("at %v: %g, the inverse square gives %g", place, got,
				want)
		}
	}
}

// slack is how far above its bound an order may be put by the grid: the
// cells are points to each other and a cell's own light is a ball's, so
// the sums are the integrals to a few hundredths.
const slack = 1.05

// Energy is never created. The air takes a share of the light of the
// sources, no more than all of it; and each next order is made of light
// the air scattered, which is the albedo of the light it took and no
// more.
func TestTheSolverNeverCreatesEnergy(t *testing.T) {
	air := cityAir()
	made, err := newGrid(air, testRegion(), testNodes)
	if err != nil {
		t.Fatal(err)
	}
	sources := []Source{aLamp(maths.V(0, 100, 0), 1e6, false),
		aLamp(maths.V(150, 60, -80), 4e5, false),
		{At: maths.V(-90, 30, 120), Normal: maths.Up,
			Power: maths.Grey(2e5)}}
	emitted := 0.0
	for _, each := range sources {
		emitted += each.Power.Luminance()
	}
	_, lit := made.first(air, sources, 2)
	previous := made.scattered(lit)
	if previous > emitted {
		t.Fatalf("the air took %g of %g emitted", previous, emitted)
	}
	carried := lit
	for order := 2; order <= 5; order++ {
		carried = made.scatter(air, carried, 2)
		taken := made.scattered(carried)
		if taken > air.Albedo*previous*slack {
			t.Errorf("order %d took %g of the %g the one before did, which"+
				" is more than the albedo %g", order, taken, previous,
				air.Albedo)
		}
		previous = taken
	}
}

// Glow falls with distance from the emitters: one source's, along a
// street away from it at one height, never rises.
func TestGlowFallsWithDistanceFromTheEmitters(t *testing.T) {
	air := cityAir()
	streetLamp := aLamp(maths.V(0, 60, 0), 1e6, false)
	field, err := Solve(air, testRegion(),
		[]Source{streetLamp}, testOptions())
	if err != nil {
		t.Fatal(err)
	}
	last := math.Inf(1)
	for _, east := range []float64{50, 100, 200, 300, 450, 590} {
		got := field.Mean(maths.V(east, 60, 0)).G
		if !(got > 0 && got < last) {
			t.Errorf("%g m east the glow is %g, after %g nearer", east, got,
				last)
		}
		last = got
	}
}

// Zero emitters give a dark sky: air with no light in it holds none, at
// any place, in every channel.
func TestNoEmittersGiveADarkSky(t *testing.T) {
	field, err := Solve(cityAir(), testRegion(), nil, testOptions())
	if err != nil {
		t.Fatal(err)
	}
	for _, at := range []maths.Vec{maths.V(0, 100, 0), maths.V(500, 20, -300),
		maths.V(-590, 390, 590), maths.V(1e6, 1e6, 1e6)} {
		if got := field.Mean(at); got != (maths.Light{}) {
			t.Errorf("at %v the dark field is %v", at, got)
		}
	}
	report := field.Report()
	if report.Sources != 0 || report.Peak != (maths.Light{}) ||
		report.Tail != 0 {
		t.Errorf("the report of nothing shining: %+v", report)
	}
}

// Each order of scattering is less than the one before it, the shares
// are all of the light between them, and what lies past the last is
// some share of it.
func TestEachOrderIsLessThanTheLastAndTheSharesAreWhole(t *testing.T) {
	sources := []Source{aLamp(maths.V(0, 100, 0), 1e6, false),
		aLamp(maths.V(150, 60, -80), 4e5, false)}
	field, err := Solve(cityAir(), testRegion(), sources, testOptions())
	if err != nil {
		t.Fatal(err)
	}
	report := field.Report()
	if len(report.Shares) < 2 {
		t.Fatalf("a haze of 3 km gave %d orders", len(report.Shares))
	}
	total := 0.0
	for index, share := range report.Shares {
		total += share
		if index > 0 && !(share < report.Shares[index-1]) {
			t.Errorf("order %d holds %g, after %g", index+1, share,
				report.Shares[index-1])
		}
	}
	if math.Abs(total-1) > powerTolerance*1e3 || report.Tail < 0 ||
		report.Tail >= 1 {
		t.Errorf("shares %v sum to %g; tail %g", report.Shares, total,
			report.Tail)
	}
}

// A lamp the tracer scatters itself is not in the first order of the
// glow, which would count it twice, and is in the orders past it, which
// the tracer does not work.
func TestATracedLampIsOnlyCarriedToTheOrdersPastTheFirst(t *testing.T) {
	sources := []Source{aLamp(maths.V(0, 100, 0), 1e6, true)}
	options := testOptions()
	options.Orders = 1
	first, err := Solve(cityAir(), testRegion(), sources, options)
	if err != nil {
		t.Fatal(err)
	}
	if peak := first.Report().Peak; peak != (maths.Light{}) {
		t.Errorf("a traced lamp is in the first order: %v", peak)
	}
	later, err := Solve(cityAir(), testRegion(), sources, testOptions())
	if err != nil {
		t.Fatal(err)
	}
	if later.Report().Peak.G <= 0 || later.Report().Traced != 1 {
		t.Errorf("a traced lamp left no later orders: %+v", later.Report())
	}
}

// The field holds its edge value beyond the grid, which is the value at
// the middle of the last cell.
func TestTheFieldHoldsItsEdgeBeyondTheGrid(t *testing.T) {
	streetLamp := aLamp(maths.V(0, 100, 0), 1e6, false)
	field, err := Solve(cityAir(), testRegion(),
		[]Source{streetLamp}, testOptions())
	if err != nil {
		t.Fatal(err)
	}
	made := field.grid
	edge := made.low.X + (float64(made.count[0])-half)*made.step[0]
	near, far := field.Mean(maths.V(edge, 100, 0)), field.Mean(
		maths.V(1e7, 100, 0))
	if near != far {
		t.Errorf("at the last cell %v, far past it %v", near, far)
	}
}

// The same solve on one worker or on three is the same field, bit for
// bit, since no two workers write one place.
func TestTheFieldIsTheSameOnAnyNumberOfWorkers(t *testing.T) {
	sources := []Source{aLamp(maths.V(0, 100, 0), 1e6, false),
		aLamp(maths.V(150, 60, -80), 4e5, true)}
	one, three := testOptions(), testOptions()
	one.Workers, three.Workers = 1, 3
	first, err := Solve(cityAir(), testRegion(), sources, one)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Solve(cityAir(), testRegion(), sources, three)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first.values, second.values) {
		t.Error("one worker and three made different fields")
	}
}

// binTolerance is how far a bin of a hundred lamps, in one cell, is from
// the lamps separately, at some cells' distance: the spread of the lamps
// over their distance, squared, is a few thousandths.
const binTolerance = 0.02

// A bin shines as its lamps do at a distance of some cells: the error the
// package states for gathering emitters.
func TestABinShinesAsItsLampsDoAtADistance(t *testing.T) {
	cluster := NewCluster(25)
	apart := []Source{}
	for east := range 10 {
		for south := range 10 {
			at := maths.V(float64(east)*2, 100, float64(south)*2)
			cluster.Point(at, maths.Grey(1e4), false)
			apart = append(apart, aLamp(at, 1e4, false))
		}
	}
	options := testOptions()
	options.Orders = 1
	bin, err := Solve(cityAir(), testRegion(), cluster.Sources(), options)
	if err != nil {
		t.Fatal(err)
	}
	each, err := Solve(cityAir(), testRegion(), apart, options)
	if err != nil {
		t.Fatal(err)
	}
	probe := maths.V(450, 100, 300)
	got, want := bin.Mean(probe).G, each.Mean(probe).G
	if math.Abs(got-want) > binTolerance*want {
		t.Errorf("the bin gives %g at %v, its lamps %g", got, probe, want)
	}
}

// Anti-test: a solve that cannot be made is refused, by what is wrong
// with it, and not made wrongly.
func TestASolveThatCannotBeMadeIsRefused(t *testing.T) {
	sources := []Source{aLamp(maths.V(0, 100, 0), 1e6, false)}
	flat := testRegion()
	flat.High.Y = flat.Low.Y
	badOptions := func(change func(*Options)) Options {
		options := testOptions()
		change(&options)
		return options
	}
	cases := map[string]func() error{
		"air that is not air": func() error {
			_, err := Solve(Air{}, testRegion(), sources, testOptions())
			return err
		},
		"a region with no volume": func() error {
			_, err := Solve(cityAir(), flat, sources, testOptions())
			return err
		},
		"no orders": func() error {
			_, err := Solve(cityAir(), testRegion(), sources,
				badOptions(func(o *Options) { o.Orders = 0 }))
			return err
		},
		"no workers": func() error {
			_, err := Solve(cityAir(), testRegion(), sources,
				badOptions(func(o *Options) { o.Workers = 0 }))
			return err
		},
		"a tolerance of one": func() error {
			_, err := Solve(cityAir(), testRegion(), sources,
				badOptions(func(o *Options) { o.Tolerance = 1 }))
			return err
		},
		"a node to an axis": func() error {
			_, err := Solve(cityAir(), testRegion(), sources,
				badOptions(func(o *Options) { o.Nodes = [3]int{1, 6, 12} }))
			return err
		},
		"more nodes than a solve can sum": func() error {
			_, err := Solve(cityAir(), testRegion(), sources,
				badOptions(func(o *Options) { o.Nodes = [3]int{40, 40, 40} }))
			return err
		},
	}
	for name, try := range cases {
		if try() == nil {
			t.Errorf("%s: the solve was made", name)
		}
	}
}

// The field is the light of its own air and no other: the tracer asks it
// of each volume it crosses.
func TestAFieldLightsItsOwnAirAndNoOther(t *testing.T) {
	own, other := medium.Puff{Radius: 1}, medium.Puff{Radius: 2}
	field := &Field{Medium: own}
	if !field.Lights(own) || field.Lights(other) || field.Lights(nil) {
		t.Error("the field lit an air that is not its own, or not its own")
	}
}
