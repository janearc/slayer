package main

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/slayer/beats"
	"github.com/janearc/slayer/interfaces/movie"
	yaml "gopkg.in/yaml.v3"
)

// analyse validates selection before opening audio and never replaces a score.
func analyse(call pzm.Call) error {
	if len(call.Args()) != 1 || call.Value("out") == "" {
		return fmt.Errorf("analyse needs one audio file and --out SCORE.yaml")
	}
	tempo, tempoErr := number(call, "tempo", 0)
	meter, meterErr := number(call, "meter", 4)
	first, firstErr := number(call, "first", -1)
	if err := errors.Join(tempoErr, meterErr, firstErr); err != nil {
		return err
	}
	if math.IsNaN(meter) || math.IsInf(meter, 0) || meter < 1 ||
		meter > 32 || math.Trunc(meter) != meter {
		return fmt.Errorf("meter is a whole count in 1..32")
	}
	settings := beats.Settings{Tempo: tempo, Meter: int(meter), First: first}
	if err := settings.Check(); err != nil {
		return err
	}
	if call.DryRun() {
		fmt.Fprintln(call.Out(), "would analyse", call.Args()[0])
		return nil
	}
	return analyseFile(call, settings)
}

// analyseFile keeps the file identity with the estimated grid. Analysis
// never edits the audio, normalizes its playback or changes its speed.
func analyseFile(call pzm.Call, settings beats.Settings) error {
	path, err := filepath.Abs(call.Args()[0])
	if err != nil {
		return err
	}
	hash, err := audioHash(path)
	if err != nil {
		return err
	}
	sound, err := movie.ReadSound(path)
	if err != nil {
		return err
	}
	analysis, stem, err := analysisSound(call.Value("drums"), sound)
	if err != nil {
		return err
	}
	grid, err := beats.Detect(analysis.Samples(), analysis.Rate(), settings)
	if err != nil {
		return err
	}
	after, err := audioHash(path)
	if err != nil || hash != after {
		return fmt.Errorf("audio changed while analysing: %s: %v", path, err)
	}
	grid.Score.Audio, grid.Score.SHA256 = path, hash
	grid.Score.Duration = sound.Duration().Seconds()
	grid.Score.Analysis = stem
	if stem != nil {
		grid.Score.Method = "drum-stem-energy-dp-v1"
	}
	if err := grid.Check(); err != nil {
		return err
	}
	raw, err := yaml.Marshal(grid)
	if err != nil {
		return err
	}
	if err := writeNew(call.Value("out"), raw); err != nil {
		return err
	}
	fmt.Fprintf(call.Out(), "%d estimated beats over %.3f seconds; "+
		"check the visual metronome: %s\n", len(grid.Score.Beats.Times),
		grid.Score.Duration, call.Value("out"))
	return nil
}

// analysisSound keeps the stem on the original audio timeline. A stem of
// different length is refused: blindly aligning starts would hide a trim.
func analysisSound(path string, original movie.Sound) (movie.Sound,
	*beats.Input, error) {
	if path == "" {
		return original, nil, nil
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return movie.Sound{}, nil, err
	}
	before, err := audioHash(path)
	if err != nil {
		return movie.Sound{}, nil, err
	}
	stem, err := movie.ReadSound(path)
	if err != nil {
		return movie.Sound{}, nil, err
	}
	after, err := audioHash(path)
	if err != nil || before != after {
		return movie.Sound{}, nil, fmt.Errorf("analysis stem changed during read")
	}
	if stem.Rate() != original.Rate() || stem.Length() != original.Length() {
		return movie.Sound{}, nil, fmt.Errorf("drum stem must retain original "+
			"sample rate and length: stem %d/%d, original %d/%d",
			stem.Rate(), stem.Length(), original.Rate(), original.Length())
	}
	return stem, &beats.Input{File: path, SHA256: before}, nil
}

// audioHash identifies the encoded source, streaming rather than copying it.
func audioHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	_, err = io.Copy(hash, file)
	return fmt.Sprintf("%x", hash.Sum(nil)), err
}

// writeNew refuses an existing score, leaving human corrections intact.
func writeNew(path string, raw []byte) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	_, writeErr := file.Write(raw)
	err = errors.Join(writeErr, file.Sync(), file.Close())
	if err != nil {
		os.Remove(path)
	}
	return err
}
