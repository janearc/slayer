package main

import (
	"context"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/janearc/slayer/cheeptune"
	"github.com/janearc/slayer/improv/host"
	"github.com/janearc/slayer/interfaces/movie"
	"gopkg.in/yaml.v3"
)

// The short in the house form: zappy's intro, then the scene's frames,
// drawn by slayer's jobs into folders, with zappy laid small at their
// foot by package host, in time with the tune. This file reads the plan
// and hands host what it asks for.

// the film's frame and the host in it, in pixels: a phone's tall frame,
// the host's square, and the box it drifts in at the foot, clear of the
// sentences above it. Picked by the author of this scene, 2026-11-12.
var (
	filmSize = image.Pt(1080, 1920)
	hostSize = image.Pt(280, 280)
	hostBox  = image.Rect(200, 1480, 880, 1900)
)

// filmRate is the film's frames a second, host's and walk's, defined.
const filmRate = 24

// hostSeed seeds the host's drift, SGR 1806-20's number, picked.
const hostSeed = 1806

// plan is magnetar-film.yaml as it is on disk.
type plan struct {
	Tune struct {
		Score   string   `yaml:"score"`
		Cuts    string   `yaml:"cuts"`
		Sound   string   `yaml:"sound"`
		Kicking []string `yaml:"kicking"`
	} `yaml:"tune"`
	Lifts []string `yaml:"lifts"`
	Clips string   `yaml:"clips"`
	Intro struct {
		Clip   string `yaml:"clip"`
		Words  string `yaml:"words"`
		Frames int    `yaml:"frames"`
	} `yaml:"intro"`
	Segments []struct {
		Frames string `yaml:"frames"`
		Count  int    `yaml:"count"`
	} `yaml:"segments"`
	Lines []struct {
		From  float64 `yaml:"from"`
		Until float64 `yaml:"until"`
		Say   string  `yaml:"say"`
	} `yaml:"lines"`
}

// readPlan is the plan at a path.
func readPlan(path string) (plan, error) {
	var read plan
	raw, err := os.ReadFile(path)
	if err == nil {
		err = yaml.Unmarshal(raw, &read)
	}
	return read, err
}

// film writes the short from the plan, its segments' frames found under
// frames, to out, saying how many frames it wrote.
func film(planPath, frames, out string, said io.Writer) error {
	read, err := readPlan(planPath)
	if err != nil {
		return err
	}
	show, err := read.show(frames)
	if err != nil {
		return err
	}
	written, err := show.Write(context.Background(), out)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(said, "%s: %d frames\n", out, written)
	return err
}

// show is the host's Show for the plan.
func (read plan) show(frames string) (host.Show, error) {
	tune, err := read.tune()
	if err != nil {
		return host.Show{}, err
	}
	clips, err := host.ReadClips(read.Clips, hostSize)
	if err != nil {
		return host.Show{}, err
	}
	box, err := host.NewBox(hostBox, hostSize)
	if err != nil {
		return host.Show{}, err
	}
	source, count, err := read.source(frames)
	if err != nil {
		return host.Show{}, err
	}
	show := host.Show{Frames: source, Count: count, Tune: tune,
		Clips: clips, Box: box, Picture: image.Rectangle{Max: filmSize},
		Seed: hostSeed, Size: filmSize, Lifts: map[string]bool{},
		Intro: host.Intro{Words: read.Intro.Words,
			Seconds: float64(read.Intro.Frames) / filmRate}}
	if clip, found := clips[read.Intro.Clip]; found {
		show.Intro.Clip = &clip
	}
	for _, part := range read.Lifts {
		show.Lifts[part] = true
	}
	for _, line := range read.Lines {
		show.Speaks = append(show.Speaks, host.Span{From: line.From,
			Until: line.Until})
	}
	return show, nil
}

// tune is the plan's tune: its score, its parts and its own sound.
func (read plan) tune() (host.Tune, error) {
	score, err := cheeptune.ReadScore(read.Tune.Score)
	if err != nil {
		return host.Tune{}, err
	}
	sheet, err := cheeptune.ReadCuts(read.Tune.Cuts, score)
	if err != nil {
		return host.Tune{}, err
	}
	sound, err := movie.ReadSound(read.Tune.Sound)
	if err != nil {
		return host.Tune{}, err
	}
	return host.NewTune(score, sheet, sound, read.Tune.Kicking)
}

// source is the segments' frames as one run, and how many: each
// segment's pictures in name order, as many as the plan says it holds.
func (read plan) source(frames string) (func(int) (*image.RGBA, error),
	int, error) {
	var all []string
	for _, segment := range read.Segments {
		pictures, err := picturesIn(filepath.Join(frames, segment.Frames))
		if err != nil {
			return nil, 0, err
		}
		if len(pictures) != segment.Count {
			return nil, 0, fmt.Errorf("%s holds %d pictures, the plan "+
				"says %d; draw its job again", segment.Frames,
				len(pictures), segment.Count)
		}
		all = append(all, pictures...)
	}
	return func(i int) (*image.RGBA, error) { return readPNG(all[i]) },
		len(all), nil
}

// picturesIn is the png files in a folder, in name order.
func picturesIn(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var pictures []string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".png") {
			pictures = append(pictures, filepath.Join(dir, entry.Name()))
		}
	}
	slices.Sort(pictures)
	return pictures, nil
}

// readPNG is a png as RGBA.
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
	made := image.NewRGBA(decoded.Bounds())
	draw.Draw(made, made.Bounds(), decoded, decoded.Bounds().Min, draw.Src)
	return made, nil
}
