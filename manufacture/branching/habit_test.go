package branching

import (
	"errors"
	"math"
	"testing"

	"github.com/janearc/celestial/measure"
)

func TestHabitRejectsInvalidBranchChoices(t *testing.T) {
	good := Habit{Turn: measure.Radians(1), Start: .1, Top: .9, Fewest: 1,
		Reach: .5, Girth: .2, Low: measure.Degrees(90)}
	if err := good.Sound(); err != nil {
		t.Fatal(err)
	}
	for name, spoil := range map[string]func(*Habit){
		"turn":   func(h *Habit) { h.Turn = measure.Radians(math.NaN()) },
		"top":    func(h *Habit) { h.Top = h.Start },
		"fewest": func(h *Habit) { h.Fewest = 0 },
		"reach":  func(h *Habit) { h.Reach = 0 },
		"own":    func(h *Habit) { h.Own = math.Inf(1) },
	} {
		t.Run(name, func(t *testing.T) {
			broken := good
			spoil(&broken)
			if err := broken.Sound(); !errors.Is(err, ErrHabit) {
				t.Fatalf("accepted invalid habit: %v", err)
			}
		})
	}
}
