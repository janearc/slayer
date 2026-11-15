package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"time"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/puzzlemouth/verb"
	"github.com/janearc/slayer/hillside"
	compositor "github.com/janearc/slayer/improv/moonlake/omnipositor"
	"github.com/janearc/slayer/interfaces/file"
	"github.com/janearc/slayer/interfaces/movie"
	"github.com/janearc/slayer/raw"
)

// scenePage documents the generic set path separately from Moonlake stretch.
const scenePage = `# scene

scene PLAN --out NEW-DIRECTORY [--reference] [--raw] [--movie]

Reads a checked compositing YAML plan pointing to an ordinary Slayer set job.
Draws full-resolution scenery plates and smaller 3D foreground transport,
then combines scattered light plus RGB transmission times the plate. This
uses the scene's actual geometry, beams, mist and camera; there is no lake
or shore requirement. --reference draws the complete scene at every frame.

The plan explicitly chooses transport-scale (1, 2 or 4), transport-rays
(per axis), and plate-every. Reusing a plate freezes geometry and surface
lighting, including volume shadows on surfaces. Camera moves and zooms
require plate-every: 1. Reduced transport can miss thin foreground objects
or beams. This is an artistic preview mode, not an exact or scientific mode.

Writes numbered PNGs and a JSON receipt in a new directory. --raw also keeps
linear PFM frames. --movie encodes silent H.264 at the job's rate. Sound,
titles, stocks, open shutters and lens rigs are refused; add the soundtrack
and titles in the edit. No resume or replacement of existing output is offered.
Use Game's GPU lane. --dry-run validates the plan and job without rendering.
`

// declareScene exposes the compositor's set adapter through Puzzlemouth.
func declareScene(tool *pzm.Tool) {
	facts := verb.NewFacts(verb.WritesTo("a new directory of scene frames and their receipt"), verb.LeavesAlone, verb.ThisMachine, verb.NoOne, verb.FullGPU)
	tool.Verb("scene", verb.NoShort, "composite a GPU set job", scenePage, "Example_scene", facts, sceneCommand,
		pzm.Value("out", verb.NoShort, "new output directory"),
		pzm.Switch("reference", verb.NoShort, "draw the complete scene at each frame"),
		pzm.Switch("raw", verb.NoShort, "also retain linear-light PFM frames"),
		pzm.Switch("movie", verb.NoShort, "also encode a silent H.264 MP4"))
}

// sceneReceipt binds completed output to the actual executable and input
// job. Input hashes are the job's declared reads, not a recursive shelf audit.
type sceneReceipt struct {
	Plan         compositor.ScenePlan     `json:"plan"`
	Reference    bool                     `json:"reference"`
	Build        string                   `json:"build"`
	BinarySHA256 string                   `json:"binary_sha256"`
	JobKey       string                   `json:"job_key"`
	Inputs       map[string]string        `json:"inputs"`
	Frames       []compositor.SceneTiming `json:"frames"`
	Hashes       map[string]string        `json:"png_sha256"`
	Seconds      float64                  `json:"seconds"`
}

// sceneCommand owns publication. A fresh directory prevents accidental
// mixing with an earlier producer or an interrupted comparison.
func sceneCommand(call pzm.Call) error {
	if len(call.Args()) != 1 || call.Value("out") == "" {
		return fmt.Errorf("scene: give PLAN and --out NEW-DIRECTORY")
	}
	plan, wanted, err := compositor.ReadScenePlan(call.Args()[0])
	if err != nil {
		return err
	}
	if call.DryRun() {
		_, err := fmt.Fprintf(call.Out(), "would draw frames [%d,%d), transport scale %d, plate every %d; reference=%t\n", plan.From, plan.To, plan.Scale, plan.PlateEvery, call.On("reference"))
		return err
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	binary, err := os.Executable()
	if err != nil {
		return err
	}
	content, err := os.ReadFile(binary)
	if err != nil {
		return err
	}
	hash := sha256.Sum256(content)
	receipt := sceneReceipt{Plan: plan, Reference: call.On("reference"), Build: sceneBuild(), BinarySHA256: hex.EncodeToString(hash[:]), JobKey: wanted.Key(), Inputs: wanted.Inputs(), Hashes: map[string]string{}}
	out := call.Value("out")
	if err := os.Mkdir(out, 0755); err != nil {
		return err
	}
	planBytes, err := os.ReadFile(call.Args()[0])
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(out, "plan.txt"), planBytes, 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(out, "job.json"), wanted.Canonical(), 0644); err != nil {
		return err
	}
	var writer *movie.Writer
	if call.On("movie") {
		rate, ok := wanted.Clock().Rate()
		if !ok {
			return fmt.Errorf("scene: a movie needs a film job with fps")
		}
		width, _ := wanted.Film().Width()
		height, _ := wanted.Film().Height()
		settings, err := movie.NewSettings(movie.Request{Path: filepath.Join(out, "animation.mp4"), Width: width, Height: height, Rate: rate, Frames: plan.To - plan.From, Pad: true})
		if err != nil {
			return err
		}
		writer, err = movie.New(ctx, settings)
		if err != nil {
			return err
		}
		defer writer.Abort()
	}
	started := time.Now()
	err = compositor.RenderScene(ctx, plan, wanted, hillside.Stamped(build, built, who, sessions), call.On("reference"), func(result compositor.SceneResult) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		name := fmt.Sprintf("frame-%06d.png", result.Timing.Index)
		digest, err := file.SequencePNGHash(filepath.Join(out, name), result.Image)
		if err != nil {
			return err
		}
		receipt.Hashes[name] = hex.EncodeToString(digest[:])
		if call.On("raw") {
			if err := raw.Write(filepath.Join(out, fmt.Sprintf("frame-%06d.pfm", result.Timing.Index)), result.Linear); err != nil {
				return err
			}
		}
		if writer != nil {
			if err := writer.Frame(result.Timing.Index-plan.From, result.Image); err != nil {
				return err
			}
		}
		receipt.Frames = append(receipt.Frames, result.Timing)
		_, err = fmt.Fprintf(call.Out(), "frame %d: plate %.3fs transport %.3fs composite %.3fs develop %.3fs\n", result.Timing.Index, result.Timing.PlateSeconds, result.Timing.TransportSeconds, result.Timing.CompositeSeconds, result.Timing.DevelopSeconds)
		return err
	})
	if err != nil {
		return err
	}
	if writer != nil {
		if err := writer.Close(); err != nil {
			return err
		}
	}
	receipt.Seconds = time.Since(started).Seconds()
	data, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(out, "render.json"), data, 0644)
}

// sceneBuild retains the compiler's VCS revision when the Game target did
// not supply a linker stamp. The executable hash remains the byte identity.
func sceneBuild() string {
	if build != "" {
		return build
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unavailable"
	}
	revision, modified := "unavailable", false
	for _, setting := range info.Settings {
		if setting.Key == "vcs.revision" {
			revision = setting.Value
		}
		if setting.Key == "vcs.modified" {
			modified = setting.Value == "true"
		}
	}
	if modified {
		revision += "+modified"
	}
	return revision
}
