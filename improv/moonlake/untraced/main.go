// Command untraced writes "slayer: untraced": moonlake's light show in
// four panes, in step with the song. The viewer's pane is the finished
// film, the show as rendered. The explaining pane is the explainer's
// panel, drawn to fill its pane from the explainer's own parts.
//
// The two others are the walkthrough's stills from above and from the
// ridge, lit for
// inspection, with the rig drawn on them live from the cue sheet by
// package rig, the same geometry lightshow lights the show with: a
// drawing, and tagged so.
//
//	untraced -layout wide|tall -from S -until S -mp4 OUT ...
//	untraced -layout wide|tall -at S -png OUT ...
//
// The sources: -film (the 4k film), -script (the explainer's
// explain.yaml), -score and -cues, -song, -stills (the walkthrough's
// recipe folder: stills/plan.png, stills/ridge.png and sets/), and
// -places (the walkthrough's places table).
package main

import (
	"context"
	"flag"
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/janearc/slayer/improv/moonlake/explain"
	"github.com/janearc/slayer/improv/moonlake/rig"
	"github.com/janearc/slayer/improv/moonlake/show"
	"github.com/janearc/slayer/improv/walk"
	"github.com/janearc/slayer/interfaces/movie"
	"github.com/janearc/slayer/set"
)

// asked is what the command was asked for.
type asked struct {
	layout, film, script, score, cues, song, stills, places string
	passes                                                  string
	mp4, png                                                string
	from, until, at                                         float64
}

// main reads the flags and does what they ask, or says why not.
func main() {
	var given asked
	flag.StringVar(&given.layout, "layout", "wide", "wide or tall")
	flag.StringVar(&given.film, "film", "", "the finished 4k film")
	flag.StringVar(&given.script, "script", "",
		"the explainer's script, explain.yaml")
	flag.StringVar(&given.score, "score", "", "the song's score")
	flag.StringVar(&given.cues, "cues", "", "the cue sheet")
	flag.StringVar(&given.song, "song", "", "the song")
	flag.StringVar(&given.stills, "stills", "",
		"the walkthrough's recipe folder")
	flag.StringVar(&given.places, "places", "", "the places table")
	flag.StringVar(&given.passes, "passes", "",
		"the traced passes' folder; none draws the two panes only")
	flag.StringVar(&given.mp4, "mp4", "", "the film to write")
	flag.StringVar(&given.png, "png", "", "the one frame to write")
	flag.Float64Var(&given.from, "from", 0, "the song's second to start")
	flag.Float64Var(&given.until, "until", 0, "the song's second to stop")
	flag.Float64Var(&given.at, "at", 0, "the song's second of one frame")
	flag.Parse()
	if err := run(given); err != nil {
		fmt.Fprintln(os.Stderr, "untraced:", err)
		os.Exit(1)
	}
}

// run does what was asked.
func run(given asked) error {
	parts, err := partsOf(given)
	if err != nil {
		return err
	}
	if given.png != "" {
		return parts.still(given)
	}
	return parts.film(given)
}

// filmRate is the films' frames a second. Defined by the 4k film.
const filmRate = 24

// frameSlack is how far past a frame's time a start may be and still
// be that frame, in frames: float64's rounding of a time times the
// rate. Picked here.
const frameSlack = 1e-6

// parts is everything a frame is made from.
type parts struct {
	layout      walk.Layout
	sheet       *show.Sheet
	above, moon view
	rigLayout   rig.Layout
	filmPath    string
	viewerSize  image.Point
	explainer   *explain.Explainer
	tones       lampTones
	// aboveTraced and moonTraced are the cameras' traced passes at
	// their panes' size, or nil where none were given
	aboveTraced, moonTraced *traced
}

// The show's eye over the lake and its tower decks, as the 4k film ran
// lightshow: MOONLAKE_EYE and RIG_DECK in the recipe's overnight.sh.
const (
	eyeOverLake = 71.7
	towerDeck   = 20.4
)

// partsOf reads the sources.
func partsOf(given asked) (parts, error) {
	layout := walk.Wide()
	if given.layout == "tall" {
		layout = walk.Tall()
	} else if given.layout != "wide" {
		return parts{}, fmt.Errorf("a layout of %q; say wide or tall",
			given.layout)
	}
	score, err := show.ReadScore(given.score)
	if err != nil {
		return parts{}, err
	}
	sheet, err := show.ReadSheet(given.cues, score)
	if err != nil {
		return parts{}, err
	}
	table, err := os.ReadFile(given.places)
	if err != nil {
		return parts{}, err
	}
	places, err := walk.ReadPlaces(table)
	if err != nil {
		return parts{}, err
	}
	explainer, err := explainerOf(score, sheet, given.script)
	if err != nil {
		return parts{}, err
	}
	tones, err := tonesOf()
	if err != nil {
		return parts{}, err
	}
	above, moon, err := drawnViews(given, places)
	if err != nil {
		return parts{}, err
	}
	viewer, _ := layout.Pane(walk.PaneViewer)
	aboveTraced, moonTraced, err := tracedPanes(given)
	if err != nil {
		return parts{}, err
	}
	rigLayout := rig.NewLayout(eyeOverLake, towerDeck)
	for _, each := range []struct {
		pane  *view
		name  string
		shots []shot
	}{{&above, walk.PaneAbove, aboveShots},
		{&moon, walk.PaneMoon, ridgeShots}} {
		place, _ := layout.Pane(each.name)
		moves, err := movesOf(*each.pane, each.shots, score, rigLayout,
			float64(place.Dx())/float64(place.Dy()))
		if err != nil {
			return parts{}, err
		}
		each.pane.moves = moves
	}
	return parts{aboveTraced: aboveTraced, moonTraced: moonTraced,
		layout: layout, sheet: sheet, above: above, moon: moon,
		rigLayout: rig.NewLayout(eyeOverLake, towerDeck),
		filmPath:  given.film, viewerSize: filmSize(viewer.Size()),
		explainer: explainer, tones: tones}, nil
}

// The parts of the two stills each layout shows, in the stills' pixels
// (1620 square): the wide panes take a band 16 by 9, the tall ones a
// square, each holding the viewer's spot and the rig; the ridge's below
// its sky's pale band. Picked by the untraced film's author,
// 2026-11-12, by looking at the stills.
var stillParts = map[string][2]image.Rectangle{
	"wide": {image.Rect(0, 355, 1620, 1266), image.Rect(0, 640, 1620, 1551)},
	"tall": {image.Rect(110, 120, 1460, 1470),
		image.Rect(310, 600, 1310, 1600)},
}

// drawnViews is the two drawn panes' views: the north plan, graded
// plainly, and the ridge, its white sky graded dark.
func drawnViews(given asked, places walk.Places) (view, view, error) {
	kept := stillParts[given.layout]
	above, err := viewOf(given.stills, "walk-north.yaml", "north plan",
		"north.png", places)
	if err != nil {
		return view{}, view{}, err
	}
	moon, err := viewOf(given.stills, "walk-ridge.yaml", "ridge",
		"ridge.png", places)
	if err != nil {
		return view{}, view{}, err
	}
	above.part, above.grade = kept[0], blueprint(false)
	moon.part, moon.grade = kept[1], blueprint(true)
	return above, moon, nil
}

// filmSize is the size the 4k film is read at to fill a pane, its shape
// (4096 by 3008) kept.
func filmSize(pane image.Point) image.Point {
	width := max(pane.X, (pane.Y*filmWidth+filmHeight-1)/filmHeight)
	return image.Pt(width, (width*filmHeight+filmWidth-1)/filmWidth)
}

// The 4k film's coded size. Measured with movie.OpenFilm, 2026-11-12.
const filmWidth, filmHeight = 4096, 3008

// viewOf is a drawn pane's camera and still: the set copy and still in
// the walkthrough's recipe folder, the whole still shown.
func viewOf(dir, setFile, camera, still string, places walk.Places) (view,
	error) {
	path := filepath.Join(dir, "sets", setFile)
	source, err := os.ReadFile(path)
	if err != nil {
		return view{}, err
	}
	read, err := set.Read(path, source)
	if err != nil {
		return view{}, err
	}
	picture, err := readPNG(filepath.Join(dir, "stills", still))
	if err != nil {
		return view{}, err
	}
	seen, err := walk.CameraOf(read, camera, places, picture.Rect.Dx())
	if err != nil {
		return view{}, err
	}
	return view{camera: seen, still: picture, part: picture.Rect}, nil
}

// readPNG reads a picture into RGBA, its top left at nought.
func readPNG(path string) (*image.RGBA, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	decoded, err := png.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	bounds := decoded.Bounds()
	made := image.NewRGBA(image.Rectangle{Max: bounds.Size()})
	for y := range bounds.Dy() {
		for x := range bounds.Dx() {
			made.Set(x, y, decoded.At(bounds.Min.X+x, bounds.Min.Y+y))
		}
	}
	return made, nil
}

// still writes one frame at a moment.
func (made parts) still(given asked) error {
	at := time.Duration(given.at * float64(time.Second))
	viewer, err := movie.FrameAt(made.filmPath, at, made.viewerSize)
	if err != nil {
		return err
	}
	frame, err := made.compose(given.at, viewer)
	if err != nil {
		return err
	}
	file, err := os.Create(given.png)
	if err != nil {
		return err
	}
	if err := png.Encode(file, frame); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}

// film writes the span of the song as a film with the song under it.
func (made parts) film(given asked) error {
	// the frames fall on the finished film's own, every 1/filmRate from
	// its start, so a drawn pane and the film show the same moment: a
	// start between two of its frames would draw the rig up to a frame
	// ahead of the film
	given.from = math.Floor(given.from*filmRate+frameSlack) / filmRate
	frames := int((given.until - given.from) * filmRate)
	sound, err := songSpan(given.song, given.from, given.until)
	if err != nil {
		return err
	}
	size := made.layout.Size()
	settings, err := movie.NewSettings(movie.Request{Path: given.mp4,
		Width: size.X, Height: size.Y, Rate: filmRate, Frames: frames,
		Sound: sound, Replace: true, Pad: true})
	if err != nil {
		return err
	}
	start := time.Duration(given.from * float64(time.Second))
	viewer, err := movie.OpenFilm(made.filmPath, start, made.viewerSize)
	if err != nil {
		return err
	}
	defer viewer.Close()
	writer, err := movie.New(context.Background(), settings)
	if err != nil {
		return err
	}
	for i := range frames {
		seconds := given.from + float64(i)/filmRate
		frame, err := made.next(seconds, viewer)
		if err == nil {
			err = writer.Frame(i, frame)
		}
		if err != nil {
			writer.Abort()
			return fmt.Errorf("frame %d: %w", i, err)
		}
	}
	if err := writer.Close(); err != nil {
		return err
	}
	fmt.Printf("%s: %d frames, %.1f s\n", given.mp4, frames,
		given.until-given.from)
	return nil
}

// next is the frame at a moment, the film read on to it. A film that
// has skipped past the moment is refused.
func (made parts) next(seconds float64, viewer *movie.Film) (
	*image.RGBA, error) {
	shown, err := viewer.Next()
	if err != nil {
		return nil, err
	}
	// a picture the film shows twice, or one before the frame's moment,
	// is read past to the one at the frame's moment
	for viewer.Shown().Seconds() < seconds-0.5/filmRate {
		if shown, err = viewer.Next(); err != nil {
			return nil, err
		}
	}
	if err := inStep(viewer.Shown().Seconds(), seconds); err != nil {
		return nil, err
	}
	return made.compose(seconds, shown)
}

// inStep refuses a pane's picture shown at another moment than the
// frame's: every pane of a frame shows the same moment, to within half
// a frame.
func inStep(shown, seconds float64) error {
	if math.Abs(shown-seconds) > 0.5/filmRate {
		return fmt.Errorf("a pane shows %.3f s in the frame for %.3f s; "+
			"read its film from the frame's moment", shown, seconds)
	}
	return nil
}

// songSpan is the song from one second to another, as it is.
func songSpan(path string, from, until float64) (movie.Sound, error) {
	song, err := movie.ReadSound(path)
	if err != nil {
		return movie.Sound{}, err
	}
	rate := float64(song.Rate())
	first, last := int(from*rate), int(until*rate)
	if from < 0 || last > song.Length() || last <= first {
		return movie.Sound{}, fmt.Errorf("%.2f to %.2f s of a song %.2f s "+
			"long", from, until, song.Duration().Seconds())
	}
	return movie.NewSound(song.Rate(),
		song.Samples()[first*stereo:last*stereo])
}

// stereo is a sound's channels. Defined by package movie.
const stereo = 2

// tracedPanes is the two cameras' traced passes; none without -passes.
func tracedPanes(given asked) (*traced, *traced, error) {
	if given.passes == "" {
		return nil, nil, nil
	}
	above, err := tracedOf(given.passes, "north")
	if err != nil {
		return nil, nil, err
	}
	moon, err := tracedOf(given.passes, "ridge")
	if err != nil {
		return nil, nil, err
	}
	return &above, &moon, nil
}
