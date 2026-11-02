package gpu

import (
	"encoding/binary"
	"math"
	"strings"
	"testing"
)

var pair = structure{"Pair", []field{
	floatField("where", "radius in w"),
	intField("what", "material, clear"),
}}

func TestAStructureIsDeclaredToTheKernelAsItIsWritten(t *testing.T) {
	want := "struct Pair {\n" +
		"\tfloat4 where; // radius in w\n" +
		"\tint4 what; // material, clear\n" +
		"};\n"
	if got := pair.msl(); got != want {
		t.Errorf("declared as\n%s\nwant\n%s", got, want)
	}
	if pair.size() != 32 {
		t.Errorf("a pair is %d bytes, want 32", pair.size())
	}
}

func TestARowIsWrittenInTheOrderOfTheDeclaration(t *testing.T) {
	var buffer packer
	buffer.row(pair, values{"what": {7, 1}, "where": {1.5, 2, 3, 4}})
	if len(buffer.bytes) != pair.size() {
		t.Fatalf("a row is %d bytes, want %d", len(buffer.bytes),
			pair.size())
	}
	word := func(place int) uint32 {
		return binary.LittleEndian.Uint32(buffer.bytes[place*4:])
	}
	if got := math.Float32frombits(word(0)); got != 1.5 {
		t.Errorf("the first number of where is %v, want 1.5", got)
	}
	if got := int32(word(4)); got != 7 {
		t.Errorf("the first number of what is %v, want 7", got)
	}
	if got := word(7); got != 0 {
		t.Errorf("a number not given is %v, want nought", got)
	}
}

// refusal is what a row of these values is refused with, if it is.
func refusal(given values) (said string) {
	defer func() { said, _ = recover().(string) }()
	var buffer packer
	buffer.row(pair, given)
	return ""
}

func TestARowThatIsWrongIsRefusedByName(t *testing.T) {
	cases := map[string]values{
		"no value for what": {"where": {1}},
		"no field why":      {"where": {1}, "what": {1}, "why": {1}},
		"holds 4":           {"where": {1, 2, 3, 4, 5}, "what": {1}},
	}
	for want, given := range cases {
		if said := refusal(given); !strings.Contains(said, want) {
			t.Errorf("refused as %q, want %q in it", said, want)
		}
	}
}
