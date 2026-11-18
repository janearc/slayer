package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/slayer/beats"
	"github.com/janearc/slayer/cheeptune"
)

// music renders an authored score once and binds a browser-readable beat
// grid to those exact samples. Existing takes and source files stay intact.
func music(call pzm.Call) error {
	if len(call.Args()) != 1 || !call.Given("voicing") || !call.Given("out") {
		return fmt.Errorf("music requires SCORE, --voicing FILE and --out NEW_DIRECTORY")
	}
	scoreBytes, err := os.ReadFile(call.Args()[0])
	if err != nil {
		return err
	}
	voiceBytes, err := os.ReadFile(call.Value("voicing"))
	if err != nil {
		return err
	}
	score, err := cheeptune.ParseScore(scoreBytes)
	if err != nil {
		return err
	}
	if score.Loop || score.Seconds() > 600 || score.Bars()*score.Meter < 2 {
		return fmt.Errorf("music proof requires a non-looping score of two or more beats, at most ten minutes")
	}
	voice, err := cheeptune.ParseVoicing(voiceBytes)
	if err != nil {
		return err
	}
	out, err := filepath.Abs(call.Value("out"))
	if err != nil {
		return err
	}
	if _, err := os.Lstat(out); !os.IsNotExist(err) {
		return fmt.Errorf("music output must be a new directory")
	}
	if call.DryRun() {
		return nil
	}
	if err := resources(filepath.Dir(out)); err != nil {
		return err
	}
	// Claim the new directory before rendering; a failed synthesis retains no
	// completion receipt, and never replaces a concurrent take.
	if err := os.Mkdir(out, 0o755); err != nil {
		return err
	}
	for name, data := range map[string][]byte{"composition.json.txt": scoreBytes, "voicing.json.txt": voiceBytes} {
		if err := os.WriteFile(filepath.Join(out, name), data, 0o644); err != nil {
			return err
		}
	}
	piece, err := cheeptune.Render(score, voice)
	if err != nil {
		return err
	}
	if err := resources(out); err != nil {
		return err
	}
	wav := filepath.Join(out, "music.wav")
	if err := piece.WriteWAV(wav); err != nil {
		return err
	}
	raw, err := os.ReadFile(wav)
	if err != nil {
		return err
	}
	grid, err := compositionGrid(score, float64(len(piece.Left))/float64(piece.Rate), wav, fmt.Sprintf("%x", sha256.Sum256(raw)))
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(grid, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	for _, name := range []string{"score.json", "score.json.txt"} {
		if err := os.WriteFile(filepath.Join(out, name), data, 0o644); err != nil {
			return err
		}
	}
	receipt := map[string]any{"source_commit": build, "composition_sha256": fmt.Sprintf("%x", sha256.Sum256(scoreBytes)),
		"voicing_sha256": fmt.Sprintf("%x", sha256.Sum256(voiceBytes)),
		"audio_sha256":   grid.Score.SHA256, "sample_rate": piece.Rate, "samples": len(piece.Left),
		"duration_seconds": grid.Score.Duration, "beats": len(grid.Score.Beats.Times),
		"note": "Exact authored score clock; counted-through rests, no onset measurement or ear approval."}
	data, err = json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(out, "music.json"), append(data, '\n'), 0o644); err != nil {
		return err
	}
	return json.NewEncoder(call.Out()).Encode(receipt)
}

// compositionGrid keeps tempo changes and silent bars on the score clock.
// Estimated means editorial approval remains open, not that tempo is guessed.
func compositionGrid(score *cheeptune.Score, duration float64, audio, hash string) (*beats.Grid, error) {
	grid := &beats.Grid{Score: beats.Score{Audio: audio, SHA256: hash, Duration: duration, Meter: score.Meter,
		Status: "estimated", Method: "authored score clock, including tempo changes; no measured onset marks"}}
	for beat := 0; beat < score.Bars()*score.Meter; beat++ {
		grid.Score.Beats.Times = append(grid.Score.Beats.Times, score.SecondsAt(float64(beat)))
		grid.Score.Beats.Measured = append(grid.Score.Beats.Measured, 0)
	}
	return grid, grid.Check()
}
