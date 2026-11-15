package door

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/render/frame"
	"go.uber.org/mock/gomock"
)

// bench is a film's runner over mocks: the workshop, the source, its
// developer, titles and reel, and a clock each stage moves on by its own
// amount, so a stage's time measured as another's shows.
type bench struct {
	shop      *Mockworkshop
	source    *Mocksource
	developer *Mockdeveloper
	titles    *Mocktitler
	reel      *Mockreel
	said      *bytes.Buffer
	clock     *stepping
	run       runner
}

// stepping is a clock that stands still until moved on.
type stepping struct{ at time.Time }

// now is the clock's time.
func (clock *stepping) now() time.Time { return clock.at }

// on moves the clock on by seconds.
func (clock *stepping) on(seconds int) {
	clock.at = clock.at.Add(time.Duration(seconds) * time.Second)
}

// What each stage costs on the bench's clock, in seconds: each its own,
// so a stage's time read as another's shows. Picked by the door slice's
// author, 2026-11-12.
const (
	drawSeconds    = 1
	developSeconds = 2
	titleSeconds   = 3
	encodeSeconds  = 5
)

// benched is a runner of a test film over mocks, writing its frames and
// its movie in a folder of its own; the workshop gives its pieces.
func benched(t *testing.T, ctx context.Context, after string) bench {
	t.Helper()
	control := gomock.NewController(t)
	directory := t.TempDir()
	wanted, err := job.Read(written(t, directory, "film.yaml",
		filmJob(t, after)))
	if err != nil {
		t.Fatal(err)
	}
	made := bench{shop: NewMockworkshop(control),
		source:    NewMocksource(control),
		developer: NewMockdeveloper(control),
		titles:    NewMocktitler(control), reel: NewMockreel(control),
		said: &bytes.Buffer{}, clock: &stepping{at: noon()}}
	made.run = runner{ctx: ctx, out: made.said, now: made.clock.now,
		shop:   made.shop,
		wanted: wanted, at: places{frames: filepath.Join(directory, "f"),
			movie: filepath.Join(directory, "film.mp4")}}
	made.shop.EXPECT().Source(ctx, wanted).Return(made.source, nil)
	made.source.EXPECT().Tone().Return(colour.Tone{})
	made.shop.EXPECT().Developer(wanted, colour.Tone{}).Return(
		made.developer, nil)
	made.source.EXPECT().Close().Return(nil)
	return made
}

// frameAt expects frame index drawn, developed with its own number,
// overlaid and titled at its film time, and gives its picture.
func (made bench) frameAt(t *testing.T, index int,
	also func()) *image.RGBA {
	t.Helper()
	control := gomock.NewController(t)
	light := &frame.Frame{}
	picture := image.NewRGBA(image.Rect(nothing, nothing,
		number(t, "width"), number(t, "height")))
	picture.Pix[index] = byte(index + one)
	drawn := NewMockdrawn(control)
	made.source.EXPECT().Frame(made.run.ctx, index).Return(drawn,
		nil).Do(func(context.Context, int) {
		made.clock.on(drawSeconds)
		if also != nil {
			also()
		}
	})
	drawn.EXPECT().Light().Return(light).AnyTimes()
	made.developer.EXPECT().Develop(index, []*frame.Frame{light}).Return(
		picture, nil).Do(func(int, []*frame.Frame) {
		made.clock.on(developSeconds)
	})
	drawn.EXPECT().Overlay(picture).Return(nil)
	made.titles.EXPECT().Draw(picture, filmTime(index,
		number(t, "fps"))).Return(nil).Do(func(*image.RGBA,
		time.Duration) {
		made.clock.on(titleSeconds)
	})
	return picture
}

// a film that writes no movie opens none, and one whose
// job says no titles draws none.
// anti-test of TestRunnerFilmBookkeeping
func TestRunnerOpensNoMovieItIsNotAskedFor(t *testing.T) {
	made := benched(t, context.Background(), "")
	made.run.at.movie = ""
	made.shop.EXPECT().Titler(made.run.wanted).Return(nil, false, nil)
	control := gomock.NewController(t)
	for index := range filmFrames(t) {
		drawn := NewMockdrawn(control)
		made.source.EXPECT().Frame(made.run.ctx, index).Return(drawn, nil)
		drawn.EXPECT().Light().Return(&frame.Frame{}).AnyTimes()
		picture := image.NewRGBA(image.Rect(nothing, nothing, one, one))
		made.developer.EXPECT().Develop(index, gomock.Any()).Return(
			picture, nil)
		drawn.EXPECT().Overlay(picture).Return(nil)
	}
	if err := made.run.draw(); err != nil {
		t.Fatal(err)
	}
}

// a cancelled context stops between frames: the movie is thrown away,
// never closed, and the frames folder resumes with the frame written.
func TestRunnerCancelLeavesNoMovie(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	made := benched(t, ctx, "")
	made.shop.EXPECT().Titler(made.run.wanted).Return(made.titles, true,
		nil)
	made.shop.EXPECT().Reel(ctx, gomock.Any()).Return(made.reel, nil)
	picture := made.frameAt(t, nothing, cancel)
	made.reel.EXPECT().Frame(nothing, picture).Return(nil)
	made.reel.EXPECT().Abort().Return(nil)
	err := made.run.draw()
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("a cancelled film gave %v", err)
	}
	book, _, err := openLedger(made.run.at.frames, made.run.keys.frames)
	if err != nil {
		t.Fatal(err)
	}
	if !book.kept(fmt.Sprintf(frameName, nothing)) {
		t.Error("the frame written before the cancel is not kept")
	}
}

// a frame that fails after the movie is open throws the
// movie away and gives the failure; one that fails before opens none.
// anti-test of TestRunnerFilmBookkeeping
func TestRunnerFailedFrameAbortsTheMovie(t *testing.T) {
	made := benched(t, context.Background(), "")
	made.shop.EXPECT().Titler(made.run.wanted).Return(made.titles, true,
		nil)
	made.shop.EXPECT().Reel(made.run.ctx, gomock.Any()).Return(made.reel,
		nil)
	picture := made.frameAt(t, nothing, nil)
	made.reel.EXPECT().Frame(nothing, picture).Return(nil)
	broken := errors.New("the lens fell off")
	made.source.EXPECT().Frame(made.run.ctx, one).Return(nil, broken)
	made.reel.EXPECT().Abort().Return(nil)
	if err := made.run.draw(); !errors.Is(err, broken) {
		t.Errorf("a failed frame gave %v", err)
	}
}

// a developer or titles that cannot be made closes the source again and
// draws nothing.
// anti-test of TestRunnerFilmBookkeeping
func TestRunnerClosesTheSourceWhenItCannotDraw(t *testing.T) {
	made := benched(t, context.Background(), "")
	refused := errors.New("no such title role")
	made.shop.EXPECT().Titler(made.run.wanted).Return(nil, false, refused)
	if err := made.run.draw(); !errors.Is(err, refused) {
		t.Errorf("titles refused gave %v", err)
	}
}

// benchedIn is a bench whose film's frames are working files in folder,
// for a movie beside it.
func benchedIn(t *testing.T, ctx context.Context, folder string) bench {
	t.Helper()
	made := benched(t, ctx, "")
	made.run.at.frames, made.run.at.work = folder, true
	made.shop.EXPECT().Titler(made.run.wanted).Return(made.titles, true,
		nil)
	return made
}

// an interrupted film's working frames stay, so the next run resumes
// from them: it draws only the frames not written and reads the rest
// back for the movie; once the movie is closed the frames are gone.
func TestRunnerResumesFromWorkingFrames(t *testing.T) {
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
	if _, err := os.Stat(filepath.Join(folder,
		fmt.Sprintf(frameName, nothing))); err != nil {
		t.Fatalf("the interrupted film left no frame: %v", err)
	}
	second := benchedIn(t, context.Background(), folder)
	second.shop.EXPECT().Reel(second.run.ctx, gomock.Any()).Return(
		second.reel, nil)
	second.reel.EXPECT().Frame(nothing, gomock.Any()).Return(nil)
	for index := one; index < filmFrames(t); index++ {
		drawn := second.frameAt(t, index, nil)
		second.reel.EXPECT().Frame(index, drawn).Return(nil)
	}
	second.reel.EXPECT().Close().Return(nil)
	if err := second.run.draw(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(second.said.String(), fmt.Sprintf(
		"frame 1 of %d: kept", filmFrames(t))) {
		t.Errorf("the first frame was not kept: %q", second.said)
	}
	if _, err := os.Stat(folder); err == nil {
		t.Error("the working frames are still there after the movie")
	}
}
