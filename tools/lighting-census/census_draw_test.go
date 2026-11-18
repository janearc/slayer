//go:build draw && darwin

package census

import (
	"bufio"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image/png"
	"io"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/gpu"
	"github.com/janearc/slayer/raw"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/smoke"
	"github.com/janearc/slayer/trace"
)

// The driver is a test solely so it can use Game's existing build and GPU
// lane. Without the explicit switch it does no work, even with -tags draw.
func TestLightingCensus(t *testing.T) {
	if os.Getenv("SLAYER_CENSUS_RUN") != "1" {
		t.Skip("opt in with SLAYER_CENSUS_RUN=1; see README")
	}
	opts, err := readOptions()
	if err != nil {
		t.Fatal(err)
	}
	if opts.Mode != "cpu" {
		cache, err := os.UserCacheDir()
		if err != nil {
			t.Fatal(err)
		}
		if err := requireGPULane(os.Getenv("GAME_LANES"), cache); err != nil {
			t.Fatal(err)
		}
	}
	if err := CreateOutput(opts.Output); err != nil {
		t.Fatal(err)
	}
	if err := runCensus(opts); err != nil {
		t.Fatal(err)
	}
}

// requireGPULane validates Game's inherited mark and tests that its normal
// lane file is locked. The mark is inherited, not proof of lock ownership.
func requireGPULane(marks, cache string) error {
	want := filepath.Join(cache, "game", "lanes", "gpu")
	for _, mark := range filepath.SplitList(marks) {
		name, path, ok := strings.Cut(mark, "=")
		if !ok || name != "gpu" {
			continue
		}
		if filepath.Dir(path) != want || !strings.HasPrefix(filepath.Base(path), "lane-") {
			return fmt.Errorf("census: unsafe GPU lane mark")
		}
		f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
		if err != nil {
			return err
		}
		err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		closeErr := f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return closeErr
		}
		return fmt.Errorf("census: Game GPU lane is not held")
	}
	return fmt.Errorf("census: a held Game GPU lane is required")
}

func TestOptionsRefuseUnknownModesAndAcceptExplicitSceneTimes(t *testing.T) {
	t.Setenv("SLAYER_CENSUS_SCENE", "0009")
	t.Setenv("SLAYER_CENSUS_MODE", "gpu-fast-lighting")
	t.Setenv("SLAYER_CENSUS_TIMES", "0s,1.5s")
	o, err := readOptions()
	if err != nil || len(o.Times) != 2 || o.Times[1] != 1500*time.Millisecond || o.Scene != "9" {
		t.Fatalf("options %+v: %v", o, err)
	}
	t.Setenv("SLAYER_CENSUS_MODE", "invented")
	if _, err := readOptions(); err == nil {
		t.Fatal("unknown mode accepted")
	}
	t.Setenv("SLAYER_CENSUS_MODE", "cpu")
	t.Setenv("SLAYER_CENSUS_TIMES", "-1s")
	if _, err := readOptions(); err == nil {
		t.Fatal("negative clock accepted")
	}
}

func TestGPULaneRequiresAnInheritedMarkAndActualLock(t *testing.T) {
	cache := t.TempDir()
	dir := filepath.Join(cache, "game", "lanes", "gpu")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "lane-1")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := requireGPULane("gpu="+path, cache); err == nil {
		t.Fatal("unlocked lane accepted")
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Fatal(err)
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	if err := requireGPULane("gpu="+path, cache); err != nil {
		t.Fatal(err)
	}
	if err := requireGPULane("gpu="+filepath.Join(cache, "other"), cache); err == nil {
		t.Fatal("wrong path accepted")
	}
	if err := requireGPULane("", cache); err == nil {
		t.Fatal("missing mark accepted")
	}
}

func TestCapacityFailureStopsBeforeConstructingFurtherCases(t *testing.T) {
	dir := t.TempDir()
	o := options{Output: dir, Scene: "all", Mode: "all", Times: []time.Duration{0}}
	called := 0
	err := runCensusWith(o, func(string, int, int) error { called++; return ErrCapacity })
	if !errors.Is(err, ErrCapacity) || called != 1 {
		t.Fatalf("capacity stop: %v, checks %d", err, called)
	}
	data, err := os.ReadFile(filepath.Join(dir, "scene-001", "cpu", "result.json"))
	if err != nil {
		t.Fatal(err)
	}
	var result caseResult
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	if result.Status != "failed" || result.WorldBuildNS != 0 {
		t.Fatalf("case was not stopped before scene construction: %+v", result)
	}
	if _, err := os.Stat(filepath.Join(dir, "scene-001", "gpu-software")); !os.IsNotExist(err) {
		t.Fatalf("later case started: %v", err)
	}
	data, err = os.ReadFile(filepath.Join(dir, "summary.json"))
	if err != nil {
		t.Fatal(err)
	}
	var counts map[string]int
	if err := json.Unmarshal(data, &counts); err != nil || counts["failed"] != 1 {
		t.Fatalf("partial summary %q: %v", data, err)
	}
}

func TestSyntheticArtifactsPreserveLightCoverageAndDepth(t *testing.T) {
	dir := t.TempDir()
	want := frame.New(2, 1)
	want.Pix[0] = frame.Pixel{R: 3, G: 2, B: 1, Cover: 0.25, Depth: 42}
	want.Pix[1] = frame.Pixel{R: 1, G: 4, B: 2, Cover: 0.75, Depth: float32(math.Inf(1))}
	if err := saveFrame(dir, "frame-000", want, smoke.Smoke{}); err != nil {
		t.Fatal(err)
	}
	got, err := ReadFrame(filepath.Join(dir, "frame-000.pfm"), filepath.Join(dir, "frame-000.cd32"))
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Pix) != len(want.Pix) || got.Pix[0] != want.Pix[0] || got.Pix[1] != want.Pix[1] {
		t.Fatalf("frame not preserved: %+v", got)
	}
}

type options struct {
	Output string          `json:"output"`
	Scene  string          `json:"scene_selection"`
	Mode   string          `json:"mode_selection"`
	Width  int             `json:"width"`
	Height int             `json:"height"`
	Times  []time.Duration `json:"times_ns"`
}

func readOptions() (options, error) {
	o := options{Output: os.Getenv("SLAYER_CENSUS_OUT"), Scene: os.Getenv("SLAYER_CENSUS_SCENE"), Mode: os.Getenv("SLAYER_CENSUS_MODE")}
	if o.Scene == "" {
		o.Scene = "all"
	}
	if o.Mode == "" {
		o.Mode = "all"
	}
	for name, target := range map[string]*int{"SLAYER_CENSUS_WIDTH": &o.Width, "SLAYER_CENSUS_HEIGHT": &o.Height} {
		if text := os.Getenv(name); text != "" {
			n, err := strconv.Atoi(text)
			if err != nil || n < 1 || n > 8192 {
				return o, fmt.Errorf("%s needs a dimension from 1 to 8192", name)
			}
			*target = n
		}
	}
	if (o.Width == 0) != (o.Height == 0) {
		return o, fmt.Errorf("both dimensions must be supplied together")
	}
	if o.Scene != "all" {
		n, err := strconv.Atoi(o.Scene)
		if err != nil {
			return o, fmt.Errorf("invalid scene %q", o.Scene)
		}
		if _, ok := smoke.Get(n); !ok {
			return o, fmt.Errorf("unknown smoke %d", n)
		}
		o.Scene = strconv.Itoa(n)
	}
	if o.Mode != "all" {
		found := false
		for _, mode := range Modes() {
			found = found || mode.Name == o.Mode
		}
		if !found {
			return o, fmt.Errorf("unknown mode %q", o.Mode)
		}
	}
	times := os.Getenv("SLAYER_CENSUS_TIMES")
	if times == "" {
		times = "0s"
	}
	for _, part := range strings.Split(times, ",") {
		at, err := time.ParseDuration(strings.TrimSpace(part))
		if err != nil || at < 0 {
			return o, fmt.Errorf("invalid nonnegative scene time %q", part)
		}
		o.Times = append(o.Times, at)
	}
	return o, nil
}

type identity struct {
	ExecutableSHA256 string            `json:"executable_sha256"`
	GoVersion        string            `json:"go_version"`
	GOOS             string            `json:"goos"`
	GOARCH           string            `json:"goarch"`
	Host             string            `json:"host"`
	BuildSettings    map[string]string `json:"build_settings"`
	OSVersion        string            `json:"os_version"`
	SourceRevision   string            `json:"source_revision"`
	SourceDirty      string            `json:"source_dirty"`
}

func provenance() (identity, error) {
	i := identity{GoVersion: runtime.Version(), GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, BuildSettings: map[string]string{}, OSVersion: "unavailable", SourceRevision: "unavailable", SourceDirty: "unavailable"}
	i.Host, _ = os.Hostname()
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, s := range info.Settings {
			i.BuildSettings[s.Key] = s.Value
			if s.Key == "vcs.revision" {
				i.SourceRevision = s.Value
			}
			if s.Key == "vcs.modified" {
				i.SourceDirty = s.Value
			}
		}
	}
	path, err := os.Executable()
	if err != nil {
		return i, err
	}
	i.ExecutableSHA256, err = hashFile(path)
	return i, err
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

type frameRecord struct {
	Index               int          `json:"index"`
	TimeNS              int64        `json:"time_ns"`
	Phase               string       `json:"phase"`
	Raw                 string       `json:"raw"`
	CoverageDepth       string       `json:"coverage_depth"`
	PNG                 string       `json:"png"`
	DurationNS          int64        `json:"duration_ns"`
	HeapBefore          uint64       `json:"go_heap_before_bytes"`
	HeapAfter           uint64       `json:"go_heap_after_bytes"`
	ProcessPeakRSSBytes *int64       `json:"process_lifetime_peak_rss_bytes,omitempty"`
	Profile             *gpu.Profile `json:"profile,omitempty"`
}

type bandRecord struct {
	Frame      int     `json:"frame"`
	Band       int     `json:"band"`
	Nanometres float64 `json:"nanometres"`
	Raw        string  `json:"raw"`
	DurationNS int64   `json:"duration_ns"`
}

type caseResult struct {
	Case             Case              `json:"case"`
	Status           string            `json:"status"`
	Reason           string            `json:"reason,omitempty"`
	BandStatus       string            `json:"spectral_band_status"`
	WorldBuildNS     int64             `json:"world_build_ns"`
	Inputs           map[string]string `json:"input_sha256,omitempty"`
	Frames           []frameRecord     `json:"frames"`
	Bands            []bandRecord      `json:"bands"`
	Camera           string            `json:"camera"`
	Width            int               `json:"width"`
	Height           int               `json:"height"`
	ExecutableSHA256 string            `json:"executable_sha256"`
}

func runCensus(o options) error {
	return runCensusWith(o, checkSpace)
}

func runCensusWith(o options, check func(string, int, int) error) error {
	id, err := provenance()
	if err != nil {
		return err
	}
	matrix := Matrix()
	if err := writeJSON(filepath.Join(o.Output, "manifest.json"), struct {
		Version  int       `json:"version"`
		Created  time.Time `json:"created"`
		Options  options   `json:"options"`
		Identity identity  `json:"identity"`
		Cases    []Case    `json:"cases"`
		Scope    string    `json:"scope"`
	}{1, time.Now().UTC(), o, id, matrix, "GPU is visible-light only; CPU records declared bands separately. PNGs are unit-exposure visible diagnostics, not authored tone, palette, gauge or composite. Camera is fixed; retained repeat uses the same scene time. Process RSS is lifetime peak, not per case. Binary digest does not establish complete source/input provenance. No movie is produced."}); err != nil {
		return err
	}
	counts := map[string]int{}
	for _, c := range matrix {
		if o.Scene != "all" && o.Scene != strconv.Itoa(c.Scene) || o.Mode != "all" && o.Mode != c.Mode.Name {
			continue
		}
		status, err := runCase(o, c, check, id.ExecutableSHA256)
		counts[status]++
		if err != nil {
			if writeErr := writeJSON(filepath.Join(o.Output, "summary.json"), counts); writeErr != nil {
				return errors.Join(err, writeErr)
			}
			return err
		}
	}
	if err := writeJSON(filepath.Join(o.Output, "summary.json"), counts); err != nil {
		return err
	}
	if counts["failed"] > 0 {
		return fmt.Errorf("census: %d cases failed; see summary.json", counts["failed"])
	}
	return nil
}

func runCase(o options, c Case, check func(string, int, int) error, executableSHA256 string) (status string, failure error) {
	dir := filepath.Join(o.Output, fmt.Sprintf("scene-%03d", c.Scene), c.Mode.Name)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "failed", err
	}
	r := caseResult{Case: c, Status: "running", Camera: "fixed smoke registry eye and field of view", Frames: []frameRecord{}, Bands: []bandRecord{}, ExecutableSHA256: executableSHA256}
	resultPath := filepath.Join(dir, "result.json")
	defer func() {
		if p := recover(); p != nil {
			r.Status = "failed"
			r.Reason = fmt.Sprintf("scene panic: %v", p)
		}
		if r.Status == "running" {
			r.Status = "failed"
			if r.Reason == "" {
				r.Reason = "case ended before completion"
			}
		}
		status = r.Status
		if err := writeJSON(resultPath, r); err != nil {
			failure = fmt.Errorf("write case result: %w", err)
		}
	}()
	if err := writeJSON(resultPath, r); err != nil {
		return "failed", err
	}
	made, ok := smoke.Get(c.Scene)
	if !ok {
		return "failed", fmt.Errorf("smoke %d disappeared", c.Scene)
	}
	r.BandStatus = "none"
	if len(made.Bands) > 0 {
		r.BandStatus = "unsupported-on-gpu"
		if c.Mode.Name == "cpu" {
			r.BandStatus = "pending"
		}
	}
	initialWidth, initialHeight := o.Width, o.Height
	if initialWidth == 0 {
		initialWidth, initialHeight = 2148, 1200
	}
	if err := check(o.Output, initialWidth, initialHeight); err != nil {
		r.Reason = err.Error()
		return "failed", err
	}
	paths, err := made.RequireInputs()
	if err != nil {
		r.Reason = err.Error()
		return "", nil
	}
	r.Inputs = map[string]string{}
	for _, path := range paths {
		sum, err := hashFile(path)
		if err != nil {
			r.Reason = err.Error()
			return "", nil
		}
		r.Inputs[path] = sum
	}
	start := time.Now()
	world, eye := made.Scene()
	r.WorldBuildNS = time.Since(start).Nanoseconds()
	if o.Width == 0 {
		o.Width, o.Height = 1074, 600
		if len(world.Fields) > 0 {
			o.Width *= 2
			o.Height *= 2
		}
	}
	r.Width, r.Height = o.Width, o.Height
	lens, err := film.PinholeOf(made.Wide)
	if err != nil {
		r.Reason = err.Error()
		return "", nil
	}
	var drawer *gpu.Frames
	if c.Mode.Name != "cpu" {
		mode := map[string]gpu.Mode{"gpu-software": gpu.Software, "gpu-native-shadows": gpu.NativeShadows, "gpu-native-meshes": gpu.NativeMeshes, "gpu-cached-lighting": gpu.CachedLighting, "gpu-fast-lighting": gpu.FastLighting, "gpu-faster-lighting": gpu.FasterLighting}[c.Mode.Name]
		drawer, err = gpu.NewFrames(mode)
		if err != nil {
			r.Reason = err.Error()
			return "", nil
		}
		defer drawer.Close()
	}
	times := append([]time.Duration{o.Times[0], o.Times[0]}, o.Times[1:]...)
	for n, at := range times {
		phase := "scene-time"
		if n == 0 {
			phase = "cold"
		}
		if n == 1 {
			phase = "same-time-repeat"
		}
		if err := check(o.Output, o.Width, o.Height); err != nil {
			r.Reason = err.Error()
			return "failed", err
		}
		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		start = time.Now()
		var picture *frame.Frame
		var profile gpu.Profile
		if drawer == nil {
			picture, err = trace.Sampled(world, eye, lens, at, o.Width, o.Height, c.Samples)
		} else {
			picture, profile, err = drawer.Draw(world, 1, eye, lens, at, o.Width, o.Height, c.Samples)
		}
		duration := time.Since(start)
		runtime.ReadMemStats(&after)
		if err != nil {
			r.Reason = err.Error()
			if strings.HasPrefix(r.Reason, "gpu: cannot draw ") {
				r.Status = "unsupported"
			}
			return "", nil
		}
		stem := fmt.Sprintf("frame-%03d", n)
		if err := check(o.Output, o.Width, o.Height); err != nil {
			r.Reason = err.Error()
			return "failed", err
		}
		if err := saveFrame(dir, stem, picture, made); err != nil {
			r.Reason = err.Error()
			return "", nil
		}
		var usage syscall.Rusage
		var peak *int64
		if syscall.Getrusage(syscall.RUSAGE_SELF, &usage) == nil {
			value := usage.Maxrss
			peak = &value
		}
		fr := frameRecord{Index: n, TimeNS: int64(at), Phase: phase, Raw: stem + ".pfm", CoverageDepth: stem + ".cd32", PNG: stem + ".png", DurationNS: duration.Nanoseconds(), HeapBefore: before.HeapAlloc, HeapAfter: after.HeapAlloc, ProcessPeakRSSBytes: peak}
		if drawer != nil {
			fr.Profile = &profile
		}
		r.Frames = append(r.Frames, fr)
		if err := writeJSON(resultPath, r); err != nil {
			return "failed", err
		}
		if drawer == nil {
			for b, band := range made.Bands {
				if e := check(o.Output, o.Width, o.Height); e != nil {
					r.Reason = e.Error()
					r.BandStatus = "failed"
					return "failed", e
				}
				bandStart := time.Now()
				bandframe, e := trace.RenderIn(world, eye, lens, at, o.Width, o.Height, band)
				bandDuration := time.Since(bandStart)
				if e != nil {
					r.Reason = e.Error()
					r.BandStatus = "failed"
					return "", nil
				}
				if e = check(o.Output, o.Width, o.Height); e != nil {
					r.Reason = e.Error()
					r.BandStatus = "failed"
					return "failed", e
				}
				bandStem := fmt.Sprintf("frame-%03d-band-%02d", n, b)
				if e = saveFrame(dir, bandStem, bandframe, made); e != nil {
					r.Reason = e.Error()
					r.BandStatus = "failed"
					return "", nil
				}
				r.Bands = append(r.Bands, bandRecord{Frame: n, Band: b, Nanometres: band.Nanometres, Raw: bandStem + ".pfm", DurationNS: bandDuration.Nanoseconds()})
				if err := writeJSON(resultPath, r); err != nil {
					return "failed", err
				}
			}
		}
	}
	if drawer == nil && len(made.Bands) > 0 {
		r.BandStatus = "cpu-raw-bands-recorded"
	}
	for path, original := range r.Inputs {
		now, e := hashFile(path)
		if e != nil || now != original {
			r.Status = "failed"
			r.Reason = "declared input changed or became unreadable during case: " + path
			return "", nil
		}
	}
	r.Status = "ok"
	if r.BandStatus == "unsupported-on-gpu" {
		r.Status = "partial"
		r.Reason = "GPU modes do not expose spectral-band rendering"
	}
	return "", nil
}

func checkSpace(path string, width, height int) error {
	needed, err := RequiredFree(width, height)
	if err != nil {
		return err
	}
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return fmt.Errorf("%w: free space unavailable: %v", ErrCapacity, err)
	}
	if stat.Bsize <= 0 || uint64(stat.Bavail) > math.MaxUint64/uint64(stat.Bsize) {
		return fmt.Errorf("%w: invalid free-space reading", ErrCapacity)
	}
	free := uint64(stat.Bavail) * uint64(stat.Bsize)
	if free < needed {
		return fmt.Errorf("%w: free space %d below required %d bytes", ErrCapacity, free, needed)
	}
	return nil
}

func saveFrame(dir, stem string, image *frame.Frame, made smoke.Smoke) error {
	pfm := filepath.Join(dir, stem+".pfm")
	if err := raw.Write(pfm+".tmp", image); err != nil {
		return err
	}
	if err := os.Rename(pfm+".tmp", pfm); err != nil {
		return err
	}
	cd := filepath.Join(dir, stem+".cd32")
	f, err := os.OpenFile(cd+".tmp", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	buffer := bufio.NewWriterSize(f, 64*1024)
	for _, p := range image.Pix {
		var pair [8]byte
		binary.LittleEndian.PutUint32(pair[:4], math.Float32bits(p.Cover))
		binary.LittleEndian.PutUint32(pair[4:], math.Float32bits(p.Depth))
		if _, err = buffer.Write(pair[:]); err != nil {
			f.Close()
			return err
		}
	}
	if err := buffer.Flush(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(cd+".tmp", cd); err != nil {
		return err
	}
	print := filepath.Join(dir, stem+".png")
	f, err = os.OpenFile(print+".tmp", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	err = png.Encode(f, colour.Develop(image, 1))
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(print+".tmp", print)
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp := path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
