package omnipositor

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sync"

	"github.com/janearc/slayer/improv/moonlake/cues"
	"github.com/janearc/slayer/improv/moonlake/lights"
	"github.com/janearc/slayer/improv/moonlake/omnipositor/identity"
	"github.com/janearc/slayer/improv/moonlake/omnipositor/settings"
)

const MaxExecutionSourceBytes = 16 << 20

type ExecutionSnapshot struct {
	config    Config
	stage     *RendererLightStage
	assets    *LoadedAssets
	spec      identity.PrehashedSpec
	renderMu  sync.Mutex
	prepareMu sync.Mutex
}

type PreparedFrame struct {
	owner   *ExecutionSnapshot
	request LightRequest
	key     identity.Key
}

// NewExecutionSnapshot builds a snapshot for rendering on the CPU. It refuses
// unless binarySHA256 matches the running executable and backend and device
// match the CPU on this machine, and unless c has no show, light stage or
// masks of its own, since they are filled in from score, cues and static
// assets. Score and cues must be nonempty and at most MaxExecutionSourceBytes.
func NewExecutionSnapshot(ctx context.Context, s settings.Settings, c Config, l lights.Config, scoreBytes, cueBytes []byte, binarySHA256, backend, device string) (*ExecutionSnapshot, error) {
	if ctx == nil {
		return nil, fmt.Errorf("execution: nil context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	executable, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("execution: executable: %w", err)
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		return nil, fmt.Errorf("execution: executable path: %w", err)
	}
	actualHash, err := identity.HashFile(executable)
	if err != nil {
		return nil, fmt.Errorf("execution: executable hash: %w", err)
	}
	actualDevice := "cpu/" + runtime.GOOS + "/" + runtime.GOARCH
	if binarySHA256 != actualHash || backend != "cpu" || device != actualDevice {
		return nil, fmt.Errorf("%w: executable or CPU runtime claim differs from actual execution", ErrUnboundExecutionSnapshot)
	}
	if c.Show != nil || c.Light != nil || c.TreeMask != nil || c.MoonReference.Width != 0 || c.MoonReference.Height != 0 || c.MoonReference.Light != nil || c.WaterMask.Width != 0 || c.WaterMask.Height != 0 || c.WaterMask.Light != nil {
		return nil, ErrUnboundExecutionSnapshot
	}
	if len(scoreBytes) == 0 || len(cueBytes) == 0 || len(scoreBytes) > MaxExecutionSourceBytes || len(cueBytes) > MaxExecutionSourceBytes {
		return nil, fmt.Errorf("execution: score and cues must be nonempty and at most %d bytes each", MaxExecutionSourceBytes)
	}
	scoreBytes = append([]byte(nil), scoreBytes...)
	cueBytes = append([]byte(nil), cueBytes...)
	show, err := cues.Parse(scoreBytes, cueBytes)
	if err != nil {
		return nil, fmt.Errorf("execution: show: %w", err)
	}
	if c.Width != s.Frame.Width || c.Height != s.Frame.Height {
		return nil, fmt.Errorf("execution: settings dimensions differ")
	}
	scalar := c
	scalar.From, scalar.To = 0, 0
	scalar.Ocean.Width, scalar.Ocean.Height, scalar.Ocean.Seconds = 0, 0, 0
	a, err := LoadStaticAssets(s)
	if err != nil {
		return nil, err
	}
	l.MaxFireBytes = lights.FireByteBudget(l.MaxFireBytes, a.budget)
	l.Fixtures.FloodPasses = a.floodPasses
	if a.fireDirs[0] != "" {
		stops := l.Fire.Stops
		l.Fire = a.fireConfig
		l.Fire.Stops = stops
	}
	recipe, err := json.Marshal(struct {
		Version    int
		Compositor Config
		Lighting   lights.Config
	}{1, scalar, l})
	if err != nil {
		return nil, fmt.Errorf("execution: config: %w", err)
	}
	inputs := map[string]identity.HashedInput{"score": {SHA256: sourceDigest(scoreBytes)}, "cues": {SHA256: sourceDigest(cueBytes)}}
	for name, asset := range a.Identities() {
		inputs[name] = identity.HashedInput{SHA256: asset.SHA256}
	}
	spec := identity.PrehashedSpec{BinarySHA256: actualHash, Settings: recipe, Inputs: inputs, FPS: c.FPS, Frame: int64(c.From), Backend: "cpu", Device: actualDevice, Width: c.Width, Height: c.Height}
	if _, err := identity.BuildPrehashed(spec); err != nil {
		return nil, err
	}
	renderer, err := lights.NewStaticRenderer(l, a.static, a.staticNames)
	if err != nil {
		return nil, err
	}
	stage := &RendererLightStage{renderer: renderer, width: c.Width, height: c.Height, maxFireFrames: l.MaxFireFrames, maxFireBytes: l.MaxFireBytes}
	c.Show, c.Light = show, stage
	c.MoonReference = a.MoonReference
	c.WaterMask = *a.Lighting.Water
	c.TreeMask = a.Lighting.Trees
	if err := c.validate(); err != nil {
		return nil, fmt.Errorf("execution: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return &ExecutionSnapshot{config: c, stage: stage, assets: a, spec: spec}, nil
}

// PrepareFrame works out the cues, fire frames and identity key for frame
// index, which must lie in the config's From to To range. It refuses if
// another preparation on this snapshot is running.
func (e *ExecutionSnapshot) PrepareFrame(ctx context.Context, index int) (*PreparedFrame, error) {
	if ctx == nil {
		return nil, fmt.Errorf("execution: nil context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if e == nil || e.stage == nil {
		return nil, ErrUnboundExecutionSnapshot
	}
	if !e.prepareMu.TryLock() {
		return nil, fmt.Errorf("%w: execution preparation already active", lights.ErrFireResources)
	}
	defer e.prepareMu.Unlock()
	c := e.config
	if index < c.From || index >= c.To {
		return nil, fmt.Errorf("execution: frame outside range")
	}
	seconds := quantize(float64(index)/c.FPS, 4)
	cue, err := c.Show.At(seconds, c.FPS)
	if err != nil {
		return nil, err
	}
	cue = cue.Quantized()
	request := LightRequest{Index: index, Seconds: seconds, Width: c.Width, Height: c.Height, Cues: cue, Stops: quantize(c.Develop.Exposure+cue.Exposure, 3), WaterMask: c.WaterMask}
	fire, err := e.assets.prepareFire(ctx, request, e.stage.maxFireFrames, e.stage.maxFireBytes)
	if err != nil {
		return nil, err
	}
	request.FireSnapshot = fire
	spec := e.spec
	spec.Frame = int64(index)
	spec.Inputs = make(map[string]identity.HashedInput, len(e.spec.Inputs)+len(fire.inputs))
	for name, input := range e.spec.Inputs {
		spec.Inputs[name] = input
	}
	for name, input := range fire.inputs {
		spec.Inputs[name] = input
	}
	key, err := identity.BuildPrehashed(spec)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return &PreparedFrame{owner: e, request: request, key: key}, nil
}

// Key is the identity key of the prepared frame, or the zero key if p is nil.
func (p *PreparedFrame) Key() identity.Key {
	if p == nil {
		return identity.Key{}
	}
	return p.key
}

// CacheHits names the fire frames that came from the cache when the frame
// was prepared. It is nil if there are none or p is nil.
func (p *PreparedFrame) CacheHits() []string {
	if p == nil || p.request.FireSnapshot == nil {
		return nil
	}
	return p.request.FireSnapshot.CacheHits()
}

// Render draws the prepared frame on the CPU and passes the result to
// consume, using the fire frames loaded when it was prepared. It refuses if
// the frame is not prepared, or another render on the snapshot is running.
func (p *PreparedFrame) Render(ctx context.Context, consume func(Result) error) error {
	if ctx == nil {
		return fmt.Errorf("execution: nil context")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if p == nil || p.owner == nil || p.request.FireSnapshot == nil {
		return ErrUnboundExecutionSnapshot
	}
	e := p.owner
	if !e.renderMu.TryLock() {
		return fmt.Errorf("%w: execution render already active", lights.ErrFireResources)
	}
	defer e.renderMu.Unlock()
	c := e.config
	c.From, c.To = p.request.Index, p.request.Index+1
	c.Light = preparedLight{p: p}
	return Run(ctx, c, consume)
}

type preparedLight struct{ p *PreparedFrame }

// Light returns the lighting for the prepared frame. It refuses with
// ErrUnboundExecutionSnapshot if request is not the one that was prepared.
func (s preparedLight) Light(ctx context.Context, request LightRequest) (LightResult, error) {
	expected := s.p.request
	expected.FireSnapshot = nil
	if !reflect.DeepEqual(request, expected) {
		return LightResult{}, ErrUnboundExecutionSnapshot
	}
	return s.p.owner.stage.light(ctx, s.p.request, true)
}
