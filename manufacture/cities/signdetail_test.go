package cities

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/typography"
)

// tolerance is how far a length in these tests may be from what it should
// be: only rounding.
const detailTolerance = 1e-9

// madeSign is a sign of a kind drawn on its own at a detail, with its
// design, or the test's end.
func madeSign(t *testing.T, kind MountKind, seed Seed,
	level Detail) (Design, []Rendered) {
	t.Helper()
	design, err := FreeSign(kind, seed, level)
	if err != nil {
		t.Fatalf("%s from seed %d: %v", kind, seed, err)
	}
	drawn, err := design.Rendered(level)
	if err != nil {
		t.Fatalf("%s from seed %d: %v", kind, seed, err)
	}
	return design, drawn
}

// makeOf is how a design says its sign was made: the last word of its name.
func makeOf(design Design) string {
	words := strings.Fields(design.Name)
	return words[len(words)-1]
}

func TestEveryMakeOfEveryKindOfSignTurnsUpAndNoOtherDoes(t *testing.T) {
	drawing := drawerOf(t, signCulture, 1)
	for _, kind := range signKinds {
		name := "sign-make-" + string(kind)
		want := map[string]bool{}
		for _, one := range drawing.options(name) {
			want[one.Name] = false
		}
		for seed := range Seed(makeSeeds) {
			design, _ := madeSign(t, kind, seed, "quick")
			if _, known := want[makeOf(design)]; !known {
				t.Errorf("%s from seed %d is made %q, which the culture does "+
					"not say", kind, seed, makeOf(design))
			}
			want[makeOf(design)] = true
		}
		for made, seen := range want {
			if !seen {
				t.Errorf("no %s of %d seeds is made %q", kind, makeSeeds, made)
			}
		}
	}
}

func TestANearSignHasWhatHoldsItAndFeedsItAndAFarOneHasNone(t *testing.T) {
	held := []string{"sign plate", "sign bolt"}
	fed := map[MountKind]string{MountRoof: "sign conduit",
		MountScreen: "sign conduit", MountBlade: "sign conduit"}
	for _, kind := range signKinds {
		for seed := range Seed(signSeeds) {
			_, near := madeSign(t, kind, seed, "quick")
			for _, name := range held {
				if meshesNamed(near, name) == 0 {
					t.Errorf("%s from seed %d has no %s", kind, seed, name)
				}
			}
			if name, feeds := fed[kind]; feeds && meshesNamed(near, name) == 0 {
				t.Errorf("%s from seed %d has no %s", kind, seed, name)
			}
			_, far := madeSign(t, kind, seed, hullLevel)
			for _, name := range append(held, "sign conduit",
				"awning stalk") {
				if got := meshesNamed(far, name); got != 0 {
					t.Errorf("%s from seed %d at the hull has %d %s", kind,
						seed, got, name)
				}
			}
		}
	}
}

func TestTubeLettersStandOffTheirBackingOnPostsAndEndInElectrodes(t *testing.T) {
	seen := 0
	for seed := range Seed(makeSeeds) {
		design, drawn := madeSign(t, MountRoof, seed, "quick")
		if makeOf(design) != lookTube {
			continue
		}
		seen++
		tubes := meshesNamed(drawn, tubeName)
		if posts := meshesNamed(drawn, "sign standoff"); posts < tubes {
			t.Errorf("seed %d: %d tubes on %d posts", seed, tubes, posts)
		}
		if ends := meshesNamed(drawn, "sign electrode"); ends != 2*tubes {
			t.Errorf("seed %d: %d tubes with %d electrodes", seed, tubes, ends)
		}
		for _, one := range drawn {
			if one.Mesh.Name != tubeName {
				continue
			}
			for _, triangle := range one.Mesh.Triangles {
				for _, corner := range triangle {
					if math.IsNaN(corner.X+corner.Y+corner.Z) ||
						math.IsInf(corner.X+corner.Y+corner.Z, 0) {
						t.Fatalf("seed %d: a tube has a corner at %v", seed,
							corner)
					}
				}
			}
		}
	}
	if seen == 0 {
		t.Errorf("no roof sign of %d seeds was made of tubes", makeSeeds)
	}
}

// distanceToPath is how far a point is from the nearest step of a path,
// closed on itself or open.
func distanceToPath(point maths.Vec, path []maths.Vec, closed bool) float64 {
	best := math.Inf(1)
	steps := len(path) - 1
	if closed {
		steps = len(path)
	}
	for index := range steps {
		from, to := path[index], path[(index+1)%len(path)]
		way := to.Sub(from)
		along := point.Sub(from).Dot(way) / way.Dot(way)
		share := math.Max(0, math.Min(1, along))
		foot := from.Add(way.Scale(share))
		gap := point.Sub(foot).Len()
		best = math.Min(best, gap)
	}
	return best
}

func TestASweptTubeWalksRoundACornerAndCapsAnOpenEnd(t *testing.T) {
	square := []maths.Vec{maths.V(0, 0, 0), maths.V(1, 0, 0),
		maths.V(1, 1, 0), maths.V(0, 1, 0)}
	closed := sweep(square, true, 0.05)
	if want := 4 * tubeSides * 2; len(closed) != want {
		t.Errorf("a closed square tube has %d triangles, want %d",
			len(closed), want)
	}
	open := sweep(square, false, 0.05)
	if want := 3*tubeSides*2 + 2*(tubeSides-2); len(open) != want {
		t.Errorf("an open square tube has %d triangles, want %d", len(open),
			want)
	}
	for _, triangle := range closed {
		first, second := triangle[1].Sub(triangle[0]), triangle[2].Sub(triangle[0])
		normal := first.Cross(second).Unit()
		centre := middle([]maths.Vec{triangle[0], triangle[1], triangle[2]})
		near := distanceToPath(centre, square, true)
		above := centre.Add(normal.Scale(0.001))
		if far := distanceToPath(above, square,
			true); far < near {
			t.Fatalf("a triangle of a closed tube faces in: %v", triangle)
		}
	}
}

func TestAnOpenedRingHasItsEndsAGapApartOnItsLongestStep(t *testing.T) {
	ring := []maths.Vec{maths.V(0, 0, 0), maths.V(4, 0, 0), maths.V(4, 1, 0),
		maths.V(0, 1, 0)}
	path := opened(ring, 0.2)
	if len(path) != len(ring)+2 {
		t.Fatalf("an opened ring of %d points has %d", len(ring), len(path))
	}
	last := path[len(path)-1]
	if gap := path[0].Sub(last).Len(); math.Abs(gap-0.2) > detailTolerance {
		t.Errorf("the ends are %v apart, want 0.2", gap)
	}
	if path[0].Y != last.Y || (path[0].Y != 0 && path[0].Y != 1) ||
		math.Abs(path[0].X-2) > 0.2 {
		t.Errorf("the gap is at %v and %v, not in the middle of a long step",
			path[0], last)
	}
	short := opened([]maths.Vec{maths.V(0, 0, 0), maths.V(0.1, 0, 0),
		maths.V(0, 0.1, 0)}, 5)
	if gap := short[0].Sub(short[len(short)-1]).Len(); gap >=
		math.Hypot(0.1, 0.1) {
		t.Errorf("a gap of %v cuts through the step it is in", gap)
	}
}

func TestABarRunsBetweenItsEndsWhateverWayItPoints(t *testing.T) {
	steel := &thing.Material{Name: "steel"}
	for _, ends := range [][2]maths.Vec{
		{maths.V(0, 0, 0), maths.V(0, 3, 4)},
		{maths.V(1, 0, 1), maths.V(1, 4, 1)},
		{maths.V(2, 1, 0), maths.V(-2, 1, 0)},
		{maths.V(0, 5, 0), maths.V(0, 2, 3)}} {
		placed := strut("bar", ends[0], ends[1], 0.2, steel)
		box, isBox := placed.Element.(solid).Form.(Box)
		length := ends[1].Sub(ends[0]).Len()
		if !isBox || math.Abs(box.Width-length) > detailTolerance {
			t.Errorf("a bar from %v to %v is %+v long", ends[0], ends[1], box)
		}
		centre := placed.Place.Out(maths.V(box.Width/half, box.Height/half,
			box.Depth/half))
		want := ends[0].Add(ends[1]).Scale(0.5)
		if !centre.Near(want, detailTolerance) {
			t.Errorf("a bar from %v to %v is centred at %v", ends[0], ends[1],
				centre)
		}
	}
}

func TestWearIsOneThingOfOneKindAndNeverElsewhere(t *testing.T) {
	if shareOf(0, 3) != 0 || shareOf(0.999, 3) != 2 || shareOf(1, 3) != 2 {
		t.Errorf("shares of three: %d, %d, %d", shareOf(0, 3),
			shareOf(0.999, 3), shareOf(1, 3))
	}
	plan := facePlan{lines: []textLine{{}, {}},
		spot: wornSpot{kind: wearTube, line: 0.9, item: 0.5}}
	if got := plan.dark(wearTube, 1, 4); got != 2 {
		t.Errorf("the dark tube is %d, want 2", got)
	}
	if got := plan.dark(wearTube, 0, 4); got != -1 {
		t.Errorf("a tube of the other line is dark: %d", got)
	}
	if got := plan.dark(wearLetter, 1, 4); got != -1 {
		t.Errorf("a letter is dark where the wear is a tube: %d", got)
	}
	if got := (facePlan{lines: []textLine{{}}}).dark(wearTube, 0, 4); got != -1 {
		t.Errorf("a sign that has not worn has a dark tube: %d", got)
	}
}

func TestAFewSignsAreWornAndAtMostOneThingOnEachIsOut(t *testing.T) {
	worn := 0
	for seed := range Seed(makeSeeds) {
		_, drawn := madeSign(t, MountRoof, seed, "quick")
		out := 0
		for _, one := range drawn {
			if strings.HasSuffix(one.Material.Name, " out") &&
				one.Mesh.Name != "sign tube" {
				out++
			}
		}
		if out > 0 {
			worn++
		}
		if out > 1 {
			t.Errorf("seed %d has %d solid things out", seed, out)
		}
	}
	if worn == 0 || worn > makeSeeds*3/4 {
		t.Errorf("%d of %d roof signs are worn, which is not a few", worn,
			makeSeeds)
	}
}

func TestEveryWordOfEveryTradeSetsInEveryFontASignUses(t *testing.T) {
	drawing := drawerOf(t, signCulture, 1)
	for _, trade := range drawing.options("sign-trade") {
		for _, kind := range []string{"word", "short"} {
			name := "sign-" + kind + "-" + trade.Name
			if len(drawing.options(name)) == 0 {
				t.Errorf("trade %s has no %s words", trade.Name, kind)
			}
			for _, one := range drawing.options(name) {
				for _, letter := range one.Name {
					if !strings.ContainsRune(
						"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 ", letter) {
						t.Errorf("%s %q has %q, which is not a capital, a "+
							"digit or a space", name, one.Name, letter)
					}
				}
				for _, font := range []string{"go", "go-bold", "go-medium",
					"go-italic", "go-mono"} {
					face, err := typography.Named(font, "")
					if err != nil {
						t.Fatal(err)
					}
					if _, err := typography.Outlines(face, typography.Text{
						Words: one.Name, Size: 1}); err != nil {
						t.Errorf("%q in %s: %v", one.Name, font, err)
					}
				}
			}
		}
	}
}

func TestAnAwningHasStripesAValanceWithWordsAndLampsUnderIt(t *testing.T) {
	for seed := range Seed(signSeeds) {
		design, drawn := madeSign(t, MountAwning, seed, "quick")
		if meshesNamed(drawn, "awning canopy") < 2 ||
			meshesNamed(drawn, "awning valance") != 1 ||
			meshesNamed(drawn, "awning rib") == 0 ||
			meshesNamed(drawn, "awning stay") != 2 ||
			lettersOf(drawn) == 0 || len(lamps(drawn)) == 0 {
			t.Errorf("%s from seed %d is missing a part", design.Name, seed)
		}
	}
	_, far := madeSign(t, MountAwning, 1, hullLevel)
	if meshesNamed(far, "awning rib") != 0 || lettersOf(far) != 0 {
		t.Error("an awning from far off has ribs or letters")
	}
}

func TestSignsOfOneBuildingAreNotAllMadeTheSameWay(t *testing.T) {
	makes := map[string]bool{}
	mounts, err := boxOf(120, 120, 120).SignMounts()
	if err != nil {
		t.Fatal(err)
	}
	for seed := range Seed(signSeeds) {
		signing, err := newSigner(seed, ConstantLevel("quick"))
		if err != nil {
			t.Fatal(err)
		}
		for index, mount := range mounts {
			signing.hang(mount, index)
		}
		for _, made := range signing.made {
			makes[made] = true
		}
	}
	if len(makes) < 4 {
		t.Errorf("signs of %d buildings were made %d ways: %v", signSeeds,
			len(makes), makes)
	}
}
