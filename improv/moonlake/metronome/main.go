// Command metronome is improv/moonlake's: a visual metronome for
// checking the score by eye against the song. It writes raw rgb24 frames
// to stdout for ffmpeg to put the song under.
//
// One square per beat in the bar, as many as the score's meter (four
// when it names none), flashes on each measured beat time, beat one
// white and the rest green; the bar and beat are written large; a dot
// says whether the beat was measured or counted through a break; each
// cue from the sheet flashes a band with what it sets.
//
//	game run metronome -- draw --score score.yaml --cues cues.yaml \
//	  --fps FPS --width W --height H | ffmpeg -f rawvideo \
//	  -pix_fmt rgb24 -s WxH -r FPS -i - -i song.m4a ... out.mp4
//
// If the white square lands on the one you feel, the bar's phase is
// right; if not, count the beats it is off by.
package main

import (
	"bufio"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"strconv"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/slayer/improv/moonlake/show"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// the hud's colours
var (
	ground   = color.RGBA{8, 8, 10, 255}
	oneOn    = [3]float64{255, 255, 255}
	oneOff   = [3]float64{55, 55, 60}
	beatOn   = [3]float64{60, 255, 90}
	beatOff  = [3]float64{18, 48, 24}
	ink      = color.RGBA{220, 220, 225, 255}
	faint    = color.RGBA{110, 110, 118, 255}
	killBand = [3]float64{255, 40, 40}
	cueBand  = [3]float64{80, 160, 255}
)

// flash is how long a beat's flash takes to fall to a third, in seconds
const flash = 0.12

// main writes the metronome's frames, from the song's start to its end.
func render(call pzm.Call) error {
	score, err := show.ReadScore(call.Value("score"))
	if err != nil {
		return err
	}
	sheet, err := show.ReadSheet(call.Value("cues"), score)
	if err != nil {
		return err
	}
	fps, err1 := strconv.Atoi(call.Value("fps"))
	width, err2 := strconv.Atoi(call.Value("width"))
	height, err3 := strconv.Atoi(call.Value("height"))
	if err1 != nil || err2 != nil || err3 != nil || fps < 1 ||
		width < 320 || height < 180 {
		return fmt.Errorf("positive fps and at least 320 by 180 required")
	}
	if call.DryRun() {
		return nil
	}
	cues := sheet.Fired()
	out := bufio.NewWriterSize(call.Out(), 1<<20)
	frame := image.NewRGBA(image.Rect(0, 0, width, height))
	pixels := make([]byte, width*height*3)
	frames := int(math.Ceil(score.Duration() * float64(fps)))
	for n := range frames {
		t := float64(n) / float64(fps)
		draw.Draw(frame, frame.Bounds(), image.NewUniform(ground), image.Point{}, draw.Src)
		drawFrame(frame, t, score, cues)
		for i := 0; i < width*height; i++ {
			pixels[3*i] = frame.Pix[4*i]
			pixels[3*i+1] = frame.Pix[4*i+1]
			pixels[3*i+2] = frame.Pix[4*i+2]
		}
		if _, err := out.Write(pixels); err != nil {
			return err
		}
	}
	return out.Flush()
}

// drawFrame lays the hud for one moment.
func drawFrame(frame *image.RGBA, t float64, score *show.Score,
	cues []show.Fired) {
	w, h := frame.Bounds().Dx(), frame.Bounds().Dy()
	unit := float64(h) / 540
	beats := score.Beats()
	beat := score.BeatAt(t)
	b := beat.Index()
	meter := score.Meter()
	// the four squares
	gap := int(36 * unit)
	gap = min(gap, w/(meter*5))
	side := min(int(130*unit), (w-(meter+1)*gap)/meter)
	left := (w - (meter*side + (meter-1)*gap)) / 2
	top := int(220 * unit)
	for k := range meter {
		on, off := beatOn, beatOff
		if k == 0 {
			on, off = oneOn, oneOff
		}
		level := 0.0
		if b >= 0 && b%meter == k {
			level = math.Exp(-(t - beats[b]) / flash)
		}
		fill(frame, image.Rect(left+k*(side+gap), top, left+k*(side+gap)+side,
			top+side), mix(off, on, level))
		if b >= 0 && b%meter == k {
			outline(frame, image.Rect(left+k*(side+gap)-4, top-4,
				left+k*(side+gap)+side+4, top+side+4), mix(off, on, 0.6))
		}
	}
	// the bar and beat, or the intro's clock
	scale := max(1, int(5*unit))
	if b < 0 {
		text(frame, fmt.Sprintf("INTRO %5.1f s", t), w/2, int(110*unit), scale, ink)
		if len(beats) > 0 {
			text(frame, fmt.Sprintf("drums at %.2f s", beats[0]), w/2,
				int(170*unit), max(1, int(2*unit)), faint)
		}
	} else {
		text(frame, fmt.Sprintf("BAR %d . %d", b/meter+1, b%meter+1), w/2,
			int(110*unit), scale, ink)
		measured := beat.Measured()
		word, tint := "measured", color.RGBA{60, 255, 90, 255}
		if score.Estimated() {
			word = "estimated onset"
		}
		if !measured {
			word, tint = "counted through", faint
		}
		text(frame, word, w/2, int(170*unit), max(1, int(2*unit)), tint)
	}
	// the cue band, a second after each cue
	for _, c := range cues {
		since := t - c.At()
		if since < 0 || since >= 1.2 {
			continue
		}
		tone := cueBand
		if c.Kill() {
			tone = killBand
		}
		level := 0.25 + 0.75*math.Exp(-since/0.25)
		fill(frame, image.Rect(0, int(400*unit), w, int(470*unit)),
			mix([3]float64{8, 8, 10}, tone, level))
		text(frame, c.Words(), w/2, int(445*unit), max(1, int(3*unit)), color.RGBA{0, 0, 0, 255})
	}
	// where in the song
	progress := t / score.Duration()
	fill(frame, image.Rect(0, h-int(6*unit), int(float64(w)*progress), h),
		color.RGBA{90, 90, 100, 255})
	text(frame, fmt.Sprintf("%d:%05.2f", int(t)/60, math.Mod(t, 60)),
		w-int(70*unit), h-int(20*unit), max(1, int(2*unit)), faint)
}

// mix is off toward on by a level from 0 to 1.
func mix(off, on [3]float64, level float64) color.RGBA {
	level = math.Max(0, math.Min(1, level))
	return color.RGBA{uint8(off[0] + (on[0]-off[0])*level),
		uint8(off[1] + (on[1]-off[1])*level),
		uint8(off[2] + (on[2]-off[2])*level), 255}
}

// fill paints a rectangle of the frame one colour.
func fill(frame *image.RGBA, r image.Rectangle, c color.RGBA) {
	draw.Draw(frame, r, image.NewUniform(c), image.Point{}, draw.Src)
}

// outline paints a rectangle's edges, three pixels wide.
func outline(frame *image.RGBA, r image.Rectangle, c color.RGBA) {
	const t = 3
	fill(frame, image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+t), c)
	fill(frame, image.Rect(r.Min.X, r.Max.Y-t, r.Max.X, r.Max.Y), c)
	fill(frame, image.Rect(r.Min.X, r.Min.Y, r.Min.X+t, r.Max.Y), c)
	fill(frame, image.Rect(r.Max.X-t, r.Min.Y, r.Max.X, r.Max.Y), c)
}

// text writes words centred on a point, the 7 by 13 bitmap face blown
// up by a whole scale, nearest neighbour, so it stays crisp.
func text(frame *image.RGBA, words string, cx, cy, scale int, c color.RGBA) {
	face := basicfont.Face7x13
	small := image.NewAlpha(image.Rect(0, 0, 7*len(words), 13))
	d := font.Drawer{Dst: small, Src: image.Opaque, Face: face,
		Dot: fixed.P(0, 11)}
	d.DrawString(words)
	w, h := small.Bounds().Dx()*scale, 13*scale
	x0, y0 := cx-w/2, cy-h/2
	bounds := frame.Bounds()
	for y := range 13 {
		for x := range small.Bounds().Dx() {
			if small.AlphaAt(x, y).A < 128 {
				continue
			}
			r := image.Rect(x0+x*scale, y0+y*scale, x0+(x+1)*scale, y0+(y+1)*scale).Intersect(bounds)
			fill(frame, r, c)
		}
	}
}
