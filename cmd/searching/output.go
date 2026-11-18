package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"math"
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
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
	started := time.Now()
	var writer *movie.Writer
	if !call.On("defer-movie") {
		writer, err = openMovie(ctx, edit, into, from, count)
		if err != nil {
			return err
		}
		defer writer.Abort()
	}
	renderer := searching.NewRenderer(edit)
	defer renderer.Close()
	timing, err := pipeline(ctx, count, func(ctx context.Context, index int) (func() error, error) {
		at := float64(from+index) / float64(edit.Plan.FPS)
		exposure, state, err := renderer.Expose(ctx, at, "")
		if err != nil {
			return nil, errors.Join(err, context.Cause(ctx))
		}
		return func() error {
			picture := exposure.Develop()
			name := filepath.Join(into, "frames", fmt.Sprintf("frame-%06d.png", from+index))
			if err := writePNG(name, picture); err != nil {
				return err
			}
			if writer != nil {
				if err := writer.Frame(index, picture); err != nil {
					return errors.Join(err, context.Cause(ctx))
				}
			}
			if index%12 == 0 || index == count-1 {
				fmt.Fprintf(call.Out(), "%d/%d %.3fs %s %s elapsed=%s\n", index+1, count, at, state.Chapter, state.Site, time.Since(started).Round(time.Millisecond))
			}
			return nil
		}, nil
	}, call.On("serial"))
	if err != nil {
		return err
	}
	renderer.Close()
	debug.FreeOSMemory()
	if writer != nil {
		if err := writer.Close(); err != nil {
			return err
		}
	} else {
		begun := time.Now()
		if err := encodeFrames(ctx, edit, into, from, count); err != nil {
			return err
		}
		timing.EncodeSeconds = time.Since(begun).Seconds()
		timing.Mode += "-deferred-movie"
	}
	if err := receipt(edit, into, from, count, time.Since(started), timing); err != nil {
		return err
	}
	return publishBehind(into, false)
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
	type copiedInput struct {
		source, target, sha string
		dependency          bool
	}
	inputs := []copiedInput{{source: call.Args()[0], target: "shoot.yaml"},
		{source: edit.Plan.Score, target: "score.yaml"}}
	if edit.Plan.CatalogueStudy != nil {
		inputs = append(inputs, copiedInput{source: edit.Plan.CatalogueStudy.File,
			target: "catalogue.json", sha: edit.Plan.CatalogueStudy.SHA})
	}
	if edit.Plan.FlowStudy != nil && edit.Plan.FlowStudy.Geometry != nil {
		reference := edit.Plan.FlowStudy.Geometry
		inputs = append(inputs, copiedInput{source: filepath.Join(filepath.Dir(call.Args()[0]), reference.File),
			target: reference.File, sha: reference.SHA, dependency: true})
	}
	// These paths are either copied inputs or produced later in the render and
	// behind workflow. Check ancestors too: a dependency named "frames" or
	// "behind-the-scenes.html/child.yaml" would block a producer-owned path.
	reserved := []string{"shoot.yaml", "shoot.yaml.txt", "score.yaml", "score.yaml.txt",
		"catalogue.json", "catalogue.json.txt", "render.json", "animation.mp4", "frames",
		"behind-the-scenes.json", "behind-the-scenes.js", "behind-the-scenes.html"}
	outputs := map[string]bool{}
	for _, input := range inputs {
		if input.target == "" || !filepath.IsLocal(input.target) ||
			filepath.Clean(input.target) != input.target ||
			filepath.ToSlash(filepath.Clean(input.target)) != input.target {
			return fmt.Errorf("input copy target %q is not a clean relative path", input.target)
		}
		for _, name := range []string{input.target, input.target + ".txt"} {
			if input.dependency {
				for _, output := range reserved {
					if name == output || strings.HasPrefix(name, output+string(filepath.Separator)) ||
						strings.HasPrefix(output, name+string(filepath.Separator)) {
						return fmt.Errorf("input dependency target %q overlaps render output %q", name, output)
					}
				}
			}
			if outputs[name] {
				return fmt.Errorf("input copy target %q collides with another output", name)
			}
			outputs[name] = true
		}
	}
	type preparedInput struct {
		copiedInput
		raw []byte
	}
	prepared := make([]preparedInput, 0, len(inputs))
	for _, input := range inputs {
		var raw []byte
		var err error
		if input.dependency {
			file, openErr := os.Open(input.source)
			if openErr != nil {
				return openErr
			}
			raw, err = io.ReadAll(io.LimitReader(file, searching.FlowGeometryMaxBytes+1))
			err = errors.Join(err, file.Close())
			if err == nil && len(raw) > searching.FlowGeometryMaxBytes {
				err = fmt.Errorf("%s exceeds %d bytes", input.target, searching.FlowGeometryMaxBytes)
			}
		} else {
			raw, err = os.ReadFile(input.source)
		}
		if err != nil {
			return err
		}
		if input.sha != "" && fmt.Sprintf("%x", sha256.Sum256(raw)) != input.sha {
			return fmt.Errorf("%s changed after validation", input.target)
		}
		prepared = append(prepared, preparedInput{copiedInput: input, raw: raw})
	}
	for _, input := range prepared {
		path := filepath.Join(into, input.target)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		_, written := file.Write(input.raw)
		if err := errors.Join(written, file.Close()); err != nil {
			return err
		}
		view, err := os.OpenFile(path+".txt", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		_, written = view.Write(input.raw)
		if err := errors.Join(written, view.Close()); err != nil {
			return err
		}
	}
	return nil
}

// receipt records actual output facts; an incomplete run has no completion.
func receipt(edit *searching.Edit, into string, from, count int, spent time.Duration, timing pipelineTiming) error {
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
	plan, err := os.ReadFile(filepath.Join(into, "shoot.yaml"))
	if err != nil {
		return err
	}
	provenance := "authored observatory and blazar proof"
	if edit.Plan.FlowStudy != nil {
		provenance = "authored-previs: no measured galaxy or velocity catalogue"
	}
	if edit.Plan.CatalogueStudy != nil {
		provenance = "Gaia display sample; inverse-parallax distances, enlarged markers and compressed travel; catalogue SHA256 " + edit.Plan.CatalogueStudy.SHA
	}
	if edit.Plan.FullCut {
		provenance = "Full-song rough cut: authored observatories, named science-fiction scenes, explicit missing-shot slates, a separately pinned Gaia sample, and authored flow geometry"
	}
	flowGeometrySHA := ""
	if edit.Plan.FlowStudy != nil && edit.Plan.FlowStudy.Geometry != nil {
		flowGeometrySHA = edit.Plan.FlowStudy.Geometry.SHA
	}
	for _, shot := range edit.Plan.Shots {
		if shot.Stage != nil {
			provenance = "Heard stage proof: manufactured instruments; USGS terrain with authored distant ridges; authored weather, Griffith massing, respondents and larger-scale field; separately pinned Gaia star sample. Fastlight/native traversal with retained GPU resources; primary beam transport where launchers are present."
			break
		}
	}
	raw, err := json.MarshalIndent(map[string]any{"name": edit.Plan.Name, "source_commit": build,
		"source_start_frame": from, "frames": count, "fps": edit.Plan.FPS,
		"width": edit.Plan.Width, "height": edit.Plan.Height, "wall_seconds": spent.Seconds(),
		"pipeline": timing,
		"probed_movie": map[string]any{"frames": facts.Frames(), "width": facts.Width(),
			"height": facts.Height(), "fps": facts.Rate(), "duration_seconds": facts.Duration().Seconds(),
			"audio_seconds": facts.AudioDuration().Seconds(), "audio_channels": facts.AudioChannels(),
			"video_codec": facts.VideoCodec(), "audio_codec": facts.AudioCodec()},
		"score_sha256":         fmt.Sprintf("%x", sha256.Sum256(score)),
		"plan_sha256":          fmt.Sprintf("%x", sha256.Sum256(plan)),
		"flow_geometry_sha256": flowGeometrySHA,
		"scene_provenance":     provenance,
		"music_sha256":         edit.Grid.Score.SHA256, "status": "completed preview",
		"spectra": "authored components, not physical band renders"}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(into, "render.json"), append(raw, '\n'), 0644)
}
