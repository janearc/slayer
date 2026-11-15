package explain

// line is one sung line of the chant: its words, and when it is up, as
// the song's own lyric track times it.
type line struct {
	words       string
	from, until float64
}

// Chant is the line being sung at a moment, if any: up from its start to
// its end, and not outside. The line is shown whole: no word in it is
// singled out, since nothing here times the words within a line.
func (of *Script) Chant(seconds float64) (string, bool) {
	for _, each := range of.chant {
		if seconds >= each.from && seconds < each.until {
			return each.words, true
		}
	}
	return "", false
}
