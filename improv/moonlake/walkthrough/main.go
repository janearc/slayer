// Command walkthrough writes moonlake's walkthrough shorts from
// improv/moonlake/walkthrough.yaml with package walk, the song's quiet
// opening under each, cut on its bars; or one frame of a short, or one
// picture of the finished film, as a PNG.
//
//	walkthrough -script S -pictures DIR -short NAME -score SCORE \
//		-song WAV -mp4 OUT
//	walkthrough -script S -short NAME -frame K -png OUT
//	walkthrough -film FILM -at SECONDS -png OUT
//
// The song's bars before the score's first, its quiet opening, are
// counted back from bar 1 at the length of bar 1, as the score counts
// beats through a break: bar 0 is the bar before bar 1.
package main

import (
	"context"
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"time"

	"github.com/janearc/slayer/improv/moonlake/show"
	"github.com/janearc/slayer/improv/walk"
	"github.com/janearc/slayer/interfaces/movie"
)

// asked is what the command was asked for.
type asked struct {
	script, short, score, song, mp4, png, film, pictures string
	frame                                                int
	at                                                   float64
}

// main reads the flags and does what they ask, or says why not.
func main() {
	var given asked
	flag.StringVar(&given.script, "script", "", "the walkthrough script")
	flag.StringVar(&given.short, "short", "", "the short's name")
	flag.StringVar(&given.pictures, "pictures", "",
		"the folder of the shots' stills and sets")
	flag.StringVar(&given.score, "score", "", "the song's score")
	flag.StringVar(&given.song, "song", "",
		"the song, any file AVFoundation reads")
	flag.StringVar(&given.mp4, "mp4", "", "the short to write")
	flag.StringVar(&given.png, "png", "", "the picture to write")
	flag.StringVar(&given.film, "film", "", "a film to take a picture from")
	flag.IntVar(&given.frame, "frame", -1, "a frame of the short to write")
	flag.Float64Var(&given.at, "at", 0, "the film's moment, in seconds")
	flag.Parse()
	if err := run(given); err != nil {
		fmt.Fprintln(os.Stderr, "walkthrough:", err)
		os.Exit(1)
	}
}

// run does what was asked.
func run(given asked) error {
	if given.film != "" {
		return filmPicture(given)
	}
	script, err := walk.ReadScript(given.script, given.pictures)
	if err != nil {
		return err
	}
	short, found := script.Short(given.short)
	if !found {
		return fmt.Errorf("the script has no short %q", given.short)
	}
	if given.frame >= 0 {
		frame, err := short.Frame(given.frame, walk.NewStills())
		if err != nil {
			return err
		}
		return writePNG(given.png, frame)
	}
	sound, err := soundFor(short, given)
	if err != nil {
		return err
	}
	written, err := short.Write(context.Background(), given.mp4, sound)
	fmt.Printf("%s: %d frames, %.1f s\n", given.mp4, written,
		short.Seconds())
	return err
}

// soundFor is the song under a short, from its bar.
func soundFor(short walk.Short, given asked) (movie.Sound, error) {
	bar, sounded := short.Bar()
	if !sounded {
		return movie.Sound{}, nil
	}
	score, err := show.ReadScore(given.score)
	if err != nil {
		return movie.Sound{}, err
	}
	from, err := barTime(score, bar)
	if err != nil {
		return movie.Sound{}, err
	}
	song, err := movie.ReadSound(given.song)
	if err != nil {
		return movie.Sound{}, err
	}
	fmt.Printf("sound: bar %d, from %.3f s\n", bar, from)
	return walk.SongSpan(song, from, short.Seconds())
}

// barTime is when a bar starts: the score's own from bar 1, and before
// it counted back at the length of bar 1.
func barTime(score *show.Score, bar int) (float64, error) {
	one, err := score.BarTime(1, 1)
	if err != nil || bar >= 1 {
		if bar >= 1 {
			return score.BarTime(bar, 1)
		}
		return 0, err
	}
	two, err := score.BarTime(2, 1)
	if err != nil {
		return 0, err
	}
	return one - float64(1-bar)*(two-one), nil
}

// filmPicture writes the picture of a film at a moment, at its own size.
func filmPicture(given asked) error {
	film, err := movie.OpenFilm(given.film, 0, oneByOne)
	if err != nil {
		return err
	}
	size := film.Native()
	film.Close()
	picture, err := movie.FrameAt(given.film,
		time.Duration(given.at*float64(time.Second)), size)
	if err != nil {
		return err
	}
	return writePNG(given.png, picture)
}

// writePNG writes a picture.
func writePNG(path string, picture image.Image) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(file, picture); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}

// oneByOne is the size a film is opened at to ask its own size.
var oneByOne = image.Pt(1, 1)
