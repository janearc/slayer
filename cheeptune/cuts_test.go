package cheeptune

import (
	"strings"
	"testing"
	"time"
)

// cutScore is eight bars of four at 120 a minute: two seconds a bar, sixteen
// seconds in all.
const cutScore = `{"name":"t","tempo":120,"meter":4,"parts":[
 {"role":"lead","bars":["a4:4","a4:4","a4:4","a4:4","a4:4","a4:4","a4:4","a4:4"]}]}`

// cutSheet is a sheet that matches cutScore, with the parts' lines given.
func cutSheet(parts string) string {
	return `{"group":"calm","mood":"calm and open","feeling":"easy","intent":"a title",
 "listen_for":"the drums","voicing":"@dmg-pad","tempo":120,"meter":4,
 "seconds":16,"parts":[` + parts + `]}`
}

const goodParts = `
 {"name":"open","bar":1,"second":0,"change":"one voice","feels":"f"},
 {"name":"lift","bar":3,"second":4,"change":"drums arrive","feels":"f"},
 {"name":"end","bar":8,"second":14,"change":"a held chord","feels":"f"}`

func cutsOf(t *testing.T, sheet string) (*CutSheet, error) {
	t.Helper()
	s, err := ParseScore([]byte(cutScore))
	if err != nil {
		t.Fatal(err)
	}
	return parseCuts([]byte(sheet), s)
}

func TestACutSheetThatMatchesItsScoreGivesTheTimesOfItsParts(t *testing.T) {
	c, err := cutsOf(t, cutSheet(goodParts))
	if err != nil {
		t.Fatal(err)
	}
	spans := c.Spans()
	want := []Span{
		{"open", 1, 0, 4 * time.Second, "one voice", "f"},
		{"lift", 3, 4 * time.Second, 14 * time.Second, "drums arrive", "f"},
		{"end", 8, 14 * time.Second, 16 * time.Second, "a held chord", "f"},
	}
	if len(spans) != len(want) {
		t.Fatalf("%d spans, want %d", len(spans), len(want))
	}
	for i := range want {
		if spans[i] != want[i] {
			t.Errorf("span %d is %+v, want %+v", i, spans[i], want[i])
		}
	}
}

func TestASecondWrittenToTwoDecimalsIsAccepted(t *testing.T) {
	// 140 a minute: a bar is 1.714285 s, written 1.71.
	faster := strings.Replace(cutScore, "120", "140", 1)
	s, err := ParseScore([]byte(faster))
	if err != nil {
		t.Fatal(err)
	}
	sheet := `{"group":"g","mood":"m m","feeling":"a","intent":"b","listen_for":"c","voicing":"@dmg-pad","tempo":140,"meter":4,
 "seconds":13.71,"parts":[
 {"name":"a","bar":1,"second":0,"change":"x","feels":"f"},
 {"name":"b","bar":2,"second":1.71,"change":"y","feels":"f"}]}`
	if _, err := parseCuts([]byte(sheet), s); err != nil {
		t.Fatal(err)
	}
}

// refused holds each sheet that must be refused and the words its error
// must carry, so an error that names the wrong thing fails too.
var refused = map[string]struct{ sheet, says string }{
	"a bar that is not in the score": {cutSheet(`
 {"name":"a","bar":1,"second":0,"change":"x","feels":"f"},
 {"name":"b","bar":9,"second":16,"change":"y","feels":"f"}`), "bar 9 is not in the score"},
	"bar nought": {cutSheet(`
 {"name":"a","bar":0,"second":0,"change":"x","feels":"f"},
 {"name":"b","bar":2,"second":2,"change":"y","feels":"f"}`), "bar 0 is not in the score"},
	"parts out of order": {cutSheet(`
 {"name":"a","bar":1,"second":0,"change":"x","feels":"f"},
 {"name":"b","bar":5,"second":8,"change":"y","feels":"f"},
 {"name":"c","bar":3,"second":4,"change":"z","feels":"f"}`), "parts are in order"},
	"the same bar twice": {cutSheet(`
 {"name":"a","bar":1,"second":0,"change":"x","feels":"f"},
 {"name":"b","bar":1,"second":0,"change":"y","feels":"f"}`), "is not after bar 1"},
	"a first part that is not bar one": {cutSheet(`
 {"name":"a","bar":2,"second":2,"change":"x","feels":"f"},
 {"name":"b","bar":3,"second":4,"change":"y","feels":"f"}`), "first part starts on bar 1"},
	"seconds that the bar does not make": {cutSheet(`
 {"name":"a","bar":1,"second":0,"change":"x","feels":"f"},
 {"name":"b","bar":3,"second":5,"change":"y","feels":"f"}`), "bar 3 starts at 4.00 s"},
	"a name used twice": {cutSheet(`
 {"name":"a","bar":1,"second":0,"change":"x","feels":"f"},
 {"name":"a","bar":3,"second":4,"change":"y","feels":"f"}`), "used twice"},
	"a part with no name": {cutSheet(`
 {"name":"","bar":1,"second":0,"change":"x","feels":"f"},
 {"name":"b","bar":3,"second":4,"change":"y","feels":"f"}`), "has no name"},
	"a part with no change": {cutSheet(`
 {"name":"a","bar":1,"second":0,"change":"","feels":"f"},
 {"name":"b","bar":3,"second":4,"change":"y","feels":"f"}`), "what changes there"},
	"one part only": {cutSheet(`
 {"name":"a","bar":1,"second":0,"change":"x","feels":"f"}`), "has 1 parts"},
	"the mood": {strings.Replace(cutSheet(goodParts), "calm and open", " ", 1),
		"the mood"},
	"a voicing with no at": {strings.Replace(cutSheet(goodParts),
		"@dmg-pad", "dmg-pad", 1), "is not a built-in"},
	"another tempo": {strings.Replace(cutSheet(goodParts), "120", "90", 1),
		"the score says 120"},
	"another meter": {strings.Replace(cutSheet(goodParts),
		`"meter":4`, `"meter":3`, 1), "and 4"},
	"another length": {strings.Replace(cutSheet(goodParts), `"seconds":16`,
		`"seconds":20`, 1), "the score's 8 bars make 16.00 s"},
	"no feeling": {strings.Replace(cutSheet(goodParts), `"feeling":"easy"`,
		`"feeling":""`, 1), "no feeling"},
	"no intent": {strings.Replace(cutSheet(goodParts), `"intent":"a title"`,
		`"intent":""`, 1), "no intent"},
	"nothing to listen for": {strings.Replace(cutSheet(goodParts),
		`"listen_for":"the drums"`, `"listen_for":""`, 1), "no listen_for"},
	"a mood of six words": {strings.Replace(cutSheet(goodParts),
		"calm and open", "a b c d e f", 1), "is 6 words"},
	"a mood of one word": {strings.Replace(cutSheet(goodParts),
		"calm and open", "calm", 1), "is 1 words"},
	"a feels with no words": {strings.Replace(cutSheet(goodParts),
		`"feels":"f"`, `"feels":""`, 1), "how the part feels"},
	"a feels too long": {strings.Replace(cutSheet(goodParts), `"feels":"f"`,
		`"feels":"a b c d e f g h i j k"`, 1), "is 11 words"},
	"no group": {strings.Replace(cutSheet(goodParts), `"group":"calm"`,
		`"group":""`, 1), "the group"},
	"a group of six words": {strings.Replace(cutSheet(goodParts),
		`"group":"calm"`, `"group":"a b c d e f"`, 1), "is 6 words"},
	"an unknown field": {strings.Replace(cutSheet(goodParts), `"mood"`,
		`"moood"`, 1), "not a cut sheet"},
}

func TestACutSheetThatDisagreesWithItsScoreIsRefusedByName(t *testing.T) {
	for name, tc := range refused {
		t.Run(name, func(t *testing.T) {
			_, err := cutsOf(t, tc.sheet)
			if err == nil {
				t.Fatal("accepted")
			}
			if !strings.Contains(err.Error(), tc.says) {
				t.Errorf("the error is %q, want it to say %q", err, tc.says)
			}
		})
	}
}

func TestEveryProblemInASheetIsReportedAtOnce(t *testing.T) {
	_, err := cutsOf(t, cutSheet(`
 {"name":"a","bar":1,"second":0,"change":"","feels":"f"},
 {"name":"b","bar":9,"second":16,"change":"","feels":"f"}`))
	if err == nil {
		t.Fatal("accepted")
	}
	if n := strings.Count(err.Error(), "\n"); n < 2 {
		t.Errorf("%d lines, want every problem:\n%v", n+1, err)
	}
}

func TestACutSheetIsReadFromAPathAgainstItsScore(t *testing.T) {
	s, err := ReadScore("@dusk")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ReadCuts("no-such-sheet.json", s); err == nil {
		t.Error("a missing file was accepted")
	}
	if _, err := ReadCuts("@dusk", s); err == nil ||
		!strings.Contains(err.Error(), "dusk") {
		t.Errorf("a score read as a sheet gave %v, want the path named", err)
	}
}

func TestAPathToARealSheetIsReadAgainstItsScore(t *testing.T) {
	s, err := ReadScore("examples/ember-rising.json")
	if err != nil {
		t.Fatal(err)
	}
	c, err := ReadCuts("examples/ember-rising.cuts.json", s)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Spans()) != 6 {
		count := len(c.Spans())
		t.Errorf("%d parts, want 6", count)
	}
	// the same sheet against another score is refused.
	other, err := ReadScore("@dusk")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ReadCuts("examples/ember-rising.cuts.json", other); err == nil {
		t.Error("a sheet was accepted against the wrong score")
	}
}

func TestABadBarIsReportedOnceAndDoesNotPoisonThePartAfterIt(t *testing.T) {
	bad := `
 {"name":"open","bar":1,"second":0,"change":"c","feels":"f"},
 {"name":"lost","bar":99,"second":4,"change":"c","feels":"f"},
 {"name":"lift","bar":3,"second":4,"change":"c","feels":"f"}`
	_, err := cutsOf(t, cutSheet(bad))
	if err == nil {
		t.Fatal("a bar outside the score was accepted")
	}
	lines := strings.Split(err.Error(), "\n")
	if len(lines) != 1 || !strings.Contains(lines[0], "bar 99 is not in") {
		t.Errorf("the errors are %q, want bar 99 reported once and "+
			"nothing for the part after it", lines)
	}
	// anti: a part truly out of order after a good bar is still caught,
	// naming the good bar before it.
	out := `
 {"name":"open","bar":1,"second":0,"change":"c","feels":"f"},
 {"name":"lift","bar":5,"second":8,"change":"c","feels":"f"},
 {"name":"back","bar":3,"second":4,"change":"c","feels":"f"}`
	_, err = cutsOf(t, cutSheet(out))
	if err == nil || !strings.Contains(err.Error(), "bar 3 is not after bar 5") {
		t.Errorf("an out of order part gave %v", err)
	}
}
