package hillside

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"io"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/puzzlemouth/verb"
	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/camera"
	"github.com/janearc/slayer/gpu"
	"github.com/janearc/slayer/interfaces/file"
	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
	yaml "gopkg.in/yaml.v3"
)

// build, built, who and sessions say which build is running; Run sets them
// from its Stamp.
var build, built, who, sessions string

// Stamp says which build is running: its version, when it was built, by
// whom, and from which sessions. A Stamp is made by Stamped.
type Stamp struct {
	build, built, who, sessions string
}

// Stamped returns the Stamp for a build with the given version, build time,
// builder and sessions.
func Stamped(build, built, who, sessions string) Stamp {
	return Stamp{build: build, built: built, who: who, sessions: sessions}
}

//go:embed sequence.schema.json
var sequenceSchemaRaw []byte
var sequenceSchema, sequenceSchemaError = yamlfile.NewSchema(
	"https://github.com/janearc/slayer/schemas/sequence-v1.json", sequenceSchemaRaw)

type sequencePlan struct {
	scene, output       string
	fps, frames         int
	start               time.Duration
	knee, top, bloom    float64
	hash                string
	camera              *camera.Path
	zoom                *camera.Zoom
	sceneSpan           time.Duration
	smoke               int
	width, height, rays int
	// lifted is the plan as a job, which job.Lift made of it
	lifted job.Job
}
type sequenceQuantity struct {
	Value float64
	Unit  string
}
type sequenceInput struct {
	Scene, Output   string
	FPS             int
	Start, Duration sequenceQuantity
	SceneDuration   *sequenceQuantity `yaml:"scene-duration"`
	Print           struct{ Knee, Top, Bloom float64 }
	Camera          yaml.Node `json:"camera"`
	Smoke           int       `json:"smoke"`
	Film            struct {
		Width  int `json:"width"`
		Height int `json:"height"`
		Rays   int `json:"rays"`
	} `json:"film"`
}

// readSequence resolves checked paths relative to the plan, not process cwd.
func readSequence(path string) (*sequencePlan, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(absolute)
	if err != nil {
		return nil, err
	}
	if sequenceSchemaError != nil {
		return nil, sequenceSchemaError
	}
	node, err := sequenceSchema.Read(path, raw)
	if err != nil {
		return nil, err
	}
	var in sequenceInput
	if err = node.Decode(&in); err != nil {
		return nil, err
	}
	lifted, err := job.Lift(in.lifted(path, absolute))
	if err != nil {
		return nil, err
	}
	lens, _ := lifted.Camera()
	cameraPath, _ := lens.Path()
	plan := in.plan(absolute, raw, cameraPath)
	plan.zoom, _ = lens.Zoom()
	plan.lifted = lifted
	return plan, nil
}

// lifted is the plan's values for job.Lift, which decodes the camera, so
// a sequence plan and a job have one camera decoder between them.
func (in sequenceInput) lifted(path, absolute string) job.Plan {
	location := sequenceLocation(absolute)
	var block []byte
	if !in.Camera.IsZero() {
		// the node was decoded from YAML just now, so it encodes
		block, _ = yaml.Marshal(&in.Camera)
	}
	plan := job.Plan{File: path, Scene: location.resolve(in.Scene),
		Smoke: in.Smoke, Width: in.Film.Width, Height: in.Film.Height,
		Rays: in.Film.Rays, FPS: in.FPS, Start: in.Start.Value,
		Duration: in.Duration.Value, Knee: in.Print.Knee,
		Top: in.Print.Top, Bloom: in.Print.Bloom, Camera: block,
		Output: location.resolve(in.Output)}
	if in.SceneDuration != nil {
		plan.SceneDuration = in.SceneDuration.Value
	}
	return plan
}

// plan assembles the checked wire values without a second configuration parse.
func (in sequenceInput) plan(absolute string, raw []byte,
	cameraPath *camera.Path) *sequencePlan {
	location := sequenceLocation(absolute)
	sum := sha256.Sum256(raw)
	wanted := in.Duration.Value * float64(in.FPS)
	rounded := int(math.Round(wanted))
	frames := max(sequenceMinimumFrames, rounded)
	plan := &sequencePlan{
		scene:  location.resolve(in.Scene),
		output: location.resolve(in.Output), fps: in.FPS, frames: frames,
		start: time.Duration(in.Start.Value * float64(time.Second)),
		knee:  in.Print.Knee, top: in.Print.Top, bloom: in.Print.Bloom,
		hash: hex.EncodeToString(sum[:]), camera: cameraPath,
		smoke: in.Smoke, width: in.Film.Width,
		height: in.Film.Height, rays: in.Film.Rays}
	if in.SceneDuration != nil {
		nanoseconds := in.SceneDuration.Value * float64(time.Second)
		plan.sceneSpan = time.Duration(nanoseconds)
	}
	return plan
}

// sequenceLocation resolves paths against the plan that declared them.
type sequenceLocation string

// resolve preserves an absent scene when a numbered smoke was selected.
func (location sequenceLocation) resolve(value string) string {
	if value == "" {
		return ""
	}
	if filepath.IsAbs(value) {
		return filepath.Clean(value)
	}
	directory := filepath.Dir(string(location))
	return filepath.Join(directory, value)
}

// sequenceTool declares the new door without extending legacy hillside flags.
func sequenceTool() *pzm.Tool {
	tool := pzm.New("hillside", "draw procedural scenery and frame sequences",
		pzm.Build(build), pzm.Built(built), pzm.Who(who), pzm.Sessions(sessions))
	facts := verb.NewFacts(verb.WritesTo("the sequence output directory"),
		verb.Changes, verb.ThisMachine, verb.NoOne, verb.FullGPU)
	tool.Verb("sequence", verb.NoShort, "render a checked sequence of timed frames",
		"# sequence\n\nreads a YAML plan, retains scenery, and writes timed PNG frames.\nresume requires matching scene, plan and renderer stamps.\n",
		"Example_sequence", facts, sequence,
		pzm.Value("plan", verb.NoShort, "the checked sequence YAML"),
		pzm.Value("output", verb.NoShort, "override the plan's output directory"),
		pzm.Switch("resume", verb.NoShort, "resume matching completed frames"))
	return tool
}

// sequenceRun delegates every new argument and flag to Puzzlemouth, and
// says after what the tree cache did, when it was on.
func sequenceRun(arguments []string, out, wrong io.Writer) int {
	code := sequenceTool().Run(arguments, out, wrong)
	reportTrees(out)
	return code
}

// sequence validates the plan and scene before it creates output.
func sequence(call pzm.Call) error {
	if !call.Given("plan") || len(call.Args()) != 0 {
		return errors.New("sequence wants --plan and no positional arguments")
	}
	plan, err := readSequence(call.Value("plan"))
	if err != nil {
		return err
	}
	if call.Given("output") {
		if call.Value("output") == "" {
			return errors.New("sequence output needs a directory")
		}
		plan.output, err = filepath.Abs(call.Value("output"))
		if err != nil {
			return err
		}
	}
	if plan.smoke > sequenceAbsent {
		return sequenceSmoke(call, *plan)
	}
	return sequenceScenery(call, *plan)
}

// sequenceScenery validates the existing terrain-scene route.
func sequenceScenery(call pzm.Call, plan sequencePlan) error {
	raw, err := os.ReadFile(plan.scene)
	if err != nil {
		return err
	}
	checked, err := set.Read(plan.scene, raw)
	if err != nil {
		return err
	}
	if _, has := checked.Ground(); !has {
		return errNoGround
	}
	if call.DryRun() {
		return sequenceDryRun(call.Out(), plan)
	}
	return drawSequence(plan, raw, call.On("resume"), call.Out())
}

// sequenceDryRun reports checked timing without preparing or drawing scenery.
func sequenceDryRun(out io.Writer, plan sequencePlan) error {
	if plan.sceneSpan > 0 {
		_, err := fmt.Fprintf(out,
			"scene span: %s in %.9gs of playback\n",
			plan.sceneSpan, float64(plan.frames)/float64(plan.fps))
		if err != nil {
			return err
		}
	}
	if plan.camera != nil {
		_, err := fmt.Fprintf(out,
			"camera path: %s, then holds its endpoint\n",
			plan.camera.Duration())
		if err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(out, "would render %d frames at %d fps\n",
		plan.frames, plan.fps)
	return err
}

type sequenceEntry struct {
	SHA256        string
	AtNanoseconds int64
	Seconds       float64
}
type sequenceState struct {
	Plan, Scene, Build, Built    string
	Executable, Sources, Terrain string
	Width, Height, FPS, Frames   int
	Complete                     map[int]sequenceEntry
	// Key is the key of the job a film was drawn from. A sequence-v1
	// plan sets none, and its manifest is written without it, byte for
	// byte as before.
	Key string `json:",omitempty"`
}

// errNoGround is the refusal of a film of a set with no ground.
var errNoGround = errors.New("sequence currently requires ground scenery")

// keyed refuses a manifest found on a resume whose job key is not the
// key of the job resuming it: a film drawn from one job is never
// finished by another. A plan's manifest and a plan both say none.
func keyed(found sequenceState, key string) error {
	if found.Key == key {
		return nil
	}
	return fmt.Errorf("sequence output was drawn from the job of key "+
		"%q, and this job's key is %q; draw it to a new directory",
		found.Key, key)
}

// sequencePreparation names the command, staging and executable boundaries.
// Production supplies their real implementations; failure checks can exercise
// propagation and ownership without replacing rendering or hashing arithmetic.
type sequencePreparation struct {
	parse      func([]string, io.Writer) (asked, error)
	stage      func(asked) (*thing.World, tally, asked, error)
	executable func() (string, error)
}

// drawSequence builds scenery once and saves each complete frame atomically.
func drawSequence(plan sequencePlan, raw []byte, resume bool, out io.Writer) error {
	return drawPreparedSequence(plan, raw, resume, out,
		sequencePreparation{parse: parse, stage: staged, executable: os.Executable})
}

// drawPreparedSequence keeps setup failure handling around the retained scene.
func drawPreparedSequence(plan sequencePlan, raw []byte, resume bool, out io.Writer,
	prepare sequencePreparation) error {
	given, err := prepare.parse([]string{"-set", plan.scene}, io.Discard)
	if err != nil {
		return err
	}
	_, _, given, err = prepare.stage(given)
	if err != nil {
		return err
	}
	if given.live != nil {
		defer given.live.closeBeds()
	}
	if err = retainable(given); err != nil {
		return err
	}
	current, err := os.ReadFile(plan.scene)
	if err != nil {
		return err
	}
	if string(current) != string(raw) {
		return errors.New("scene changed while preparing the sequence")
	}
	executable, err := prepare.executable()
	if err != nil {
		return err
	}
	expected, err := sequenceInputs(plan, raw, given, executable)
	if err != nil {
		return err
	}
	state, err := sequenceFolder(plan.output, expected, resume)
	if err != nil {
		return err
	}
	if given.kept, err = filmKeeper(given); err != nil {
		return err
	}
	defer given.kept.close()
	return sequencePictures(plan, state, out, func(at time.Duration) (
		*image.RGBA, gpu.Profile, tally, error) {
		return sequencePicture(plan, given, at)
	})
}

// retainable refuses a staged set a film cannot be drawn from: one seen
// through a rig's lens, and one whose lighting is not retained from frame
// to frame.
func retainable(given asked) error {
	if given.glass != nil {
		return errors.New("sequence currently uses pinhole frames")
	}
	if given.live == nil || given.live.unsupported != "" {
		return errors.New("sequence cannot retain this scene's lighting")
	}
	return nil
}

// sequenceInputs fingerprints the actual files used to prepare a sequence.
// Taking the executable path explicitly lets checks use a real changed file.
func sequenceInputs(plan sequencePlan, raw []byte, given asked,
	executable string) (sequenceState, error) {
	expected := sequenceIdentity(plan, raw, given)
	read, err := set.Read(plan.scene, raw)
	if err != nil {
		return sequenceState{}, err
	}
	ground, _ := read.Ground()
	shelf, listed, err := pinnedSource(ground.From(), ground.File(),
		filepath.Dir(plan.scene))
	if err != nil {
		return sequenceState{}, err
	}
	for _, input := range []struct {
		path string
		sum  *string
	}{
		{executable, &expected.Executable},
		{listed, &expected.Sources},
		{filepath.Join(shelf, ground.File()), &expected.Terrain},
	} {
		*input.sum, err = sequenceHash(input.path)
		if err != nil {
			return sequenceState{}, err
		}
	}
	return expected, nil
}

// sequencePicture advances retained scenery and develops one timed picture.
func sequencePicture(plan sequencePlan, given asked, at time.Duration) (
	*image.RGBA, gpu.Profile, tally, error) {
	light, err := sequenceLight(plan, given, at)
	if err != nil {
		return nil, light.profile, light.made, err
	}
	tone := colour.Tone{Knee: plan.knee, Top: plan.top, Bloom: plan.bloom}
	picture := tone.Develop(light.drawn)
	if err = light.overlay(picture); err != nil {
		return nil, light.profile, light.made, err
	}
	return picture, light.profile, light.made, nil
}

// sequenceLight is the first half of sequencePicture: the retained scene
// at a time, seen through the plan's camera, drawn and not developed.
// Where it fails, what it holds is what was made before the failure.
func sequenceLight(plan sequencePlan, given asked,
	at time.Duration) (drawnLight, error) {
	world, made, err := given.live.frameAt(at)
	if err != nil {
		return drawnLight{}, err
	}
	world, err = sceneClockOceanWorldAt(world, at)
	if err != nil {
		return drawnLight{made: made}, err
	}
	given.at = at
	if err = plan.exposing(&given, at); err != nil {
		return drawnLight{made: made}, err
	}
	given.footprint(world)
	drawn, profile, err := given.kept.picture(world, given, made.across)
	return drawnLight{drawn: drawn, profile: profile, made: made,
		world: world, given: given}, err
}

// sequencePictures owns frame files, verification and atomic progress.
// draw is called only for absent, altered or incorrectly timed frames.
func sequencePictures(plan sequencePlan, state *sequenceState, out io.Writer,
	draw func(time.Duration) (*image.RGBA, gpu.Profile, tally, error)) error {
	for index := 0; index < plan.frames; index++ {
		when := plan.frameTime(index)
		target := filepath.Join(plan.output, fmt.Sprintf("frame-%06d.png", index))
		if entry, ok := state.Complete[index]; ok {
			content, err := os.ReadFile(target)
			if err == nil {
				hash := sha256.Sum256(content)
				if entry.SHA256 == hex.EncodeToString(hash[:]) && entry.AtNanoseconds == int64(when) {
					continue
				}
			}
			delete(state.Complete, index)
		}
		started := time.Now()
		picture, profile, made, err := draw(when)
		if err != nil {
			return err
		}
		temporary := target + ".part"
		hash, err := file.SequencePNGHash(temporary, picture)
		if err != nil {
			return err
		}
		if err = os.Rename(temporary, target); err != nil {
			return err
		}
		spent := time.Since(started).Seconds()
		state.Complete[index] = sequenceEntry{SHA256: hex.EncodeToString(hash[:]), AtNanoseconds: int64(when), Seconds: spent}
		if err = writeSequenceState(plan.output, state); err != nil {
			return err
		}
		if _, err = fmt.Fprintf(out, "frame %d/%d at %.9fs: %.3fs (GPU %.3fs), %d sparks, %d lamps\n",
			index+1, plan.frames, when.Seconds(), spent, (profile.TraceGPU + profile.LightingGPU).Seconds(), made.sparks, made.lamps); err != nil {
			return err
		}
		if err = reportFrameStages(out, profile); err != nil {
			return err
		}
	}
	return nil
}

// reportFrameStages says where a gpu frame's time went, and for a frame
// of the film's retained drawer what it sent to the device, so the
// saving is visible frame by frame. A frame not drawn on a device says
// nothing.
func reportFrameStages(out io.Writer, profile gpu.Profile) error {
	if profile.Device == "" {
		return nil
	}
	_, err := fmt.Fprintf(out, "  encode %.3fs, bake %.3fs, lighting "+
		"%.3fs, dispatch %.3fs, finish %.3fs, total %.3fs\n",
		profile.Encode.Seconds(), profile.BakeWall.Seconds(),
		profile.LightingWall.Seconds(), profile.DispatchWall.Seconds(),
		profile.Finish.Seconds(), profile.Total.Seconds())
	if err != nil || profile.Retained.Held == 0 {
		return err
	}
	_, err = fmt.Fprintf(out, "  retained: sent %d of %d inputs held, "+
		"%d bytes\n", profile.Retained.Sent, profile.Retained.Held,
		profile.Retained.SentBytes)
	return err
}

// sequenceFolder refuses mismatched resumes and nonempty unowned outputs.
func sequenceFolder(dir string, expected sequenceState, resume bool) (*sequenceState, error) {
	path := filepath.Join(dir, "sequence.json")
	raw, err := os.ReadFile(path)
	if err == nil {
		if !resume {
			return nil, errors.New("sequence output exists; use --resume")
		}
		var state sequenceState
		if err = json.Unmarshal(raw, &state); err != nil {
			return nil, err
		}
		if state.Executable != expected.Executable || state.Sources != expected.Sources || state.Terrain != expected.Terrain ||
			state.Plan != expected.Plan || state.Scene != expected.Scene || state.Build != expected.Build ||
			state.Built != expected.Built || state.Width != expected.Width || state.Height != expected.Height ||
			state.FPS != expected.FPS || state.Frames != expected.Frames || state.Complete == nil {
			return nil, errors.New("sequence output does not match this plan, scene or renderer")
		}
		if err = keyed(state, expected.Key); err != nil {
			return nil, err
		}
		return &state, nil
	}
	if !os.IsNotExist(err) {
		return nil, err
	}
	if entries, err := os.ReadDir(dir); err == nil && len(entries) != 0 {
		return nil, errors.New("sequence output is not empty and has no matching manifest")
	}
	if err = os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	if err = writeSequenceState(dir, &expected); err != nil {
		return nil, err
	}
	return &expected, nil
}

// writeSequenceState commits progress after a successfully renamed PNG.
func writeSequenceState(dir string, state *sequenceState) error {
	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	temporary := filepath.Join(dir, "sequence.json.part")
	if err = os.WriteFile(temporary, append(raw, '\n'), 0644); err != nil {
		return err
	}
	return os.Rename(temporary, filepath.Join(dir, "sequence.json"))
}

// sequenceHash streams large dependencies, including the unpacked DEM actually
// read by the renderer rather than relying only on its archive's pin.
func sequenceHash(path string) (string, error) {
	input, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer input.Close()
	hash := sha256.New()
	if _, err = io.Copy(hash, input); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
