package omnipositor

import (
	"container/list"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/janearc/slayer/improv/moonlake/lights"
	"github.com/janearc/slayer/improv/moonlake/omnipositor/settings"
)

const MaxFireDirectoryEntries = 4096

const MaxAssetIdentities = 3*MaxFireDirectoryEntries + 15

type AssetIdentity struct {
	Path     string
	Size     int64
	Modified time.Time
	SHA256   string
}

type LoadedAssets struct {
	Lighting           lights.Assets
	static             *lights.StaticAssets
	staticNames        lights.StaticAssetNames
	MoonReference      lights.Image
	mu                 sync.Mutex
	fireRequestMu      sync.Mutex
	width, height      int
	floodPasses        [2]string
	fireConfig         lights.FireConfig
	fireDirs           [3]string
	counts             [3]int
	identities         map[string]AssetIdentity
	lastFireWorkingSet lights.FireWorkingSet
	budget, cached     int64
	cache              map[string]*list.Element
	lru                list.List
}

type assetCacheEntry struct {
	path     string
	image    lights.Image
	bytes    int64
	identity AssetIdentity
}

// LoadAssets reads every pass, moon reference and fire directory named in s
// and returns them with the images held in memory for the caller to use.
func LoadAssets(s settings.Settings) (*LoadedAssets, error) {
	return loadAssets(s, false)
}

// LoadStaticAssets is LoadAssets, but the images go into a private store
// that the lighting renderer reads. Only snapshots of water and trees are
// left in Lighting.
func LoadStaticAssets(s settings.Settings) (*LoadedAssets, error) {
	return loadAssets(s, true)
}

// loadAssets is the body of LoadAssets and LoadStaticAssets; private picks
// the second. It refuses a bad frame size or a cache budget that is not
// positive. A file named under two settings is read once. Fire directories
// are only counted here; their frames are read later, on demand.
func loadAssets(s settings.Settings, private bool) (*LoadedAssets, error) {
	if s.Frame.Width <= 0 || s.Frame.Height <= 0 || s.Frame.Width > int(^uint(0)>>1)/24/s.Frame.Height {
		return nil, fmt.Errorf("assets frame: invalid or excessive dimensions")
	}
	if s.Output.MemoryCacheBytes <= 0 {
		return nil, fmt.Errorf("output.memory_cache_bytes: must be positive")
	}
	a := &LoadedAssets{width: s.Frame.Width, height: s.Frame.Height, floodPasses: s.Night.FloodPasses,
		identities: make(map[string]AssetIdentity), budget: s.Output.MemoryCacheBytes, cache: make(map[string]*list.Element)}
	if private {
		a.static = new(lights.StaticAssets)
	}
	a.Lighting.Passes = make(map[string]lights.Image)
	seen := make(map[string]lights.Image)
	load := func(name, path string) (lights.Image, error) {
		absolute, err := filepath.Abs(path)
		if err != nil || strings.TrimSpace(path) == "" {
			return lights.Image{}, fmt.Errorf("%s: invalid path %q", name, path)
		}
		if image, ok := seen[absolute]; ok {
			for _, identity := range a.identities {
				if identity.Path == absolute {
					a.identities[name] = identity
					break
				}
			}
			return image, nil
		}
		image, identity, err := readAssetInto(name, absolute, a.width, a.height, a.static)
		if err != nil {
			return lights.Image{}, err
		}
		seen[absolute], a.identities[name] = image, identity
		return image, nil
	}
	var disc, water, trees lights.Image
	for _, item := range []struct {
		name, path string
		target     *lights.Image
	}{
		{"passes.nodisc", s.Passes.Nodisc, &a.Lighting.Plate},
		{"passes.disc", s.Passes.Disc, &disc},
		{"passes.mask", s.Passes.Mask, &a.Lighting.Mask},
		{"passes.water", s.Passes.Water, &water},
		{"passes.trees", s.Passes.Trees, &trees},
	} {
		image, err := load(item.name, item.path)
		if err != nil {
			return nil, err
		}
		*item.target = image
	}
	a.Lighting.Disc, a.Lighting.Water, a.Lighting.Trees = &disc, &water, &trees
	for _, item := range []struct{ name, path, pass string }{
		{"passes.back_left", s.Passes.BackLeft, s.Night.FloodPasses[0]},
		{"passes.back_right", s.Passes.BackRight, s.Night.FloodPasses[1]},
		{"passes.overhead", s.Passes.Overhead, "overhead"},
	} {
		if item.pass == "" {
			return nil, fmt.Errorf("%s: empty renderer pass name", item.name)
		}
		if _, exists := a.Lighting.Passes[item.pass]; exists {
			return nil, fmt.Errorf("%s: duplicate renderer pass name %q", item.name, item.pass)
		}
		image, err := load(item.name, item.path)
		if err != nil {
			return nil, err
		}
		a.Lighting.Passes[item.pass] = image
	}
	for _, item := range []struct {
		name, pass string
		source     *settings.OptionalPath
	}{
		{"passes.spot_left_0", "spot-left-0", s.Passes.SpotLeft0},
		{"passes.spot_left_1", "spot-left-1", s.Passes.SpotLeft1},
		{"passes.spot_left_2", "spot-left-2", s.Passes.SpotLeft2},
		{"passes.spot_right_0", "spot-right-0", s.Passes.SpotRight0},
		{"passes.spot_right_1", "spot-right-1", s.Passes.SpotRight1},
		{"passes.spot_right_2", "spot-right-2", s.Passes.SpotRight2},
	} {
		if item.source == nil || item.source.Off {
			continue
		}
		if _, exists := a.Lighting.Passes[item.pass]; exists {
			return nil, fmt.Errorf("%s: duplicate renderer pass name %q", item.name, item.pass)
		}
		image, err := load(item.name, item.source.Path)
		if err != nil {
			return nil, err
		}
		a.Lighting.Passes[item.pass] = image
	}
	a.MoonReference = disc
	if s.MoonRef != nil && !s.MoonRef.Off {
		image, err := load("moon_ref", s.MoonRef.Path)
		if err != nil {
			return nil, err
		}
		a.MoonReference = image
	}
	for sequence, source := range [3]*settings.OptionalPath{s.Fire.Frames, s.Fire.Swell, s.Fire.Burst} {
		if source == nil || source.Off {
			continue
		}
		name := "fire." + [3]string{"steady", "swell", "burst"}[sequence]
		if strings.TrimSpace(source.Path) == "" {
			return nil, fmt.Errorf("%s: empty directory", name)
		}
		dir, err := filepath.Abs(source.Path)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		count, err := fireFrameCount(dir)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		a.fireDirs[sequence], a.counts[sequence] = dir, count
	}
	if a.fireDirs[0] == "" && (a.fireDirs[1] != "" || a.fireDirs[2] != "") {
		return nil, fmt.Errorf("fire.frames: steady directory required when swell or burst is enabled")
	}
	a.fireConfig = lights.FireConfig{Gain: s.Fire.Gain, Decay: s.Fire.Decay, FPS: s.Fire.FPS,
		Percussive: a.fireDirs[1] != "", Burst: a.fireDirs[2] != "", SteadyFrom: float64(s.Fire.SteadyFrom),
		Pulse: s.Fire.Pulse, Flash: s.Fire.Flash, Snap: s.Fire.Snap, SwellFall: s.Fire.SwellFall, BurstGain: s.Fire.BurstGain, Top: s.Fire.Top}
	if private {
		path := func(name string) string { return a.identities[name].Path }
		a.staticNames = lights.StaticAssetNames{Plate: path("passes.nodisc"), Mask: path("passes.mask"), Disc: path("passes.disc"), Water: path("passes.water"), Trees: path("passes.trees"), Passes: make(map[string]string)}
		for _, item := range []struct{ name, pass string }{
			{"passes.back_left", s.Night.FloodPasses[0]}, {"passes.back_right", s.Night.FloodPasses[1]}, {"passes.overhead", "overhead"},
			{"passes.spot_left_0", "spot-left-0"}, {"passes.spot_left_1", "spot-left-1"}, {"passes.spot_left_2", "spot-left-2"},
			{"passes.spot_right_0", "spot-right-0"}, {"passes.spot_right_1", "spot-right-1"}, {"passes.spot_right_2", "spot-right-2"},
		} {
			if name := path(item.name); name != "" {
				a.staticNames.Passes[item.pass] = name
			}
		}
		water, trees := a.static.Snapshot(a.staticNames.Water), a.static.Snapshot(a.staticNames.Trees)
		a.Lighting = lights.Assets{Water: &water, Trees: &trees}
		moon := path("moon_ref")
		if moon == "" {
			moon = a.staticNames.Disc
		}
		a.MoonReference = a.static.Snapshot(moon)
	}
	return a, nil
}

// readAsset reads the PFM file at path as a width by height image, and
// returns it with the identity of the bytes that were read.
func readAsset(name, path string, width, height int) (lights.Image, AssetIdentity, error) {
	return readAssetInto(name, path, width, height, nil)
}

// readAssetInto reads the PFM file at path, hashing what it reads, and
// returns the identity (path, size, mod time, SHA-256). It refuses a file
// that is not regular, or whose size is not the pixel bytes plus at most
// 4096 of header, or that changes while it is read. If stored is not nil
// the pixels are decoded into it under path and the returned image is empty.
func readAssetInto(name, path string, width, height int, stored *lights.StaticAssets) (image lights.Image, identity AssetIdentity, err error) {
	fail := func(cause error) (lights.Image, AssetIdentity, error) {
		return lights.Image{}, AssetIdentity{}, fmt.Errorf("%s (%s): %w", name, path, cause)
	}
	file, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return fail(err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("%s (%s): %w", name, path, closeErr))
		}
	}()
	before, err := file.Stat()
	if err != nil {
		return fail(err)
	}
	if !before.Mode().IsRegular() {
		return fail(fmt.Errorf("not a regular PFM file"))
	}
	if width <= 0 || height <= 0 || width > int(^uint(0)>>1)/24/height {
		return fail(fmt.Errorf("invalid expected dimensions"))
	}
	pixelBytes := int64(width) * int64(height) * 12
	if before.Size() < pixelBytes {
		return fail(fmt.Errorf("expected %dx%d: %w", width, height, io.ErrUnexpectedEOF))
	}
	if before.Size() > pixelBytes+4096 {
		return fail(fmt.Errorf("PFM exceeds expected pixel bytes plus 4096-byte header bound"))
	}
	hash := sha256.New()
	reader := io.TeeReader(io.LimitReader(file, before.Size()+1), hash)
	if stored == nil {
		image, _, err = lights.DecodePFMExpected(reader, width, height)
	} else {
		err = stored.Decode(path, reader, width, height)
	}
	if err != nil {
		return fail(err)
	}
	if stored == nil {
		if err := validateLightImage(image, width, height); err != nil {
			return fail(err)
		}
	}
	after, err := file.Stat()
	if err != nil {
		return fail(err)
	}
	current, err := os.Stat(path)
	if err != nil {
		return fail(err)
	}
	if !os.SameFile(before, current) || current.Size() != after.Size() || !current.ModTime().Equal(after.ModTime()) || !sameChangeTime(current, after) || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) || !sameChangeTime(before, after) {
		return fail(fmt.Errorf("asset changed while loading"))
	}
	return image, AssetIdentity{Path: path, Size: before.Size(), Modified: before.ModTime(), SHA256: hex.EncodeToString(hash.Sum(nil))}, nil
}

// sameChangeTime reports whether two stat results show the same inode
// change time. It is false if the system stat has no such field.
func sameChangeTime(before, after os.FileInfo) bool {
	left, right := reflect.ValueOf(before.Sys()), reflect.ValueOf(after.Sys())
	if left.Kind() != reflect.Pointer || right.Kind() != reflect.Pointer {
		return false
	}
	for _, field := range []string{"Ctim", "Ctimespec"} {
		a, b := left.Elem().FieldByName(field), right.Elem().FieldByName(field)
		if a.IsValid() && b.IsValid() {
			return reflect.DeepEqual(a.Interface(), b.Interface())
		}
	}
	return false
}

// fireFrameCount returns how many frames the fire directory dir holds. It
// refuses more than MaxFireDirectoryEntries entries of any kind, a .pfm
// that is not a regular file, and frames not named 000.pfm, 001.pfm and so
// on without a gap.
func fireFrameCount(dir string) (int, error) {
	file, err := os.OpenFile(dir, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	entries, err := file.ReadDir(MaxFireDirectoryEntries + 1)
	if err != nil && err != io.EOF {
		return 0, err
	}
	if len(entries) > MaxFireDirectoryEntries {
		return 0, fmt.Errorf("fire directory entries exceed maximum %d (all entries count toward the bound)", MaxFireDirectoryEntries)
	}
	names := make(map[string]bool)
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".pfm") {
			continue
		}
		if !entry.Type().IsRegular() {
			return 0, fmt.Errorf("%s: not a regular PFM file", filepath.Join(dir, entry.Name()))
		}
		names[entry.Name()] = true
	}
	for index := 0; index < len(names); index++ {
		name := fmt.Sprintf("%03d.pfm", index)
		if !names[name] {
			return 0, fmt.Errorf("%s: expected contiguous numbered fire frames, missing %s", dir, name)
		}
	}
	return len(names), nil
}

// Identities returns a copy of the identity of each asset loaded so far,
// keyed by asset name, so the caller cannot change the stored ones.
func (a *LoadedAssets) Identities() map[string]AssetIdentity {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make(map[string]AssetIdentity, len(a.identities))
	for name, identity := range a.identities {
		out[name] = identity
	}
	return out
}

// CacheBytes is how many bytes of fire frames the cache holds now.
func (a *LoadedAssets) CacheBytes() int64 {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.cached
}

// LastFireWorkingSet is the working set measured by the most recent fire
// load. It is reset at the start of each load.
func (a *LoadedAssets) LastFireWorkingSet() lights.FireWorkingSet {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.lastFireWorkingSet
}

// LoadFire returns the fire frames that request needs, one set for each of
// the steady, swell and burst sequences. The pixel data is copied, so the
// caller owns it. It refuses at once if another fire request is running.
func (a *LoadedAssets) LoadFire(ctx context.Context, request LightRequest) ([3]lights.FireFrames, error) {
	if !a.fireRequestMu.TryLock() {
		return [3]lights.FireFrames{}, fmt.Errorf("%w: asset fire request already active", lights.ErrFireResources)
	}
	defer a.fireRequestMu.Unlock()
	return a.loadFire(ctx, request, true)
}

// loadFire is loadFireBudget with the default frame limit, or the first of
// limits if one is given, and the default byte budget cut to the cache budget.
func (a *LoadedAssets) loadFire(ctx context.Context, request LightRequest, copyImages bool, limits ...int) ([3]lights.FireFrames, error) {
	maximum := lights.DefaultConfig().MaxFireFrames
	if len(limits) != 0 {
		maximum = limits[0]
	}
	return a.loadFireBudget(ctx, request, copyImages, maximum, lights.FireByteBudget(lights.DefaultMaxFireBytes, a.budget))
}

// loadFireBudget is loadFireBudgetTracked without a record of cache hits.
func (a *LoadedAssets) loadFireBudget(ctx context.Context, request LightRequest, copyImages bool, maximum int, byteBudget int64) ([3]lights.FireFrames, error) {
	return a.loadFireBudgetTracked(ctx, request, copyImages, maximum, byteBudget, nil)
}

// loadFireBudgetTracked plans which fire frames request needs and refuses if
// they pass maximum frames or byteBudget bytes. It then takes each from the
// cache or reads it, and appends the name of each cache hit to hits if hits
// is not nil. It returns no frames when the fire level is near zero or no
// steady directory is set. It refuses a frame whose identity has changed.
func (a *LoadedAssets) loadFireBudgetTracked(ctx context.Context, request LightRequest, copyImages bool, maximum int, byteBudget int64, hits *[]string) ([3]lights.FireFrames, error) {
	var frames [3]lights.FireFrames
	if ctx == nil {
		return frames, fmt.Errorf("fire: nil context")
	}
	if err := ctx.Err(); err != nil {
		return frames, err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.lastFireWorkingSet = lights.FireWorkingSet{}
	for sequence, count := range a.counts {
		frames[sequence] = lights.FireFrames{Count: count, Images: make(map[int]lights.Image)}
	}
	cue := request.Cues.Quantized()
	if !finite(cue.Fire) {
		return frames, fmt.Errorf("fire: nonfinite level")
	}
	if cue.Fire <= 1e-4 || a.fireDirs[0] == "" {
		return frames, nil
	}
	layers, err := lights.PlanFire(a.fireConfig, lights.FireCues{Level: cue.Fire, Lit: cue.FireLit, Since: cue.FireSince}, a.counts)
	if err != nil {
		return frames, fmt.Errorf("fire plan: %w", err)
	}
	workingSet, err := lights.MeasureFireWorkingSet(layers, a.width, a.height, maximum)
	if err != nil {
		return [3]lights.FireFrames{}, err
	}
	if err := lights.CheckFireBudget(workingSet, byteBudget); err != nil {
		return [3]lights.FireFrames{}, err
	}
	workingSet.TransientBytes = workingSet.SelectedBytes
	if copyImages {
		if workingSet.SelectedBytes > (int64(^uint64(0)>>1)-int64(a.width)*int64(a.height)*12)/2 {
			return [3]lights.FireFrames{}, fmt.Errorf("fire working set: byte count overflow")
		}
		workingSet.TransientBytes += workingSet.SelectedBytes
	}
	if workingSet.Frames != 0 {
		workingSet.TransientBytes += int64(a.width) * int64(a.height) * 12
	}
	a.lastFireWorkingSet = workingSet
	for _, layer := range layers {
		if err := ctx.Err(); err != nil {
			return [3]lights.FireFrames{}, err
		}
		if _, exists := frames[layer.Sequence].Images[layer.Index]; exists {
			continue
		}
		name := fmt.Sprintf("fire.%s[%d]", [3]string{"steady", "swell", "burst"}[layer.Sequence], layer.Index)
		path := filepath.Join(a.fireDirs[layer.Sequence], fmt.Sprintf("%03d.pfm", layer.Index))
		var image lights.Image
		if entry, exists := a.cache[path]; exists {
			a.lru.MoveToFront(entry)
			cached := entry.Value.(assetCacheEntry)
			image = cached.image
			if original, exists := a.identities[name]; exists && original.SHA256 != cached.identity.SHA256 {
				return [3]lights.FireFrames{}, fmt.Errorf("%s: cached asset identity changed", name)
			}
			if _, exists := a.identities[name]; !exists && len(a.identities) >= MaxAssetIdentities {
				return [3]lights.FireFrames{}, fmt.Errorf("asset identities exceed maximum %d", MaxAssetIdentities)
			}
			a.identities[name] = cached.identity
			if hits != nil {
				*hits = append(*hits, name)
			}
		} else {
			var identity AssetIdentity
			image, identity, err = readAsset(name, path, a.width, a.height)
			if err != nil {
				return [3]lights.FireFrames{}, err
			}
			if err := ctx.Err(); err != nil {
				return [3]lights.FireFrames{}, err
			}
			if original, exists := a.identities[name]; exists {
				if original.Path != identity.Path || original.SHA256 != identity.SHA256 {
					return [3]lights.FireFrames{}, fmt.Errorf("%s (%s): asset identity changed since first load", name, path)
				}
			} else {
				if len(a.identities) >= MaxAssetIdentities {
					return [3]lights.FireFrames{}, fmt.Errorf("asset identities exceed maximum %d", MaxAssetIdentities)
				}
				a.identities[name] = identity
			}
			bytes := int64(len(image.Light)) * 8
			if bytes <= a.budget {
				for a.cached > a.budget-bytes {
					old := a.lru.Back()
					entry := old.Value.(assetCacheEntry)
					delete(a.cache, entry.path)
					a.cached -= entry.bytes
					a.lru.Remove(old)
				}
				a.cache[path] = a.lru.PushFront(assetCacheEntry{path: path, image: image, bytes: bytes, identity: identity})
				a.cached += bytes
			}
		}
		if copyImages {
			image.Light = append([]float64(nil), image.Light...)
		}
		frames[layer.Sequence].Images[layer.Index] = image
	}
	return frames, nil
}

// NewLightStage makes a lighting stage from the loaded assets, with the
// flood passes and fire settings taken from them and the fire byte limit cut
// to the cache budget. The stage reads fire frames from these assets.
func (a *LoadedAssets) NewLightStage(config lights.Config) (*RendererLightStage, error) {
	config.MaxFireBytes = lights.FireByteBudget(config.MaxFireBytes, a.budget)
	config.Fixtures.FloodPasses = a.floodPasses
	if a.fireDirs[0] != "" {
		stops := config.Fire.Stops
		config.Fire = a.fireConfig
		config.Fire.Stops = stops
	}
	assets := a.Lighting
	if a.fireDirs[0] != "" {
		assets.Fire = nil
	}
	var stage *RendererLightStage
	var err error
	if a.static != nil {
		renderer, renderErr := lights.NewStaticRenderer(config, a.static, a.staticNames)
		err = renderErr
		if err == nil {
			stage = &RendererLightStage{renderer: renderer, width: a.width, height: a.height}
		}
	} else {
		stage, err = NewLightStage(config, assets)
	}
	if err != nil {
		return nil, err
	}
	stage.fireLoader = a
	stage.maxFireFrames = config.MaxFireFrames
	stage.maxFireBytes = config.MaxFireBytes
	return stage, nil
}
