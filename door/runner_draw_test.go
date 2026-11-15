//go:build draw

package door

import (
	"context"
	"fmt"
	"image"
	"strings"
	"testing"

	"github.com/janearc/slayer/interfaces/movie"
	"go.uber.org/mock/gomock"
)

// a film is drawn frame by frame in order: each developed with its own
// number, titled at its own film time, handed to the movie in order, the
// movie opened once at the film's size, padded, rate and count with its
// sound, and closed once, after the last; each stage's time is its own.
func TestRunnerFilmBookkeeping(t *testing.T) {
	made := benched(t, context.Background(),
		"sound: {tune: \"@crypt\", voicing: \"@crypt-dmg\"}\n")
	made.shop.EXPECT().Titler(made.run.wanted).Return(made.titles, true,
		nil)
	made.shop.EXPECT().Reel(made.run.ctx, gomock.Cond(func(asked any) bool {
		settings := asked.(movie.Settings)
		return settings.Picture() == image.Pt(number(t, "width"),
			number(t, "height")) &&
			settings.Coded() == image.Pt(number(t, "width"), padded(t)) &&
			settings.Rate() == number(t, "fps") &&
			settings.Frames() == filmFrames(t) &&
			settings.Sound().Length() > nothing &&
			settings.Path() == made.run.at.movie
	})).Return(made.reel, nil)
	var order []any
	for index := range filmFrames(t) {
		picture := made.frameAt(t, index, nil)
		order = append(order, made.reel.EXPECT().Frame(index,
			picture).Do(func(int, *image.RGBA) {
			made.clock.on(encodeSeconds)
		}))
	}
	order = append(order, made.reel.EXPECT().Close())
	gomock.InOrder(order...)
	if err := made.run.draw(); err != nil {
		t.Fatal(err)
	}
	said := made.said.String()
	if strings.Count(said, "padded") != one {
		t.Errorf("the padding was not said once: %q", said)
	}
	stages := fmt.Sprintf("frame 1 of %d: draw %d.00 s, develop %d.00 s, "+
		"titles %d.00 s, write 0.00 s, encode %d.00 s", filmFrames(t),
		drawSeconds, developSeconds, titleSeconds, encodeSeconds)
	if !strings.Contains(said, stages) {
		t.Errorf("the stages are not each their own time: %q", said)
	}
}
