package door

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/janearc/slayer/diskcache"
	"go.uber.org/mock/gomock"
)

// The anti-tests that answer a positive no test beside it answered:
// each is the failing case of the same check.

// a second kind, kept in the same home as verbKind, so a clear of one
// can be told from a clear of the other.
var otherKind = diskcache.MustKind("door-other-test", mebibyte, "another test's")

// anti-test of TestAssessOpensTheTitles
func TestAssessWithTitlesThatOpenWillDraw(t *testing.T) {
	control := gomock.NewController(t)
	shop := NewMockworkshop(control)
	opened := NewMocksource(control)
	shop.EXPECT().Source(gomock.Any(), gomock.Any()).Return(opened, nil)
	opened.EXPECT().Tone().AnyTimes()
	opened.EXPECT().Close()
	shop.EXPECT().Developer(gomock.Any(), gomock.Any()).Return(nil, nil)
	shop.EXPECT().Titler(gomock.Any()).Return(nil, false, nil)
	env := envIn(t, context.Background(), t.TempDir())
	got := running(tool(env, shop, machine{}), "assess", widow(t))
	if got.code != nothing || !strings.Contains(got.said,
		"widow.yaml will draw") {
		t.Errorf("opened titles gave %d: %q %q", got.code, got.said,
			got.wrongSaid)
	}
}

// anti-test of TestCacheClearEmptiesOneKind
func TestCacheClearOneKindLeavesTheOthers(t *testing.T) {
	env, store := cacheEnv(t)
	parent := filepath.Dir(store.Folder())
	other, err := diskcache.Open(parent, otherKind)
	if err != nil {
		t.Fatal(err)
	}
	if err := other.Put(diskcache.NewKey(otherKind).Text("a", "1"),
		[]byte("kept")); err != nil {
		t.Fatal(err)
	}
	got := running(Tool(env), "cache", "--clear", "door-verb-test")
	if got.code != nothing {
		t.Fatalf("%d: %s%s", got.code, got.said, got.wrongSaid)
	}
	if usage, _ := other.Usage(); usage.Entries() != one {
		t.Errorf("clearing one kind left %d entries of another",
			usage.Entries())
	}
}

// anti-test of TestCacheClearEmptiesEveryKind
func TestCacheClearDryRunOfEveryKindRemovesNothing(t *testing.T) {
	env, store := cacheEnv(t)
	got := running(Tool(env), "cache", "--clear", "--dry-run")
	usage, _ := store.Usage()
	if got.code != nothing || usage.Entries() != one ||
		!strings.Contains(got.said, "would remove 1 entries") {
		t.Fatalf("%d: %s%s", got.code, got.said, got.wrongSaid)
	}
}

// anti-test of TestDevelopTwoRawsSummedAreOneRawAStopHigher
func TestDevelopTwoRawsNotSummedAreTwoPictures(t *testing.T) {
	directory := t.TempDir()
	input := rawFile(t, directory, "input.pfm", pair)
	plain := written(t, directory, "plain.yaml", stillJob(t, plainPrint))
	alone := filepath.Join(directory, "alone.png")
	if got := developed(t, directory, plain, input, "--out", alone); got.code !=
		nothing {
		t.Fatalf("one raw gave %q", got.wrongSaid)
	}
	second := rawFile(t, directory, "second.pfm", pair)
	if got := developed(t, directory, plain, input, second, "--out",
		"two"); got.code != nothing {
		t.Fatalf("two raws gave %q", got.wrongSaid)
	}
	want := pictureOf(t, alone).Pix
	for _, name := range []string{"input.png", "second.png"} {
		if got := pictureOf(t, filepath.Join(directory, "two",
			name)).Pix; !slices.Equal(got, want) {
			t.Errorf("a raw not summed is %v, not its own %v", got, want)
		}
	}
}

// anti-test of TestDrawSaysWhyANoteIsNotWritten
func TestDrawSaysNothingOfANoteWhereItIsWritten(t *testing.T) {
	directory, home := t.TempDir(), t.TempDir()
	path := written(t, directory, "lake.yaml", stillJob(t,
		"outputs: {picture: lake.png, note: the lake}\n"))
	env := envIn(t, context.Background(), directory, rendersVariable,
		home, teamVariable, "team")
	if dry := drew(t, env, path, "--dry-run"); strings.Contains(dry,
		"would not write the note") {
		t.Errorf("a note that would be written is said not: %q", dry)
	}
}

// anti-test of TestEditDryRunWritesNothing
func TestEditWithoutDryRunWritesTheJob(t *testing.T) {
	control := gomock.NewController(t)
	directory := t.TempDir()
	editor := editingIn(t, directory, finding(control), shellVar, "/bin/zsh")
	running(toolWith(editor), "edit", "a.yaml")
	if found := files(t, directory); !slices.Contains(found, "a.yaml") ||
		!slices.Contains(found, guideName) {
		t.Errorf("edit wrote %v", found)
	}
}

// anti-test of TestCopyFile
func TestCopyFileLeavesTheTargetWhenTheSourceIsGone(t *testing.T) {
	directory := t.TempDir()
	to := written(t, directory, "job.yaml", "words")
	if err := copyFile(filepath.Join(directory, "gone"), to); err == nil {
		t.Error("a file not there was copied")
	}
	if raw, err := os.ReadFile(to); err != nil || string(raw) != "words" {
		t.Errorf("the target is %q, %v after a copy that failed", raw, err)
	}
}

// anti-test of TestCopyFile
func TestReplacedLeavesTheFileWhenItCannotBeWritten(t *testing.T) {
	directory := t.TempDir()
	path := written(t, directory, "manifest.json", "the old one")
	if err := os.Mkdir(path+partSuffix, folderMode); err != nil {
		t.Fatal(err)
	}
	if err := replaced(path, []byte("the new one")); err == nil {
		t.Error("a write that could not be made was said to be")
	}
	if raw, _ := os.ReadFile(path); string(raw) != "the old one" {
		t.Errorf("the file is %q after a failed replace", raw)
	}
}

// anti-test of TestLedgerDiscardsStaleFramesOnly
func TestLedgerNamesTheFrameItCannotRemove(t *testing.T) {
	folder := t.TempDir()
	stuck := fmt.Sprintf(frameName, nothing)
	free := fmt.Sprintf(frameName, one)
	stuckFolder := mkdir(t, filepath.Join(folder, stuck))
	written(t, stuckFolder, "inside.txt", "x")
	written(t, folder, free, "a stale frame")
	written(t, folder, manifestName, `{"key": "theirs", "frames": {"`+
		stuck+`": "x", "`+free+`": "x"}}`)
	_, _, err := openLedger(folder, "ours")
	if err == nil || !strings.Contains(err.Error(), stuck) {
		t.Errorf("a frame that would not go gave %v", err)
	}
	if _, err := os.Stat(filepath.Join(folder, free)); err == nil {
		t.Error("the frame that could go was kept for the one that could not")
	}
}

// anti-test of TestRunnerResumesFromWorkingFrames
func TestFramesLineKeepsWhatIsNotItsOwn(t *testing.T) {
	folder := filepath.Join(t.TempDir(), "work")
	book, _, err := openLedger(folder, "ours")
	if err != nil {
		t.Fatal(err)
	}
	written(t, folder, "keep.txt", "someone's")
	going := &filming{run: runner{at: places{frames: folder, work: true}},
		book: book}
	_, err = going.framesLine()
	if err == nil || !strings.Contains(err.Error(),
		"the working frames are removed and the folder is not") {
		t.Errorf("a folder with a stranger's file gave %v", err)
	}
	if _, err := os.Stat(filepath.Join(folder, "keep.txt")); err != nil {
		t.Errorf("the stranger's file was removed: %v", err)
	}
}

// anti-test of TestTheExamplesHere
func TestTheExamplesAreFoundOut(t *testing.T) {
	env := envIn(t, context.Background(), t.TempDir())
	if err := Tool(env).Examples(fstest.MapFS{}); err == nil {
		t.Error("a tool with no examples in sight passed")
	}
}

// anti-test of TestRunnerCancelLeavesNoMovie
func TestRunnerCancelledBeforeTheFirstFrameDrawsNone(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	made := benched(t, ctx, "")
	made.shop.EXPECT().Titler(made.run.wanted).Return(made.titles, true,
		nil)
	made.shop.EXPECT().Reel(ctx, gomock.Any()).Return(made.reel,
		nil).AnyTimes()
	made.reel.EXPECT().Abort().Return(nil).AnyTimes()
	if err := made.run.draw(); !errors.Is(err, context.Canceled) {
		t.Fatalf("a film cancelled before it began gave %v", err)
	}
	name := fmt.Sprintf(frameName, nothing)
	if _, err := os.Stat(filepath.Join(made.run.at.frames,
		name)); err == nil {
		t.Error("a cancelled film wrote a frame")
	}
}

// anti-test of TestRunnerResumesFromWorkingFrames
func TestRunnerDrawsAgainAWorkingFrameThatChanged(t *testing.T) {
	folder := filepath.Join(t.TempDir(), "work")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	first := benchedIn(t, ctx, folder)
	first.shop.EXPECT().Reel(ctx, gomock.Any()).Return(first.reel, nil)
	picture := first.frameAt(t, nothing, cancel)
	first.reel.EXPECT().Frame(nothing, picture).Return(nil)
	first.reel.EXPECT().Abort().Return(nil)
	if err := first.run.draw(); !errors.Is(err, context.Canceled) {
		t.Fatalf("the interrupted film gave %v", err)
	}
	written(t, folder, fmt.Sprintf(frameName, nothing), "tampered")
	second := benchedIn(t, context.Background(), folder)
	second.shop.EXPECT().Reel(second.run.ctx, gomock.Any()).Return(
		second.reel, nil)
	for index := range filmFrames(t) {
		drawn := second.frameAt(t, index, nil)
		second.reel.EXPECT().Frame(index, drawn).Return(nil)
	}
	second.reel.EXPECT().Close().Return(nil)
	if err := second.run.draw(); err != nil {
		t.Fatal(err)
	}
	kept := fmt.Sprintf("frame 1 of %d: kept", filmFrames(t))
	if strings.Contains(second.said.String(), kept) {
		t.Errorf("a changed frame was kept: %q", second.said)
	}
}

// anti-test of TestNoVerbsLineBeginsWithItsName
func TestALineThatBeginsWithItsNameIsFoundOut(t *testing.T) {
	if !namesItself("  draw draw a picture") {
		t.Error("a line that begins with its own name was passed")
	}
	if namesItself("  draw  draws a picture") {
		t.Error("a line that does not was refused")
	}
}

// anti-test of TestTunesStopsCleanlyWithARequestOpen
func TestTunesServerThatDiesOnItsOwnIsNotACleanStop(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	listener.Close()
	err = serveOn(context.Background(), io.Discard, listener, "x",
		http.NotFoundHandler())
	if err == nil {
		t.Error("a server whose listener failed reported a clean stop")
	}
}

// anti-test of TestTunesServedAsLocalhostAnswersAtTheAddressItSays
func TestTunesServedAsLocalhostRefusesAnotherHost(t *testing.T) {
	dir := library(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	reading, writing := io.Pipe()
	done := make(chan struct{})
	go func() {
		var wrong strings.Builder
		Tool(envIn(t, ctx, dir)).RunWith([]string{"tunes", dir,
			"--address", "localhost:0"}, strings.NewReader(""), writing,
			&wrong)
		writing.Close()
		close(done)
	}()
	line, err := bufio.NewReader(reading).ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	var address string
	if _, err := fmt.Sscanf(line, "open http://%s", &address); err != nil {
		t.Fatalf("serving said %q: %v", line, err)
	}
	asked, err := http.NewRequest(http.MethodGet, "http://"+address+
		"votes", nil)
	if err != nil {
		t.Fatal(err)
	}
	asked.Host = "elsewhere.example"
	answer, err := http.DefaultClient.Do(asked)
	if err != nil {
		t.Fatal(err)
	}
	answer.Body.Close()
	if answer.StatusCode == http.StatusOK {
		t.Error("a request for another host was answered")
	}
	cancel()
	<-done
}

// anti-test of TestStatusFolderIsBesideTheJob
func TestStatusFolderIsOfThatJobAlone(t *testing.T) {
	one, other := statusFolder("/jobs/a.yaml"), statusFolder("/jobs/b.yaml")
	if one == other || statusFolder("/other/a.yaml") == one {
		t.Errorf("two jobs share a status folder: %s %s", one, other)
	}
}
