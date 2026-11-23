package door

import (
	"context"
	"errors"
	"fmt"
	"image"
	"strings"
	"testing"

	"github.com/janearc/slayer/render/frame"
	"go.uber.org/mock/gomock"
)

// a film whose width the movie writer cannot code is
// refused by the writer's own words before any frame is drawn, whether
// it writes the movie alone or frames beside it: nothing is left.
// anti-test of TestRunnerFilmBookkeeping
func TestDrawRefusesAnOddWidthForAMovie(t *testing.T) {
	for _, outputs := range []string{"{movie: film.mp4}",
		"{frames: film, movie: film.mp4}"} {
		directory := t.TempDir()
		path := written(t, directory, "film.yaml",
			filmJob(t, "outputs: "+outputs+"\n"))
		odd := fmt.Sprint(number(t, "width") - one)
		env := envIn(t, context.Background(), directory)
		got := running(Tool(env),
			"draw", path, "--width", odd)
		if got.code == nothing || !strings.Contains(got.wrongSaid, "odd") ||
			strings.Contains(got.said, "frame 1") {
			t.Errorf("%s at an odd width gave %d: %q %q", outputs, got.code,
				got.said, got.wrongSaid)
		}
		if found := files(t, directory); len(found) != one {
			t.Errorf("%s at an odd width left %v", outputs, found)
		}
	}
}

// a frame whose hud or titles fail gives the failure, the movie thrown
// away.
// anti-test of TestRunnerFilmBookkeeping
func TestRunnerFailedHudOrTitles(t *testing.T) {
	for _, each := range []struct {
		name    string
		hud     error
		titling error
	}{
		{"hud", errors.New("the hud has no font"), nil},
		{"titles", nil, errors.New("the title has no ink")},
	} {
		t.Run(each.name, func(t *testing.T) {
			made := benched(t, context.Background(), "")
			made.shop.EXPECT().Titler(made.run.wanted).Return(made.titles,
				true, nil)
			control := gomock.NewController(t)
			drawn := NewMockdrawn(control)
			made.source.EXPECT().Frame(made.run.ctx, nothing).Return(
				drawn, nil)
			drawn.EXPECT().Light().Return(&frame.Frame{}).AnyTimes()
			picture := image.NewRGBA(image.Rect(nothing, nothing, one,
				one))
			made.developer.EXPECT().Develop(nothing, gomock.Any()).Return(
				picture, nil)
			drawn.EXPECT().Overlay(picture).Return(each.hud)
			if each.hud == nil {
				made.titles.EXPECT().Draw(picture, gomock.Any()).Return(
					each.titling)
			}
			err := made.run.draw()
			if err == nil || (!errors.Is(err, each.hud) &&
				!errors.Is(err, each.titling)) {
				t.Errorf("a failed %s gave %v", each.name, err)
			}
		})
	}
}

// a frame that does not develop gives the developer's refusal.
// anti-test of TestRunnerFilmBookkeeping
func TestRunnerFailedDevelop(t *testing.T) {
	made := benched(t, context.Background(), "")
	made.shop.EXPECT().Titler(made.run.wanted).Return(nil, false, nil)
	control := gomock.NewController(t)
	drawn := NewMockdrawn(control)
	made.source.EXPECT().Frame(made.run.ctx, nothing).Return(drawn, nil)
	drawn.EXPECT().Light().Return(&frame.Frame{}).AnyTimes()
	refused := errors.New("the stock is spent")
	made.developer.EXPECT().Develop(nothing, gomock.Any()).Return(nil,
		refused)
	if err := made.run.draw(); !errors.Is(err, refused) {
		t.Errorf("a failed develop gave %v", err)
	}
}
