package cacti

import (
	_ "embed"
	"fmt"
	"maps"
	"math"
	"slices"

	"github.com/janearc/slayer/manufacture/branching"
	"github.com/janearc/slayer/manufacture/profilecheck"
	"github.com/janearc/slayer/maths"
)

//go:embed cactus-profiles.yaml
var profilesRaw []byte

// Policy is the caller's checked construction policy, separate from species.
type Policy struct {
	Lookup                     func(string) float64
	Details                    []string
	MinimumSides, MaximumSides int
}

type cactusProfile struct {
	Kind, Status, By, Date, Source string
	Values                         map[string]float64
}
type Book struct {
	profiles map[string]cactusProfile
	policy   Policy
}
type Recipe struct {
	values map[string]float64
	policy Policy
}
type Column struct {
	Stem Stem
	Pose maths.Subspace
}

var cactusSpeciesFields = []string{
	"height-minimum", "height-maximum", "radius-minimum", "radius-maximum",
	"arms-minimum", "arms-maximum", "ribs", "pleat-depth", "attachment-minimum",
	"attachment-maximum", "root-radius-share", "arm-radius-share", "arm-turn-variation",
	"arm-run-minimum", "arm-run-maximum", "bend-minimum", "bend-maximum",
	"spine-length", "spine-radius", "spine-spacing", "spine-root-share",
	"green-r", "green-g", "green-b", "spine-r", "spine-g", "spine-b",
}

// valid refuses an incomplete construction policy.
func (policy Policy) valid() error {
	if policy.Lookup == nil || policy.MinimumSides < 3 || policy.MaximumSides < policy.MinimumSides || len(policy.Details) == 0 {
		return fmt.Errorf("invalid cactus construction policy")
	}
	return nil
}

// Embedded returns an independent copy of the default species book.
func Embedded() []byte { return slices.Clone(profilesRaw) }

func Read(file string, raw []byte, policy Policy) (*Book, error) {
	policy.Details = slices.Clone(policy.Details)
	if err := policy.valid(); err != nil {
		return nil, err
	}
	rows, err := profilecheck.PlantRows[cactusProfile](file, raw, nil)
	if err != nil {
		return nil, err
	}
	book := &Book{profiles: make(map[string]cactusProfile, len(rows)), policy: policy}
	for _, profile := range rows {
		if !profilecheck.Authored(profile.Kind, profile.Status, profile.By, profile.Date, profile.Source) {
			return nil, fmt.Errorf("manufacture: %s: invalid cactus provenance for %q", file, profile.Kind)
		}
		if _, ok := book.profiles[profile.Kind]; ok {
			return nil, fmt.Errorf("manufacture: %s: duplicate cactus %q", file, profile.Kind)
		}
		if len(profile.Values) != len(cactusSpeciesFields) {
			return nil, fmt.Errorf("manufacture: %s: wrong cactus values for %q", file, profile.Kind)
		}
		for _, name := range cactusSpeciesFields {
			if _, ok := profile.Values[name]; !ok {
				return nil, fmt.Errorf("manufacture: %s: cactus %q needs a numeric %s", file, profile.Kind, name)
			}
		}
		recipe := &Recipe{values: profile.Values, policy: policy}
		if err := recipe.Validate(); err != nil {
			return nil, fmt.Errorf("manufacture: %s: cactus %q: %w", file, profile.Kind, err)
		}
		book.profiles[profile.Kind] = profile
	}
	return book, nil
}

func (book *Book) New(name string) (*Recipe, error) {
	if book == nil {
		return nil, fmt.Errorf("manufacture: no cactus book")
	}
	profile, ok := book.profiles[name]
	if !ok {
		return nil, fmt.Errorf("manufacture: no cactus profile %q", name)
	}
	return &Recipe{values: profile.Values, policy: book.policy}, nil
}
func (book *Book) Species() []string {
	if book == nil {
		return nil
	}
	return slices.Sorted(maps.Keys(book.profiles))
}
func (plant *Recipe) Value(name string) float64 {
	if value, ok := plant.values[name]; ok {
		return value
	}
	return plant.policy.Lookup(name)
}

// Columns selects a seeded trunk and its arms in physical rest space.
func (plant *Recipe) Columns(from branching.Seed) []Column {
	height := plant.pick(from, "height", 0)
	radius := plant.pick(from, "radius", 1)
	columns := []Column{{Stem: Stem{Run: height, Radius: radius, Ribs: int(plant.Value("ribs")), Depth: plant.Value("pleat-depth")}, Pose: maths.Local()}}
	low, high := plant.Value("arms-minimum"), plant.Value("arms-maximum")
	count := maths.CountBetween(low, high, from.Draw(0, 0, 2))
	for index := range count {
		columns = append(columns, plant.arm(from, index, count, height, radius))
	}
	return columns
}
func (plant *Recipe) pick(from branching.Seed, name string, index int) float64 {
	low, high := plant.Value(name+"-minimum"), plant.Value(name+"-maximum")
	return low + (high-low)*from.Draw(0, index, 0)
}
func (plant *Recipe) arm(from branching.Seed, index, count int, height, radius float64) Column {
	branch := from.Sprout(index, 1)
	turn := 2 * math.Pi * (float64(index)/float64(count) + plant.Value("arm-turn-variation")*branch.Swing(1, index, 0))
	sine, cosine := math.Sincos(turn)
	attachment := plant.pick(branch, "attachment", 0) * height
	inset := radius * plant.Value("root-radius-share")
	at := maths.V(inset*cosine, attachment, -inset*sine)
	pose := maths.NewSubspace(at, maths.Rotation(maths.YAxis, turn), 1)
	return Column{Stem: Stem{Run: plant.pick(branch, "arm-run", 1), Radius: radius * plant.Value("arm-radius-share"), Bend: plant.pick(branch, "bend", 2), Ribs: int(plant.Value("ribs")), Depth: plant.Value("pleat-depth")}, Pose: pose}
}

// Surface applies the chosen detail without changing physical dimensions.
func (plant *Recipe) Surface(column Column, detail string) (Stem, int) {
	stem := column.Stem
	stem.Curve = int(plant.Value("curve-" + detail))
	stem.Cap = int(plant.Value("cap-" + detail))
	sides := int(plant.Value("sides-" + detail))
	if detail == "hull" || detail == "clumps" {
		stem.Depth = 0
	}
	return stem, sides
}
func positive(n float64) bool { return n > 0 && !math.IsInf(n, 0) }

// validCactusChoices bounds private data edits before integer loops and sweeps.
func (plant *Recipe) Validate() error {
	// Tiny positive dimensions can collapse after rounding when a stem is
	// meshed. These are numerical margins, not botanical measurements.
	largest := math.Max(plant.Value("height-maximum"), math.Max(plant.Value("radius-maximum"), plant.Value("arm-run-maximum")))
	minimum := math.Max(1e-9, largest*1e-9)
	for _, name := range []string{"height", "radius", "arm-run", "bend", "attachment"} {
		low, high := plant.Value(name+"-minimum"), plant.Value(name+"-maximum")
		if !positive(low) || low < minimum || !positive(high) || high < low ||
			high > plant.policy.Lookup("maximum-cactus-metres") {
			return fmt.Errorf("manufacture: invalid cactus range %s", name)
		}
	}
	for _, name := range []string{"ribs", "maximum-spines", "arms-minimum", "arms-maximum"} {
		value := plant.Value(name)
		if !cactusBetween(value, 0, plant.policy.Lookup("maximum-cactus-count")) ||
			value != math.Trunc(value) {
			return fmt.Errorf("manufacture: invalid cactus count %s", name)
		}
	}
	if err := plant.validCactusShape(); err != nil {
		return err
	}
	return plant.validCactusDetail()
}

// cactusBetween explicitly refuses NaN as well as infinities outside a range.
func cactusBetween(value, low, high float64) bool { return value >= low && value <= high }

// validCactusShape prevents an arm's inner bend from folding over itself.
func (plant *Recipe) validCactusShape() error {
	if plant.Value("arms-minimum") > plant.Value("arms-maximum") ||
		plant.Value("arms-maximum") > plant.policy.Lookup("maximum-stems") ||
		plant.Value("ribs") < float64(plant.policy.MinimumSides) || plant.Value("ribs") > float64(plant.policy.MaximumSides) ||
		plant.Value("maximum-spines") < 1 {
		return fmt.Errorf("manufacture: excessive cactus construction")
	}
	armRadius := plant.Value("radius-maximum") * plant.Value("arm-radius-share")
	rootEnvelope := math.Hypot(plant.Value("root-radius-share"), plant.Value("arm-radius-share"))
	innerSkin := 1 - 2*plant.Value("pleat-depth")
	if !positive(armRadius) || armRadius >= plant.Value("bend-minimum") ||
		!(rootEnvelope < innerSkin) ||
		plant.Value("attachment-maximum") >= 1 ||
		!cactusBetween(plant.Value("pleat-depth"), 0, plant.policy.Lookup("maximum-pleat-depth")) ||
		!cactusBetween(plant.Value("root-radius-share"), 0, 1) ||
		!cactusBetween(plant.Value("arm-turn-variation"), 0, .5) {
		return fmt.Errorf("manufacture: invalid cactus arm or pleat")
	}
	return plant.validCactusSpines()
}

// validCactusSpines checks fixed physical microgeometry and its painted
// colours.
func (plant *Recipe) validCactusSpines() error {
	for _, name := range []string{"spine-length", "spine-radius", "spine-spacing"} {
		if !positive(plant.Value(name)) || plant.Value(name) > plant.policy.Lookup("maximum-cactus-metres") {
			return fmt.Errorf("manufacture: invalid cactus %s", name)
		}
	}
	// The cone's cap has area proportional to radius squared; finite but
	// vanishing radii otherwise become zero-area triangles.
	if plant.Value("spine-radius") < math.Max(1e-9, plant.Value("spine-length")*1e-9) || plant.Value("spine-length") < 1e-9 {
		return fmt.Errorf("manufacture: cactus spine is too small to mesh")
	}
	if !cactusBetween(plant.Value("spine-root-share"), 0, 1) {
		return fmt.Errorf("manufacture: invalid cactus spine root")
	}
	// Check the worst possible selected size and arm count before converting
	// ratios to integers or allocating any spine instances.
	rows := math.Floor(plant.Value("height-maximum")/plant.Value("spine-spacing")) +
		plant.Value("arms-maximum")*math.Floor(plant.Value("arm-run-maximum")/plant.Value("spine-spacing"))
	if math.IsNaN(rows) || math.IsInf(rows, 0) ||
		rows > plant.Value("maximum-spines")/plant.Value("ribs") {
		return fmt.Errorf("manufacture: cactus exceeds spine budget")
	}
	for _, name := range []string{"green-r", "green-g", "green-b", "spine-r", "spine-g", "spine-b"} {
		if !cactusBetween(plant.Value(name), 0, 1) {
			return fmt.Errorf("manufacture: invalid cactus colour %s", name)
		}
	}
	return nil
}

// validCactusDetail bounds all path and ring subdivisions before allocation.
func (plant *Recipe) validCactusDetail() error {
	for _, detail := range plant.policy.Details {
		for _, field := range []string{"curve-", "cap-", "sides-"} {
			value := plant.Value(field + detail)
			if !cactusBetween(value, 1, float64(plant.policy.MaximumSides)) || value != math.Trunc(value) {
				return fmt.Errorf("manufacture: invalid cactus detail %s%s", field, detail)
			}
		}
		if plant.Value("sides-"+detail) < float64(plant.policy.MinimumSides) {
			return fmt.Errorf("manufacture: cactus ring cannot close")
		}
	}
	for _, detail := range []string{"quick", "demo"} {
		value := plant.Value("spine-sides-" + detail)
		if !cactusBetween(value, float64(plant.policy.MinimumSides), float64(plant.policy.MaximumSides)) || value != math.Trunc(value) {
			return fmt.Errorf("manufacture: invalid cactus spine detail %s", detail)
		}
	}
	return nil
}
