package hillside

import (
	"strconv"
	"testing"
)

// number is a size or a count as the flags give it, as a number.
func number(t *testing.T, given string) int {
	t.Helper()
	made, err := strconv.Atoi(given)
	if err != nil {
		t.Fatal(err)
	}
	return made
}
