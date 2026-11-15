package main

import (
	"math"
	"math/rand"
	"strings"
	"testing"
)

// knownAdvances are widths read off the Adobe Core 14 metrics by
// hand: a few letters, a digit, a space and a mark of each kind.
var knownAdvances = []struct {
	set     face
	letter  rune
	advance int
}{
	{helvetica, 'i', 222}, {helvetica, 'M', 833}, {helvetica, ' ', 278},
	{helvetica, '5', 556}, {helvetica, 'é', 556},
	{helvetica, '°', 400}, {helvetica, '—', 1000},
	{helveticaBold, 'W', 944}, {helveticaBold, 'r', 389},
	{helveticaBold, 'ç', 556}, {courier, 'i', 600},
	{courier, 'W', 600},
}

func TestAdvancesMatchTheCoreMetrics(test *testing.T) {
	for _, each := range knownAdvances {
		got, held := each.set.advance(each.letter)
		if !held || got != each.advance {
			test.Fatalf("%s %q: got %d, want %d", each.set.base,
				each.letter, got, each.advance)
		}
	}
}

func TestEveryEncodedCharacterHasAWidthInEveryFace(test *testing.T) {
	for _, set := range []face{helvetica, helveticaBold, courier} {
		var missing []rune
		for letter := range winAnsiByte {
			if _, held := set.advance(letter); !held {
				missing = append(missing, letter)
			}
		}
		if len(missing) > noEntries {
			test.Fatalf("%s has no width for %q", set.base,
				string(missing))
		}
	}
}

func TestNoWidthGroupNamesALetterTwiceOrOneOutsideTheEncoding(test *testing.T) {
	for name, groups := range map[string][]widthGroup{
		"Helvetica": helveticaWidths, "Helvetica-Bold": boldWidths} {
		seen := map[rune]bool{}
		for _, group := range groups {
			for _, letter := range group.letters {
				if seen[letter] {
					test.Fatalf("%s lists %q twice", name, letter)
				}
				if _, held := winAnsiByte[letter]; !held {
					test.Fatalf("%s has %q, which is not encoded", name,
						letter)
				}
				seen[letter] = true
			}
		}
	}
}

func TestACharacterOutsideTheEncodingHasNoWidthAndIsRefused(test *testing.T) {
	arrow := '→'
	if _, held := helvetica.advance(arrow); held {
		test.Fatal("an arrow has a width")
	}
	_, err := helvetica.encode("a " + string(arrow) + " b")
	if err == nil || !strings.Contains(err.Error(), "U+2192") {
		test.Fatal("the arrow was not refused by name:", err)
	}
	coded, err := helvetica.encode("café")
	if err != nil || len(coded) != len("café")-firstName ||
		coded[len(coded)-lastPlace] != 0xe9 {
		test.Fatal("an accent was not encoded as one byte:", coded, err)
	}
}

func TestTheUnassignedBytesAreNotInTheEncoding(test *testing.T) {
	for _, letter := range "\u0081\u008d\u008f\u0090\u009d" {
		if _, held := winAnsiByte[letter]; held {
			test.Fatalf("%U is in the encoding", letter)
		}
	}
	if winAnsiByte['€'] != 0x80 || winAnsiByte['’'] != 0x92 {
		test.Fatal("the typographic marks are at the wrong bytes")
	}
}

func TestMeasureSumsTheAdvances(test *testing.T) {
	// hello is Helvetica's H e l l o: 722 556 222 222 556.
	const hello, size = 2278.0, 10.0
	got := helvetica.measure("Hello", size)
	if math.Abs(got-hello*size/em) > 1e-9 {
		test.Fatalf("measured %g", got)
	}
	if helvetica.measure("", size) != 0 {
		test.Fatal("nothing has a width")
	}
}

// randomText is words of random letters and lengths, some longer than
// a column, drawn from a seeded source.
func randomText(source *rand.Rand) string {
	// alphabet holds wide and narrow letters, accents and a mark.
	const alphabet = "iIlWMmwx0 éÆ.-/"
	// longest is picked: the longest run of letters in a test text.
	const longest = 90
	var made strings.Builder
	for made.Len() < longest*3 {
		for range source.Intn(longest) + firstName {
			made.WriteRune([]rune(alphabet)[source.Intn(
				len([]rune(alphabet)))])
		}
		made.WriteString(" ")
	}
	return made.String()
}

func TestAWrappedLineNeverExceedsTheColumn(test *testing.T) {
	// seed and trials are picked: a fixed source, many texts.
	const seed, trials = 20261112, 300
	source := rand.New(rand.NewSource(seed))
	for range trials {
		width := columnWidth / float64(source.Intn(4)+firstName)
		size := bodyStyle.size
		text := randomText(source)
		lines := wrapText(helvetica, size, text, width)
		for _, line := range lines {
			if helvetica.measure(line, size) > width {
				test.Fatalf("%q is wider than %g", line, width)
			}
		}
		if strings.Join(lines, " ") != strings.Join(strings.Fields(text),
			" ") && strings.Join(strings.Fields(strings.Join(lines, " ")),
			"") != strings.Join(strings.Fields(text), "") {
			test.Fatal("wrapping lost or added letters")
		}
	}
}

func TestWrappingKeepsWholeWordsTogetherWhenTheyFit(test *testing.T) {
	lines := wrapText(helvetica, bodyStyle.size,
		"one two three four five six", helvetica.measure("one two three",
			bodyStyle.size))
	if lines[noEntries] != "one two three" ||
		strings.Join(lines, " ") != "one two three four five six" {
		test.Fatal("words were split that fit:", lines)
	}
}

func TestAWordWiderThanTheColumnIsSplitAndNothingIsDropped(test *testing.T) {
	word := strings.Repeat("W", 40)
	width := helvetica.measure("WWWWW", bodyStyle.size)
	lines := wrapText(helvetica, bodyStyle.size, "a "+word, width)
	if strings.Join(lines, "") != "a"+word {
		test.Fatal("hard splitting lost letters:", lines)
	}
	for _, line := range lines {
		if helvetica.measure(line, bodyStyle.size) > width {
			test.Fatalf("%q runs past the column", line)
		}
		if strings.Contains(line, "...") {
			test.Fatal("an ellipsis stands in for text")
		}
	}
}

func TestAColumnNarrowerThanALetterStillMakesProgress(test *testing.T) {
	lines := wrapText(helvetica, bodyStyle.size, "WWW", 1)
	if strings.Join(lines, "") != "WWW" {
		test.Fatal("no progress in a narrow column:", lines)
	}
}

func TestAnAddressIsCutAfterAMarkAndNotLeftWithOneLetter(test *testing.T) {
	address := "https://example.com/products/127mm-fcd100-apochromatic-x"
	width := helvetica.measure(address[:len(address)-firstPlace*3],
		bodyStyle.size)
	lines := wrapText(helvetica, bodyStyle.size, address, width)
	if len(lines) != twoCards || strings.Join(lines, "") != address ||
		!strings.HasSuffix(lines[noEntries], "-") {
		test.Fatal("the address was cut badly:", lines)
	}
}

func TestAnUnbreakableWordIsCutWhereItFits(test *testing.T) {
	// word has no mark to break after.
	const word = "abcdefghijklmnopqrstuvwxyz"
	width := helvetica.measure(word[:10], bodyStyle.size)
	lines := wrapText(helvetica, bodyStyle.size, word, width)
	if strings.Join(lines, "") != word || len(lines[noEntries]) < 10 {
		test.Fatal("the word was cut short:", lines)
	}
}
