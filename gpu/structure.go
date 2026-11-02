package gpu

import (
	"encoding/binary"
	"fmt"
	"math"
	"strings"
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
	headerStruct, treeStruct, whorledStruct, instanceStruct}

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
// a scene that hands back so many records a sample.
func numbers(records int) string {
	return fmt.Sprintf(`constexpr constant int most_volumes = %d;
constexpr constant int tree_stack = %d;
constexpr constant int per_sample = %d;
constexpr constant int per_base = %d;
constexpr constant int per_record = %d;
constexpr constant int records = %d;
constexpr constant float nowhere = %g;
constexpr constant int conifer_stack = %d;
constexpr constant int most_conifer_depth = %d;
constexpr constant int lamps_asked = %d;
constexpr constant int boxed_kind = %d;
`, mostVolumes, treeStack, per+perRecord*records, per, perRecord,
		records, nowhere, coniferStack, mostConiferDepth, lampsAsked,
		boxedKind)
}

// prelude is the kernel's declaration of every number and structure
// the two share, for a scene that hands back so many records a sample.
func prelude(records int) string {
	var text strings.Builder
	text.WriteString(numbers(records))
	for _, made := range structures {
		text.WriteString(made.msl())
	}
	return text.String()
}
