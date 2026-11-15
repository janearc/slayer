//go:build draw && darwin && cgo

package door

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/janearc/slayer/cheeptune"
	"github.com/janearc/slayer/interfaces/movie"
)

// a film with a movie and sound writes an mp4 that Probe reads back at
// the film's size, padded, its rate and frame count, with an audio
// track; the padding is said once.
func TestDrawFilmWritesAMovieWithSound(t *testing.T) {
	directory := t.TempDir()
	path := written(t, directory, "film.yaml", filmJob(t,
		"sound: {tune: \"@crypt\", voicing: \"@crypt-dmg\"}\n"+
			"outputs: {movie: film.mp4}\n"))
	said := drew(t, envIn(t, context.Background(), directory), path)
	facts, err := movie.Probe(filepath.Join(directory, "film.mp4"))
	if err != nil {
		t.Fatal(err)
	}
	if facts.Width() != number(t, "width") ||
		facts.Height() != padded(t) ||
		facts.Frames() != filmFrames(t) ||
		int(facts.Rate()) != number(t, "fps") {
		t.Errorf("the movie is %dx%d, %d frames at %v", facts.Width(),
			facts.Height(), facts.Frames(), facts.Rate())
	}
	if facts.AudioCodec() == "" || facts.AudioChannels() != stereo {
		t.Errorf("the movie's sound is %q in %d channels",
			facts.AudioCodec(), facts.AudioChannels())
	}
	padded := fmt.Sprintf("padded %d rows to %d for the film",
		number(t, "height"), padded(t))
	if strings.Count(said, padded) != one {
		t.Errorf("the padding was not said once: %q", said)
	}
	if !strings.Contains(said, "the tune is cut at the film's end") {
		t.Errorf("how the tune fits was not said: %q", said)
	}
}

// a film with a movie and no sound has no audio track,
// and a movie with no frames folder writes nothing but the movie.
// anti-test of TestDrawFilmWritesAMovieWithSound
func TestDrawFilmWithoutSoundIsSilent(t *testing.T) {
	directory := t.TempDir()
	path := written(t, directory, "film.yaml",
		filmJob(t, "outputs: {movie: film.mp4}\n"))
	drew(t, envIn(t, context.Background(), directory), path)
	facts, err := movie.Probe(filepath.Join(directory, "film.mp4"))
	if err != nil {
		t.Fatal(err)
	}
	if facts.AudioCodec() != "" {
		t.Errorf("a film with no sound has %q", facts.AudioCodec())
	}
	if got := files(t, directory); len(got) != stereo {
		t.Errorf("a movie alone wrote %v", got)
	}
}

// an audio file is read from its from onward, and a from past its end
// is refused by the file's name.
func TestSoundFromAFile(t *testing.T) {
	directory := t.TempDir()
	piece, err := cheeptune.Open("@crypt", "@crypt-dmg")
	if err != nil {
		t.Fatal(err)
	}
	wav := filepath.Join(directory, "crypt.wav")
	if err := piece.WriteWAV(wav); err != nil {
		t.Fatal(err)
	}
	length := duration(t, "title-until")
	made, err := filed(wav, length, length)
	if err != nil {
		t.Fatal(err)
	}
	if made.sound.Duration() >= piece.Duration() ||
		!strings.Contains(made.said, "the sound file plays") {
		t.Errorf("the file from %s is %s: %q", length,
			made.sound.Duration(), made.said)
	}
	// anti-test of TestSoundFromAFile
	_, err = filed(wav, piece.Duration()*stereo, length)
	if err == nil || !strings.Contains(err.Error(), "crypt.wav") {
		t.Errorf("a from past the end gave %v", err)
	}
	if _, err := filed(filepath.Join(directory, "gone.wav"), nothing,
		length); err == nil {
		t.Error("a file not there was read")
	}
	path := written(t, directory, "film.yaml", filmJob(t, fmt.Sprintf(
		"sound: {file: crypt.wav, from: {value: %g, unit: s}}\n"+
			"outputs: {movie: film.mp4}\n", length.Seconds())))
	said := drew(t, envIn(t, context.Background(), directory), path)
	if !strings.Contains(said, "the sound file plays") {
		t.Errorf("a film of a sound file said %q", said)
	}
	if _, err := os.Stat(filepath.Join(directory, "film.mp4")); err != nil {
		t.Errorf("a film of a sound file wrote no movie: %v", err)
	}
}

// a film of frames and a movie drawn twice keeps both the second time
// and draws nothing; with its sound taken out, it keeps every frame and
// makes the movie again, silent; with its picture changed, it draws its
// frames and its movie again.
func TestDrawFramesAndMovieTwice(t *testing.T) {
	directory := t.TempDir()
	env := envIn(t, context.Background(), directory)
	sound := "sound: {tune: \"@crypt\", voicing: \"@crypt-dmg\"}\n"
	outputs := "outputs: {frames: film, movie: film.mp4}\n"
	path := written(t, directory, "film.yaml", filmJob(t, sound+outputs))
	drew(t, env, path)
	movieAt := filepath.Join(directory, "film.mp4")
	first := fileSum(movieAt)
	again := drew(t, env, path)
	if !strings.Contains(again, "nothing to draw") ||
		fileSum(movieAt) != first {
		t.Errorf("the second run did not keep both: %q", again)
	}
	// anti-test of TestDrawFramesAndMovieTwice
	written(t, directory, "film.yaml", filmJob(t, outputs))
	silent := drew(t, env, path)
	facts, err := movie.Probe(movieAt)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(silent, fmt.Sprintf("drew 0 frames and kept %d",
		filmFrames(t))) || facts.AudioCodec() != "" ||
		facts.Frames() != filmFrames(t) {
		t.Errorf("the silent run said %q, the movie %q with %d frames",
			silent, facts.AudioCodec(), facts.Frames())
	}
	remade := fileSum(movieAt)
	written(t, directory, "film.yaml", filmJob(t,
		"stock: {name: lantern}\n"+outputs))
	stocked := drew(t, env, path)
	if !strings.Contains(stocked, "from an earlier version") ||
		fileSum(movieAt) == remade {
		t.Errorf("a changed picture did not draw again: %q", stocked)
	}
}

// a finished film that writes a movie and no frames leaves the movie,
// the job copy and its note, and nothing else: its working frames are
// gone from the cache.
func TestDrawFinishedFilmLeavesNoFrames(t *testing.T) {
	directory, home := t.TempDir(), t.TempDir()
	env := envIn(t, context.Background(), directory,
		rendersVariable, home, teamVariable, "team")
	path := written(t, directory, "short.yaml", filmJob(t,
		"outputs: {movie: short.mp4, note: a short}\n"))
	said := drew(t, env, path)
	inside := filepath.Join(filmsFolder, today+"-short")
	want := []string{filepath.Join(inside, "short.mp4"),
		filepath.Join(inside, "short.yaml"),
		filepath.Join("team", notesName)}
	if got := files(t, home); !slices.Equal(got, want) {
		t.Errorf("a finished film left %v, not %v", got, want)
	}
	if !strings.Contains(said, "removed the working frames") {
		t.Errorf("the film did not say its frames went: %q", said)
	}
}

// with --keep-frames the working frames stay in the
// cache, every one, with their ledger, and the line says where.
// anti-test of TestDrawFinishedFilmLeavesNoFrames
func TestDrawKeepsWorkingFramesWhenAsked(t *testing.T) {
	directory, home := t.TempDir(), t.TempDir()
	env := envIn(t, context.Background(), directory,
		rendersVariable, home, teamVariable, "team")
	path := written(t, directory, "short.yaml",
		filmJob(t, "outputs: {movie: short.mp4}\n"))
	said := drew(t, env, path, "--keep-frames")
	kept := files(t, filepath.Join(home, cacheFolder))
	if len(kept) != filmFrames(t)+one ||
		!strings.Contains(said, "kept the working frames in") {
		t.Errorf("kept %v, said %q", kept, said)
	}
}
