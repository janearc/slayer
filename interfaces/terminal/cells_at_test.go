package terminal

import "testing"

// the cells are row by row from the top left, so in a grid three across
// and two down, column x of row y is number y*3+x. worked by hand: the
// runes a to f are laid out a b c / d e f, and each is asked for by place.
func TestAtIsRowByRowFromTheTopLeft(t *testing.T) {
	c := &Cells{W: 3, H: 2}
	for _, r := range "abcdef" {
		c.Cells = append(c.Cells, Cell{Rune: r})
	}
	for _, w := range []struct {
		x, y int
		want rune
	}{
		{0, 0, 'a'}, {1, 0, 'b'}, {2, 0, 'c'},
		{0, 1, 'd'}, {1, 1, 'e'}, {2, 1, 'f'},
	} {
		if got := c.At(w.x, w.y).Rune; got != w.want {
			t.Errorf("At(%d, %d) = %q, want %q", w.x, w.y, got, w.want)
		}
	}
}
