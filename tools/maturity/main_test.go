package main

import (
	"errors"
	"strings"
	"testing"

	"github.com/janearc/slayer/manufacture"
)

// three is a report of three kinds in two families, given in the order a
// real report gives them.
func three() ([]manufacture.Maturity, error) {
	return []manufacture.Maturity{
		{Name: "barge", Family: "boat", Parts: 3, Pieces: 9, MeshFaces: 40, Looks: 2, Lights: 1,
			Details: []string{"hull", "quick"}, Size: [3]float64{1.25, 2, 30.04},
			Measured: []string{"barge-beam"}, Picked: []string{"barge-paint", "barge-rake"}},
		{Name: "cart", Family: "wagon", Parts: 7, Pieces: 8, Details: []string{"quick"}},
		{Name: "skiff", Family: "boat", Parts: 1, Pieces: 2, Details: []string{"quick"},
			Picked: []string{"skiff-thwart"}},
	}, nil
}

// lines runs the command on three and returns its status, its output's
// lines with runs of spaces made one, and what it wrote to stderr.
func lines(args ...string) (int, []string, string) {
	var out, errOut strings.Builder
	status := run(append([]string{"maturity"}, args...), &out, &errOut, three)
	var got []string
	trimmed := strings.TrimRight(out.String(), "\n")
	for _, line := range strings.Split(trimmed, "\n") {
		words := strings.Fields(line)
		got = append(got, strings.Join(words, " "))
	}
	return status, got, errOut.String()
}

// TestRunPrintsEveryKindInColumns checks the heading and each kind's
// line, number for number, when nothing is asked for.
func TestRunPrintsEveryKindInColumns(t *testing.T) {
	status, got, problems := lines()
	want := []string{
		"kind family parts pieces mesh faces looks lights details size, metres measured picked",
		"barge boat 3 9 40 2 1 hull quick 1.2 x 2.0 x 30.0 1 2",
		"cart wagon 7 8 0 0 0 quick 0.0 x 0.0 x 0.0 0 0",
		"skiff boat 1 2 0 0 0 quick 0.0 x 0.0 x 0.0 0 1",
	}
	if status != 0 || problems != "" || strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("status %d, stderr %q, output:\n%s", status, problems, strings.Join(got, "\n"))
	}
}

// TestRunKeepsAFamilyAndTheKindsNamed checks that -family keeps one
// family, that naming kinds keeps those, and that both together keep only
// what is in both.
func TestRunKeepsAFamilyAndTheKindsNamed(t *testing.T) {
	for _, c := range []struct {
		args []string
		want []string
	}{
		{[]string{"-family", "boat"}, []string{"barge", "skiff"}},
		{[]string{"cart", "skiff"}, []string{"cart", "skiff"}},
		{[]string{"-family", "boat", "cart", "skiff"}, []string{"skiff"}},
		{[]string{"-family", "wagon", "skiff"}, nil},
	} {
		status, got, problems := lines(c.args...)
		var names []string
		for _, line := range got[1:] {
			names = append(names, strings.Fields(line)[0])
		}
		if status != 0 || problems != "" || strings.Join(names, " ") != strings.Join(c.want, " ") {
			t.Errorf("%v: status %d, stderr %q, kinds %v, want %v", c.args, status, problems, names, c.want)
		}
	}
}

// TestRunListsTheNumbersWhenAsked checks that -numbers puts each kind's
// measured names and then its picked names under its line, and that
// without the flag none is printed.
func TestRunListsTheNumbersWhenAsked(t *testing.T) {
	status, got, _ := lines("-numbers", "barge", "cart")
	want := []string{
		"barge boat 3 9 40 2 1 hull quick 1.2 x 2.0 x 30.0 1 2",
		"measured: barge-beam",
		"picked: barge-paint",
		"picked: barge-rake",
		"cart wagon 7 8 0 0 0 quick 0.0 x 0.0 x 0.0 0 0",
	}
	if status != 0 || strings.Join(got[1:], "\n") != strings.Join(want, "\n") {
		t.Errorf("status %d, output:\n%s", status, strings.Join(got, "\n"))
	}
	_, plain, _ := lines("barge")
	if len(plain) != 2 {
		t.Errorf("without -numbers the command printed %d lines", len(plain))
	}
}

// TestRunRefusesWhatIsNotThere checks the status and the message for a
// family and a kind the report lacks, a flag the command lacks, and a
// report that fails; and that nothing is printed on stdout for any.
func TestRunRefusesWhatIsNotThere(t *testing.T) {
	failing := func() ([]manufacture.Maturity, error) { return nil, errors.New("the table is broken") }
	for _, c := range []struct {
		name   string
		args   []string
		report func() ([]manufacture.Maturity, error)
		status int
		says   string
	}{
		{"family", []string{"-family", "plane"}, three, 2, `maturity: no family "plane"; there are boat, wagon`},
		{"kind", []string{"barge", "raft"}, three, 2, `maturity: no kind "raft"; there are barge, cart, skiff`},
		{"flag", []string{"-colour"}, three, 2, "flag provided but not defined: -colour"},
		{"report", nil, failing, 1, "maturity: the table is broken"},
	} {
		var out, errOut strings.Builder
		status := run(append([]string{"maturity"}, c.args...), &out, &errOut, c.report)
		if status != c.status || !strings.Contains(errOut.String(), c.says) || out.Len() != 0 {
			t.Errorf("%s: status %d, stderr %q, stdout %q", c.name, status, errOut.String(), out.String())
		}
	}
}

// TestRunSetsItsColumnsTwoSpacesApart checks the output as printed, not
// word by word: each column is as wide as its widest cell and two spaces
// stand before the next, and the first argument, the command's own name,
// is not read as a kind.
func TestRunSetsItsColumnsTwoSpacesApart(t *testing.T) {
	var out, errOut strings.Builder
	status := run([]string{"skiff", "-numbers", "skiff"}, &out, &errOut, three)
	want := "kind   family  parts  pieces  mesh faces  looks  lights  details  size, metres     measured  picked\n" +
		"skiff  boat    1      2       0           0      0       quick    0.0 x 0.0 x 0.0  0         1\n" +
		"       picked: skiff-thwart\n"
	if status != 0 || errOut.Len() != 0 || out.String() != want {
		t.Errorf("status %d, stderr %q, output:\n%s\nwant:\n%s", status, errOut.String(), out.String(), want)
	}
}
