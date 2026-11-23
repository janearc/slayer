package pages

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"
)

// filmName is the render folder of a test's film.
const filmName = "2026-11-15-studio-film"

// filmRate is a test film's frame rate, as its receipt says it.
const filmRate = 6 // picked: the team's heard v7 rate

// musicName is a test film's music, beside its frames.
const musicName = "music.wav"

// scoreMusic writes a film's music and a score naming it, with its
// digest when one is given, and returns the music's path.
func scoreMusic(t *testing.T, dir string, music []byte, digest string) string {
	t.Helper()
	audio := filepath.Join(dir, musicName)
	score := fmt.Sprintf(`{"score": {"audio": %q, "sha256": %q}}`, audio,
		digest)
	scorePath := filepath.Join(dir, "score.json")
	scoreBytes := []byte(score)
	err := errors.Join(os.WriteFile(audio, music, listing),
		os.WriteFile(scorePath, scoreBytes, listing))
	if err != nil {
		t.Fatal(err)
	}
	return audio
}

// filmFolder lays a film's numbered frames, one picture for each number
// given, in X/frames/, its music and score, and a receipt saying its rate
// when one is given.
func filmFolder(t *testing.T, numbers []int, rate string) (string, Folder) {
	t.Helper()
	root := filepath.Join(t.TempDir(), "renders")
	folder := Folder{Shelf: animationsShelf, Name: filmName}
	for _, number := range numbers {
		lay(t, root, fixtureNoon.Unix(), fmt.Sprintf(
			"%s/%s/frames/frame-%06d.png", folder.Shelf, folder.Name, number))
	}
	scoreMusic(t, folder.path(root), []byte("music"), "")
	if rate != "" {
		receipt := filepath.Join(folder.path(root), "receipt.json")
		err := os.WriteFile(receipt, []byte(`{"fps": `+rate+"}\n"), listing)
		if err != nil {
			t.Fatal(err)
		}
	}
	return root, folder
}

// upTo is the numbers from nought below an end.
func upTo(end int) []int {
	var numbers []int
	for number := range end {
		numbers = append(numbers, number)
	}
	return numbers
}

// viewOf is what a test's film folder holds now.
func viewOf(t *testing.T, root string, folder Folder,
	inputs []string) filmView {
	t.Helper()
	look, err := publisherOf(Settings{Renders: root}).view(folder, inputs)
	if err != nil {
		t.Fatal(err)
	}
	return look
}

// TestContiguousFramesStopAtTheFirstGap counts 0, 1 and 2 of 0, 1, 2, 4
// and 5.
func TestContiguousFramesStopAtTheFirstGap(t *testing.T) {
	root, folder := filmFolder(t, []int{0, 1, 2, 4, 5}, "6")
	if look := viewOf(t, root, folder, nil); look.count != 3 {
		t.Fatal("contiguous frames:", look.count)
	}
}

// TestContiguousFramesWithoutAGapAreAllCounted is the anti-test.
func TestContiguousFramesWithoutAGapAreAllCounted(t *testing.T) {
	root, folder := filmFolder(t, upTo(6), "6")
	if look := viewOf(t, root, folder, nil); look.count != 6 {
		t.Fatal("contiguous frames:", look.count)
	}
	root, folder = filmFolder(t, []int{1, 2}, "6")
	look := viewOf(t, root, folder, nil)
	if look.count != 0 || look.due(true) || look.reason == "" {
		t.Fatal("a film not begun at its first frame was made", look)
	}
}

// expectFilm has the mock encoder expect one film of count frames, and
// write a few bytes where it is asked to.
func expectFilm(encoder *MockEncoder, root string, folder Folder,
	count int) *gomock.Call {
	frames := filepath.Join(folder.path(root), framesFolder)
	want := gomock.Cond(func(film Film) bool {
		return film.Frames == frames && film.Count == count &&
			film.Rate == filmRate && strings.HasSuffix(film.Output, partEnding) &&
			film.Music == filepath.Join(folder.path(root), musicName)
	})
	return encoder.EXPECT().Encode(gomock.Any(), want).DoAndReturn(
		func(_ context.Context, film Film) error {
			return os.WriteFile(film.Output, []byte("film"), listing)
		})
}

// TestPartialFilmIsTheContiguousFrames encodes the frames up to the gap,
// renames the film into place and records it.
func TestPartialFilmIsTheContiguousFrames(t *testing.T) {
	root, folder := filmFolder(t, []int{0, 1, 2, 4}, "6")
	encoder := NewMockEncoder(gomock.NewController(t))
	expectFilm(encoder, root, folder, 3)
	made := publisherOf(Settings{Renders: root, Encoder: encoder})
	look := viewOf(t, root, folder, nil)
	if err := made.encode(t.Context(), folder.path(root), look); err != nil {
		t.Fatal(err)
	}
	look = viewOf(t, root, folder, nil)
	lines := strings.Join(look.lines(), "\n")
	if look.partial.Frames != 3 || look.partial.Seconds != 0.5 ||
		!strings.Contains(lines, "the first 3 frames") {
		t.Fatal("the partial film was not recorded", look, lines)
	}
	partialPath := filepath.Join(folder.path(root), partialName)
	if read(t, partialPath) != "film" {
		t.Fatal("the partial film is not in place")
	}
}

// TestFailedEncodeLeavesNoFilm is the anti-test: a failed encode leaves
// no film, no record and no temporary file.
func TestFailedEncodeLeavesNoFilm(t *testing.T) {
	root, folder := filmFolder(t, upTo(6), "6")
	encoder := NewMockEncoder(gomock.NewController(t))
	failure := errors.New("the encoder failed")
	encoder.EXPECT().Encode(gomock.Any(), gomock.Any()).Return(failure)
	made := publisherOf(Settings{Renders: root, Encoder: encoder})
	look := viewOf(t, root, folder, nil)
	if err := made.encode(t.Context(), folder.path(root),
		look); !errors.Is(err, failure) {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(folder.path(root))
	names := []string{}
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	kept := []string{"frames", musicName, "receipt.json", "score.json"}
	if err != nil || !slices.Equal(names, kept) {
		t.Fatal("a failed encode left", names, err)
	}
}

// TestPartialFilmWaitsForEnoughNewFrames: a few frames more than the last
// partial film are not worth another.
func TestPartialFilmWaitsForEnoughNewFrames(t *testing.T) {
	last := partialFilm{Frames: 100, Rate: filmRate}
	for _, count := range []int{100, 105, 109} {
		look := filmView{numbered: true, count: count, rate: filmRate,
			partial: last}
		if look.due(false) {
			t.Errorf("%d frames after 100 were due", count)
		}
	}
}

// TestPartialFilmIsDueWhenEnoughHaveCome is the anti-test: a tenth more
// frames, a first second of film, any new frame at the end, or a new
// rate, each make another.
func TestPartialFilmIsDueWhenEnoughHaveCome(t *testing.T) {
	last := partialFilm{Frames: 100, Rate: filmRate}
	for _, look := range []filmView{
		{numbered: true, count: 110, rate: filmRate, partial: last},
		{numbered: true, count: filmRate, rate: filmRate},
		{numbered: true, count: 100, rate: 2 * filmRate, partial: last},
	} {
		if !look.due(false) {
			t.Errorf("%v was not due", look)
		}
	}
	ending := filmView{numbered: true, count: 101, rate: filmRate,
		partial: last}
	if !ending.due(true) {
		t.Error("a new frame at the end was not due")
	}
}

// TestNoRateMeansNoPartialFilm: with nothing saying the rate, or two
// files disagreeing, there is no film, and the page says why.
func TestNoRateMeansNoPartialFilm(t *testing.T) {
	root, folder := filmFolder(t, upTo(12), "")
	look := viewOf(t, root, folder, nil)
	if look.due(true) || !strings.Contains(look.lines()[0], "never guesses") {
		t.Fatal("a film was due without a rate", look.lines())
	}
	root, folder = filmFolder(t, upTo(12), "6")
	set := filepath.Join(folder.path(root), "set.yaml")
	if err := os.WriteFile(set, []byte("fps: 24\n"), listing); err != nil {
		t.Fatal(err)
	}
	look = viewOf(t, root, folder, nil)
	if look.due(true) || !strings.Contains(look.reason, "disagree") {
		t.Fatal("a film was due with two rates", look.reason)
	}
}

// TestRateFromAJobInputMakesAFilm is the anti-test: the job's own set
// file, outside the folder, says the rate, and the film is due.
func TestRateFromAJobInputMakesAFilm(t *testing.T) {
	root, folder := filmFolder(t, upTo(12), "")
	set := filepath.Join(t.TempDir(), "job.yaml")
	data := []byte("scene:\n  named: heard\nfps: 6\n")
	if err := os.WriteFile(set, data, listing); err != nil {
		t.Fatal(err)
	}
	look := viewOf(t, root, folder, []string{set})
	if !look.due(false) || look.rate != filmRate {
		t.Fatal("the job's rate was not read", look)
	}
}

// TestFinishedFilmStopsThePartial: once the finished film is there it is
// linked, and no partial film is made.
func TestFinishedFilmStopsThePartial(t *testing.T) {
	root, folder := filmFolder(t, upTo(12), "6")
	lay(t, root, fixtureNoon.Unix(), folder.Shelf+"/"+folder.Name+"/film.mp4")
	look := viewOf(t, root, folder, nil)
	if look.due(true) || !strings.Contains(look.lines()[0],
		`<a href="film.mp4">`) {
		t.Fatal("the finished film did not stop the partial", look.lines())
	}
}

// TestNoFinishedFilmLeavesThePartialDue is the anti-test.
func TestNoFinishedFilmLeavesThePartialDue(t *testing.T) {
	root, folder := filmFolder(t, upTo(12), "6")
	if look := viewOf(t, root, folder, nil); !look.due(true) {
		t.Fatal("a film with no finished film was not due", look)
	}
}

// TestFfmpegArgumentsPlayInQuickTime checks the encoding QuickTime needs.
func TestFfmpegArgumentsPlayInQuickTime(t *testing.T) {
	said := strings.Join(ffmpegArguments(Film{Frames: "/frames", Count: 3,
		Rate: filmRate, Music: "/music.wav", Output: "/out.part"}), " ")
	for _, want := range []string{"-c:v hevc_videotoolbox", "-tag:v hvc1",
		"-colorspace bt709", "ceil(ih/16)*16", "-frames:v 3",
		"-framerate 6", "-start_number 0", "/frames/frame-%06d.png",
		"-i /music.wav", "-map 1:a:0", "-t 0.5", "afade=t=out:st=0:d=0.5",
		"-c:a aac", "-f mp4 /out.part"} {
		if !strings.Contains(said, want) {
			t.Errorf("ffmpeg is not given %q: %s", want, said)
		}
	}
}

// TestFfmpegWithoutAPathFails is the anti-test: with no ffmpeg found, the
// encoder fails and says so.
func TestFfmpegWithoutAPathFails(t *testing.T) {
	if err := (Ffmpeg{}).Encode(t.Context(), Film{}); !errors.Is(err,
		errNoFfmpeg) {
		t.Fatal(err)
	}
	unchecked := Ffmpeg{Path: "/usr/bin/false"}
	if err := unchecked.Encode(t.Context(), Film{}); !errors.Is(err,
		errNoFfprobe) {
		t.Fatal("a film was made that could not be checked:", err)
	}
}
