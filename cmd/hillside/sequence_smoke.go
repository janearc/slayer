package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"io"
	"os"
	"strings"
	"time"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/slayer/gpu"
	"github.com/janearc/slayer/maths"
	milestone "github.com/janearc/slayer/scene/smoke"
)

// sequenceSmoke validates a milestone and its declared input files.
func sequenceSmoke(call pzm.Call, plan sequencePlan) error {
	scene, exists := milestone.Get(plan.smoke)
	if !exists {
		return fmt.Errorf("sequence: no smoke %d", plan.smoke)
	}
	inputs, err := scene.RequireInputs()
	if err != nil {
		return err
	}
	if call.DryRun() {
		return sequenceDryRun(call.Out(), plan)
	}
	return drawSmokeSequence(plan, scene, inputs,
		call.On("resume"), call.Out())
}

// sequenceIdentity collects the image identity shared by both routes.
func sequenceIdentity(plan sequencePlan, raw []byte,
	given asked) sequenceState {
	sum := sha256.Sum256(raw)
	return sequenceState{
		Plan: plan.hash, Scene: hex.EncodeToString(sum[:]),
		Build: build, Built: built,
		Width: given.width, Height: given.height,
		FPS: plan.fps, Frames: plan.frames,
		Complete: map[int]sequenceEntry{},
	}
}

// drawSmokeSequence retains one actual smoke world across camera frames.
func drawSmokeSequence(plan sequencePlan, scene milestone.Smoke,
	inputs []string, resume bool, out io.Writer) error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	assets, err := sequenceAssetHash(inputs)
	if err != nil {
		return err
	}
	world, eye := scene.Scene()
	given := asked{width: plan.width, height: plan.height,
		samples: plan.rays,
		wide:    scene.Wide / maths.Degree, eye: &eye,
		live: &liveScene{base: *world}}
	identity := []byte(fmt.Sprintf("smoke %d", scene.Number))
	expected := sequenceIdentity(plan, identity, given)
	expected.Executable, err = sequenceHash(executable)
	if err != nil {
		return err
	}
	expected.Sources, err = sequenceAssetHash(inputs)
	if err != nil {
		return err
	}
	if expected.Sources != assets {
		return fmt.Errorf("sequence: smoke inputs changed " +
			"during preparation")
	}
	state, err := sequenceFolder(plan.output, expected, resume)
	if err != nil {
		return err
	}
	source := sequenceFrames{plan: plan, given: given}
	return sequencePictures(plan, state, out, source.draw)
}

// sequenceFrames owns the retained scene and camera plan during drawing.
type sequenceFrames struct {
	plan  sequencePlan
	given asked
}

// draw samples the shared frame pipeline at the requested absolute time.
func (source sequenceFrames) draw(at time.Duration) (
	*image.RGBA, gpu.Profile, tally, error) {
	return sequencePicture(source.plan, source.given, at)
}

// sequenceAssetHash includes the actual data files, not only their catalogue
// names. Missing files stop a resume just as a changed executable does.
func sequenceAssetHash(paths []string) (string, error) {
	var key strings.Builder
	for _, path := range paths {
		sum, err := sequenceHash(path)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&key, "%d:%s%s", len(path), path, sum)
	}
	raw := []byte(key.String())
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}
