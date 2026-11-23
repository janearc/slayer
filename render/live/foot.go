package live

import "strings"

// foot is the notes and errors shown above the status row: the last few,
// newest at the bottom, and a trouble with the job file, which stays
// as the newest line until the file reads again.
type foot struct {
	notes   []string
	trouble string
}

// note keeps a line, each line of a note with several its own, and lets
// the oldest go past the rows the foot has.
func (shown *foot) note(text string) {
	trimmed := strings.TrimSpace(text)
	shown.notes = append(shown.notes,
		strings.Split(trimmed, "\n")...)
	if extra := len(shown.notes) - FootRows; extra > none {
		shown.notes = shown.notes[extra:]
	}
}

// troubled holds a sentence about the job file until it is cleared, its
// lines joined into one.
func (shown *foot) troubled(text string) {
	shown.trouble = strings.Join(strings.Fields(text), " ")
}

// cleared lets the trouble go, and says whether there was one.
func (shown *foot) cleared() bool {
	had := shown.trouble != ""
	shown.trouble = ""
	return had
}

// lines is the foot's rows, top first: as many as it has, the trouble
// last, padded with empty lines above to the foot's rows.
func (shown *foot) lines() []string {
	held := append([]string(nil), shown.notes...)
	if shown.trouble != "" {
		held = append(held, shown.trouble)
	}
	if extra := len(held) - FootRows; extra > none {
		held = held[extra:]
	}
	padded := make([]string, FootRows-len(held), FootRows)
	return append(padded, held...)
}
