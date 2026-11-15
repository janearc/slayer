// Package facts is every number the quasar shorts say, and the check that
// a sentence says no other.
//
// A Number is either registered, read from research by its name (and so
// held to its source by sources verify), or computed here from
// registered and defined values by a function with a test. Each knows
// how it is written on screen and where it came from.
//
// A Sentence is words and the numbers it uses. NewSentence refuses a
// sentence with a numeral that none of its numbers is written as, and a
// number it was handed and does not say: a sentence naming a number that
// is neither registered nor computed cannot be made.
package facts

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/janearc/slayer/research"
)

// Number is a number a short says: its value, how it is written, and
// where it came from.
type Number struct {
	value float64
	shown string
	from  string
}

// Value is the number's value.
func (number Number) Value() float64 { return number.value }

// Shown is the number as it is written on screen.
func (number Number) Shown() string { return number.shown }

// From is where the number came from: a registered name, or the
// function that computed it and what from.
func (number Number) From() string { return number.from }

// registered is a value from research by its name, written to so many
// places after the point.
func registered(name string, places int) Number {
	value := research.Value(name)
	return Number{value: value, shown: written(value, places),
		from: "research " + name}
}

// computed is a value worked out here, written to so many places after
// the point, with what it was worked out from.
func computed(value float64, places int, from string) Number {
	return Number{value: value, shown: written(value, places), from: from}
}

// Computed is a value worked out by a caller from these numbers, such as
// a width in pixels the rings package measured, written to so many
// places after the point. It refuses a value that is not a number or is
// infinite, and fewer than no places.
func Computed(value float64, places int, from string) (Number, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) || places < none {
		return Number{}, fmt.Errorf("facts: %v to %d places, from %s, "+
			"cannot be written: give a finite value and places of "+
			"nought or more", value, places, from)
	}
	if strings.TrimSpace(from) == "" {
		return Number{}, fmt.Errorf("facts: %v has no source: say what "+
			"it was computed from", value)
	}
	return computed(value, places, from), nil
}

// written is a value to so many places after the point.
func written(value float64, places int) string {
	return strconv.FormatFloat(value, 'f', places, bits)
}

// bits is the float's width, as strconv asks: defined.
const bits = 64

// Sentence is a line a short says, and the numbers in it.
type Sentence struct {
	text    string
	numbers []Number
}

// Text is the sentence's words.
func (sentence Sentence) Text() string { return sentence.text }

// Numbers is the numbers the sentence says.
func (sentence Sentence) Numbers() []Number {
	return append([]Number(nil), sentence.numbers...)
}

// numeral is a run of digits, with a point and digits after it if any.
var numeral = regexp.MustCompile(`[0-9]+(\.[0-9]+)?`)

// names are the objects whose names hold digits that are not numbers a
// sentence says: a catalogue's names, not measurements.
var names = []string{"TON 618", "M87", "Sgr A*"}

// Numerals is every numeral in words, as NewSentence finds them: the
// objects' names taken out first, since their digits are names.
func Numerals(text string) []string {
	for _, name := range names {
		text = strings.ReplaceAll(text, name, "")
	}
	return numeral.FindAllString(text, -one)
}

// NewSentence is words that say these numbers. It refuses a numeral in
// the words that none of the numbers is written as, and a number the
// words do not say.
func NewSentence(text string, numbers ...Number) (Sentence, error) {
	said := map[string]bool{}
	for _, found := range Numerals(text) {
		said[found] = true
		if !shownBy(found, numbers) {
			return Sentence{}, fmt.Errorf("facts: %q says %s, which "+
				"no registered or computed number is written as: "+
				"hand it the number, or take it out", text, found)
		}
	}
	for _, number := range numbers {
		if !said[number.shown] {
			return Sentence{}, fmt.Errorf("facts: %q does not say %s "+
				"(%s): take the number off or say it", text,
				number.shown, number.from)
		}
	}
	return Sentence{text: text, numbers: append([]Number(nil),
		numbers...)}, nil
}

// shownBy is whether one of the numbers is written as a numeral.
func shownBy(found string, numbers []Number) bool {
	for _, number := range numbers {
		if number.shown == found {
			return true
		}
	}
	return false
}

// The counts the package is made of: defined.
const (
	none = 0
	one  = 1
)
