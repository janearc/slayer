package pan

import (
	"image"
	"math"
	"time"

	"github.com/janearc/slayer/hud"
)

// span is a shot's place in the film: its shot and when it starts and
// ends, from the film's start.
type span struct {
	shot       shot
	start, end time.Duration
}

// barStart is when a bar begins, by the score's own clock, from the
// piece's start; bars count from one.
func (made Cut) barStart(bar int) time.Duration {
	beats := float64((bar - one) * made.piece.Score.Meter)
	return seconds(made.piece.Score.SecondsAt(beats))
}

// Length is how long the film plays: from the first bar's start to the
// last bar's end.
func (made Cut) Length() time.Duration {
	return made.barStart(made.last+one) - made.barStart(made.first)
}

// Frames is how many frames the film holds.
func (made Cut) Frames() int {
	return int(math.Round(made.Length().Seconds() * float64(made.rate)))
}

// spans is each shot's place in the film, cut on its bar lines.
func (made Cut) spans() []span {
	origin := made.barStart(made.first)
	bar := made.first
	var laid []span
	for _, each := range made.shots {
		start := made.barStart(bar) - origin
		bar += each.bars
		laid = append(laid, span{shot: each, start: start,
			end: made.barStart(bar) - origin})
	}
	return laid
}

// at is the film's time at frame index.
func (made Cut) at(index int) time.Duration {
	return time.Duration(index) * time.Second / time.Duration(made.rate)
}

// picture is frame index: its shot's window at its place, faded, and the
// title over it while it shows.
func (made Cut) picture(index int) (*image.RGBA, error) {
	when := made.at(index)
	laid := made.spans()
	current := laid[len(laid)-one]
	for _, each := range laid {
		if when >= each.start && when < each.end {
			current = each
			break
		}
	}
	picture := current.window(when)
	current.fade(picture, when)
	return picture, made.titled(picture, when)
}

// window is the shot's window at a film time, a crop of the still the
// film's size, its corner moved from its start to its end, eased.
func (laid span) window(when time.Duration) *image.RGBA {
	share := float64(when-laid.start) / float64(laid.end-laid.start)
	eased := share * share * (smoothThree - smoothTwo*share)
	from, to := laid.shot.from, laid.shot.to
	corner := image.Pt(from.X+int(math.Round(eased*float64(to.X-from.X))),
		from.Y+int(math.Round(eased*float64(to.Y-from.Y))))
	still := laid.shot.still
	size := laid.shot.size
	made := image.NewRGBA(image.Rectangle{Max: size})
	for row := range size.Y {
		begin := still.PixOffset(corner.X, corner.Y+row)
		copy(made.Pix[row*made.Stride:row*made.Stride+size.X*rgba],
			still.Pix[begin:begin+size.X*rgba])
	}
	return made
}

// fade darkens a picture toward black at its shot's ends, as its fades
// say: none is a cut.
func (laid span) fade(picture *image.RGBA, when time.Duration) {
	light := whole
	if in := laid.shot.fadeIn; in > nothing {
		light = math.Min(light, float64(when-laid.start)/float64(in))
	}
	if out := laid.shot.fadeOut; out > nothing {
		light = math.Min(light, float64(laid.end-when)/float64(out))
	}
	if light >= whole {
		return
	}
	light = math.Max(light, nothing)
	for i := range picture.Pix {
		if i%rgba != alpha {
			picture.Pix[i] = uint8(float64(picture.Pix[i]) * light)
		}
	}
}

// titled draws the title, where the script says one, while it shows.
func (made Cut) titled(picture *image.RGBA, when time.Duration) error {
	if made.title == nil {
		return nil
	}
	rows := made.title.Rows
	if rows == nothing {
		rows = defaultRows
	}
	words := hud.Hud{Elements: []hud.Element{hud.Text{
		Words: hud.Words(made.title.Words),
		At:    hud.Point{Across: titleAcross, Down: titleDown},
		Align: hud.Centre, Ink: titleInk,
		Within: hud.Until(seconds(made.title.Until))}}}
	return words.Picture(picture, rows, when, hud.DefaultTheme(),
		hud.View{})
}
