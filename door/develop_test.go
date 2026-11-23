package door

import (
	"context"
	"errors"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/raw"
	"github.com/janearc/slayer/render/frame"
	"go.uber.org/mock/gomock"
)

// The light of the tests' raws, linear, and the depth and cover every
// pixel of them has. Picked, as cmd/develop's tests had them: a cool
// grey well under white.
const (
	rawRed, rawGreen, rawBlue = 0.08, 0.12, 0.2
	rawDepth                  = 2
)

// rawFile writes a raw width pixels across and one down, of the tests'
// light, under a name in directory, and says where.
func rawFile(t *testing.T, directory, name string, width int) string {
	t.Helper()
	picture := frame.New(width, one)
	for i := range picture.Pix {
		picture.Pix[i] = frame.Pixel{R: rawRed, G: rawGreen, B: rawBlue,
			Cover: one, Depth: rawDepth}
	}
	path := filepath.Join(directory, name)
	if err := raw.Write(path, picture); err != nil {
		t.Fatal(err)
	}
	return path
}

// developed runs develop in a working directory.
func developed(t *testing.T, directory string, args ...string) ran {
	t.Helper()
	env := envIn(t, context.Background(), directory)
	return running(Tool(env), append([]string{"develop"}, args...)...)
}

// the print keys of the tests' jobs: cmd/develop's own development, at
// nought stops and at one.
const (
	plainPrint = "print: {knee: 0.7, top: 8}\n"
	stopPrint  = "print: {knee: 0.7, top: 8, stops: 1}\n"
)

// two raws summed are one raw a stop higher: the light adds, and the
// job's print develops it. Moved from cmd/develop.
func TestDevelopTwoRawsSummedAreOneRawAStopHigher(t *testing.T) {
	directory := t.TempDir()
	input := rawFile(t, directory, "input.pfm", pair)
	plain := written(t, directory, "plain.yaml", stillJob(t, plainPrint))
	stop := written(t, directory, "stop.yaml", stillJob(t, stopPrint))
	onePath := filepath.Join(directory, "one.png")
	twoPath := filepath.Join(directory, "two.png")
	for _, each := range [][]string{
		{stop, input, "--out", onePath},
		{plain, input, input, "--sum", "--out", twoPath},
	} {
		got := developed(t, directory, each...)
		if got.code != nothing || !strings.HasPrefix(got.said, "wrote ") {
			t.Fatalf("%v gave %d: %q %q", each, got.code, got.said,
				got.wrongSaid)
		}
	}
	once, twice := pictureOf(t, onePath), pictureOf(t, twoPath)
	if once.Bounds() != image.Rect(nothing, nothing, pair, one) ||
		!slices.Equal(once.Pix, twice.Pix) {
		t.Errorf("one raw a stop up is %v, two summed %v", once.Pix,
			twice.Pix)
	}
}

// --white balances the light before it is developed, and a raw given
// alone goes to out/ under its own name.
func TestDevelopBalancesTheWhite(t *testing.T) {
	directory := t.TempDir()
	input := rawFile(t, directory, "input.pfm", one)
	job := written(t, directory, "job.yaml", stillJob(t, plainPrint))
	developed(t, directory, job, input)
	plain := pictureOf(t, filepath.Join(directory, outFolder, "input.png"))
	developed(t, directory, job, input, "--white", "3200", "--out",
		"white.png")
	white := pictureOf(t, filepath.Join(directory, "white.png"))
	if slices.Equal(plain.Pix, white.Pix) {
		t.Error("the white balance was ignored")
	}
}

// a balance keeps green and the light's depth and cover, warms a cool
// light's red down and its blue up; a temperature with no white leaves
// the light as it was. Moved from cmd/develop.
func TestBalanceKeepsGreen(t *testing.T) {
	light := frame.New(one, one)
	light.Pix[nothing] = frame.Pixel{R: .2, G: .3, B: .4, Cover: one,
		Depth: rawDepth}
	if err := balance(light, 3200); err != nil {
		t.Fatal(err)
	}
	pixel := light.Pix[nothing]
	if pixel.G != .3 || pixel.R >= .2 || pixel.B <= .4 ||
		pixel.Cover != one || pixel.Depth != rawDepth {
		t.Errorf("balanced, the pixel is %+v", pixel)
	}
	// anti-test of TestBalanceKeepsGreen
	before := light.Pix[nothing]
	if err := balance(light, 1e-300); err == nil ||
		light.Pix[nothing] != before {
		t.Errorf("a temperature with no white gave %v, %+v", err,
			light.Pix[nothing])
	}
}

// several raws are several frames, each written in a folder named for
// the job, under its own name; a folder of plates is read as a frames
// scene reads it.
func TestDevelopSeveralRawsAndAFolder(t *testing.T) {
	directory := t.TempDir()
	frames := filepath.Join(directory, "frames")
	if err := os.Mkdir(frames, folderMode); err != nil {
		t.Fatal(err)
	}
	first := rawFile(t, frames, "a.pfm", one)
	second := rawFile(t, frames, "b.pfm", one)
	job := written(t, directory, "job.yaml", stillJob(t, plainPrint))
	got := developed(t, directory, job, first, second)
	if got.code != nothing {
		t.Fatalf("two raws gave %q", got.wrongSaid)
	}
	got = developed(t, directory, job, "frames", "--out", "again")
	if got.code != nothing {
		t.Fatalf("a folder gave %q", got.wrongSaid)
	}
	want := []string{"again/a.png", "again/b.png", "frames/a.pfm",
		"frames/b.pfm", "job.yaml", "out/job/a.png", "out/job/b.png"}
	if !slices.Equal(files(t, directory), want) {
		t.Errorf("develop wrote %v", files(t, directory))
	}
}

// same is a raw named frame.pfm in a folder of its own under directory,
// so two of them would write one picture.
func same(t *testing.T, directory, folder string) string {
	t.Helper()
	inside := filepath.Join(directory, folder)
	if err := os.MkdirAll(inside, folderMode); err != nil {
		t.Fatal(err)
	}
	return rawFile(t, inside, "frame.pfm", one)
}

// one picture with --out a folder already there goes into it under its
// raw's name, as the help says, and nothing is refused after the work.
func TestDevelopIntoAFolderThere(t *testing.T) {
	directory := t.TempDir()
	input := rawFile(t, directory, "input.pfm", one)
	job := written(t, directory, "job.yaml", stillJob(t, plainPrint))
	into := filepath.Join(directory, "into")
	if err := os.Mkdir(into, folderMode); err != nil {
		t.Fatal(err)
	}
	dry := developed(t, directory, job, input, "--out", into, "--dry-run")
	if !strings.HasPrefix(dry.said, "would write "+filepath.Join(into,
		"input.png")) {
		t.Errorf("a dry run into a folder said %q %q", dry.said,
			dry.wrongSaid)
	}
	got := developed(t, directory, job, input, "--out", into)
	if got.code != nothing {
		t.Fatalf("into a folder gave %q", got.wrongSaid)
	}
	pictureOf(t, filepath.Join(into, "input.png"))
}

// a raw named in draw's frame naming is that frame: frame 42 developed
// alone through a stock is frame 42 of a whole ordered run, grain and
// all; the anti-test, that the developer is asked for frame 42 and not
// frame 0, shows the number is used. A raw with no number in its name
// takes --frame's.
func TestDevelopNumbersAFrameByItsName(t *testing.T) {
	directory := t.TempDir()
	frames := filepath.Join(directory, "frames")
	if err := os.Mkdir(frames, folderMode); err != nil {
		t.Fatal(err)
	}
	const last = 42 // picked: the frame the review named
	for index := range last + one {
		rawFile(t, frames, fmt.Sprintf(frameStem, index)+".pfm",
			grainWide)
	}
	job := written(t, directory, "film.yaml", filmJob(t,
		"stock: {name: soot}\n"))
	got := developed(t, directory, job, "frames", "--out", "all")
	if got.code != nothing {
		t.Fatalf("the whole run gave %q", got.wrongSaid)
	}
	alone := developed(t, directory, job, filepath.Join(frames,
		"frame-000042.pfm"), "--out", "alone.png")
	if alone.code != nothing || !strings.Contains(alone.said,
		"as frame 42\n") {
		t.Fatalf("frame 42 alone said %q %q", alone.said, alone.wrongSaid)
	}
	whole := pictureOf(t, filepath.Join(directory, "all",
		"frame-000042.png"))
	picturePath := filepath.Join(directory, "alone.png")
	if !slices.Equal(pictureOf(t, picturePath).Pix, whole.Pix) {
		t.Error("frame 42 alone is not frame 42 of the run")
	}
	// anti-test of TestDevelopNumbersAFrameByItsName
	numberedAs(t, directory, job, filepath.Join(frames,
		"frame-000042.pfm"))
	other := rawFile(t, directory, "take.pfm", grainWide)
	given := developed(t, directory, job, other, "--frame", "42", "--out",
		"take.png")
	taken := pictureOf(t, filepath.Join(directory, "take.png"))
	if !strings.Contains(given.said, "as frame 42\n") ||
		!slices.Equal(taken.Pix, whole.Pix) {
		t.Errorf("--frame 42 said %q %q", given.said, given.wrongSaid)
	}
}

// numberedAs runs develop on one raw with a mock developer that wants
// frame 42, and fails the test if it is asked for any other.
func numberedAs(t *testing.T, directory, job, raw string) {
	t.Helper()
	control := gomock.NewController(t)
	shop := NewMockworkshop(control)
	develops := NewMockdeveloper(control)
	shop.EXPECT().Developer(gomock.Any(), gomock.Any()).Return(develops,
		nil)
	shop.EXPECT().Titler(gomock.Any()).Return(nil, false, nil)
	blank := image.NewRGBA(image.Rect(nothing, nothing, one, one))
	develops.EXPECT().Develop(42, gomock.Any()).Return(blank, nil)
	env := envIn(t, context.Background(), directory)
	got := running(tool(env, shop, machine{}), "develop", job, raw, "--out",
		"mocked.png")
	if got.code != nothing {
		t.Errorf("frame 42 mocked gave %q", got.wrongSaid)
	}
}

// grainWide is the tests' raws' width where grain must show. Picked: a
// few pixels is enough for two rolls of grain to differ.
const grainWide = 8

// a dry run says each picture and what it is made from, and writes
// nothing.
func TestDevelopDryRunWritesNothing(t *testing.T) {
	directory := t.TempDir()
	input := rawFile(t, directory, "input.pfm", one)
	job := written(t, directory, "job.yaml", stillJob(t, plainPrint))
	got := developed(t, directory, job, input, input, input, "--sum",
		"--dry-run")
	want := "would write " + filepath.Join(directory, outFolder,
		"input.png") + " from input.pfm as frame 0 (its name holds no " +
		"frame number; say --frame N to number it) and 2 more, summed\n"
	if got.code != nothing || got.said != want {
		t.Errorf("a dry run said %q %q", got.said, got.wrongSaid)
	}
	if len(files(t, directory)) != pair {
		t.Errorf("a dry run wrote %v", files(t, directory))
	}
}

// what cannot be developed is refused, saying why. Moved
// from cmd/develop, with the door's own refusals.
// anti-test of TestDevelopBalancesTheWhite
// anti-test of TestDevelopSeveralRawsAndAFolder
// anti-test of TestDevelopDryRunWritesNothing
// anti-test of TestDevelopIntoAFolderThere
func TestDevelopRefusesWhatItCannot(t *testing.T) {
	directory := t.TempDir()
	input := rawFile(t, directory, "input.pfm", one)
	wider := rawFile(t, directory, "wider.pfm", pair)
	bad := written(t, directory, "bad.pfm", "not a PFM\n")
	job := written(t, directory, "job.yaml", stillJob(t, plainPrint))
	stock := written(t, directory, "stock.yaml", stillJob(t,
		"stock: {name: nostock}\n"))
	for _, refusal := range []struct {
		args []string
		said string
	}{
		{nil, "develop takes a job and its raws"},
		{[]string{job}, "develop takes a job and its raws"},
		{[]string{"gone.yaml", input}, "gone.yaml"},
		{[]string{job, "missing.pfm"}, "missing.pfm"},
		{[]string{job, bad}, "raw:"},
		{[]string{job, input, wider, "--sum"}, "frame among"},
		{[]string{job, input, "--white", "-1"}, "--white takes a " +
			"temperature in kelvin above nought"},
		{[]string{job, input, "--white", "inf"}, "--white takes"},
		{[]string{job, input, "--white", "0"}, "--white takes"},
		{[]string{job, input, "--white", "NaN"}, "--white takes"},
		{[]string{job, input, "--frame", "-1"}, "--frame takes a frame " +
			"number, nought or more"},
		{[]string{job, input, "--frame", "x"}, "--frame takes"},
		{[]string{job, "missing.pfm", "--dry-run"}, "the raw missing.pfm " +
			"is not there"},
		{[]string{job, same(t, directory, "a"), same(t, directory, "b")},
			"would both be written to"},
		{[]string{job, input, "--white", "1e-300"}, "has no white"},
		{[]string{job, input, "--out", filepath.Join(job, "x.png")},
			"not a directory"},
		{[]string{job, directory, input}, "one folder of plates"},
		{[]string{stock, input}, "value must be one of"},
	} {
		got := developed(t, directory, refusal.args...)
		if got.code != one || !strings.Contains(got.wrongSaid,
			refusal.said) {
			t.Errorf("develop %v gave %d: %q, wanted %q", refusal.args,
				got.code, got.wrongSaid, refusal.said)
		}
	}
}

// each frame is developed by its number and titled at its film time,
// one frame of the job's rate apart; a development or titles the job
// cannot have are refused before anything is read.
func TestDevelopTitlesEachFrameAtItsTime(t *testing.T) {
	directory := t.TempDir()
	job := written(t, directory, "film.yaml", filmJob(t, ""))
	first := rawFile(t, directory, "a.pfm", one)
	second := rawFile(t, directory, "b.pfm", one)
	control := gomock.NewController(t)
	shop := NewMockworkshop(control)
	develops := NewMockdeveloper(control)
	titles := NewMocktitler(control)
	shop.EXPECT().Developer(gomock.Any(), gomock.Any()).Return(develops,
		nil)
	shop.EXPECT().Titler(gomock.Any()).Return(titles, true, nil)
	rate := time.Duration(number(t, "fps"))
	for index := range pair {
		blank := image.NewRGBA(image.Rect(nothing, nothing, one, one))
		develops.EXPECT().Develop(index, gomock.Any()).Return(blank, nil)
		titles.EXPECT().Draw(gomock.Any(),
			time.Duration(index)*time.Second/rate).Return(nil)
	}
	env := envIn(t, context.Background(), directory)
	got := running(tool(env, shop, machine{}), "develop", job, first, second)
	if got.code != nothing {
		t.Errorf("a titled film gave %q", got.wrongSaid)
	}
	// anti-test of TestDevelopTitlesEachFrameAtItsTime
	broken := errors.New("no theme is called x")
	for _, each := range []func(){
		func() {
			shop.EXPECT().Developer(gomock.Any(), gomock.Any()).Return(
				nil, broken)
		},
		func() {
			shop.EXPECT().Developer(gomock.Any(), gomock.Any()).Return(
				develops, nil)
			shop.EXPECT().Titler(gomock.Any()).Return(nil, false, broken)
		},
		func() {
			shop.EXPECT().Developer(gomock.Any(), gomock.Any()).Return(
				develops, nil)
			shop.EXPECT().Titler(gomock.Any()).Return(titles, true, nil)
			develops.EXPECT().Develop(nothing, gomock.Any()).Return(nil,
				broken)
		},
		func() {
			shop.EXPECT().Developer(gomock.Any(), gomock.Any()).Return(
				develops, nil)
			shop.EXPECT().Titler(gomock.Any()).Return(titles, true, nil)
			blank := image.NewRGBA(image.Rect(nothing, nothing, one, one))
			develops.EXPECT().Develop(nothing, gomock.Any()).Return(blank, nil)
			titles.EXPECT().Draw(gomock.Any(), time.Duration(nothing)).
				Return(broken)
		},
	} {
		each()
		got := running(tool(env, shop, machine{}), "develop", job, first)
		if got.code != one || !strings.Contains(got.wrongSaid,
			broken.Error()) {
			t.Errorf("a refusal gave %d: %q", got.code, got.wrongSaid)
		}
	}
}

// a still job, with no rate, titles every frame at nought.
func TestDevelopTitlesAStillAtNought(t *testing.T) {
	// anti-test of TestDevelopTitlesAStillAtNought
	room := darkroom{titles: nil}
	blank := image.NewRGBA(image.Rect(nothing, nothing, one, one))
	if err := room.title(blank, one); err != nil {
		t.Error(err)
	}
	control := gomock.NewController(t)
	titles := NewMocktitler(control)
	titles.EXPECT().Draw(gomock.Any(), time.Duration(nothing))
	room = darkroom{titles: titles}
	if err := room.title(nil, one); err != nil {
		t.Error(err)
	}
}

// develop's example: a raw developed dry through widow's print.
func Example_develop() {
	here, _ := os.Getwd()
	env, _ := NewEnv(Given{Context: context.Background(),
		Environment: func(string) (string, bool) { return "", false },
		Now:         noon, Directory: here})
	var said strings.Builder
	Tool(env).Run([]string{"develop", "../sets/widow.yaml",
		"testdata/frame-000042.pfm", "--dry-run"}, &said, &said)
	shown := strings.ReplaceAll(said.String(), here, ".")
	fmt.Print(shown)
	// Output: would write ./out/frame-000042.png from frame-000042.pfm as frame 42
}
