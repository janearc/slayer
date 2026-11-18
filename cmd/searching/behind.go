package main

import (
	"crypto/sha256"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/slayer/improv/searching"
	"github.com/janearc/slayer/interfaces/movie"
)

// behindPage plays the original movie; the panel has no independent clock.
//
//go:embed behind.html
var behindPage []byte

// hashFile fingerprints a finished movie without holding its bytes in memory.
func hashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// behindFrame is explanatory metadata on one actual movie frame.
type behindFrame struct {
	Song  float64 `json:"song"`
	Bar   int     `json:"bar"`
	Beat  int     `json:"beat"`
	Onset bool    `json:"onset"`
	Says  string  `json:"says"`
}

// behind prepares a companion for an existing complete take, refusing a
// changed recipe or score. It does not rerender or re-encode the picture.
func behind(call pzm.Call) error {
	if len(call.Args()) != 1 {
		return fmt.Errorf("one completed render directory is required")
	}
	return publishBehind(call.Args()[0], call.DryRun())
}

// publishBehind binds the explanatory panel to the actual movie and its
// frozen inputs. The generator revision is separate from the movie producer.
func publishBehind(into string, dry bool) error {
	raw, err := os.ReadFile(filepath.Join(into, "render.json"))
	if err != nil {
		return err
	}
	var receipt struct {
		From     int    `json:"source_start_frame"`
		Frames   int    `json:"frames"`
		FPS      int    `json:"fps"`
		Plan     string `json:"plan_sha256"`
		Score    string `json:"score_sha256"`
		Producer string `json:"source_commit"`
	}
	if err := json.Unmarshal(raw, &receipt); err != nil {
		return err
	}
	if receipt.Frames < 1 || receipt.FPS < 1 || receipt.From < 0 {
		return fmt.Errorf("invalid completed take")
	}
	for name, want := range map[string]string{"shoot.yaml": receipt.Plan, "score.yaml": receipt.Score} {
		bytes, err := os.ReadFile(filepath.Join(into, name))
		if err != nil {
			return err
		}
		if fmt.Sprintf("%x", sha256.Sum256(bytes)) != want {
			return fmt.Errorf("%s changed since rendering", name)
		}
	}
	edit, err := searching.Read(filepath.Join(into, "shoot.yaml"))
	if err != nil {
		return err
	}
	// Read resolves the score named by the plan. Verify that it is still
	// the frozen score before using it, even if the plan names another path.
	score, err := os.ReadFile(edit.Plan.Score)
	if err != nil {
		return err
	}
	if fmt.Sprintf("%x", sha256.Sum256(score)) != receipt.Score {
		return fmt.Errorf("plan's score changed")
	}
	facts, err := movie.Probe(filepath.Join(into, "animation.mp4"))
	if err != nil {
		return err
	}
	if facts.Frames() != receipt.Frames || facts.Rate() != float64(receipt.FPS) {
		return fmt.Errorf("movie differs from receipt")
	}
	if dry {
		return nil
	}
	frames := make([]behindFrame, receipt.Frames)
	for index := range frames {
		frames[index] = explainFrame(edit, float64(receipt.From+index)/float64(receipt.FPS))
	}
	movieHash, err := hashFile(filepath.Join(into, "animation.mp4"))
	if err != nil {
		return err
	}
	data := map[string]any{"fps": receipt.FPS, "frames": frames, "movie_sha256": movieHash,
		"plan_sha256": receipt.Plan, "score_sha256": receipt.Score, "movie_producer": receipt.Producer,
		"explanation_generator": build, "grid_status": edit.Grid.Score.Status}
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(into, "behind-the-scenes.json"), append(bytes, '\n'), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(into, "behind-the-scenes.js"), append(append([]byte("const heard = "), bytes...), ';', '\n'), 0644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(into, "behind-the-scenes.html"), behindPage, 0644)
}

// explainFrame takes cue timing from the same checked edit as the picture.
// These sentences describe implemented studies, never future screenplay shots.
func explainFrame(edit *searching.Edit, at float64) behindFrame {
	beat := edit.Grid.BeatAt(at)
	index := int(math.Floor(beat))
	frame := behindFrame{Song: at, Bar: index/4 + 1, Beat: index%4 + 1}
	if index >= 0 && index < len(edit.Grid.Score.Beats.Measured) {
		frame.Onset = edit.Grid.Score.Beats.Measured[index] == 1
	}
	if beat < 0 {
		frame.Bar, frame.Beat = 0, 0
	}
	if shot := edit.ShotAt(at); shot != nil {
		frame.Says = shot.Caption
		if shot.Scene == "slate" && shot.Caption != "heard." {
			frame.Says = "This shot is still missing from the rough cut: " + shot.Caption
		}
	} else if edit.Plan.CatalogueStudy != nil && (!edit.Plan.FullCut || at >= edit.CatalogueStart && at < edit.CatalogueEnd) {
		frame.Says = "Gaia supplies the star positions. The camera pulls away from the cyan Sun marker; the dots are enlarged so we can still see them."
	} else if edit.FlowAt(at) {
		frame.Says = "These filaments are authored geometry. Light travels along their lengths; this is not a measured cosmic flow field."
		if edit.Plan.FlowStudy.Travel != nil {
			frame.Says = "The camera retreats from a fictional cyan home marker. Fixed lights provide parallax; the geometry is authored, not a measured Laniakea map."
		}

		if at >= edit.FlowFold {
			frame.Says = "The inner strands fold while the outer landmarks hold their places. The opening is an asymmetric dark surface."
		}
		if at >= edit.FlowAnswer {
			frame.Says = "One packet travels back toward the cyan marker. That reversal is the answer. Its apparent speed is a storytelling choice."
		}
	} else {
		state := edit.At(at)
		frame.Says = "The shot and its motion follow the song's retained beat grid. The sky runs on a separate, accelerated clock."
		if state.Chapter == "lock" {
			frame.Says = "Both observatories settle on the authored shared direction at this take's lock cue."
		}
		if state.Chapter == "radio response" {
			frame.Says = "The array follows the pointing cue. This first proof still turns whole assemblies; articulated mounts are unfinished."
		}
		if state.Site == "blazar" {
			frame.Says = "This older proof changes authored emission components. These are illustrations, not measured spectral images."
		}
	}
	return frame
}
