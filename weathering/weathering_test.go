package weathering_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/weathering"
)

// the numbers the tests lay a layer out by, each a choice of the tests
const (
	// testSeed is the seed of every layer unless a test changes it.
	testSeed = 7
	// testAge is the age, in years, of a layer unless a test changes it.
	testAge = 40
	// testBlock is how wide the shelter's cells are, in metres.
	testBlock = 2
	// testProbe is how far off a surface a look overhead starts, in metres.
	testProbe = 0.03
	// ledgeHeight is how high the test ledge's underside is, in metres.
	ledgeHeight = 2.0
	// ledgeDepth is how far the test ledge stands out of its wall.
	ledgeDepth = 0.5
	// ledgeSpan is how far the test ledge runs either way along its wall.
	ledgeSpan = 40.0
	// gridStep is the step of the grid of sites a test walks, in metres.
	gridStep = 0.37
	// gridCount is how many steps a grid has along each side.
	gridCount = 24
	// slack is how much a sum may differ and still be called the same.
	slack = 1e-12
)

// shelterWithLedge is a shelter holding one ledge, as a flat plate over
// the wall at z of nought, named for a material.
func shelterWithLedge(name string) *weathering.Shelter {
	shelter := weathering.NewShelter(testBlock)
	plate(shelter, maths.V(-ledgeSpan, ledgeHeight, 0),
		maths.V(ledgeSpan, ledgeHeight, ledgeDepth), name)
	return shelter
}

// plate adds a flat plate between two corners to a shelter, as two
// triangles.
func plate(shelter *weathering.Shelter, low, high maths.Vec, name string) {
	across := maths.V(low.X, low.Y, high.Z)
	along := maths.V(high.X, low.Y, low.Z)
	shelter.Add(low, along, high, name)
	shelter.Add(low, high, across, name)
}

// layerOf is a layer of agents under the sun from the south, with the
// wind from the west, over a shelter.
func layerOf(shelter *weathering.Shelter, agents ...weathering.Agent) weathering.Layer {
	return weathering.Layer{Setting: weathering.Setting{Seed: testSeed,
		Age: testAge, Up: maths.YAxis, North: maths.V(0, 0, -1),
		Sun: maths.V(0, 0.5, 1).Unit(), Wind: maths.V(1, 0, 0),
		Probe: testProbe, Block: testBlock, Over: shelter}, Agents: agents}
}

// the faces the grid of sites has, by which way they point
var (
	faceSouth = maths.V(0, 0, 1)
	faceNorth = maths.V(0, 0, -1)
	faceUp    = maths.V(0, 1, 0)
)

// sites is a grid of sites on a surface of a material, facing a way, as
// a wall at z of nought or as the ground just above it.
func sites(name string, normal maths.Vec) []*weathering.Site {
	var all []*weathering.Site
	for x := range gridCount {
		for y := range gridCount {
			for z := range gridCount / 4 {
				at := maths.V(float64(x)*gridStep-gridStep*gridCount/2,
					float64(y)*gridStep/2+0.05, float64(z)*gridStep)
				if normal != faceUp {
					at.Z = 0
				}
				all = append(all, &weathering.Site{Point: at, Normal: normal,
					Name: name})
			}
		}
	}
	return all
}

// total is how much an agent covers over every site.
func total(layer weathering.Layer, agent weathering.Agent,
	all []*weathering.Site) float64 {
	sum := 0.0
	for _, site := range all {
		sum += layer.Amount(agent, site)
	}
	return sum
}

// TestMossNeverLandsOnASunlitFace says that moss is nowhere on a wall
// that faces the sun and has nothing over it, and is somewhere on one
// that faces away.
func TestMossNeverLandsOnASunlitFace(t *testing.T) {
	moss := weathering.Agent{Kind: weathering.Moss, Name: "moss",
		Strength: 1, Settle: 10, Reach: 6, Share: 0.9, Grain: 0.4,
		North: 0.5, Shade: 0.3, Shelter: 1}
	layer := layerOf(weathering.NewShelter(testBlock), moss)
	if sunlit := total(layer, moss, sites("wall", faceSouth)); sunlit != 0 {
		t.Errorf("moss covered %v of a sunlit face, and should be none", sunlit)
	}
	if shaded := total(layer, moss, sites("wall", faceNorth)); !(shaded > 0) {
		t.Errorf("moss covered none of a face turned from the sun")
	}
}

// TestMossStaysLow says that moss does not climb past its reach.
func TestMossStaysLow(t *testing.T) {
	moss := weathering.Agent{Kind: weathering.Moss, Name: "moss",
		Strength: 1, Settle: 10, Reach: 1, Share: 0.9, Grain: 0.4,
		Shade: 0.3}
	layer := layerOf(weathering.NewShelter(testBlock), moss)
	for _, site := range sites("wall", faceNorth) {
		if site.Point.Y >= moss.Reach && layer.Amount(moss, site) != 0 {
			t.Fatalf("moss at %v, past its reach of %v", site.Point, moss.Reach)
		}
	}
}

// TestDustNeverLandsUnderAnOverhang says that dust is nowhere on an upward
// face with something over it, and is somewhere on one in the open.
func TestDustNeverLandsUnderAnOverhang(t *testing.T) {
	shelter := weathering.NewShelter(testBlock)
	plate(shelter, maths.V(-ledgeSpan, ledgeHeight, -ledgeSpan),
		maths.V(0, ledgeHeight, ledgeSpan), "roof")
	dust := weathering.Agent{Kind: weathering.Dust, Name: "dust",
		Strength: 1, Settle: 5, Tilt: 0.5, Share: 0.9, Grain: 0.5,
		Shelter: 2 * ledgeHeight}
	layer := layerOf(shelter, dust)
	under, open := 0.0, 0.0
	for _, site := range sites("ground", faceUp) {
		switch {
		case site.Point.Y >= ledgeHeight: // above the roof, in the open
			continue
		case site.Point.X < 0:
			under += layer.Amount(dust, site)
		default:
			open += layer.Amount(dust, site)
		}
	}
	if under != 0 {
		t.Errorf("dust covered %v under an overhang, and should be none", under)
	}
	if !(open > 0) {
		t.Errorf("dust covered none of the open ground")
	}
}

// TestDustOnlyOnUpwardFaces says that dust is nowhere on a wall.
func TestDustOnlyOnUpwardFaces(t *testing.T) {
	dust := weathering.Agent{Kind: weathering.Dust, Name: "dust",
		Strength: 1, Settle: 5, Tilt: 0.5, Share: 0.9, Grain: 0.5}
	layer := layerOf(weathering.NewShelter(testBlock), dust)
	if wall := total(layer, dust, sites("wall", faceSouth)); wall != 0 {
		t.Errorf("dust covered %v of a wall, and should be none", wall)
	}
}

// TestRainRunsBelowALedgeOnly says that a streak is only on a wall that
// has a ledge over it within a streak's length, and fades down it.
func TestRainRunsBelowALedgeOnly(t *testing.T) {
	rain := weathering.Agent{Kind: weathering.Rain, Name: "rain",
		Strength: 1, Settle: 10, Level: 0.2, Length: 1.5, Spacing: 0.4,
		Share: 1}
	layer := layerOf(shelterWithLedge("sill"), rain)
	near, far, none := 0.0, 0.0, 0.0
	for _, site := range sites("wall", faceSouth) {
		switch below := ledgeHeight - site.Point.Y; {
		case below < 0:
			none += layer.Amount(rain, site)
		case below < rain.Length/3:
			near += layer.Amount(rain, site)
		case below > rain.Length:
			far += layer.Amount(rain, site)
		}
	}
	if none != 0 || far != 0 {
		t.Errorf("streaks above the ledge (%v) or past a length (%v)", none,
			far)
	}
	if !(near > 0) {
		t.Errorf("no streak just under the ledge")
	}
	bare := layerOf(weathering.NewShelter(testBlock), rain)
	if got := total(bare, rain, sites("wall", faceSouth)); got != 0 {
		t.Errorf("streaks %v on a wall with no ledge", got)
	}
}

// TestRustIsOnSteelAndRunsBelowIt says that rust is on steel, and on
// other surfaces only below steel within its run.
func TestRustIsOnSteelAndRunsBelowIt(t *testing.T) {
	rust := weathering.Agent{Kind: weathering.Rust, Name: "rust",
		On: []string{"steel"}, Strength: 1, Settle: 10, Level: 0.2,
		Length: 1.5, Spacing: 0.4, Share: 0.9, Grain: 0.4}
	layer := layerOf(shelterWithLedge("steel beam"), rust)
	if on := total(layer, rust, sites("steel rail", faceSouth)); !(on > 0) {
		t.Errorf("no rust on steel")
	}
	stain := 0.0
	for _, site := range sites("plaster", faceSouth) {
		amount := layer.Amount(rust, site)
		if ledgeHeight-site.Point.Y > rust.Length && amount != 0 {
			t.Fatalf("rust stain %v farther below steel than a run", amount)
		}
		stain += amount
	}
	if !(stain > 0) {
		t.Errorf("no stain below steel")
	}
	plain := layerOf(shelterWithLedge("timber beam"), rust)
	if got := total(plain, rust, sites("plaster", faceSouth)); got != 0 {
		t.Errorf("rust stain %v below timber", got)
	}
}

// TestAPointSourceThins says that mineral dust is thinner the farther
// downwind of its source, thinner upwind than downwind, and thinner off
// the line of the wind than on it.
func TestAPointSourceThins(t *testing.T) {
	mine := weathering.Agent{Kind: weathering.Mineral, Name: "mine",
		Strength: 1, Settle: 5, Tilt: 0.5, Reach: 30, Spread: 0.5,
		Upwind: 0.1, Grain: 1}
	layer := layerOf(weathering.NewShelter(testBlock), mine)
	at := func(x, z float64) float64 {
		return layer.Amount(mine, &weathering.Site{
			Point: maths.V(x, 0.05, z), Normal: faceUp, Name: "ground"})
	}
	previous := math.Inf(1)
	for _, distance := range []float64{1, 5, 10, 20, 40} {
		if got := at(distance, 0); !(got < previous) {
			t.Errorf("at %v m downwind %v is not under %v", distance, got,
				previous)
		} else {
			previous = got
		}
	}
	if up, down := at(-10, 0), at(10, 0); !(up < down) {
		t.Errorf("upwind %v is not under downwind %v", up, down)
	}
	if off, on := at(10, 8), at(10, 0); !(off < on) {
		t.Errorf("off the wind %v is not under on it %v", off, on)
	}
}

// TestGlitterThinsWithDistance says that flecks are fewer farther
// downwind of their source.
func TestGlitterThinsWithDistance(t *testing.T) {
	factory := weathering.Agent{Kind: weathering.Glitter, Name: "factory",
		Strength: 1, Settle: 1, Tilt: 0.5, Reach: 25, Spread: 1, Upwind: 0,
		Grain: 1, Fleck: 0.2, Share: 1}
	layer := layerOf(weathering.NewShelter(testBlock), factory)
	count := func(from, to float64) int {
		flecks := 0
		for x := from; x < to; x += factory.Fleck {
			for z := -2.0; z < 2; z += factory.Fleck {
				amount := layer.Amount(factory,
					&weathering.Site{Point: maths.V(x, 0.05, z),
						Normal: faceUp, Name: "ground"})
				flecks += int(math.Ceil(amount))
			}
		}
		return flecks
	}
	if near, far := count(1, 8), count(30, 37); !(near > far) {
		t.Errorf("%d flecks near the source, and %d far", near, far)
	}
}

// agentsAging is one agent of each kind that lays colour, each placed
// where the grid of sites reaches it.
func agentsAging() []weathering.Agent {
	around := maths.V(0, 0, 1)
	return []weathering.Agent{
		{Kind: weathering.Rain, Name: "rain", Strength: 0.8, Settle: 10,
			Level: 0.2, Length: 2, Spacing: 0.4, Share: 0.8},
		{Kind: weathering.Soot, Name: "soot", Strength: 0.7, Settle: 20,
			Reach: 3, Share: 0.4, Grain: 0.5, Shelter: 2, Catch: 0.2},
		{Kind: weathering.Dust, Name: "dust", Strength: 0.6, Settle: 5,
			Tilt: 0.5, Share: 0.8, Grain: 0.5, Shelter: 1.5},
		{Kind: weathering.Moss, Name: "moss", Strength: 1, Settle: 10,
			Reach: 6, Share: 0.9, Grain: 0.4, North: 0.5, Shade: 0.3,
			Shelter: 1},
		{Kind: weathering.Ivy, Name: "ivy", Strength: 0.9, Settle: 15,
			Level: 0.2, Spacing: 0.5, Share: 0.7, Climb: 3, Catch: 0.3},
		{Kind: weathering.Graffiti, Name: "graffiti", Strength: 1,
			Settle: 10, Level: 0.2, Grain: 0.5, Low: 0.5, High: 2.4,
			Share: 0.5, Catch: 1},
		{Kind: weathering.Rust, Name: "rust", On: []string{"steel"},
			Strength: 1, Settle: 10, Level: 0.2, Length: 2, Spacing: 0.4,
			Share: 0.9, Grain: 0.4},
		{Kind: weathering.Spall, Name: "spall", Strength: 1, Settle: 30,
			Grain: 0.5, Crack: 0.05, Share: 0.9, Spalled: 0.3, Catch: 0.5},
		{Kind: weathering.Mineral, Name: "mine", Strength: 0.9, Settle: 5,
			Tilt: 0.5, Reach: 30, Spread: 0.5, Upwind: 0.1, Grain: 1},
		{Kind: weathering.Glitter, Name: "glitter", Strength: 1, Settle: 3,
			Tilt: 0.5, Reach: 30, Spread: 1, Upwind: 0.2, Grain: 1,
			Fleck: 0.2, Share: 1},
		{Kind: weathering.Nano, Name: "nano", Strength: 1, Grain: 1,
			Rate: 0.5, Turn: 1, At: maths.V(0, 1, 0), Facing: around},
	}
}

// TestAgeOnlyLaysMore says that for every agent, at every site, more years
// never cover less, and by the last age some agent has covered some site.
func TestAgeOnlyLaysMore(t *testing.T) {
	shelter := shelterWithLedge("steel beam")
	ages := []float64{0, 1, 3, 10, 30, 100, 300}
	var grid []*weathering.Site
	for _, name := range []string{"wall", "steel rail"} {
		for _, normal := range []maths.Vec{faceSouth, faceNorth, faceUp} {
			grid = append(grid, sites(name, normal)...)
		}
	}
	for _, agent := range agentsAging() {
		last := make([]float64, len(grid))
		for _, age := range ages {
			layer := layerOf(shelter, agent)
			layer.Age = age
			for index, site := range grid {
				got := layer.Amount(agent, site)
				if got < last[index]-slack {
					t.Fatalf("%s at %v: %v at %v years is under %v before",
						agent.Name, site.Point, got, age, last[index])
				}
				last[index] = got
			}
		}
		covered := 0.0
		for _, got := range last {
			covered += got
		}
		if !(covered > 0) {
			t.Errorf("%s covered nothing by %v years", agent.Name,
				ages[len(ages)-1])
		}
	}
}

// TestAFestivalOnlyFades says that a festival's splatter is never more
// with age, and is some at the start.
func TestAFestivalOnlyFades(t *testing.T) {
	festival := weathering.Agent{Kind: weathering.Festival, Name: "plaza",
		Strength: 1, Settle: 8, Reach: 12, Grain: 0.4, Share: 0.9, Tilt: 0.5,
		Level: 0.2, High: 1.5, At: maths.V(0, 0, 3)}
	grid := sites("ground", faceUp)
	last := math.Inf(1)
	for index, age := range []float64{0, 1, 3, 10, 30, 100} {
		layer := layerOf(weathering.NewShelter(testBlock), festival)
		layer.Age = age
		sum := total(layer, festival, grid)
		if index == 0 && !(sum > 0) {
			t.Fatalf("a festival that has just been left no splatter")
		}
		if sum > last+slack {
			t.Errorf("splatter grew from %v to %v by %v years", last, sum, age)
		}
		last = sum
	}
}

// TestTheSameSeedLaysTheSame says that a layer is the same each time it
// is asked, and another seed lays another.
func TestTheSameSeedLaysTheSame(t *testing.T) {
	dust := weathering.Agent{Kind: weathering.Dust, Name: "dust",
		Strength: 1, Settle: 5, Tilt: 0.5, Share: 0.5, Grain: 0.5}
	grid := sites("ground", faceUp)
	firstLayer := layerOf(weathering.NewShelter(testBlock), dust)
	first := total(firstLayer, dust, grid)
	againLayer := layerOf(weathering.NewShelter(testBlock), dust)
	again := total(againLayer, dust, grid)
	if first != again {
		t.Errorf("the same seed laid %v and then %v", first, again)
	}
	other := layerOf(weathering.NewShelter(testBlock), dust)
	other.Seed++
	if got := total(other, dust, grid); got == first {
		t.Errorf("another seed laid the same %v", got)
	}
}

// TestMortarHoldsMore says that an agent with a catch lodges more of
// itself in a joint than on a brick.
func TestMortarHoldsMore(t *testing.T) {
	dust := weathering.Agent{Kind: weathering.Dust, Name: "dust",
		Strength: 0.3, Settle: 5, Tilt: 0.5, Share: 1, Grain: 0, Catch: 1}
	layer := layerOf(weathering.NewShelter(testBlock), dust)
	site := &weathering.Site{Point: maths.V(0, 0.05, 0), Normal: faceUp,
		Name: "brick"}
	brick := layer.Amount(dust, site)
	site.Joint = true
	if mortar := layer.Amount(dust, site); !(mortar > brick) {
		t.Errorf("a joint held %v and a brick %v", mortar, brick)
	}
}

// TestAgentsLandOnlyWhereTheyAreAsked says that an agent that names
// materials is nowhere on others.
func TestAgentsLandOnlyWhereTheyAreAsked(t *testing.T) {
	dust := weathering.Agent{Kind: weathering.Dust, Name: "dust",
		On: []string{"quay"}, Strength: 1, Settle: 5, Tilt: 0.5, Share: 1}
	layer := layerOf(weathering.NewShelter(testBlock), dust)
	if got := total(layer, dust, sites("water", faceUp)); got != 0 {
		t.Errorf("dust %v on water", got)
	}
	if got := total(layer, dust, sites("quay stone", faceUp)); !(got > 0) {
		t.Errorf("no dust on the quay")
	}
}

// TestNanomachinesSpreadFromTheirSeed says that the contamination is
// nearest the seed, and reaches farther the older the layer.
func TestNanomachinesSpreadFromTheirSeed(t *testing.T) {
	seed := weathering.Agent{Kind: weathering.Nano, Name: "nano",
		Strength: 1, Grain: 1, Rate: 0.2, Turn: 1,
		At: maths.V(0, 1, 0), Facing: faceSouth}
	layer := layerOf(weathering.NewShelter(testBlock), seed)
	at := func(layer weathering.Layer, across float64) float64 {
		return layer.Amount(seed, &weathering.Site{
			Point: maths.V(across, 1, 0), Normal: faceSouth, Name: "wall"})
	}
	if near, far := at(layer, 0.5), at(layer, 12); !(near > far) {
		t.Errorf("%v near the seed and %v far from it", near, far)
	}
	young := layer
	young.Age = 4
	if older, younger := at(layer, 4), at(young, 4); !(older > younger) {
		t.Errorf("%v at forty years and %v at four, four metres off", older,
			younger)
	}
}

// TestAShelterFindsWhatIsOver says that what stands over a point is found
// by its height above, and not past the reach.
func TestAShelterFindsWhatIsOver(t *testing.T) {
	shelter := shelterWithLedge("sill")
	rise, name, found := shelter.Above(maths.V(1, 0.5, 0.2), 5)
	if !found || name != "sill" || math.Abs(rise-(ledgeHeight-0.5)) > slack {
		t.Errorf("found %v %q %v, and want 1.5 over, a sill", rise, name, found)
	}
	if _, _, found := shelter.Above(maths.V(1, 0.5, 0.2), 1); found {
		t.Errorf("found a sill past the reach")
	}
	if _, _, found := shelter.Above(maths.V(1, 0.5, 3), 5); found {
		t.Errorf("found a sill over a place it does not cover")
	}
	if _, _, found := shelter.Above(maths.V(1, 2.5, 0.2), 5); found {
		t.Errorf("found a sill under a place above it")
	}
}
