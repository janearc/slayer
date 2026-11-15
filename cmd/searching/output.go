package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/slayer/improv/searching"
	"github.com/janearc/slayer/interfaces/movie"
)

// still makes a small actual render before the full music film is attempted.
func still(ctx context.Context, call pzm.Call) error {
	edit, err := read(call)
	if err != nil {
		return err
	}
	at := 0.0
	if call.Given("at") {
		at, err = strconv.ParseFloat(call.Value("at"), 64)
	}
	if err != nil || math.IsNaN(at) || math.IsInf(at, 0) || at < 0 || at >= edit.Grid.Score.Duration {
		return fmt.Errorf("at is outside the song")
	}
	if call.Value("out") == "" {
		return fmt.Errorf("still needs --out")
	}
	if call.DryRun() {
		return describe(call, edit)
	}
	ctx, stop, err := guard(ctx, call, filepath.Dir(call.Value("out")))
	if err != nil {
		return err
	}
	defer stop()
	renderer := searching.NewRenderer(edit)
	defer renderer.Close()
	picture, state, err := renderer.Frame(ctx, at, call.Value("site"))
	if err != nil {
		return err
	}
	if err := writePNG(call.Value("out"), picture); err != nil {
		return err
	}
	return json.NewEncoder(call.Out()).Encode(state)
}

// render keeps its range on the original frame grid, including an excerpt's
// source-audio trim. Cancellation closes the movie instead of publishing it.
func render(ctx context.Context, call pzm.Call) error {
	edit, err := read(call)
	if err != nil {
		return err
	}
	from, err := integer(call, "from", 0)
	if err != nil {
		return err
	}
	count, err := integer(call, "frames", edit.Frames()-from)
	if err != nil {
		return err
	}
	if from < 0 || count < 1 || from >= edit.Frames() || count > edit.Frames()-from {
		return fmt.Errorf("frame range outside song")
	}
	into := call.Value("out")
	if into == "" {
		return fmt.Errorf("render needs --out")
	}
	if call.DryRun() {
		return describe(call, edit)
	}
	if err := os.MkdirAll(filepath.Join(into, "frames"), 0755); err != nil {
		return err
	}
	ctx, stop, err := guard(ctx, call, into)
	if err != nil {
		return err
	}
	defer stop()
	if err := copyInputs(call, edit, into); err != nil {
		return err
	}
	sound, err := movie.ReadSound(edit.Plan.Music)
	if err != nil {
		return err
	}
	skip := int(float64(from)*float64(sound.Rate())/float64(edit.Plan.FPS)) * 2
	samples := sound.Samples()
	if skip >= len(samples) {
		return fmt.Errorf("audio trim past end")
	}
	trimmed, err := movie.NewSound(sound.Rate(), samples[skip:])
	if err != nil {
		return err
	}
	settings, err := movie.NewSettings(movie.Request{Path: filepath.Join(into, "animation.mp4"),
		Width: edit.Plan.Width, Height: edit.Plan.Height, Rate: edit.Plan.FPS, Frames: count, Sound: trimmed, Pad: true})
	if err != nil {
		return err
	}
	writer, err := movie.New(ctx, settings)
	if err != nil {
		return err
	}
	defer writer.Close()
	renderer := searching.NewRenderer(edit)
	defer renderer.Close()
	started := time.Now()
	for index := range count {
		at := float64(from+index) / float64(edit.Plan.FPS)
		picture, state, err := renderer.Frame(ctx, at, "")
		if err != nil {
			return err
		}
		name := filepath.Join(into, "frames", fmt.Sprintf("frame-%06d.png", from+index))
		if err := writePNG(name, picture); err != nil {
			return err
		}
		if err := writer.Frame(index, picture); err != nil {
			return err
		}
		if index%12 == 0 || index == count-1 {
			fmt.Fprintf(call.Out(), "%d/%d %.3fs %s %s elapsed=%s\n", index+1, count, at, state.Chapter, state.Site, time.Since(started).Round(time.Millisecond))
		}
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return receipt(edit, into, from, count, time.Since(started))
}

// writePNG refuses replacement and writes only a single completed image.
func writePNG(path string, picture image.Image) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	encoder := png.Encoder{CompressionLevel: png.BestSpeed}
	err = errors.Join(encoder.Encode(file, picture), file.Close())
	if err != nil {
		os.Remove(path)
	}
	return err
}

// copyInputs keeps recipes with outputs, without copying the source audio.
func copyInputs(call pzm.Call, edit *searching.Edit, into string) error {
	for source, name := range map[string]string{call.Args()[0]: "shoot.yaml", edit.Plan.Score: "score.yaml"} {
		raw, err := os.ReadFile(source)
		if err != nil {
			return err
		}
		path := filepath.Join(into, name)
		file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		_, written := file.Write(raw)
		if err := errors.Join(written, file.Close()); err != nil {
			return err
		}
	}
	return nil
}

// receipt records actual output facts; an incomplete run has no completion.
func receipt(edit *searching.Edit, into string, from, count int, spent time.Duration) error {
	facts, err := movie.Probe(filepath.Join(into, "animation.mp4"))
	if err != nil {
		return err
	}
	if facts.Frames() != count || facts.Width() != edit.Plan.Width ||
		facts.Rate() != float64(edit.Plan.FPS) || facts.AudioChannels() != 2 {
		return fmt.Errorf("finished movie differs from requested frames, width, rate or stereo soundtrack")
	}
	score, err := os.ReadFile(filepath.Join(into, "score.yaml"))
	if err != nil {
		return err
	}
	raw, err := json.MarshalIndent(map[string]any{"name": edit.Plan.Name, "source_commit": build,
		"source_start_frame": from, "frames": count, "fps": edit.Plan.FPS,
		"width": edit.Plan.Width, "height": edit.Plan.Height, "wall_seconds": spent.Seconds(),
		"probed_movie": map[string]any{"frames": facts.Frames(), "width": facts.Width(),
			"height": facts.Height(), "fps": facts.Rate(), "duration_seconds": facts.Duration().Seconds(),
			"audio_seconds": facts.AudioDuration().Seconds(), "audio_channels": facts.AudioChannels(),
			"video_codec": facts.VideoCodec(), "audio_codec": facts.AudioCodec()},
		"score_sha256": fmt.Sprintf("%x", sha256.Sum256(score)),
		"music_sha256": edit.Grid.Score.SHA256, "status": "completed preview",
		"spectra": "authored components, not physical band renders"}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(into, "render.json"), append(raw, '\n'), 0644)
}
