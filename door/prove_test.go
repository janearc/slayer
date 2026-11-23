package door

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/fermat"
	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/session"
	"github.com/janearc/slayer/trace"
	"go.uber.org/mock/gomock"
)

// The numbers the prove tests pick.
const (
	// proveWide is how many pixels across the tests' raw is. It is one
	// pixel down, so it has four pixels, fewer than the sixty-four
	// fermat picks, and every one of them is picked.
	proveWide = 4
	// proveFrame, proveLeft and proveRight are a frame and two pixels
	// of it, picked: the left and the right end of the raw.
	proveFrame, proveLeft, proveRight = 3, 1, 3
	// proveBeside is where the mock's sheet is written where no folder
	// is asked, picked.
	proveBeside = "/beside/the/raw-proof"
)

// seenProve is what the mock source was asked, to be checked once the
// tool has run.
type seenProve struct {
	wanted  job.Job
	index   int
	proving session.Proving
	dir     string
	sheeted bool
}

// holds is the replay that makes every pixel as the tests' raw has it.
func holds(int, int) (frame.Pixel, []trace.Step, error) {
	return frame.Pixel{R: rawRed, G: rawGreen, B: rawBlue}, nil, nil
}

// fails is the replay that makes every pixel a little redder than the
// tests' raw has it.
func fails(int, int) (frame.Pixel, []trace.Step, error) {
	return frame.Pixel{R: rawRed + rawRed, G: rawGreen, B: rawBlue},
		nil, nil
}

// proveRun runs prove on widow and a raw of the tests', with the flags
// given, against a mock source that proves as the replay does, and says
// what it was asked and what the tool said.
func proveRun(t *testing.T, replay fermat.Replay, flags ...string) (ran,
	*seenProve, string) {
	t.Helper()
	directory := t.TempDir()
	path := rawFile(t, directory, "hill.pfm", proveWide)
	control := gomock.NewController(t)
	shop := NewMockworkshop(control)
	opened := NewMocksource(control)
	seen := &seenProve{}
	shop.EXPECT().Source(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, wanted job.Job) (source, error) {
			seen.wanted = wanted
			return opened, nil
		})
	opened.EXPECT().Prove(gomock.Any(), gomock.Any()).DoAndReturn(
		func(index int, proving session.Proving) (fermat.Proof, error) {
			seen.index, seen.proving = index, proving
			return fermat.Prove(proving.Raw, proving.Asked,
				proving.Pixels, proving.Many, replay)
		})
	opened.EXPECT().Sheet(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ fermat.Proof, dir string) (string, error) {
			seen.dir, seen.sheeted = dir, true
			if dir == "" {
				return proveBeside, nil
			}
			return dir, nil
		})
	opened.EXPECT().Close()
	env := envIn(t, context.Background(), directory)
	args := append([]string{"prove", widow(t), path}, flags...)
	return running(tool(env, shop, machine{}), args...), seen, directory
}

// a raw that holds is said to hold, with where its sheet went, and the
// tool exits 0. The raw has four pixels, all of them picked, and each is
// as the replay makes it. With nothing said, the frame is 0, no pixel
// is named, the picks are fermat's own sixty-four, and the sheet is
// left to go beside the raw.
// anti-test of TestProveFailsWhereAPixelDoesNotHold
func TestProveSaysAProofHoldsAndWhereTheSheetWent(t *testing.T) {
	got, seen, directory := proveRun(t, holds)
	if got.code != nothing || !strings.Contains(got.said,
		"holds: 4 of 4 pixels") || !strings.Contains(got.said,
		proveBeside+": the sheet, and the raw marked") {
		t.Fatalf("a holding proof gave %d: %q %q", got.code, got.said,
			got.wrongSaid)
	}
	if seen.index != nothing || seen.proving.Pixels != nil ||
		seen.proving.Many != fermat.Many || seen.dir != "" ||
		seen.proving.Raw != filepath.Join(directory, "hill.pfm") {
		t.Errorf("unasked, the source was asked frame %d, %v: %+v",
			seen.index, seen.dir, seen.proving)
	}
}

// a raw that does not hold exits 1 after saying the proof, so the sheet
// is written and the failure is where it can be read. Every pixel is
// made redder than the raw has it, so none of the four holds.
// anti-test of TestProveSaysAProofHoldsAndWhereTheSheetWent
func TestProveFailsWhereAPixelDoesNotHold(t *testing.T) {
	got, seen, _ := proveRun(t, fails)
	if got.code != one || !strings.Contains(got.said,
		"fails: 0 of 4 pixels") || !strings.Contains(got.wrongSaid,
		"the proof does not hold") || !seen.sheeted {
		t.Errorf("a failing proof gave %d: %q %q, sheet written %v",
			got.code, got.said, got.wrongSaid, seen.sheeted)
	}
}

// the flags are what the source is asked: the frame, the pixels named,
// in the order named between semicolons, how many more, and the folder,
// a relative one under the working directory; and the proof records the
// words. A --width is laid over the job as draw lays it.
// anti-test of TestProveSaysAProofHoldsAndWhereTheSheetWent, unasked
func TestProveAsksTheSourceWhatTheFlagsSay(t *testing.T) {
	named := fmt.Sprintf("%d,%d; %d,%d", proveRight, nothing, proveLeft,
		nothing)
	got, seen, directory := proveRun(t, holds, "--pixel", named, "--many",
		"0", "--frame", fmt.Sprint(proveFrame), "--proof", "sheets/hill",
		"--width", "12")
	if got.code != nothing || !strings.Contains(got.said,
		"holds: 2 of 2 pixels") {
		t.Fatalf("the flags gave %d: %q %q", got.code, got.said,
			got.wrongSaid)
	}
	wantPixels := [][2]int{{proveRight, nothing}, {proveLeft, nothing}}
	if seen.index != proveFrame || seen.proving.Many != nothing ||
		fmt.Sprint(seen.proving.Pixels) != fmt.Sprint(wantPixels) ||
		seen.dir != filepath.Join(directory, "sheets", "hill") {
		t.Errorf("asked frame %d, %v, %d more, into %q", seen.index,
			seen.proving.Pixels, seen.proving.Many, seen.dir)
	}
	if width, said := seen.wanted.Film().Width(); !said || width != 12 {
		t.Errorf("the job's width is %d, said %v, not 12", width, said)
	}
	for _, words := range []string{"--pixel " + named, "--many 0",
		"--frame 3", "--width 12"} {
		if !strings.Contains(seen.proving.Asked, words) {
			t.Errorf("the proof records %q, without %q",
				seen.proving.Asked, words)
		}
	}
}

// a missing raw, a third word, and a job and raw that are not there are
// refused, each by name and with exit 1, and nothing is written.
func TestProveRefusesWhatItCannotProve(t *testing.T) {
	directory := t.TempDir()
	for _, refusal := range []struct {
		args []string
		said string
	}{
		{nil, "prove takes a job and a raw"},
		{[]string{widow(t)}, "prove takes a job and a raw"},
		{[]string{widow(t), "a.pfm", "b.pfm"}, "prove takes a job and " +
			"a raw"},
		{[]string{"absent.yaml", "a.pfm"}, "absent.yaml"},
	} {
		env := envIn(t, context.Background(), directory)
		got := running(tool(env, NewMockworkshop(gomock.NewController(
			t)), machine{}), append([]string{"prove"}, refusal.args...)...)
		if got.code != one || !strings.Contains(got.wrongSaid,
			refusal.said) {
			t.Errorf("prove %v gave %d: %q, wanted %q", refusal.args,
				got.code, got.wrongSaid, refusal.said)
		}
	}
	if len(files(t, directory)) != nothing {
		t.Errorf("a refusal wrote %v", files(t, directory))
	}
}

// a pixel that is not X,Y, a count that is not a number of nought or
// more, and a flag given twice are refused by the flag's name. The
// mock workshop expects no call, so none of them opens a scene.
// anti-test of TestProveAsksTheSourceWhatTheFlagsSay, whose pixels and
// count are good
func TestProveRefusesAPixelOrCountItCannotRead(t *testing.T) {
	directory := t.TempDir()
	for _, refusal := range []struct {
		flags []string
		said  string
	}{
		{[]string{"--pixel", "3"}, "--pixel takes pixels as X,Y"},
		{[]string{"--pixel", "a,b"}, "--pixel takes pixels as X,Y"},
		{[]string{"--pixel", "1,2;3"}, `not "3"`},
		{[]string{"--pixel", "1,2", "--pixel", "3,4"}, "given twice"},
		{[]string{"--many", "-1"}, "--many takes a whole number"},
		{[]string{"--many", "x"}, "--many takes a whole number"},
		{[]string{"--frame", "x"}, "--frame takes a whole number"},
	} {
		env := envIn(t, context.Background(), directory)
		args := append([]string{"prove", widow(t), "a.pfm"},
			refusal.flags...)
		got := running(tool(env, NewMockworkshop(gomock.NewController(
			t)), machine{}), args...)
		if got.code == nothing || !strings.Contains(got.wrongSaid,
			refusal.said) {
			t.Errorf("prove %v gave %d: %q, wanted %q", refusal.flags,
				got.code, got.wrongSaid, refusal.said)
		}
	}
}

// a source that cannot prove, here a refusal of the session's, is why
// the tool exits 1, and the scene it opened is let go.
// anti-test of TestProveSaysAProofHoldsAndWhereTheSheetWent
func TestProveSaysWhyASourceCannotProve(t *testing.T) {
	control := gomock.NewController(t)
	shop := NewMockworkshop(control)
	opened := NewMocksource(control)
	shop.EXPECT().Source(gomock.Any(), gomock.Any()).Return(opened, nil)
	opened.EXPECT().Prove(gomock.Any(), gomock.Any()).Return(
		fermat.Proof{}, errors.New("hillside: this session draws on the "+
			"gpu"))
	opened.EXPECT().Close()
	directory := t.TempDir()
	path := rawFile(t, directory, "hill.pfm", proveWide)
	env := envIn(t, context.Background(), directory)
	got := running(tool(env, shop, machine{}), "prove", widow(t), path)
	if got.code != one || got.said != "" || !strings.Contains(
		got.wrongSaid, "this session draws on the gpu") {
		t.Errorf("a refused proof gave %d: %q %q", got.code, got.said,
			got.wrongSaid)
	}
}

// a dry run says what would be proved and where the sheet would go, and
// opens no scene: the mock workshop expects no call.
func TestProveDryRunSaysWhatItWouldProve(t *testing.T) {
	directory := t.TempDir()
	env := envIn(t, context.Background(), directory)
	got := running(tool(env, NewMockworkshop(gomock.NewController(t)),
		machine{}), "prove", widow(t), "hill.pfm", "--pixel", "1,0",
		"--proof", "sheets", "--dry-run")
	// fermat picks sixty-four where the flags say no count
	want := fmt.Sprintf("would prove %s\nas frame 0 of widow.yaml, with 1 "+
		"named pixels and 64 more picked\nwould write the sheet in %s\n",
		filepath.Join(directory, "hill.pfm"),
		filepath.Join(directory, "sheets"))
	if got.code != nothing || got.said != want {
		t.Errorf("a dry run gave %d: %q, not %q", got.code, got.said, want)
	}
	if len(files(t, directory)) != nothing {
		t.Errorf("a dry run wrote %v", files(t, directory))
	}
}

// prove's example: a dry run of a raw against widow, the sheet beside it.
func Example_prove() {
	here, _ := os.Getwd()
	env, _ := NewEnv(Given{Context: context.Background(),
		Environment: func(string) (string, bool) { return "", false },
		Now:         noon, Directory: here})
	var said strings.Builder
	Tool(env).Run([]string{"prove", "../sets/widow.yaml",
		"testdata/frame-000042.pfm", "--dry-run"}, &said, &said)
	fmt.Print(strings.ReplaceAll(said.String(), here, "."))
	// Output:
	// would prove ./testdata/frame-000042.pfm
	// as frame 0 of widow.yaml, with 0 named pixels and 64 more picked
	// would write the sheet beside the raw, named for it
}
