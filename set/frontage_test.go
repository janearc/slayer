package set_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

// aRow is a stand of terraces along the canal of aCanal, as written, after
// which a test may change a part of it.
const aRow = `  set:
    stands:
      row:
        kind: terrace-groningen
        habit: frontage
        count: 8
        seed: 5
        complexity: hull
        frontage:
          of: canal
          side: left
          setback: {value: 5, unit: m}
          gap: {value: 8, unit: m}
`

// rowRead is the canal and its row, as written or changed by each pair of
// was and now, or the refusal it meets.
func rowRead(t *testing.T, changes ...string) (set.Set, error) {
	t.Helper()
	written := aCanal + aRow
	for index := 0; index+1 < len(changes); index += 2 {
		if !strings.Contains(written, changes[index]) {
			t.Fatalf("%q is not in the row", changes[index])
		}
		written = strings.Replace(written, changes[index],
			changes[index+1], 1)
	}
	return set.Read("row.yaml", []byte(written))
}

// A frontage is each part as it was written: the channel, the side, the
// setback from its wall and the gap between runs.
func TestAFrontageIsWhatItSays(t *testing.T) {
	read, err := rowRead(t)
	if err != nil {
		t.Fatal(err)
	}
	stands := read.Stands()
	if len(stands) != 1 || stands[0].Habit() != "frontage" {
		t.Fatalf("stands %+v", stands)
	}
	of, right, setback, gap, has := stands[0].Frontage()
	if !has || of != "canal" || right || setback != 5 || gap != 8 {
		t.Fatalf("frontage %q right %v setback %v gap %v has %v", of, right,
			setback, gap, has)
	}
}

// The right side is read as the right, and a setback may be negative, for
// a boat that stands in the water on the near side of a wall.
func TestAFrontageMayStandOnTheRightAndInTheWater(t *testing.T) {
	read, err := rowRead(t, "side: left", "side: right", "value: 5, unit: m",
		"value: -2, unit: m")
	if err != nil {
		t.Fatal(err)
	}
	_, right, setback, _, _ := read.Stands()[0].Frontage()
	if !right || setback != -2 {
		t.Fatalf("right %v setback %v", right, setback)
	}
}

// A stand that has no frontage says none.
func TestAStandWithNoFrontageSaysNone(t *testing.T) {
	read, err := rowRead(t, "        habit: frontage\n", "",
		"        frontage:\n          of: canal\n          side: left\n"+
			"          setback: {value: 5, unit: m}\n"+
			"          gap: {value: 8, unit: m}\n", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, _, has := read.Stands()[0].Frontage(); has {
		t.Fatal("a stand with no frontage said one")
	}
}

// rowRefusal is what the reader says of a gap said as the line is: the
// file, the line the gap stands on and the column of the word that is
// wrong, counted from one, then the words. The line and column are found
// in the text, not in what the reader answered.
func rowRefusal(t *testing.T, said, word, words string) string {
	t.Helper()
	gap := strings.Replace(aRow, "value: 8, unit: m", said, 1)
	written := strings.Split(aCanal+gap, "\n")
	for place, line := range written {
		if strings.Contains(line, said) {
			return fmt.Sprintf("row.yaml:%d:%d: %s", place+1,
				strings.Index(line, word)+1, words)
		}
	}
	t.Fatalf("%q is not in the row", said)
	return ""
}

// Anti-test: each way a frontage can be unsaid or ill said is refused.
func TestAFrontageThatIsIllSaidIsRefused(t *testing.T) {
	for name, row := range map[string]struct{ was, now, says string }{
		"no channel": {"          of: canal\n", "", "of"},
		"no side":    {"          side: left\n", "", "side"},
		"no setback": {"          setback: {value: 5, unit: m}\n", "",
			"setback"},
		"no gap":     {"          gap: {value: 8, unit: m}\n", "", "gap"},
		"wrong side": {"side: left", "side: port", "side"},
		"feet": {"value: 8, unit: m", "value: 8, unit: ft",
			rowRefusal(t, "value: 8, unit: ft", "ft", "unit takes [m]")},
		"negative gap": {"value: 8, unit: m", "value: -8, unit: m",
			rowRefusal(t, "value: -8, unit: m", "-8",
				"value takes a number from 0 to 1000")},
		"unknown key": {"          of: canal\n",
			"          of: canal\n          width: 4\n", "width"},
	} {
		_, err := rowRead(t, row.was, row.now)
		if err == nil || !strings.Contains(err.Error(), row.says) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// Anti-test: the habit frontage takes a frontage and no within region or
// area, and a frontage is for no other habit.
func TestFrontageAndItsHabitGoTogether(t *testing.T) {
	for name, row := range map[string]struct{ was, now string }{
		"habit without frontage": {"        frontage:\n          of: canal\n" +
			"          side: left\n          setback: {value: 5, unit: m}\n" +
			"          gap: {value: 8, unit: m}\n", ""},
		"frontage without habit": {"        habit: frontage\n", ""},
		"frontage on the grid":   {"habit: frontage", "habit: grid"},
		"frontage with a within": {"        count: 8\n", "        count: 8\n" +
			"        within: {of: x, from: 0, to: 360, near: 0, far: 10}\n"},
		"frontage with an area": {"        count: 8\n", "        count: 8\n" +
			"        area: {west: 0, south: 0, east: 10, north: 10}\n"},
	} {
		_, err := rowRead(t, row.was, row.now)
		if err == nil || !strings.Contains(err.Error(), "frontage") {
			t.Errorf("%s: %v", name, err)
		}
	}
}
