package workspace

import "strings"

// lastLine is the last line of transcript that has anything in it, with its
// ends trimmed. It is empty when the transcript has none.
func lastLine(transcript string) string {
	lines := strings.Split(strings.TrimSpace(transcript), "\n")
	return strings.TrimSpace(lines[len(lines)-1])
}

// failureDiagnostic is what a failed render shows: the child's own last line,
// which says what went wrong, then the error that ended the run, then
// everything the child wrote. The inline readout cuts the text to a line, so
// the order is what decides what she sees.
func failureDiagnostic(failure error, transcript string) string {
	text := failure.Error() + "\n" + transcript
	if last := lastLine(transcript); last != "" {
		return last + "\n" + text
	}
	return text
}
