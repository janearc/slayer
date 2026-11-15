package gpu

import (
	"encoding/binary"
	"fmt"
	"math"
	"strings"

	"github.com/janearc/slayer/render/beamstep"
)

// field is one member of a struct the kernel reads: four floats, or
// four ints. a struct is a run of these, which keeps every member on a
// sixteen-byte boundary in both languages.
type field struct {
	name string
	ints bool
	note string
}

// floatField is four floats, noted as the kernel's reader should read them.
func floatField(name, note string) field {
	return field{name: name, note: note}
}

// intField is four ints.
func intField(name, note string) field {
	return field{name: name, ints: true, note: note}
}

// bits is a number as the field holds it.
func (member field) bits(value float64) uint32 {
	if member.ints {
		return uint32(int32(int64(value)))
	}
	return math.Float32bits(float32(value))
}

// structure is a struct the kernel reads, declared once: the kernel's
// declaration is made from it, and the go side writes by its field
// names, so the two cannot disagree about order or size.
type structure struct {
	name   string
	fields []field
}

// msl is the struct as the kernel declares it.
func (made structure) msl() string {
	var text strings.Builder
	fmt.Fprintf(&text, "struct %s {\n", made.name)
	for _, member := range made.fields {
		kind := "float4"
		if member.ints {
			kind = "int4"
		}
		fmt.Fprintf(&text, "\t%s %s;", kind, member.name)
		if member.note != "" {
			fmt.Fprintf(&text, " // %s", member.note)
		}
		text.WriteString("\n")
	}
	text.WriteString("};\n")
	return text.String()
}

// size is how many bytes one of the struct takes.
func (made structure) size() int { return len(made.fields) * 16 }

// values are what to write in a struct's fields, by name.
type values map[string][]float64

// wrong is a mistake in the code, and not in a scene, so it panics and
// a test finds it.
func (made structure) wrong(format string, more ...any) {
	panic("gpu: " + made.name + ": " + fmt.Sprintf(format, more...))
}

// check refuses what a row cannot be: a field left out, one of more
// than four numbers, or a name the structure lacks.
func (made structure) check(given values) {
	known := map[string]bool{}
	for _, member := range made.fields {
		known[member.name] = true
		numbers, ok := given[member.name]
		if !ok {
			made.wrong("no value for %s", member.name)
		}
		if len(numbers) > 4 {
			made.wrong("%s holds 4, is given %d", member.name,
				len(numbers))
		}
	}
	for name := range given {
		if !known[name] {
			made.wrong("no field %s", name)
		}
	}
}

// word appends one thirty-two bit word.
func (buffer *packer) word(bits uint32) {
	buffer.bytes = binary.LittleEndian.AppendUint32(buffer.bytes, bits)
}

// row appends one struct, each field in the order the declaration gives,
// whatever order the values were written in. a field given fewer than four
// numbers has zero for the rest.
func (buffer *packer) row(made structure, given values) {
	made.check(given)
	for _, member := range made.fields {
		var numbers [4]float64
		copy(numbers[:], given[member.name])
		for _, value := range numbers {
			buffer.word(member.bits(value))
		}
	}
}

// structures is every struct the kernel reads that is declared here, in
// the order the kernel meets them.
var structures = []structure{sphereStruct,
	triangleStruct, bendStruct, nodeStruct,
	materialStruct, lampStruct, volumeStruct,
	headerStruct, treeStruct, whorledStruct, instanceStruct,
	fieldStruct}

// treeStack is how many boxes the kernel's walk of the tree can hold
// waiting: the split at the middle face keeps a tree's depth near
// log2(faces / 4), so 64 holds any tree that fits in memory.
const treeStack = 64

// nowhere is the kernel's depth for a ray into the sky, which has no
// infinity of its own; it becomes the tracer's +Inf here
const nowhere = 1e30

// lampsAsked is how many lamps a point asks when the world has more:
// trace's own lampsAsked, so the two pick alike
const lampsAsked = 8

// refractions is how many clear surfaces a look may bend through:
// trace's own refractions, so the two bend alike
const refractions = 8

// per is how many numbers the kernel gives back for every sample, and perRecord
// how many more for each place it hands back to be painted.
//
// a record is the material, the place in the thing's own space, how far the ray
// came over the thing's scale, and the light its colour and its shine are each
// to be taken by
const (
	per       = 12
	perRecord = 11
)

// numbers is what the kernel is told of the numbers the two share, for
// a scene that hands back so many records a sample and holds so many
// volumes. The kernel keeps a ray's crossings in arrays sized by
// most_volumes, so a scene pays thread memory for the volumes it has
// and no more.
func numbers(records, volumes int) string {
	return fmt.Sprintf(`constexpr constant int most_volumes = %d;
constexpr constant int most_crossings = 2 * most_volumes;
constexpr constant int refractions = `+fmt.Sprint(refractions)+`;
constexpr constant int march_steps = 256;
constexpr constant float march_touch = 1e-6;
constexpr constant int tree_stack = %d;
constexpr constant int per_sample = %d;
constexpr constant int per_base = %d;
constexpr constant int per_record = %d;
constexpr constant int records = %d;
constexpr constant float nowhere = %g;
constexpr constant int conifer_stack = %d;
constexpr constant int most_conifer_depth = %d;
constexpr constant int lamps_asked = %d;
constexpr constant int field_seed_words = 16;
constexpr constant int boxed_kind = %d;
constexpr constant int shell_kind = %d;
constexpr constant float beam_zero = %g;
constexpr constant float beam_one = %g;
constexpr constant float beam_half = %g;
constexpr constant float beam_bound_radii = %g;
constexpr constant float beam_radial_samples = %g;
constexpr constant int beam_max_splits = %d;
constexpr constant int lamp_polar = %d;
constexpr constant int lamp_flood = %d;
constexpr constant float flood_whole = %g;
constexpr constant float flood_half = %g;
constexpr constant int gas_on = %d;
constexpr constant float gas_rayleigh = %.9g;
constexpr constant float gas_four_pi = %.9g;
constexpr constant float gas_lobe_power = %g;
constexpr constant float gas_twice = %g;
constexpr constant float clip_share = %g;
constexpr constant float clip_rounding = %g;
`, max(volumes, 1), treeStack, per+perRecord*records, per, perRecord,
		records, nowhere, coniferStack, mostConiferDepth, lampsAsked,
		boxedKind, shellKind, float64(beamZero), float64(beamWhole), beamHalf,
		beamstep.Radii(), beamstep.RadialSamples(),
		beamstep.MaximumSplits(), polarKind, floodKind, float64(beamWhole), beamHalf,
		gasOn, rayleighLobe, fourPi, lobePower, float64(lobeTwice),
		clipShare, clipRounding)
}

// clipShare and clipRounding are how far a hardware query's segment is
// lengthened at each end past where it crosses the sphere round the scene.
// clipShare is that share of the sphere's radius, picked as one percent so
// a hit at the sphere's edge is never cut off by the hardware's own box
// tests.
//
// clipRounding is that much of the ray's distance from the sphere, for the
// rounding of the arithmetic that finds the crossing: clipUnits steps of
// one float's last place, picked as eight times the two or three roundings
// a dot or a cross product of a far origin makes.
const (
	clipShare    = 0.01
	clipRounding = clipUnits * floatLastPlace
	clipUnits    = 16
	// floatLastPlace is the gap above one to the next 32-bit float,
	// two to the minus twenty-three, true by definition of the format.
	floatLastPlace = 1.0 / (1 << 23)
)

// beamWhole and beamHalf are defined as a unit cell and its midpoint.
const beamWhole, beamHalf = 1, 0.5

// the counting numbers the kernel counts and indexes by, and the same
// numbers as floats for its sums, each true by definition.
const (
	wholeZero = iota
	wholeOne
	wholeTwo
	wholeThree
	wholeFour
	wholeFive
)

// beamTwo to beamFive are wholeTwo to wholeFive, as the floats the kernel's
// sums use.
const (
	beamTwo   = float64(wholeTwo)
	beamThree = float64(wholeThree)
	beamFour  = float64(wholeFour)
	beamFive  = float64(wholeFive)
)

// flatUnasked, flatYes and flatNo are what a ring's crossing knows of its
// own flatness while a ray steps through it: not yet asked, flat, and not
// flat. they are three names for three states, true by definition.
const (
	flatUnasked = iota
	flatYes
	flatNo
)

// seedHighWord is how far past a seed's low word its high word sits, in a
// card's words, and seedHalfBits is how many bits the low half of a seed
// holds. both are true by definition: a sixty-four bit seed is two
// thirty-two bit words, low first.
const (
	seedHighWord = wholeOne
	seedHalfBits = 32
)

// wholeNumbers is the kernel's declaration of the whole numbers and the
// small floats its arithmetic is made of, the ones the numbers above name.
func wholeNumbers() string {
	var text strings.Builder
	for _, one := range []struct {
		name  string
		whole int
	}{{"whole_zero", wholeZero}, {"whole_one", wholeOne},
		{"whole_two", wholeTwo}, {"whole_three", wholeThree},
		{"flat_unasked", flatUnasked}, {"flat_yes", flatYes},
		{"flat_no", flatNo}, {"seed_high_word", seedHighWord},
		{"seed_half_bits", seedHalfBits}, {"hash_drop", hashDrop}} {
		fmt.Fprintf(&text, "constexpr constant int %s = %d;\n", one.name,
			one.whole)
	}
	for _, one := range []struct {
		name string
		part float64
	}{{"beam_two", beamTwo}, {"beam_three", beamThree},
		{"beam_four", beamFour}, {"beam_five", beamFive}} {
		fmt.Fprintf(&text, "constexpr constant float %s = %g;\n", one.name,
			one.part)
	}
	return text.String()
}

// prelude is the kernel's declaration of every number and structure
// the two share, for a scene that hands back so many records a sample
// and holds so many volumes.
func prelude(records, volumes int) string {
	var text strings.Builder
	text.WriteString(numbers(records, volumes))
	text.WriteString(wholeNumbers())
	text.WriteString(reachNumbers())
	text.WriteString(ringNumbers())
	text.WriteString(wakeNumbers())
	text.WriteString(exposureNumbers())
	for _, made := range structures {
		text.WriteString(made.msl())
	}
	return text.String()
}
