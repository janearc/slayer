package walk

import (
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
)

// Stills keeps the stills a short draws from, each read once.
type Stills struct {
	read map[string]*image.RGBA
}

// NewStills is an empty keeping of stills.
func NewStills() *Stills { return &Stills{read: map[string]*image.RGBA{}} }

// of is a still's picture, read the first time it is asked for.
func (kept *Stills) of(picture still) (*image.RGBA, error) {
	if made, found := kept.read[picture.path]; found {
		return made, nil
	}
	file, err := os.Open(picture.path)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrWalk, err)
	}
	defer file.Close()
	decoded, err := png.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrWalk, picture.path, err)
	}
	made := image.NewRGBA(decoded.Bounds().Sub(decoded.Bounds().Min))
	for y := range made.Rect.Dy() {
		for x := range made.Rect.Dx() {
			made.Set(x, y, decoded.At(decoded.Bounds().Min.X+x,
				decoded.Bounds().Min.Y+y))
		}
	}
	kept.read[picture.path] = made
	return made, nil
}

// moment is where a frame falls in a short: its shot, how far through
// the shot as a share, and how far the picture is dipped to the ground.
type moment struct {
	shot       int
	share, dip float64
}

// momentOf is where frame i falls in the short. The picture dips to the
// ground over dipSeconds either side of a cut between two shots; the
// short's own start and end do not dip.
func (short Short) momentOf(i int) moment {
	seconds := float64(i) / filmRate
	start := 0.0
	for k, each := range short.shots {
		last := k == len(short.shots)-1
		if seconds >= start+each.seconds && !last {
			start += each.seconds
			continue
		}
		in := math.Min(seconds-start, each.seconds)
		edge := math.Inf(1)
		if k > 0 {
			edge = in
		}
		if !last {
			edge = math.Min(edge, each.seconds-in)
		}
		return moment{shot: k, share: in / each.seconds,
			dip: math.Max(0, 1-edge/(dipSeconds*half))}
	}
	return moment{}
}

// Frame is frame i of the short: the shot's still through its window,
// its labels, scale bar and tag, dipped at a cut, over the panel.
func (short Short) Frame(i int, kept *Stills) (*image.RGBA, error) {
	if i < 0 || i >= short.Frames() {
		return nil, fmt.Errorf("%w: frame %d of a short of %d; ask for one "+
			"of its frames", ErrWalk, i, short.Frames())
	}
	tones, err := colours()
	if err != nil {
		return nil, err
	}
	now := short.momentOf(i)
	shot := short.shots[now.shot]
	frame := image.NewRGBA(image.Rectangle{Max: frameSize})
	fill(frame, frame.Bounds(), tones["ground"])
	source, err := kept.of(shot.picture)
	if err != nil {
		return nil, err
	}
	part := shot.windowAt(now.share)
	sampled(frame, source, part)
	if err := shot.overlay(frame, part, tones); err != nil {
		return nil, err
	}
	dipped(frame, tones, now.dip)
	if err := shot.panel(frame, tones, len(short.host) > 0); err != nil {
		return nil, err
	}
	return frame, short.hosted(frame, i, kept)
}

// hosted lays the host's clip into its corner of the panel, frame i of
// the short showing frame i of the clip, looped, scaled to fill the
// corner. A short with no host keeps nothing and lays nothing.
func (short Short) hosted(frame *image.RGBA, i int, kept *Stills) error {
	if len(short.host) == 0 {
		return nil
	}
	clip, err := kept.of(still{path: short.host[i%len(short.host)]})
	if err != nil {
		return err
	}
	size := clip.Rect.Size()
	for y := range hostBox.Dy() {
		for x := range hostBox.Dx() {
			sx := (float64(x)+half)*float64(size.X)/float64(hostBox.Dx()) -
				half
			sy := (float64(y)+half)*float64(size.Y)/float64(hostBox.Dy()) -
				half
			colour := bilinear(clip, sx, sy)
			if clip.RGBAAt(int(sx+half), int(sy+half)).A == 0 {
				continue
			}
			frame.SetRGBA(hostBox.Min.X+x, hostBox.Min.Y+y, colour)
		}
	}
	return nil
}

// windowAt is the shot's window a share of the way through it, eased in
// and out.
func (shot Shot) windowAt(share float64) window {
	eased := share * share * (3 - 2*share)
	from, to := shot.travel[0], shot.travel[1]
	between := func(a, b float64) float64 { return a + (b-a)*eased }
	return window{left: between(from.left, to.left),
		top: between(from.top, to.top), width: between(from.width, to.width)}
}

// sampled lays a window of a still over the picture's part of a frame,
// each pixel the still's light at its middle, between its four nearest.
func sampled(frame, source *image.RGBA, part window) {
	scale := part.width / float64(pictureSize.X)
	for y := range pictureSize.Y {
		for x := range pictureSize.X {
			sx := part.left + (float64(x)+half)*scale - half
			sy := part.top + (float64(y)+half)*scale - half
			frame.SetRGBA(x, pictureTop+y, bilinear(source, sx, sy))
		}
	}
}

// dipped mixes the picture toward the ground by a share.
func dipped(frame *image.RGBA, tones palette, share float64) {
	if share <= 0 {
		return
	}
	for y := range pictureSize.Y {
		for x := range pictureSize.X {
			frame.SetRGBA(x, pictureTop+y, mix(frame.RGBAAt(x, pictureTop+y),
				tones["ground"], share))
		}
	}
}

// onFrame is a still's pixel where it falls on the frame through a
// window.
func onFrame(at Pixel, part window) image.Point {
	scale := float64(pictureSize.X) / part.width
	return image.Pt(int(math.Round((at.x+half-part.left)*scale-half)),
		pictureTop+int(math.Round((at.y+half-part.top)*scale-half)))
}
