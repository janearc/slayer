package door

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/render/compare"
	"go.uber.org/mock/gomock"
)

// ball writes the ball's set file, changed by each pair of was and now,
// where the test keeps its files, and says where.
func ball(t *testing.T, changes ...string) string {
	t.Helper()
	source, err := os.ReadFile("testdata/ball.yaml")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	for i := nothing; i+one < len(changes); i += pair {
		if !strings.Contains(text, changes[i]) {
			t.Fatalf("%q is not in the ball", changes[i])
		}
		text = strings.Replace(text, changes[i], changes[i+one], one)
	}
	return written(t, t.TempDir(), "ball.yaml", text)
}

// assessing runs assess in a working directory of the test's own.
func assessed(t *testing.T, directory string, args ...string) ran {
	t.Helper()
	env := envIn(t, context.Background(), directory)
	return running(Tool(env), append([]string{"assess"}, args...)...)
}

// the titles draw would draw with are opened, and a refusal of them is
// why the file will not draw; the scene opened is let go.
func TestAssessOpensTheTitles(t *testing.T) {
	control := gomock.NewController(t)
	shop := NewMockworkshop(control)
	opened := NewMocksource(control)
	shop.EXPECT().Source(gomock.Any(), gomock.Any()).Return(opened, nil)
	opened.EXPECT().Tone().AnyTimes()
	opened.EXPECT().Close()
	shop.EXPECT().Developer(gomock.Any(), gomock.Any()).Return(nil, nil)
	shop.EXPECT().Titler(gomock.Any()).Return(nil, false,
		errors.New("no theme is called x"))
	env := envIn(t, context.Background(), t.TempDir())
	got := running(tool(env, shop, machine{}), "assess", widow(t))
	if got.code != one || !strings.Contains(got.wrongSaid,
		"widow.yaml will not draw: no theme is called x") {
		t.Errorf("a refused title gave %d: %q", got.code, got.wrongSaid)
	}
}

// what assess cannot do is refused by name: no file or two, and a flag
// the way of assessing does not take.
// anti-test of TestAssessSaysAFileWillDraw
// anti-test of TestAssessDryRunsTheSmokes
func TestAssessRefusesWhatItCannotDo(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	directory := t.TempDir()
	for _, refusal := range []struct {
		args []string
		said string
	}{
		{nil, "assess takes one file"},
		{[]string{"a.yaml", "b.yaml"}, "assess takes one file"},
		{[]string{widow(t), "--k", "2"}, "--k does nothing with assess " +
			"FILE"},
		{[]string{widow(t), "--judge", "--width", "3"}, "--width does " +
			"nothing with judge"},
		{[]string{"--smokes", "--camera", "front"}, "--camera does " +
			"nothing with smokes"},
		{[]string{"--smokes", "--k", "0"}, "--k is at least 1, not 0"},
		{[]string{"--smokes", "--k", "x"}, `--k takes a whole number, ` +
			`not "x"`},
		{[]string{"--smokes", "fire"}, `smoke: no group "fire"; the ` +
			"groups are light,"},
		{[]string{"--smokes", "9999", "--dry-run"}, "slayer has no smoke " +
			"9999"},
		{[]string{"--smokes", "1", "--k", "1", "--out", filepath.Join(
			written(t, t.TempDir(), "held", "held"), "copies")},
			"not a directory"},
	} {
		got := assessed(t, directory, refusal.args...)
		if got.code != one || !strings.Contains(got.wrongSaid,
			refusal.said) {
			t.Errorf("assess %v gave %d: %q, wanted %q", refusal.args,
				got.code, got.wrongSaid, refusal.said)
		}
	}
	if len(files(t, directory)) != nothing {
		t.Errorf("a refusal wrote %v", files(t, directory))
	}
}

// a judgement unsaid goes to out/ under the working directory, and a
// dry run of one says the files and writes none.
func TestAssessJudgesIntoOutAndDryRunsWritingNothing(t *testing.T) {
	directory := t.TempDir()
	path := ball(t)
	dry := assessed(t, directory, path, "--judge", "--dry-run")
	if dry.code != nothing || dry.said != "would write "+filepath.Join(
		directory, outFolder, "ball")+"-text.png, -real.png, -look.png "+
		"and -verdict.json\n" {
		t.Errorf("a dry judgement said %q %q", dry.said, dry.wrongSaid)
	}
	if len(files(t, directory)) != nothing {
		t.Errorf("a dry judgement wrote %v", files(t, directory))
	}
	refused := assessed(t, directory, filepath.Join(directory, "gone"),
		"--judge", "--dry-run")
	if refused.code != one {
		t.Errorf("a dry judgement of no file gave %d", refused.code)
	}
}

// a comparison whose worst pixel is past what an eye can see is not
// honest, one within it is, and one that compared no pixel says that
// nothing was compared. Moved from slayer set.
func TestTheVerdictIsSaidInAWord(t *testing.T) {
	for _, each := range []struct {
		report compare.Report
		said   string
	}{
		{compare.Report{Compared: 6, Worst: 1}, "not honest"},
		{compare.Report{Compared: 6}, "honest"},
		{compare.Report{Pixels: 6}, "not honest, nothing was compared"},
	} {
		if said := verdictWord(each.report); said != each.said {
			t.Errorf("%+v is %q, not %q", each.report, said, each.said)
		}
	}
}

// a dry run of the smokes names each smoke it would judge and draws
// nothing.
func TestAssessDryRunsTheSmokes(t *testing.T) {
	directory := t.TempDir()
	got := assessed(t, directory, "--smokes", "1", "2", "--dry-run")
	if got.code != nothing || got.said != "would judge smokes 1 2\n" {
		t.Errorf("a dry run said %q %q", got.said, got.wrongSaid)
	}
	if len(files(t, directory)) != nothing {
		t.Errorf("a dry run wrote %v", files(t, directory))
	}
}

// assess's example: widow will draw, and nothing is drawn.
func Example_assess() {
	here, _ := os.Getwd()
	env, _ := NewEnv(Given{Context: context.Background(),
		Environment: func(string) (string, bool) { return "", false },
		Now:         noon, Directory: here})
	var said strings.Builder
	Tool(env).Run([]string{"assess", "../sets/widow.yaml", "--dry-run"},
		&said, &said)
	fmt.Print(strings.ReplaceAll(said.String(), here, "."))
	// Output:
	// widow.yaml will draw, as a still; nothing was drawn
	// draw would write picture ./out/widow.png
}

// smokes are refused a flag assess does not know, as smokecheck refused
// one, with the usage word and exit two.
// anti-test of TestAssessJudgesASmokeAndServesItAgain
func TestAssessSmokesRefusesAFlagItDoesNotKnow(t *testing.T) {
	got := assessed(t, t.TempDir(), "--smokes", "1", "--wobble")
	if got.code != pair || !strings.Contains(got.wrongSaid,
		"unknown flag: --wobble") {
		t.Errorf("an unknown flag gave %d: %q", got.code, got.wrongSaid)
	}
}

// a judgement that is not honest fails with exit one and a sentence
// saying so, for a set and for the smokes; the anti-test, an honest one,
// exits nought.
// anti-test of TestTheVerdictIsSaidInAWord
// anti-test of TestAssessJudgesASet
func TestAJudgementNotHonestFails(t *testing.T) {
	keptSet, keptSmokes := judgeSetBy, judgeSmokesBy
	t.Cleanup(func() { judgeSetBy, judgeSmokesBy = keptSet, keptSmokes })
	for _, honest := range []bool{false, true} {
		judgeSetBy = func(io.Writer, string, judging) (bool, error) {
			return honest, nil
		}
		judgeSmokesBy = func([]string, io.Writer) (bool, error) {
			return honest, nil
		}
		set := assessed(t, t.TempDir(), ball(t), "--judge")
		smokes := assessed(t, t.TempDir(), "--smokes", "1")
		if honest {
			if set.code != nothing || smokes.code != nothing {
				t.Errorf("honest judgements gave %d and %d", set.code,
					smokes.code)
			}
			continue
		}
		if set.code != one || !strings.Contains(set.wrongSaid,
			"ball.yaml is not honest: the terminal's picture is not the "+
				"real one, coarser") {
			t.Errorf("a set not honest gave %d: %q", set.code,
				set.wrongSaid)
		}
		if smokes.code != one || !strings.Contains(smokes.wrongSaid,
			"not every smoke judged was honest") {
			t.Errorf("smokes not honest gave %d: %q", smokes.code,
				smokes.wrongSaid)
		}
	}
}

// mkdir makes a folder and says where.
func mkdir(t *testing.T, folder string) string {
	t.Helper()
	if err := os.MkdirAll(folder, folderMode); err != nil {
		t.Fatal(err)
	}
	return folder
}
