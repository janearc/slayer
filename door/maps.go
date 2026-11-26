package door

// A named scene drawn as maps, through the door.
//
// Design. A scene of maps (package scene/named, the weather) has no world
// and no eye: carried to each frame's scene time, it makes several
// pictures, each named as its command names them. The door draws it on a
// path of its own: no source, no developer and no titles.
//
// A frame's pictures all go into the frames folder under their own names;
// the movie is made of the one the scene shows, and a still writes that
// one to its picture.
//
// The scene keeps its moments in its own cache (the weather's is shared
// with the session), spaced a frame apart, so a second run reaches each frame
// from a checkpoint. The frames folder keeps a manifest of the job's key,
// so a folder of another job is refused as a traced film's is.
//
// What a scene of maps cannot be drawn with is refused by name: a film
// size, rays, lens or exposure, a camera, a shutter, a print, a stock,
// titles, sound, raw light or a quality. With no renderer said, the
// scene takes its own: the weather goes on the gpu where there is one.

import (
	"errors"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"time"

	"github.com/janearc/slayer/interfaces/file"
	"github.com/janearc/slayer/interfaces/movie"
	"github.com/janearc/slayer/job"
	scenes "github.com/janearc/slayer/scene/named"
)

// lonelySpacing is the spacing of a still's moments in its scene's
// cache when it is at nothing: defined, as a still at nothing takes no
// step and leaves no checkpoint, so any spacing serves.
const lonelySpacing = time.Second

// mapsOf is the job's scene when it is a named scene drawn as maps, and
// whether it is one.
func mapsOf(wanted job.Job) (*scenes.Maps, bool) {
	name, isNamed := wanted.Scene().Named()
	if !isNamed {
		return nil, false
	}
	scene, err := scenes.GetMaps(name)
	return scene, err == nil
}

// mapsRefused is what a scene of maps cannot be drawn with, each with
// whether the job says it.
func mapsRefused(wanted job.Job) []struct {
	said bool
	what string
} {
	film := wanted.Film()
	_, width := film.Width()
	_, height := film.Height()
	_, rays := film.Rays()
	_, samples := film.Samples()
	_, wide := film.Wide()
	_, tall := film.Tall()
	_, exposure := film.Exposure()
	_, camera := wanted.Camera()
	_, shutter := wanted.Shutter()
	_, printed := wanted.Print()
	_, stock := wanted.Stock()
	_, titles := wanted.Titles()
	_, sound := wanted.Sound()
	_, raw := wanted.Outputs().Raw()
	_, quality := wanted.Quality()
	return []struct {
		said bool
		what string
	}{
		{width || height, "a film size (its maps are its grid's size: " +
			"say across and rows in with, and fps for a film's rate)"},
		{rays || samples, "rays or samples"}, {wide || tall, "a lens"},
		{exposure, "an exposure"}, {camera, "a camera"},
		{shutter, "a shutter"}, {printed, "a print"}, {stock, "a stock"},
		{titles, "titles"}, {sound, "sound"}, {raw, "raw light"},
		{quality, "a quality"},
	}
}

// mapsReady refuses, by name, the first thing the job says that a scene
// of maps cannot be drawn with.
func mapsReady(wanted job.Job, scene *scenes.Maps) error {
	for _, each := range mapsRefused(wanted) {
		if each.said {
			return fmt.Errorf("%s: the %s scene is drawn as maps and "+
				"cannot be drawn with %s; take it out of the job",
				wanted.File(), scene.Name(), each.what)
		}
	}
	return nil
}

// mapper is the job's scene of maps built: its with block, the renderer
// the job says, the scene's cache spaced a frame apart, and its words
// on the verb's output.
func (run runner) mapper(scene *scenes.Maps) (scenes.Mapper, error) {
	if err := mapsReady(run.wanted, scene); err != nil {
		return nil, err
	}
	cache, err := scene.Cache()
	if err != nil {
		return nil, err
	}
	renderer, said := run.wanted.Renderer()
	mapTimes := run.wanted.Times()
	ask, err := scenes.AskMaps(scenes.Renderer(renderer), said, cache,
		spacing(mapTimes), run.out)
	if err != nil {
		return nil, err
	}
	built, err := scene.Build(run.wanted.Scene().Parameters(), ask)
	if err != nil {
		return nil, fmt.Errorf("%s: scene: %w", run.wanted.File(), err)
	}
	return built, nil
}

// spacing is how far apart a job's frames are in scene time: a film's
// span over its frames, a still's moment, or lonelySpacing for a still
// at nothing.
func spacing(times []time.Duration) time.Duration {
	last := len(times) - one
	if last > nothing {
		return (times[last] - times[nothing]) / time.Duration(last)
	}
	if times[nothing] > nothing {
		return times[nothing]
	}
	return lonelySpacing
}

// mapped draws a scene of maps: a still's shown map to its picture, or
// each frame's maps to the frames folder and its shown map to the movie.
func (run runner) mapped(scene *scenes.Maps) error {
	built, err := run.mapper(scene)
	if err != nil {
		return err
	}
	if !run.wanted.Clock().Still() {
		return run.mappedFilm(built)
	}
	if err := run.unclaimed(); err != nil {
		return err
	}
	if err := run.copied(); err != nil {
		return err
	}
	return run.mappedStill(built)
}

// shown is the map a scene shows among a moment's maps.
func shown(built scenes.Mapper, maps []scenes.Map) (*image.RGBA, error) {
	for _, each := range maps {
		if each.Kind() == built.Shows() {
			return each.Picture(), nil
		}
	}
	return nil, fmt.Errorf("the scene shows %s and made no such map",
		built.Shows())
}

// mappedStill writes a still's shown map to its picture.
func (run runner) mappedStill(built scenes.Mapper) error {
	maps, err := built.At(run.wanted.Times()[nothing])
	if err != nil {
		return err
	}
	picture, err := shown(built, maps)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(run.at.picture),
		folderMode); err != nil {
		return err
	}
	if err := file.PNG(run.at.picture, picture, square); err != nil {
		return err
	}
	_, err = fmt.Fprintf(run.out, "wrote %s\n", run.at.picture)
	return err
}

// mapping is a film of maps in flight: its frames' ledger, and its movie
// once the first shown map has said its size.
type mapping struct {
	run   runner
	built scenes.Mapper
	book  *ledger // nil where the job writes no frames
	reel  reel    // nil until the first frame, and where no movie
}

// mappedFilm draws each frame's maps in order. A cancelled context or a
// failed frame throws the movie away.
func (run runner) mappedFilm(built scenes.Mapper) error {
	going := &mapping{run: run, built: built}
	if run.at.frames != "" {
		book, stale, err := openLedger(run.at.frames, run.keys.frames)
		if err != nil {
			return err
		}
		going.book = book
		if stale {
			if _, err := fmt.Fprintln(run.out, "the maps here are from "+
				"an earlier version of this job; drawing again"); err != nil {
				return err
			}
		}
	}
	if err := run.copied(); err != nil {
		return err
	}
	times := run.wanted.Times()
	for index, at := range times {
		if err := run.ctx.Err(); err != nil {
			return going.abort(err)
		}
		if err := going.frame(index, len(times), at); err != nil {
			return going.abort(err)
		}
	}
	return going.finish(len(times))
}

// frame reaches one frame's moment, writes its maps and hands the shown
// one to the movie, and says so.
func (going *mapping) frame(index, frames int, at time.Duration) error {
	start := going.run.now()
	maps, err := going.built.At(at)
	if err != nil {
		return err
	}
	if err := going.write(maps); err != nil {
		return err
	}
	picture, err := shown(going.built, maps)
	if err != nil {
		return err
	}
	if err := going.encode(index, frames, picture); err != nil {
		return err
	}
	sinceStart := going.run.now().Sub(start)
	_, err = fmt.Fprintf(going.run.out, "frame %d of %d: %s, on the %s, "+
		"%s\n", index+one, frames, maps[nothing].Name(),
		going.built.Renderer(), hundredths(sinceStart))
	return err
}

// write writes a frame's maps to the frames folder under their own
// names, each recorded in the ledger.
func (going *mapping) write(maps []scenes.Map) error {
	if going.book == nil {
		return nil
	}
	for _, each := range maps {
		name := each.Name() + pictureExtension
		sum, err := file.SequencePNGHash(filepath.Join(going.book.folder,
			name), each.Picture())
		if err != nil {
			return err
		}
		if err := going.book.note(name, sum); err != nil {
			return err
		}
	}
	return nil
}

// encode hands the shown map to the movie, opening it at the first.
func (going *mapping) encode(index, frames int, picture *image.RGBA) error {
	run := going.run
	if run.at.movie == "" {
		return nil
	}
	if going.reel == nil {
		rate, _ := run.wanted.Clock().Rate()
		size := picture.Bounds().Size()
		settings, err := movie.NewSettings(movie.Request{
			Path: run.at.movie, Width: size.X, Height: size.Y,
			Rate: rate, Frames: frames, Pad: true, Replace: true})
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(run.at.movie),
			folderMode); err != nil {
			return err
		}
		if going.reel, err = run.shop.Reel(run.ctx, settings); err != nil {
			return err
		}
	}
	return going.reel.Frame(index, picture)
}

// abort throws the movie away and gives the reason it stopped.
func (going *mapping) abort(reason error) error {
	if going.reel == nil {
		return reason
	}
	return errors.Join(reason, going.reel.Abort())
}

// finish closes the movie and says what was written.
func (going *mapping) finish(frames int) error {
	run := going.run
	var lines []string
	if going.reel != nil {
		if err := going.reel.Close(); err != nil {
			return err
		}
	}
	if going.book != nil {
		line, err := going.mapsLine(frames)
		if err != nil {
			return err
		}
		lines = append(lines, line)
	}
	if going.reel != nil {
		mapShown := going.built.Shows()
		lines = append(lines, fmt.Sprintf("wrote %s, of the %s map",
			run.at.movie, mapShown))
	}
	for _, line := range lines {
		if _, err := fmt.Fprintln(run.out, line); err != nil {
			return err
		}
	}
	return nil
}

// mapsLine says where the maps are: written where the job says its
// frames, kept as working files where the call asked, or removed once
// the movie is written, where they were working files, so the folder's
// one thing to look at is the film.
func (going *mapping) mapsLine(frames int) (string, error) {
	run, book := going.run, going.book
	switch {
	case !run.at.work:
		return fmt.Sprintf("wrote %d frames of maps in %s", frames,
			book.folder), nil
	case run.keepFrames:
		return fmt.Sprintf("kept the working maps in %s, as asked",
			book.folder), nil
	}
	if err := book.removeAll(); err != nil {
		return "", err
	}
	if err := os.Remove(book.folder); err != nil {
		return "", fmt.Errorf("%s: the working maps are removed and the "+
			"folder is not: %w", book.folder, err)
	}
	return "removed the working maps, the movie being written", nil
}
