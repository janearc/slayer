package session

import (
	"context"
	"fmt"
	"math"

	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/trace"
)

// A set under an air that says no exposure is exposed by the sky meter:
// a level grey card under the sun and the sky. That is the light falling
// on the scene, not the light the camera sees, and a camera looking
// toward a low sun through haze sees far more: Futian at sunset metered
// 9005 and printed white.
//
// So a still whose exposure is the meter's is
// metered again once its scene is staged, by what its camera sees: a
// small picture is drawn from the staged scene, and the print's exposure
// is set so that the picture's log-average luminance is a mid grey.
//
// The lights stay as the sky meter made them; light is linear, so scaling
// the print scales them all alike.

const (
	// meterWidth is how many pixels across the picture the meter reads
	// is, its height in the film's shape. picked: enough to see sky,
	// haze, street and sea apart, a hundredth of a small picture's work.
	meterWidth = 48
	// meterGrey is the luminance the picture's log-average is printed at,
	// the grey card's 0.18, as the sky meter prints its card.
	meterGrey = 0.18
	// meterFloor is the least luminance a pixel is read at, so a black
	// pixel's logarithm is a number. picked: far under any light a
	// picture prints as other than black.
	meterFloor = 1e-6
)

// pictureMetered draws the meter's picture of the session's first frame
// from the staged scene, one ray a pixel, keeping no row, and sets the
// print's exposure so its log-average luminance prints at meterGrey. It
// says what it read among the session's notes. A picture with no light
// in it leaves the sky meter's exposure as it is.
func (session *Session) pictureMetered() error {
	look := session.given
	look.width = meterWidth
	look.height = max(singleRay, int(math.Round(float64(meterWidth)*
		float64(session.given.height)/float64(max(session.given.width,
		singleRay)))))
	look.samples = singleRay
	keeping := trace.NewKeeping(context.Background(), nil)
	look.keeping = &keeping
	made, err := session.source.light(session.times[nothing], look)
	if err != nil {
		return fmt.Errorf("%s: the picture meter: %w", session.given.set,
			err)
	}
	average, lit := logAverage(made.drawn)
	if !lit {
		return nil
	}
	session.tone.Exposure = meterGrey / average
	session.given.cached = append(session.given.cached, cacheNote{told: true,
		what: fmt.Sprintf("the picture meter: a %dx%d picture of what the "+
			"camera sees has a log-average luminance of %.4g at the sky "+
			"meter's exposure %.4g; printed at %.4g times that, so it is "+
			"%.2f", look.width, look.height, average, session.given.exposure,
			session.tone.Exposure, meterGrey)})
	return nil
}

// logAverage is the log-average luminance of a picture's pixels, each
// read at no less than meterFloor, and whether any pixel has light.
func logAverage(drawn *frame.Frame) (float64, bool) {
	if drawn == nil || len(drawn.Pix) == nothing {
		return 0, false
	}
	sum, lit := 0.0, false
	for _, pixel := range drawn.Pix {
		luminance := pixel.Light().Luminance()
		if luminance > meterFloor {
			lit = true
		}
		if !(luminance > meterFloor) {
			luminance = meterFloor
		}
		sum += math.Log(luminance)
	}
	return math.Exp(sum / float64(len(drawn.Pix))), lit
}
