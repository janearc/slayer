package pages

import (
	"cmp"
	"strings"
)

// collation is the order the Mac's sort puts printable characters in
// under en_US.UTF-8, which the scripts run in. It was measured on
// 2026-11-15 by sorting "x" + c + "y" for each one: punctuation, then
// digits, then letters with case set aside.
const collation = " _-,;:!?.'\"()[]{}@*/\\&#%`^+<=>|~$" +
	"0123456789abcdefghijklmnopqrstuvwxyz"

// beyond is the first weight of a character the measure did not cover:
// after every measured one, in the order of its code.
const beyond = len(collation) // defined: one past the last measured weight

// capital is the second weight of a capital letter; lower case is nought,
// so "ab" sorts before "aB", which sorts before "Ab".
const capital = 1 // defined: the sort puts lower case first

// caseApart is the distance from a capital to its lower case letter.
const caseApart = 'a' - 'A' // defined: the ASCII table

// firstWeight is a character's place when case is set aside.
func firstWeight(character rune) int {
	if character >= 'A' && character <= 'Z' {
		character += caseApart
	}
	if at := strings.IndexRune(collation, character); at >= 0 {
		return at
	}
	return beyond + int(character)
}

// secondWeight tells a capital from its lower case letter.
func secondWeight(character rune) int {
	if character >= 'A' && character <= 'Z' {
		return capital
	}
	return 0
}

// collate orders two strings as the Mac's sort does: by the characters
// with case set aside, then by case, and last by their bytes.
func collate(a, b string) int {
	if order := byWeight(a, b, firstWeight); order != 0 {
		return order
	}
	if order := byWeight(a, b, secondWeight); order != 0 {
		return order
	}
	return cmp.Compare(a, b)
}

// byWeight compares two strings a character at a time by one weight; a
// string that runs out first sorts first.
func byWeight(a, b string, weight func(rune) int) int {
	first, second := []rune(a), []rune(b)
	for index := range min(len(first), len(second)) {
		order := cmp.Compare(weight(first[index]), weight(second[index]))
		if order != 0 {
			return order
		}
	}
	return cmp.Compare(len(first), len(second))
}
