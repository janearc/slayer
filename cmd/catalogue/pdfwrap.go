package main

import "strings"

// wrapText breaks text into whole-word lines no wider than width
// points when set in a face at a size. A word wider than the width on
// its own is split between characters; nothing is dropped and nothing
// is replaced with an ellipsis. The words rejoin to the text.
func wrapText(set face, size float64, text string,
	width float64) []string {
	var lines []string
	line := ""
	for _, word := range strings.Fields(text) {
		for set.measure(word, size) > width {
			head, rest := splitWord(set, size, word, width)
			if line != "" {
				lines = append(lines, line)
				line = ""
			}
			lines = append(lines, head)
			word = rest
		}
		switch {
		case line == "":
			line = word
		case set.measure(line+" "+word, size) <= width:
			line += " " + word
		default:
			lines = append(lines, line)
			line = word
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}

// splitWord cuts the longest head of a word that fits the width,
// always taking at least one character so that a column narrower than
// a letter still makes progress. Where the head holds a mark that
// addresses break after, in its later half, the cut is made there, so
// an address is not left with a single letter on its last line.
func splitWord(set face, size float64, word string,
	width float64) (head, rest string) {
	letters := []rune(word)
	taken := firstPlace
	for taken < len(letters) &&
		set.measure(string(letters[:taken+firstPlace]), size) <= width {
		taken++
	}
	for cut := taken; cut > taken/halvedWord; cut-- {
		if strings.ContainsRune(breakAfter, letters[cut-firstPlace]) {
			taken = cut
			break
		}
	}
	return string(letters[:taken]), string(letters[taken:])
}

// breakAfter is picked: the marks in an address after which a line
// may end.
const breakAfter = "/-_?&=.,:;"

// halvedWord is defined as half: a cut is only moved back as far as
// the middle of the line.
const halvedWord = 2

// firstPlace is defined as one character: the least a split takes.
const firstPlace = 1
