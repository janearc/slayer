package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/slayer/maths"
)

type sequenceFrame struct {
	Directory      string        `json:"directory"`
	SceneSeconds   float64       `json:"scene-seconds"`
	SettingsSHA256 string        `json:"settings-sha256"`
	ImageSHA256    string        `json:"image-sha256"`
	Elapsed        time.Duration `json:"elapsed-nanoseconds"`
}

type sequenceRecord struct {
	Complete         bool            `json:"complete"`
	Started          time.Time       `json:"started"`
	Elapsed          time.Duration   `json:"elapsed-nanoseconds"`
	Preparation      time.Duration   `json:"preparation-nanoseconds"`
	InputSceneSHA256 string          `json:"input-scene-sha256"`
	InputPlanSHA256  string          `json:"input-plan-sha256"`
	BinarySHA256     string          `json:"binary-sha256"`
	FPS              int             `json:"fps"`
	ExpectedFrames   int             `json:"expected-frames"`
	SceneDuration    float64         `json:"scene-duration-seconds"`
	Frames           []sequenceFrame `json:"frames"`
}

// sequence shares verified sources and prototypes across sampled views. Its
// completion record is published only after every frame and source close.
func sequence(ctx context.Context, call pzm.Call) (failure error) {
	if len(call.Args()) != 0 {
		return fmt.Errorf("island sequence takes --plan and --out, without positional arguments")
	}
	name, err := required(call, "plan")
	if err != nil {
		return err
	}
	destination, err := required(call, "out")
	if err != nil {
		return err
	}
	track, err := readSequence(name)
	if err != nil {
		return err
	}
	base, err := readPlan(track.Scene)
	if err != nil {
		return err
	}
	if base.Time+track.Duration > 86400 {
		return fmt.Errorf("island: sequence clock exceeds one day")
	}
	if call.DryRun() {
		for index := 0; index < track.Frames; index++ {
			if _, err := track.sample(base, index); err != nil {
				return err
			}
		}
		_, err := fmt.Fprintf(call.Out(), "%d frames at %d fps, %.3f scene seconds, %d x %d -> %s\n", track.Frames, track.FPS, track.Duration, base.Film.Width, base.Film.Height, destination)
		return err
	}
	if err := os.Mkdir(destination, 0o755); err != nil {
		return err
	}
	started := time.Now()
	status := sequenceRecord{Started: started.UTC(), InputSceneSHA256: base.hash, InputPlanSHA256: track.hash, FPS: track.FPS, ExpectedFrames: track.Frames, SceneDuration: track.Duration}
	for name, raw := range map[string][]byte{"input-scene.yaml": base.raw, "input-sequence.yaml": track.raw, "scene.schema.json": sceneSchema, "sequence.schema.json": sequenceSchema} {
		if err := os.WriteFile(filepath.Join(destination, name), raw, 0o644); err != nil {
			return err
		}
	}
	if err := writeSequenceStatus(destination, status); err != nil {
		return err
	}
	session, err := openScene(ctx, base)
	if err != nil {
		return err
	}
	defer func() { failure = errors.Join(failure, session.Close()) }()
	status.BinarySHA256 = session.binaryHash
	status.Preparation = time.Since(started)
	// Check sampled camera positions before spending GPU time. This does not
	// prove clearance between samples or account for authored tree obstacles.
	for index := 0; index < track.Frames; index++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		frame, err := track.sample(base, index)
		if err != nil {
			return err
		}
		if err := session.checkCamera(frame); err != nil {
			return fmt.Errorf("frame %d: %w", index, err)
		}
	}
	for index := 0; index < track.Frames; index++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		frame, err := track.sample(base, index)
		if err != nil {
			return err
		}
		directory := fmt.Sprintf("frame-%06d", index)
		receipt, err := session.draw(ctx, frame, filepath.Join(destination, directory), time.Now())
		if err != nil {
			return fmt.Errorf("frame %d: %w", index, err)
		}
		status.Frames = append(status.Frames, sequenceFrame{Directory: directory, SceneSeconds: frame.Time, SettingsSHA256: receipt.SettingsSHA256, ImageSHA256: receipt.ImageSHA256, Elapsed: receipt.Elapsed})
		status.Elapsed = time.Since(started)
		if err := writeSequenceStatus(destination, status); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(call.Out(), "frame %d/%d: t=%.3fs, height=%.3fm, wide=%.3fdeg, %s\n", index+1, track.Frames, frame.Time, frame.Camera.At.Height, frame.Film.Wide, receipt.Elapsed); err != nil {
			return err
		}
	}
	if err := session.Close(); err != nil {
		return err
	}
	// A root poster keeps this sequence usable by the still gallery. Its
	// settings and record are exact copies of the first effective frame.
	for _, name := range []string{"island.png", "scene.yaml", "render.json"} {
		raw, err := os.ReadFile(filepath.Join(destination, "frame-000000", name))
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(destination, name), raw, 0o644); err != nil {
			return err
		}
	}
	resolved := *track
	resolved.Scene = "scene.yaml"
	raw, err := json.MarshalIndent(resolved, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(destination, "sequence.yaml"), append(raw, '\n'), 0o644); err != nil {
		return err
	}
	status.Complete = true
	status.Elapsed = time.Since(started)
	return writeSequenceStatus(destination, status)
}

// writeSequenceStatus replaces the progress record through a temporary file.
func writeSequenceStatus(directory string, status sequenceRecord) error {
	raw, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return err
	}
	temporary := filepath.Join(directory, "sequence.json.tmp")
	if err := os.WriteFile(temporary, append(raw, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(temporary, filepath.Join(directory, "sequence.json"))
}

// checkCamera checks the sampled pose and near clearance against source height
// and ocean level. It does not prove clearance between samples or past objects.
func (session *sceneSession) checkCamera(plan *plan) error {
	at, err := session.sphere.Position(plan.Camera.At.geographic())
	if err != nil {
		return err
	}
	look, err := session.sphere.Position(plan.Camera.Look.geographic())
	if err != nil {
		return err
	}
	basis, err := session.sphere.Tangent(at)
	if err != nil {
		return err
	}
	if _, err := maths.View(at, look, basis.Y); err != nil {
		return err
	}
	height, has, err := session.layers.Height(plan.Camera.At.Longitude, plan.Camera.At.Latitude)
	if err != nil {
		return err
	}
	if plan.Camera.At.Height <= plan.Ocean.Height+plan.Camera.Near || (has && plan.Camera.At.Height <= height+plan.Camera.Near) {
		return fmt.Errorf("island: camera height %.6f m does not clear source height %.6f m (present=%t), ocean %.6f m and near %.6f m", plan.Camera.At.Height, height, has, plan.Ocean.Height, plan.Camera.Near)
	}
	return nil
}
