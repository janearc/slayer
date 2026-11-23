package noise_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths/noise"
	"pgregory.net/rapid"
)

// hashed is Hash under a picked seed, by the seed's name.
func hashed(test reporter, seedName string, index int64) float64 {
	test.Helper()
	return noise.Hash(pickedSeed(test, seedName), index)
}

// The same inputs give the same number, and different seeds give different
// ones.
func TestHashIsRepeatableAndSeeded(test *testing.T) {
	first, other := pickedSeed(test, "coordinate-seed-x"),
		pickedSeed(test, "coordinate-seed-y")
	index := int64(picked(test, "hash-index"))
	if noise.Hash(first, index) != noise.Hash(first, index) {
		test.Fatal("a hash changed between two asks")
	}
	if noise.Hash(first, index) == noise.Hash(other, index) {
		test.Fatal("two seeds gave one number")
	}
}

// Every hash is from zero up to one, whatever the seed and index, and
// asked twice it is the same number. The three hashes are each held to it.
func TestHashesAreInTheUnitIntervalAndRepeat(test *testing.T) {
	rapid.Check(test, hashesAreInTheUnitInterval)
}

func hashesAreInTheUnitInterval(t *rapid.T) {
	seed := rapid.Uint64().Draw(t, "seed")
	first, second, third := rapid.Int64().Draw(t, "first"),
		rapid.Int64().Draw(t, "second"), rapid.Int64().Draw(t, "third")
	for name, got := range map[string]float64{
		"Hash":  noise.Hash(seed, first),
		"Hash2": noise.Hash2(seed, first, second),
		"Hash3": noise.Hash3(seed, first, second, third),
	} {
		if !(got >= nought && got < unity) {
			t.Fatalf("%s gave %v, out of the unit interval", name, got)
		}
	}
	if noise.Hash3(seed, first, second, third) !=
		noise.Hash3(seed, first, second, third) {
		t.Fatal("Hash3 changed between two asks")
	}
}

// spread is how many of the pairs of neighbouring indices a hash of a
// pair puts in each of the bins the unit interval is cut into.
func spread(test reporter, hash func(uint64, int64, int64) float64) []int {
	test.Helper()
	bins := make([]int, count(test, "spread-bins"))
	seed, points := pickedSeed(test, "spread-seed"),
		int64(picked(test, "spread-points"))
	for index := range points {
		got := hash(seed, index, index+unity)
		if got < nought || got >= unity {
			test.Fatalf("hash %v out of range", got)
		}
		width := float64(len(bins))
		bins[int(got*width)]++
	}
	return bins
}

// uneven is the bins whose count is further from their even share than
// the slack allows, as a share of the share.
func uneven(test reporter, bins []int) []int {
	test.Helper()
	share := picked(test, "spread-points") / float64(len(bins))
	slack := share * picked(test, "spread-slack")
	var found []int
	for bin, got := range bins {
		if !(math.Abs(float64(got)-share) <= slack) {
			found = append(found, bin)
		}
	}
	return found
}

// Neighbouring indices give numbers spread evenly over zero to one: each of
// the bins gets its share of the points, within the slack.
func TestHashSpreadsNeighbours(test *testing.T) {
	bins := spread(test, noise.Hash2)
	for _, bin := range uneven(test, bins) {
		test.Errorf("bin %d holds %d of %v", bin, bins[bin],
			picked(test, "spread-points"))
	}
}

// stuck is a hash that gives one number whatever it is asked.
type stuck struct{ value float64 }

// hash is the one number.
func (same stuck) hash(uint64, int64, int64) float64 { return same.value }

// anti-test: a hash stuck on one number fills one bin, and the spread
// check sees every bin uneven.
func TestTheSpreadCatchesAStuckHash(test *testing.T) {
	bins := spread(test, stuck{picked(test, "stuck-value")}.hash)
	if found := uneven(test, bins); len(found) != len(bins) {
		test.Errorf("a stuck hash left only %d of %d bins uneven",
			len(found), len(bins))
	}
}

// field is a noise the smoothness and bound checks are run over.
type field func(seed uint64, x, y, z float64) float64

// rough is the first point of a field that is past the bound, or moves
// by more than the allowed move over one step along x, or is not nought
// on the lattice point under it; and whether there is one.
func rough(test reporter, noisy field) (place, bool) {
	test.Helper()
	seed, bound := pickedSeed(test, "field-seed"), picked(test, "noise-bound")
	step, move := picked(test, "smooth-step"), picked(test, "smooth-move")
	for index := range int64(picked(test, "smooth-points")) {
		at := thrown(test, index)
		value := noisy(seed, at.x, at.y, at.z)
		if !(math.Abs(value) <= bound) {
			return at, true
		}
		if moved := math.Abs(noisy(seed, at.x+step, at.y, at.z) - value); moved > move {
			return at, true
		}
		under := place{math.Floor(at.x), math.Floor(at.y), math.Floor(at.z)}
		if noisy(seed, under.x, under.y, under.z) != nought {
			return at, true
		}
	}
	return place{}, false
}

// Gradient noise is zero on the lattice, bounded, and continuous: a step of a
// thousandth moves it by a small amount everywhere it is tried. the bound is
// one and a little, since it can pass one.
func TestGradientIsSmoothAndBounded(test *testing.T) {
	if at, found := rough(test, noise.Gradient); found {
		test.Fatalf("the noise is rough at %v", at)
	}
}

// jumping is Gradient with a jump added on one side of every lattice
// plane across x: where the fraction of x is past a half, the noise is
// moved by the jump.
type jumping struct{ jump float64 }

// at is the jumpy noise at a point.
func (jumpy jumping) at(seed uint64, x, y, z float64) float64 {
	value := noise.Gradient(seed, x, y, z)
	if x-math.Floor(x) > half {
		return value + jumpy.jump
	}
	return value
}

// jumpy is the jumping noise with the picked jump.
func jumpy(test reporter) field { return jumping{picked(test, "jump")}.at }

// anti-test: a field with a jump in it is found rough.
func TestTheSmoothnessCheckCatchesAJump(test *testing.T) {
	if _, found := rough(test, jumpy(test)); !found {
		test.Fatal("a field with a jump in it passed as smooth")
	}
}

// Every whole-numbered point is a lattice point, and the noise is nought
// there, however far out, whatever the seed.
func TestGradientIsNoughtOnEveryLatticePoint(test *testing.T) {
	rapid.Check(test, gradientIsNoughtOnTheLattice)
}

func gradientIsNoughtOnTheLattice(t *rapid.T) {
	reach := int64(picked(t, "lattice-reach"))
	seed := rapid.Uint64().Draw(t, "seed")
	x := float64(rapid.Int64Range(-reach, reach).Draw(t, "x"))
	y := float64(rapid.Int64Range(-reach, reach).Draw(t, "y"))
	z := float64(rapid.Int64Range(-reach, reach).Draw(t, "z"))
	if got := noise.Gradient(seed, x, y, z); got != nought {
		t.Fatalf("not nought on the lattice at %v, %v, %v: %v", x, y, z, got)
	}
}

// zero on the lattice far from the middle, either side, at the points of
// lattice.csv.
func TestGradientIsNoughtOnTheLatticeFarOut(test *testing.T) {
	seed := pickedSeed(test, "field-seed")
	for _, row := range read(test, latticeFile) {
		at := float(test, row, "at")
		if got := noise.Gradient(seed, at, at+unity, -at); got != nought {
			test.Errorf("%s: not nought at the lattice point %v: %v",
				row["name"], at, got)
		}
	}
}

// anti-test: a quarter of a spacing off those lattice points the noise
// is not nought, so the lattice check is not passed by a flat field.
func TestOffTheLatticeTheNoiseIsNotNought(test *testing.T) {
	seed, off := pickedSeed(test, "field-seed"), picked(test, "quarter")
	flat := nought
	for _, row := range read(test, latticeFile) {
		at := float(test, row, "at")
		if noise.Gradient(seed, at+off, at+unity+off, -at+off) == nought {
			flat++
		}
	}
	if flat == len(read(test, latticeFile)) {
		test.Error("the noise is nought a quarter off every lattice point")
	}
}

// across a lattice plane the noise does not jump: a billionth either
// side of it, anywhere, the two values are within a few billionths.
func TestGradientIsContinuousAcrossALatticePlane(test *testing.T) {
	seed := pickedSeed(test, "field-seed")
	step, jump := picked(test, "plane-step"), picked(test, "plane-jump")
	for index := range int64(picked(test, "plane-points")) {
		at := thrown(test, index)
		x := math.Floor(at.x)
		before, after := noise.Gradient(seed, x-step, at.y, at.z),
			noise.Gradient(seed, x+step, at.y, at.z)
		if !(math.Abs(before-after) <= jump) {
			test.Fatalf("across x=%v it jumps from %v to %v", x, before,
				after)
		}
	}
}

// anti-test: the jumpy field jumps across the half plane, and the same
// check, run across that plane, sees it.
func TestTheContinuityCheckCatchesAJump(test *testing.T) {
	seed := pickedSeed(test, "field-seed")
	step, jump := picked(test, "plane-step"), picked(test, "plane-jump")
	noisy := jumpy(test)
	caught := false
	for index := range int64(picked(test, "plane-points")) {
		at := thrown(test, index)
		x := math.Floor(at.x) + half
		before, after := noisy(seed, x-step, at.y, at.z),
			noisy(seed, x+step, at.y, at.z)
		if !(math.Abs(before-after) <= jump) {
			caught = true
		}
	}
	if !caught {
		test.Fatal("a jump across every half plane went unseen")
	}
}

// recordedRows are the rows of recorded.csv that are whole-number work, Hash,
// Hash2 and Hash3, or, asked for floating, the rows of Gradient and Fractal,
// whose last bit is the machine's.
func recordedRows(test reporter, floating bool) table {
	test.Helper()
	var rows table
	for _, row := range read(test, recordedFile) {
		if (row["call"] == "Gradient" || row["call"] == "Fractal") == floating {
			rows = append(rows, row)
		}
	}
	return rows
}

// holdToRecorded is every row's number, bit for bit, as recorded. It stops
// if there are no rows, so that a table read as empty does not pass.
func holdToRecorded(test reporter, rows table) {
	test.Helper()
	if len(rows) == nought {
		test.Fatal("there are no recorded rows to hold the package to")
	}
	for _, row := range rows {
		got := called(test, row)
		if found := math.Float64bits(got); found != bits(test, row, "want") {
			test.Errorf("%s is %#016x, recorded is %s", row["name"], found,
				row["want"])
		}
	}
}

// the hashes themselves, bit for bit, as recorded: what slayer draws with.
// They are whole-number work, so they are the same on every machine. A change
// to mix or to the step shows here before it shows in a picture.
func TestTheHashesAreTheRecorded(test *testing.T) {
	holdToRecorded(test, recordedRows(test, false))
}

// anti-test: with the step added to an index turned by one, the hash of each
// Hash row is not the recorded number, and with the right step it is. So
// the recorded rows are moved by a wrong constant, and the row that moves is
// the one the step is in.
func TestAWrongStepMovesTheRecordedHashes(test *testing.T) {
	counted := nought
	for _, row := range recordedRows(test, false) {
		if row["call"] != "Hash" {
			continue
		}
		counted++
		seeded, index := seed(test, row, "seed"), whole(test, row, "a")
		recorded := bits(test, row, "want")
		right := math.Float64bits(hashStepping(golden, seeded, index))
		if right != recorded {
			test.Errorf("%s: the reference hash is %#016x, recorded is %s",
				row["name"], right, row["want"])
		}
		wrong := math.Float64bits(hashStepping(golden+unity, seeded, index))
		if wrong == recorded {
			test.Errorf("%s: a wrong step still gives the recorded number",
				row["name"])
		}
	}
	if counted == nought {
		test.Fatal("no Hash rows were tried")
	}
}

// Stirred of any word is the top 53 bits of the reference mix of it over two
// to the 53, to the bit, and is from zero up to one.
func TestStirredIsTheUnitOfTheMixedWord(test *testing.T) {
	rapid.Check(test, stirredIsTheReference)
}

func stirredIsTheReference(t *rapid.T) {
	word := rapid.Uint64().Draw(t, "word")
	want := float64(mix(word)>>spareBits) / fraction
	got := noise.Stirred(word)
	if !sameBits(got, want) || !(got >= nought && got < unity) {
		t.Fatalf("Stirred of %d is %v, the reference is %v", word, got, want)
	}
}

// Stirred of the word a recorded Hash row stirs last is the recorded number,
// bit for bit: Hash is Stirred of its seed xor its placed index.
func TestStirredIsTheRecordedHash(test *testing.T) {
	counted := nought
	for _, row := range recordedRows(test, false) {
		if row["call"] != "Hash" {
			continue
		}
		counted++
		seeded, index := seed(test, row, "seed"), whole(test, row, "a")
		placed := noise.Mix(uint64(index) + noise.Golden)
		got := math.Float64bits(noise.Stirred(seeded ^ placed))
		if got != bits(test, row, "want") {
			test.Errorf("%s: Stirred is %#016x, recorded is %s", row["name"], got,
				row["want"])
		}
	}
	if counted == nought {
		test.Fatal("no Hash rows were tried")
	}
}

// anti-test: Unit of the same word left unstirred is not the recorded Hash
// number, so the test above is not passed by a Stirred that does not stir.
func TestUnitOfAWordUnstirredIsNotTheRecordedHash(test *testing.T) {
	counted := nought
	for _, row := range recordedRows(test, false) {
		if row["call"] != "Hash" {
			continue
		}
		counted++
		seeded, index := seed(test, row, "seed"), whole(test, row, "a")
		placed := noise.Mix(uint64(index) + noise.Golden)
		got := math.Float64bits(noise.Unit(seeded ^ placed))
		if got == bits(test, row, "want") {
			test.Errorf("%s: an unstirred word is the recorded number", row["name"])
		}
	}
	if counted == nought {
		test.Fatal("no Hash rows were tried")
	}
}

// called is the number a row of recorded.csv asks for: the call it names,
// with the row's seed, indices or coordinates, and octaves.
func called(test reporter, row map[string]string) float64 {
	test.Helper()
	seeded := seed(test, row, "seed")
	switch row["call"] {
	case "Hash":
		return noise.Hash(seeded, whole(test, row, "a"))
	case "Hash2":
		return noise.Hash2(seeded, whole(test, row, "a"), whole(test, row, "b"))
	case "Hash3":
		return noise.Hash3(seeded, whole(test, row, "a"), whole(test, row, "b"),
			whole(test, row, "c"))
	case "Gradient":
		return noise.Gradient(seeded, float(test, row, "a"),
			float(test, row, "b"), float(test, row, "c"))
	case "Fractal":
		octaves := int(whole(test, row, "octaves"))
		return noise.Fractal(seeded, float(test, row, "a"),
			float(test, row, "b"), float(test, row, "c"), octaves)
	}
	test.Fatalf("%s: no call named %q", row["name"], row["call"])
	return nought
}

// Gradient can pass one, a little; the doc says where, and so it is.
func TestGradientPassesOneALittle(test *testing.T) {
	got := noise.Gradient(pickedSeed(test, "past-one-seed"),
		picked(test, "past-one-x"), picked(test, "past-one-y"),
		picked(test, "past-one-z"))
	if got <= unity || got > picked(test, "noise-bound") {
		test.Errorf("Gradient there is %v", got)
	}
}

// fractalAt is Fractal under the fractal seed at a point, of the
// fractal bound test's octaves.
func fractalAt(test reporter, at place) float64 {
	test.Helper()
	return noise.Fractal(pickedSeed(test, "fractal-seed"), at.x, at.y, at.z,
		count(test, "fractal-octaves"))
}

// fractalPoints are the points the fractal bound test tries: a hash of
// each point's number under each coordinate's seed, out to the reach.
func fractalPoints(test reporter) []place {
	test.Helper()
	reach := picked(test, "fractal-reach")
	var points []place
	for index := range int64(picked(test, "fractal-points")) {
		points = append(points, place{
			hashed(test, "coordinate-seed-x", index) * reach,
			hashed(test, "coordinate-seed-y", index) * reach,
			hashed(test, "coordinate-seed-z", index) * reach})
	}
	return points
}

// Fractal noise of several octaves stays well inside one octave's range: the
// octaves seldom line up, so their sum is seldom near the most any one can be.
func TestFractalIsBounded(test *testing.T) {
	bound := picked(test, "fractal-bound")
	for _, at := range fractalPoints(test) {
		if got := fractalAt(test, at); !(math.Abs(got) <= bound) {
			test.Fatalf("fractal %v at %v", got, at)
		}
	}
}

// anti-test: the fractal doubled is past the bound somewhere, so the
// bound is not so loose that it holds anything.
func TestTheBoundCatchesADoubledFractal(test *testing.T) {
	bound, doubled := picked(test, "fractal-bound"), picked(test, "doubled")
	for _, at := range fractalPoints(test) {
		if !(math.Abs(doubled*fractalAt(test, at)) <= bound) {
			return
		}
	}
	test.Fatal("the fractal doubled is within the bound everywhere")
}

// a fractal of no octaves or fewer is zero, and of more octaves than a float
// can scale to is still a number within one: the octaves past that are not
// added. octaves.csv says which is which.
func TestFractalOfAnyOctavesIsANumber(test *testing.T) {
	seed := pickedSeed(test, "fractal-seed")
	for _, row := range read(test, octavesFile) {
		at, octaves := point(test, row), int(whole(test, row, "octaves"))
		got := noise.Fractal(seed, at.x, at.y, at.z, octaves)
		switch expect := row["expect"]; {
		case expect == "nought" && got != nought:
			test.Errorf("%s gave %v", row["name"], got)
		case expect == "bounded" && (math.IsNaN(got) || !(math.Abs(got) <= unity)):
			test.Errorf("%s gave %v", row["name"], got)
		case expect != "nought" && expect != "bounded":
			test.Fatalf("%s: expect is %q", row["name"], expect)
		}
	}
}

// anti-test: at the same points, the fractal bound test's octaves give a
// number that is not nought somewhere, so the nought rows are not passed
// by a fractal that is nought everywhere.
func TestFractalOfSomeOctavesIsNotNoughtEverywhere(test *testing.T) {
	seed, octaves := pickedSeed(test, "fractal-seed"),
		count(test, "fractal-octaves")
	for _, row := range read(test, octavesFile) {
		at := point(test, row)
		if noise.Fractal(seed, at.x, at.y, at.z, octaves) != nought {
			return
		}
	}
	test.Fatal("the fractal is nought at every point of the table")
}

// noPlaces are points with a coordinate that is not a number, one on each
// axis: NaN, and an infinity each way.
func noPlaces() map[string]place {
	return map[string]place{
		"NaN":            {math.NaN(), nought, nought},
		"plus infinity":  {nought, plusInf(), nought},
		"minus infinity": {nought, nought, -plusInf()},
	}
}

// a point with a coordinate that is NaN or infinite is no place, and the
// fractal there is NaN, as Gradient is, whatever the octaves: of the
// picked octaves and of none: a nought there would look like a lattice
// point.
func TestFractalOfNoNumberIsNoNumber(test *testing.T) {
	seed, octaves := pickedSeed(test, "fractal-seed"),
		count(test, "fractal-octaves")
	for name, at := range noPlaces() {
		for _, layers := range []int{octaves, nought} {
			got := noise.Fractal(seed, at.x, at.y, at.z, layers)
			if !math.IsNaN(got) {
				test.Errorf("a fractal of %d octaves at %s is %v, not NaN",
					layers, name, got)
			}
		}
		if got := noise.Gradient(seed, at.x, at.y, at.z); !math.IsNaN(got) {
			test.Errorf("a gradient at %s is %v, not NaN", name, got)
		}
	}
}

// anti-test: at every point of the fractal's own table, each a place, the
// fractal is a number, so the test above is not passed by a fractal that
// is NaN everywhere.
func TestFractalOfANumberIsANumber(test *testing.T) {
	for _, at := range fractalPoints(test) {
		if got := fractalAt(test, at); math.IsNaN(got) {
			test.Fatalf("the fractal at %v, a place, is NaN", at)
		}
	}
}

// Fractals at no place is NaN for every seed, as Fractal is, and what is
// in into past the seeds is left as it was.
func TestFractalsOfNoNumberAreNoNumber(test *testing.T) {
	seed, octaves := pickedSeed(test, "fractal-seed"),
		count(test, "fractal-octaves")
	seeds := []uint64{seed, seed + unity}
	for name, at := range noPlaces() {
		kept := picked(test, "noise-bound")
		into := []float64{nought, nought, kept}
		noise.Fractals(seeds, at.x, at.y, at.z, octaves, into)
		for which := range seeds {
			if !math.IsNaN(into[which]) {
				test.Errorf("seed %d of the fractals at %s is %v, not NaN",
					which, name, into[which])
			}
		}
		if into[len(seeds)] != kept {
			test.Errorf("past the seeds at %s is %v, not the %v left there",
				name, into[len(seeds)], kept)
		}
	}
}

// anti-test: Fractals at a place gives numbers for the same seeds, so the
// test above is not passed by a Fractals that is NaN everywhere.
func TestFractalsOfANumberAreNumbers(test *testing.T) {
	seed, octaves := pickedSeed(test, "fractal-seed"),
		count(test, "fractal-octaves")
	seeds := []uint64{seed, seed + unity}
	into := make([]float64, len(seeds))
	for _, at := range fractalPoints(test) {
		noise.Fractals(seeds, at.x, at.y, at.z, octaves, into)
		for which := range seeds {
			if math.IsNaN(into[which]) {
				test.Fatalf("seed %d of the fractals at %v, a place, is NaN",
					which, at)
			}
		}
	}
}
