package audition

import (
	"os/exec"
	"strings"
	"testing"
)

// The page's script has no browser here. What it decides is in pageLogic as
// plain functions, and these tests run them with node, when node is on the
// path. What they cover: which tune is next unvoted, what each key does
// given where focus is, the counts, the list of tossed tunes, and the words
// of a failure. What they do not: the fetches, the focus, the video, the
// drawing; those are only read.

// node runs a JavaScript expression after pageLogic and returns its JSON.
func node(t *testing.T, expr string) string {
	t.Helper()
	path, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is not on the path; the page's logic is not run")
	}
	cmd := exec.Command(path, "-")
	cmd.Dir = t.TempDir()
	cmd.Stdin = strings.NewReader(pageLogic +
		"\nconsole.log(JSON.stringify(" + expr + "));")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("node: %v\n%s", err, out)
	}
	return strings.TrimSpace(string(out))
}

func TestTheNextTuneIsTheNextOneNotVotedOnAndItWrapsRound(t *testing.T) {
	const names = `['a','b','c','d']`
	for _, c := range []struct{ votes, current, want string }{
		{`{}`, `''`, `"a"`},
		{`{}`, `'a'`, `"b"`},
		{`{b:{vote:'keep'}}`, `'a'`, `"c"`},
		{`{d:{vote:'toss'}}`, `'c'`, `"a"`},
		{`{a:{vote:'keep'},b:{vote:'toss'},c:{vote:'keep'}}`, `'a'`, `"d"`},
		{`{a:{vote:'keep'},b:{vote:'keep'},c:{vote:'keep'}}`, `'d'`, `"d"`},
		{`{a:{vote:'keep'}}`, `'gone'`, `"b"`},
		{`{a:{vote:'keep'},b:{vote:'keep'},c:{vote:'keep'},d:{vote:'keep'}}`,
			`'a'`, `""`},
	} {
		expr := "nextUnvoted(" + names + "," + c.votes + "," + c.current + ")"
		if got := node(t, expr); got != c.want {
			t.Errorf("%s gave %s, want %s", expr, got, c.want)
		}
	}
	if got := node(t, "nextUnvoted([],{},'')"); got != `""` {
		t.Errorf("no tunes gave %s", got)
	}
}

func TestEachKeyDoesWhatItShouldWhereFocusIs(t *testing.T) {
	for _, c := range []struct {
		key, tag string
		held     bool
		row      bool
		want     string
	}{
		{"j", "BODY", false, false, "next"},
		{"j", "ARTICLE", false, true, "next"},
		{"j", "VIDEO", false, true, "next"},
		{"ArrowDown", "ARTICLE", false, true, "next"},
		{"ArrowDown", "VIDEO", false, true, ""},
		{"k", "VIDEO", false, true, "keep"},
		{"t", "VIDEO", false, true, "toss"},
		{"k", "ARTICLE", false, true, "keep"},
		{"t", "BUTTON", false, true, "toss"},
		{"k", "BODY", false, false, ""},
		{"t", "BODY", false, false, ""},
		{" ", "ARTICLE", false, true, "toggle"},
		{" ", "BODY", false, true, "toggle"},
		{" ", "BODY", false, false, ""},
		{" ", "VIDEO", false, true, ""},
		{" ", "BUTTON", false, true, ""},
		{"ArrowLeft", "VIDEO", false, true, ""},
		{"ArrowRight", "VIDEO", false, true, ""},
		{"ArrowUp", "VIDEO", false, true, ""},
		{"x", "ARTICLE", false, true, ""},
		{"J", "ARTICLE", false, true, ""},
		{"j", "BODY", true, false, ""},
		{"k", "ARTICLE", true, true, ""},
	} {
		expr := "keyAction('" + c.key + "','" + c.tag + "'," +
			map[bool]string{true: "true", false: "false"}[c.held] + "," +
			map[bool]string{true: "true", false: "false"}[c.row] + ")"
		if got := node(t, expr); got != `"`+c.want+`"` {
			t.Errorf("%s gave %s, want %q", expr, got, c.want)
		}
	}
}

func TestTheCountsComeFromTheRecordNotFromTheRowsOnThePage(t *testing.T) {
	// two tunes on the page, one voted; two more votes are for tunes that
	// are no longer there, and they still count.
	rec := `{votes:{a:{vote:'keep'},gone:{vote:'keep'},old:{vote:'toss'}},` +
		`queue:['a','gone']}`
	got := node(t, "tally(['a','b'],"+rec+")")
	if got != `{"kept":2,"tossed":1,"left":1,"queued":2}` {
		t.Errorf("the tally is %s", got)
	}
	if got := node(t, "countsText(tally(['a','b'],"+rec+"))"); got !=
		`"2 kept, 1 tossed, 1 still to hear."` {
		t.Errorf("the counts say %s", got)
	}
	if got := node(t, "tossedNames("+rec+")"); got != `["old"]` {
		t.Errorf("the tossed list is %s", got)
	}
	empty := node(t, "tally(['a'],{votes:{},queue:[]})")
	if empty != `{"kept":0,"tossed":0,"left":1,"queued":0}` {
		t.Errorf("an empty record gave %s", empty)
	}
}

func TestAFailureSaysWhatTheServerSaidOrThatNothingAnswered(t *testing.T) {
	const what = "'The vote was not saved'"
	if got := node(t, "failText("+what+",0,'')"); got !=
		`"The vote was not saved. Is the server still running?"` {
		t.Errorf("no answer said %s", got)
	}
	got := node(t, "failText("+what+",400,'no tune \"x\" in this folder\\n')")
	if got != `"The vote was not saved: no tune \"x\" in this folder"` {
		t.Errorf("an error answer said %s", got)
	}
	if strings.Contains(node(t, "failText("+what+",500,'votes.json is not "+
		"a record')"), "running") {
		t.Error("a server error was reported as a server not running")
	}
}

func TestThePageScriptHoldsTheLogicAndSendsTheContentType(t *testing.T) {
	if !strings.Contains(pageScript, pageLogic) {
		t.Error("the page's script does not hold the logic that is tested")
	}
	if !strings.Contains(pageScript, "'Content-Type':'application/json'") {
		t.Error("the page's script does not send the content type")
	}
	if strings.Contains(pageScript, "throw 0") {
		t.Error("a failure is still thrown away")
	}
}
