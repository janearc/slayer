package cities

import (
	"math"
	"slices"
	"strings"
	"testing"
)

// halfGrown is the accretion of a planned block partly overgrown.
const halfGrown = 0.5

// planSeeds is how many seeds the plan's tests grow.
const planSeeds = 4

// rising are numbers of people, each more than the last, up to the
// picture's.
var rising = []int{1000, 3000, 6000, parcelPeople}

// doubled is twice the crowding the tests plan for.
const doubled = 2

// The kinds of fixture only the accretion hangs, which a planned block has
// none of.
var accretedKinds = []string{"bridge", "jetty", "cage", "box", "sign",
	"cable", "pipe", "rail", "line"}

func TestAPlannedBlockIsSlabsAndCoresOnAPodiumRoundACourt(t *testing.T) {
	for seed := range Seed(planSeeds) {
		block := growSuperblock(t, kowloon().at(0), seed)
		if got := len(usedAs(block, useInfill)) +
			len(usedAs(block, useTopping)); got != 0 {
			t.Errorf("seed %d: a planned block with %d grown plots", seed,
				got)
		}
		if len(usedAs(block, useFlats)) < sidesAround ||
			len(usedAs(block, useCore)) < sidesAround {
			flats := len(usedAs(block, useFlats))
			cores := len(usedAs(block, useCore))
			t.Errorf("seed %d: %d runs of flats and %d cores", seed,
				flats, cores)
		}
		kinds := kindsOf(block.Fixtures)
		for _, kind := range accretedKinds {
			if kinds[kind] != 0 {
				t.Errorf("seed %d: %d of %s on a planned block", seed,
					kinds[kind], kind)
			}
		}
		for _, kind := range []string{"deck", "canopy", "ramp"} {
			if kinds[kind] == 0 {
				t.Errorf("seed %d: no %s on a planned block", seed, kind)
			}
		}
		if len(block.Yards) == 0 || block.Court.area() <= 0 {
			t.Errorf("seed %d: a planned court of no open ground", seed)
		}
	}
}

func TestTheSlabsRiseToTheStoreysTheirPeopleNeed(t *testing.T) {
	block := growSuperblock(t, kowloon().at(0), 1)
	var footprint float64
	for _, plot := range usedAs(block, useFlats) {
		footprint += plot.Lot.area()
	}
	want := int(math.Ceil(parcelPeople * block.PerPerson / footprint))
	if block.Storeys != want {
		t.Errorf("slabs of %d storeys for %d people on %v square metres, "+
			"want %d", block.Storeys, parcelPeople, footprint, want)
	}
	for _, plot := range usedAs(block, useFlats) {
		if plot.Storeys != want {
			t.Errorf("a run of %d storeys among slabs of %d", plot.Storeys,
				want)
		}
	}
	for _, plot := range usedAs(block, useCore) {
		if plot.Storeys != want+1 {
			t.Errorf("a core of %d storeys over slabs of %d", plot.Storeys,
				want)
		}
	}
}

func TestMorePeopleNeverLowerTheSlabsAndMoreCrowdingNeverRaisesThem(
	t *testing.T) {
	last := 0
	for _, people := range rising {
		ask := kowloon().at(0)
		ask.Occupants = people
		block := growSuperblock(t, ask, 1)
		if block.Storeys < last {
			t.Errorf("%d people in %d storeys, fewer than %d", people,
				block.Storeys, last)
		}
		last = block.Storeys
		ask.Crowding = doubled
		if crowded := growSuperblock(t, ask, 1); crowded.Storeys >
			block.Storeys {
			t.Errorf("at twice the crowding %d storeys, over %d",
				crowded.Storeys, block.Storeys)
		}
	}
}

func TestAccessDecksStandEveryThirdStoreyOnTheCourtFace(t *testing.T) {
	block := growSuperblock(t, kowloon().at(0), 2)
	every := drawerOf(t, superblockCulture, 1).count("deck-every", place{})
	for _, deck := range block.Fixtures {
		if deck.Kind != "deck" {
			continue
		}
		plot := block.Plots[deck.Plot]
		thick := drawerOf(t, superblockCulture, 1).fixed("deck-thickness")
		found := false
		for storey := every; storey < plot.Storeys; storey += every {
			found = found || math.Abs(plot.floorOf(storey)-
				(deck.Low[1]+thick)) < areaSlack
		}
		if !found {
			t.Errorf("a deck at %v m on no third storey of %+v", deck.Low[1],
				plot)
		}
		if !deck.inside(block.Court) {
			t.Errorf("a deck %+v not over the court", deck)
		}
	}
}

// inside is whether a volume stands over a stretch of ground.
func (volume Volume) inside(ground rect) bool {
	return volume.Low[0] >= ground.X0-meeting &&
		volume.High[0] <= ground.X1+meeting &&
		volume.Low[2] >= ground.Z0-meeting &&
		volume.High[2] <= ground.Z1+meeting
}

func TestNoTwoVolumesMeetAtAnyAccretion(t *testing.T) {
	for _, grown := range []float64{0, halfGrown, 1} {
		volumes := growSuperblock(t, kowloon().at(grown), 3).Volumes()
		for first := range volumes {
			for second := first + 1; second < len(volumes); second++ {
				if volumes[first].meets(volumes[second]) {
					t.Fatalf("accretion %v: %+v meets %+v", grown,
						volumes[first], volumes[second])
				}
			}
		}
	}
}

func TestDensityRisesWithAccretionAndComesFromTheFacts(t *testing.T) {
	last := 0.0
	for _, grown := range []float64{0, halfGrown, 1} {
		block := growSuperblock(t, kowloon().at(grown), 4)
		density, err := block.Density()
		if err != nil {
			t.Fatal(err)
		}
		var floor float64
		for _, plot := range block.Plots {
			if plot.Use != useCore {
				floor += plot.Lot.area() * float64(plot.Storeys)
			}
		}
		want := floor / block.PerPerson / block.Parcel.area()
		if math.Abs(density-want) > areaSlack {
			t.Errorf("accretion %v: a density of %v, and the facts give %v",
				grown, density, want)
		}
		if density <= last {
			t.Errorf("accretion %v: a density of %v, not over %v", grown,
				density, last)
		}
		last = density
	}
}

func TestHalfwayIsAPlannedBlockPartlyOvergrown(t *testing.T) {
	planned := growSuperblock(t, kowloon().at(0), 5)
	half := growSuperblock(t, kowloon().at(halfGrown), 5)
	whole := growSuperblock(t, kowloon().at(1), 5)
	court := func(block Superblock) int {
		return len(usedAs(block, useInfill))
	}
	if !(court(planned) < court(half) && court(half) < court(whole)) {
		t.Errorf("the court is built over %d, %d and %d times at "+
			"nought, a half and one", court(planned), court(half),
			court(whole))
	}
	if len(usedAs(half, useFlats)) != len(usedAs(planned, useFlats)) {
		t.Error("a block partly overgrown lost its plan")
	}
}

func TestADrawnAccretionIsTheSeedsAndFromNoughtToOne(t *testing.T) {
	ask := kowloon()
	ask.Accretion = Accretion{Drawn: true}
	seen := map[float64]bool{}
	// The accretion is drawn by district, and seeds that differ by less
	// than a district draw the same district: the seeds are one district
	// apart.
	district := Seed(drawerOf(t, superblockCulture, 0).fixed(
		"rows-per-district"))
	for index := range Seed(planSeeds) {
		seed := index * district
		first := growSuperblock(t, ask, seed)
		again := growSuperblock(t, ask, seed)
		if first.Accretion != again.Accretion || first.Accretion < 0 ||
			first.Accretion > 1 {
			t.Errorf("seed %d drew %v and %v", seed, first.Accretion,
				again.Accretion)
		}
		seen[first.Accretion] = true
	}
	if len(seen) < 2 {
		t.Errorf("%d seeds drew one accretion", planSeeds)
	}
}

func TestPlannedFacesAreHouseholdsAtTheDemoLevel(t *testing.T) {
	ask := small(false).at(0)
	ask.Level = demoLevel
	drawn, err := growSuperblock(t, ask, 6).Design().Rendered(demoLevel)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, one := range drawn {
		names = append(names, one.Mesh.Name)
	}
	for _, want := range []string{"curtain", "blind"} {
		if !slices.ContainsFunc(names, func(name string) bool {
			return strings.Contains(name, want)
		}) {
			t.Errorf("no %s among the planned faces' parts", want)
		}
	}
}

func TestPeopleWhoDoNotFitAreCountedInTheRefusal(t *testing.T) {
	ask := small(false)
	ask.Occupants = crushing * smallPeople
	_, err := NewSuperblock(ask, 1)
	if err == nil || !strings.Contains(err.Error(), "do not fit") {
		t.Errorf("a block too small for its people: %v", err)
	}
}
