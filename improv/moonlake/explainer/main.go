// Command explainer is improv only: moonlake's explainer short. It
// draws package explain's panel under the finished film, frame by frame,
// and writes a vertical mp4 with the song under it; or one frame, as a
// png, to look at before the film is made.
//
//	explainer -film F -score S -cues C -script E -still SECONDS -png OUT
//	explainer -film F -score S -cues C -script E -song W -short NAME \
//	  -mp4 OUT
//
// The short covers the span the script names for it. rig.go is what
// moonlake supplies to package explain: its clock, its six lamps, the
// panel's ground. Each stage's time is said on
// stderr, so the cost of a frame is known before the whole is made.
package main

import (
	"context"
	"flag"
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"time"

	"github.com/janearc/slayer/improv/moonlake/explain"
	"github.com/janearc/slayer/improv/moonlake/show"
	"github.com/janearc/slayer/interfaces/movie"
)

// frameWidth and shortRate are the short's width and frames a second.
// Defined: a 1080 by 1920 vertical short at the film's own 24.
const (
	frameWidth = 1080
	frameHigh  = 1920
	shortRate  = 24
	frameRate  = float64(shortRate)
)

// stereo is the channels a Sound's samples are interleaved over.
// Defined by package movie.
const stereo = 2

// paths is what the command is pointed at.
type paths struct {
	film, score, cues, script, song, mp4, png, name string
	still                                           float64
}

// main reads the flags and makes the still or the short.
func main() {
	var asked paths
	flag.StringVar(&asked.film, "film", "", "the finished film")
	flag.StringVar(&asked.score, "score", "", "score.yaml")
	flag.StringVar(&asked.cues, "cues", "", "cues.yaml")
	flag.StringVar(&asked.script, "script", "", "explain.yaml")
	flag.StringVar(&asked.song, "song", "", "the song, for the short")
	flag.StringVar(&asked.mp4, "mp4", "", "the short to write")
	flag.StringVar(&asked.name, "short", "", "which of the script's shorts")
	flag.StringVar(&asked.png, "png", "", "the still to write")
	flag.Float64Var(&asked.still, "still", -1, "the still's moment, s")
	flag.Parse()
	explainer, err := explainerFor(asked)
	if err != nil {
		fail(err)
	}
	if asked.png != "" {
		err = still(explainer, asked)
	} else {
		err = short(explainer, asked)
	}
	if err != nil {
		fail(err)
	}
}

// explainerFor reads the score, the sheet and the script.
func explainerFor(asked paths) (*explain.Explainer, error) {
	score, err := show.ReadScore(asked.score)
	if err != nil {
		return nil, err
	}
	sheet, err := show.ReadSheet(asked.cues, score)
	if err != nil {
		return nil, err
	}
	show := moonlake{score: score, sheet: sheet, fired: sheet.Fired()}
	script, err := explain.ReadScript(asked.script, score)
	if err != nil {
		return nil, err
	}
	lights, err := show.lights()
	if err != nil {
		return nil, err
	}
	return explain.New(explain.Parts{Clock: show.clock, Lights: lights,
		Ground: show.ground, Script: script})
}

// filmSize is the film's pictures scaled to the frame's width, their
// shape kept.
func filmSize(path string) (image.Point, error) {
	film, err := movie.OpenFilm(path, 0, image.Pt(1, 1))
	if err != nil {
		return image.Point{}, err
	}
	defer film.Close()
	native := film.Native()
	if native.X < 1 {
		return image.Point{}, fmt.Errorf("%s says it is %v", path, native)
	}
	high := (frameWidth*native.Y + native.X/2) / native.X
	return image.Pt(frameWidth, high), nil
}

// still writes one composed frame as a png, and says what it cost.
func still(explainer *explain.Explainer, asked paths) error {
	began := time.Now()
	size, err := filmSize(asked.film)
	if err != nil {
		return err
	}
	at := time.Duration(asked.still * float64(time.Second))
	picture, err := movie.FrameAt(asked.film, at, size)
	if err != nil {
		return err
	}
	read := time.Since(began)
	frame, err := explainer.Compose(asked.still, picture)
	if err != nil {
		return err
	}
	composed := time.Since(began) - read
	out, err := os.Create(asked.png)
	if err != nil {
		return err
	}
	if err := png.Encode(out, frame); err != nil {
		out.Close()
		return err
	}
	fmt.Fprintf(os.Stderr, "explainer: read %v, composed %v, written %v\n",
		read, composed, time.Since(began)-read-composed)
	return out.Close()
}

// fail says what went wrong on stderr and stops with a failing status.
func fail(err error) {
	fmt.Fprintln(os.Stderr, "explainer:", err)
	os.Exit(1)
}

// short writes the script's span as a vertical mp4 with the song under
// it.
func short(explainer *explain.Explainer, asked paths) error {
	from, until, ok := explainer.Script().Short(asked.name)
	if !ok {
		return fmt.Errorf("no short %q in the script; it has %v",
			asked.name, explainer.Script().Shorts())
	}
	// start on a film frame, so the panel, the song and the film agree
	from = math.Floor(from*shortRate) / shortRate
	frames := int((until-from)*shortRate + 0.5)
	size, err := filmSize(asked.film)
	if err != nil {
		return err
	}
	sound, err := songSpan(asked.song, from, until)
	if err != nil {
		return err
	}
	settings, err := movie.NewSettings(movie.Request{Path: asked.mp4,
		Width: frameWidth, Height: frameHigh, Rate: shortRate,
		Frames: frames, Sound: sound, Replace: true})
	if err != nil {
		return err
	}
	film, err := movie.OpenFilm(asked.film,
		time.Duration(from*float64(time.Second)), size)
	if err != nil {
		return err
	}
	defer film.Close()
	writer, err := movie.New(context.Background(), settings)
	if err != nil {
		return err
	}
	began := time.Now()
	if err := frameAll(explainer, film, writer, from, frames); err != nil {
		writer.Abort()
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	spent := time.Since(began)
	fmt.Fprintf(os.Stderr, "explainer: %d frames in %v, %v a frame\n",
		frames, spent, spent/time.Duration(frames))
	return nil
}

// frameAll composes and writes every frame of the short.
func frameAll(explainer *explain.Explainer, film *movie.Film,
	writer *movie.Writer, from float64, frames int) error {
	for i := range frames {
		picture, err := film.Next()
		if err != nil {
			return err
		}
		frame, err := explainer.Compose(from+float64(i)/shortRate, picture)
		if err != nil {
			return err
		}
		if err := writer.Frame(i, frame); err != nil {
			return err
		}
	}
	return nil
}

// songSpan is the song from one moment to another.
func songSpan(path string, from, until float64) (movie.Sound, error) {
	whole, err := movie.ReadSound(path)
	if err != nil {
		return movie.Sound{}, err
	}
	rate := whole.Rate()
	first, last := int(from*float64(rate)), int(until*float64(rate))
	samples := whole.Samples()
	if last*stereo > len(samples) {
		return movie.Sound{}, fmt.Errorf("%s ends before %.3f s", path, until)
	}
	return movie.NewSound(rate, samples[first*stereo:last*stereo])
}
