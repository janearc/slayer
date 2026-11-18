package omnipositor

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/render/frame"
)

// sceneJobBase is a film job that a scene plan accepts; the tests add one
// line at a time to it.
const sceneJobBase = `$schema: https://github.com/janearc/slayer/schemas/job-v1.json
scene: {set: summit.yaml}
film: {width: 320, height: 180, rays: 1}
renderer: gpu
fps: 6
duration: {value: 2, unit: s}
outputs: {frames: unused}
`

// scenePlanText is a plan that is valid against sceneJobBase.
const scenePlanText = `version: 1
job: job.yaml
from: 0
to: 12
transport-scale: 2
transport-rays: 1
plate-every: 12
`

// readPlan writes the job and plan texts into a new directory and reads the
// plan from it.
func readPlan(t *testing.T, jobText, planText string) error {
	t.Helper()
	dir := t.TempDir()
	for name, text := range map[string]string{"job.yaml": jobText, "plan.yaml": planText, "theme.yaml": "theme"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	_, _, err := ReadScenePlan(filepath.Join(dir, "plan.yaml"))
	return err
}

// TestReadScenePlanAcceptsTheBase checks that the base job and plan read, so
// that the refusals below are caused by their one changed line.
func TestReadScenePlanAcceptsTheBase(t *testing.T) {
	if err := readPlan(t, sceneJobBase, scenePlanText); err != nil {
		t.Fatal(err)
	}
}

// TestReadScenePlanMissingFiles checks the errors for a plan that is not
// there and for a plan whose job is not there.
func TestReadScenePlanMissingFiles(t *testing.T) {
	dir := t.TempDir()
	if _, _, err := ReadScenePlan(filepath.Join(dir, "none.yaml")); err == nil || !os.IsNotExist(unwrapAll(err)) {
		t.Errorf("missing plan: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "plan.yaml"), []byte(scenePlanText), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ReadScenePlan(filepath.Join(dir, "plan.yaml")); err == nil || !strings.Contains(err.Error(), "job.yaml") {
		t.Errorf("missing job: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "plan.yaml"), []byte("version: [1"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ReadScenePlan(filepath.Join(dir, "plan.yaml")); err == nil {
		t.Error("accepted a plan that is not YAML")
	}
}

// unwrapAll follows errors.Unwrap to the innermost error.
func unwrapAll(err error) error {
	for {
		inner, ok := err.(interface{ Unwrap() error })
		if !ok || inner.Unwrap() == nil {
			return err
		}
		err = inner.Unwrap()
	}
}

// TestScenePlanValidateRefusals checks that each combination the plan
// refuses is named by its message.
func TestScenePlanValidateRefusals(t *testing.T) {
	replace := func(old, new string) string { return strings.Replace(scenePlanText, old, new, 1) }
	for _, c := range []struct {
		name, job, plan, want string
	}{
		{"zero rays by schema", sceneJobBase, replace("transport-rays: 1", "transport-rays: 0"), "transport-rays"},
		{"scale 3", sceneJobBase, replace("transport-scale: 2", "transport-scale: 3"), "transport-scale"},
		{"width not divisible by scale", strings.Replace(sceneJobBase, "width: 320", "width: 321", 1), scenePlanText, "must divide by transport-scale"},
		{"empty range", sceneJobBase, replace("from: 0\nto: 12", "from: 5\nto: 5"), "invalid range"},
		{"range past the clip", sceneJobBase, replace("to: 12", "to: 13"), "invalid range"},
		{"scene that is not a set", strings.Replace(sceneJobBase, "scene: {set: summit.yaml}", "scene: {smoke: 1}", 1), scenePlanText, "use a set job"},
		{"stock", sceneJobBase + "stock: {name: dusk, seed: 7, rated: 1}\n", scenePlanText, "stocks are not supported"},
		{"titles", sceneJobBase + "titles:\n  theme: theme.yaml\n  elements:\n    name: {kind: text, words: A, ink: ink, across: 0.1, down: 0.1}\n", scenePlanText, "titles"},
		{"sound", sceneJobBase + "sound: {file: take.wav}\n", scenePlanText, "silent footage"},
		{"shutter", sceneJobBase + "shutter: {angle: {value: 180, unit: deg}, glimpses: 2}\n", scenePlanText, "closed shutter"},
		{"aperture", strings.Replace(sceneJobBase, "rays: 1}", "rays: 1, exposure: 0.5, aperture: 0.1, focus: 12}", 1), scenePlanText, "pinhole"},
		{"camera zoom with reused plates", sceneJobBase + "camera:\n  name: summit\n  zoom:\n    from: {value: 60, unit: deg}\n    to: {value: 62, unit: deg}\n    duration: {value: 2, unit: s}\n", scenePlanText, "plate-every: 1"},
	} {
		err := readPlan(t, c.job, c.plan)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v", c.name, err)
		}
	}
}

// TestPlanValidateDirect checks the validator on plan values that the schema
// would stop first, by building the plan in code.
func TestPlanValidateDirect(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "job.yaml"), []byte(sceneJobBase), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "plan.yaml"), []byte(scenePlanText), 0600); err != nil {
		t.Fatal(err)
	}
	plan, wanted, err := ReadScenePlan(filepath.Join(dir, "plan.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for name, edit := range map[string]func(p *ScenePlan){
		"version":         func(p *ScenePlan) { p.Version = 2 },
		"negative from":   func(p *ScenePlan) { p.From = -1 },
		"plate interval":  func(p *ScenePlan) { p.PlateEvery = 0 },
		"too many rays":   func(p *ScenePlan) { p.Rays = 5 },
		"no rays":         func(p *ScenePlan) { p.Rays = 0 },
		"scale 8":         func(p *ScenePlan) { p.Scale = 8 },
		"scale 0":         func(p *ScenePlan) { p.Scale = 0 },
		"reversed range":  func(p *ScenePlan) { p.From, p.To = 5, 4 },
		"rays and height": func(p *ScenePlan) { p.Rays = 4; p.Scale = 4 },
	} {
		p := plan
		edit(&p)
		err := p.Validate(wanted)
		if name == "rays and height" {
			if err != nil {
				t.Errorf("%s: %v", name, err)
			}
			continue
		}
		if err == nil || !strings.Contains(err.Error(), "invalid range, version") {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// TestCompositeTransportRefusals checks the input shapes the composite
// refuses: a missing frame, transport that does not divide the plate, and
// transport of different sizes.
func TestCompositeTransportRefusals(t *testing.T) {
	plate := frame.New(4, 2)
	scatter, transmission := frame.New(2, 1), frame.New(2, 1)
	if _, err := CompositeTransport(nil, scatter, transmission); err == nil || !strings.Contains(err.Error(), "invalid frame dimensions") {
		t.Errorf("nil plate: %v", err)
	}
	for name, c := range map[string][3]*frame.Frame{
		"transport sizes differ": {plate, scatter, frame.New(1, 1)},
		"plate not divisible":    {frame.New(5, 2), scatter, transmission},
		"unequal scale":          {frame.New(4, 4), scatter, transmission},
	} {
		if _, err := CompositeTransport(c[0], c[1], c[2]); err == nil || !strings.Contains(err.Error(), "must match and divide") {
			t.Errorf("%s: %v", name, err)
		}
	}
	bad := frame.New(2, 1)
	bad.Pix[0].Depth = -1
	if _, err := CompositeTransport(plate, bad, transmission); err == nil || !strings.Contains(err.Error(), "invalid light or depth") {
		t.Errorf("negative depth: %v", err)
	}
}

// TestValidSceneFrameShapes checks the frames the validator refuses and that
// a sky depth of positive infinity is allowed.
func TestValidSceneFrameShapes(t *testing.T) {
	sky := frame.New(1, 1)
	sky.Pix[0].Depth = float32(math.Inf(1))
	if err := validSceneFrame(sky); err != nil {
		t.Errorf("sky depth refused: %v", err)
	}
	short := frame.New(2, 2)
	short.Pix = short.Pix[:3]
	huge := &frame.Frame{W: 1 << 13, H: 1 << 13}
	for name, f := range map[string]*frame.Frame{"nil": nil, "zero width": {W: 0, H: 1}, "zero height": {W: 1, H: 0}, "too many samples": huge, "short pixels": short} {
		if err := validSceneFrame(f); err == nil || !strings.Contains(err.Error(), "invalid frame dimensions") {
			t.Errorf("%s: %v", name, err)
		}
	}
	for name, edit := range map[string]func(p *frame.Pixel){
		"infinite red":  func(p *frame.Pixel) { p.R = float32(math.Inf(1)) },
		"NaN green":     func(p *frame.Pixel) { p.G = float32(math.NaN()) },
		"infinite blue": func(p *frame.Pixel) { p.B = float32(math.Inf(-1)) },
		"NaN depth":     func(p *frame.Pixel) { p.Depth = float32(math.NaN()) },
	} {
		f := frame.New(1, 1)
		edit(&f.Pix[0])
		if err := validSceneFrame(f); err == nil || !strings.Contains(err.Error(), "invalid light or depth") {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// TestResolveTransportAverages checks the average of four hand-worked samples
// and the refusals for the grid side.
func TestResolveTransportAverages(t *testing.T) {
	samples := frame.New(2, 2)
	samples.Pix[0] = frame.Pixel{R: 1, G: 0, B: 4, Cover: 1, Depth: 9}
	samples.Pix[1] = frame.Pixel{R: 2, G: 4, B: 0, Cover: 0, Depth: 3}
	samples.Pix[2] = frame.Pixel{R: 3, G: 4, B: 0, Cover: 1, Depth: 7}
	samples.Pix[3] = frame.Pixel{R: 6, G: 0, B: 0, Cover: 0, Depth: 5}
	got, err := ResolveTransport(samples, 2)
	if err != nil || got.W != 1 || got.H != 1 {
		t.Fatalf("%v %v", got, err)
	}
	want := frame.Pixel{R: 3, G: 2, B: 1, Cover: 0.5, Depth: 3}
	if got.Pix[0] != want {
		t.Errorf("average %+v, want %+v", got.Pix[0], want)
	}
	if same, err := ResolveTransport(samples, 1); err != nil || same != samples {
		t.Errorf("side 1: %v %v", same, err)
	}
	for _, side := range []int{0, 5, 3} {
		if _, err := ResolveTransport(samples, side); err == nil || !strings.Contains(err.Error(), "divide by rays") {
			t.Errorf("side %d: %v", side, err)
		}
	}
	if _, err := ResolveTransport(nil, 1); err == nil {
		t.Error("accepted no samples")
	}
}

// TestDepthGapValues checks the relative gap for hand-worked depths, and the
// sky rule that a sky sample matches only sky.
func TestDepthGapValues(t *testing.T) {
	sky := float32(math.Inf(1))
	for _, c := range []struct {
		a, b float32
		want float64
	}{
		{sky, sky, 0},
		{sky, 3, math.Inf(1)},
		{3, sky, math.Inf(1)},
		{50, 100, 0.5},
		{0.5, 0.25, 0.25},
		{7, 7, 0},
	} {
		if got := depthGap(c.a, c.b); got != c.want {
			t.Errorf("depthGap(%g, %g) = %g, want %g", c.a, c.b, got, c.want)
		}
	}
}

// TestReconstructFallsBackToTheClosestDepth checks that when no coarse sample
// matches the plate depth, the sample with the closest depth is used instead
// of an empty pixel.
func TestReconstructFallsBackToTheClosestDepth(t *testing.T) {
	plate := frame.New(4, 2)
	scatter, transmission := frame.New(2, 1), frame.New(2, 1)
	for i := range plate.Pix {
		plate.Pix[i] = frame.Pixel{R: 2, G: 2, B: 2, Depth: 100}
	}
	scatter.Pix[0] = frame.Pixel{R: 1, Depth: 2}
	scatter.Pix[1] = frame.Pixel{R: 7, Depth: 50}
	transmission.Pix[0] = frame.Pixel{R: 1, G: 1, B: 1}
	transmission.Pix[1] = frame.Pixel{R: 0.5, G: 0.5, B: 0.5}
	got, err := CompositeTransport(plate, scatter, transmission)
	if err != nil {
		t.Fatal(err)
	}
	for i, pixel := range got.Pix {
		if r := pixel.R; r != 8 {
			t.Errorf("pixel %d: red %g, want 8 (scatter 7 plus 0.5 times 2)", i, r)
		}
	}
	sky := float32(math.Inf(1))
	scatter.Pix[0] = frame.Pixel{R: 1, Depth: sky}
	scatter.Pix[1] = frame.Pixel{R: 3, Depth: sky}
	plate.Pix[3].Depth = 5
	got, err = CompositeTransport(plate, scatter, transmission)
	if err != nil {
		t.Fatal(err)
	}
	if r := got.At(3, 0).R; r != 4 {
		t.Errorf("surface against sky: red %g, want 4 (scatter 3 plus 0.5 times 2)", r)
	}
}
