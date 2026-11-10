//go:build darwin

package animation_test

import (
	"bytes"
	"crypto/sha256"
	"debug/buildinfo"
	"encoding/hex"
	"encoding/json"
	"image"
	"image/png"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/janearc/libtheme-css/spaces/ok"
	"github.com/janearc/libtheme-css/spaces/srgb"
	"github.com/janearc/slayer/raw"
	"github.com/janearc/slayer/set"
	"github.com/janearc/slayer/sources"
)

type nativeCounts struct{ Queries, Candidates, Refined, Fallbacks uint64 }
type frameMeasurement struct {
	Renderer                          string
	SceneSHA256, PNGSHA256, RawSHA256 string
	WholeSeconds                      float64
	Native                            *nativeCounts
}
type imageComparison struct {
	Pixels, BeyondOneJND, AlphaDifferences int
	WorstJND                               float64
	WorstAt                                [2]int
}
type measurement struct {
	Revision, ExecutableSHA256, SourceCatalogueSHA256, TerrainSHA256 string
	Width, Height, SamplesPerPixel, Trees                            int
	Frames                                                           []frameMeasurement
	Comparison                                                       *imageComparison
}

// BenchmarkCommitted4K compares the two documented modes with one committed
// executable. Build it at out/review-4k/hillside first; run with -benchtime=1x.
// Artifacts and the machine-written measurement stay in that ignored out/.
func BenchmarkCommitted4K(b *testing.B) {
	root, err := filepath.Abs("../..")
	if err != nil {
		b.Fatal(err)
	}
	out := filepath.Join(root, "out", "review-4k")
	binary := filepath.Join(out, "hillside")
	revision, err := exec.Command("git", "-C", root, "rev-parse", "HEAD").Output()
	if err != nil {
		b.Fatal(err)
	}
	status, err := exec.Command("git", "-C", root, "status", "--porcelain").Output()
	if err != nil || len(status) != 0 {
		b.Fatalf("benchmark needs a clean source checkout: %s %v", status, err)
	}
	info, err := buildinfo.ReadFile(binary)
	if err != nil {
		b.Fatal(err)
	}
	buildRevision, modified := "", ""
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			buildRevision = setting.Value
		case "vcs.modified":
			modified = setting.Value
		}
	}
	if buildRevision != strings.TrimSpace(string(revision)) || modified != "false" {
		b.Fatalf("renderer is not this clean commit: %s modified=%s", buildRevision, modified)
	}
	authored, err := os.ReadFile(filepath.Join(root, "doc", "observatory-4k.yaml"))
	if err != nil {
		b.Fatal(err)
	}
	if bytes.Count(authored, []byte("  renderer: gpu\n")) != 1 {
		b.Fatal("study must name exactly one software GPU mode")
	}
	read, err := set.Read("observatory-4k.yaml", authored)
	if err != nil {
		b.Fatal(err)
	}
	film, has := read.Film()
	if !has {
		b.Fatal("study has no film")
	}
	width, height, side := film.Size()
	trees := 0
	for _, stand := range read.Stands() {
		if stand.Kind() == "coast-redwood" {
			count, _ := stand.Count()
			trees += count
		}
	}
	if width != 3840 || height != 2160 || side != 4 || trees != 5000 {
		b.Fatalf("changed study: %dx%d, %d samples/side, %d redwoods", width, height, side, trees)
	}
	ground, has := read.Ground()
	if !has {
		b.Fatal("study has no measured ground")
	}
	shelf, err := sources.Here()
	if err != nil {
		b.Fatal(err)
	}
	executableHash := hashFile(b, binary)
	record := measurement{Revision: buildRevision, ExecutableSHA256: executableHash,
		SourceCatalogueSHA256: hashFile(b, filepath.Join(root, "sources.yaml")),
		TerrainSHA256:         hashFile(b, filepath.Join(shelf, ground.File())),
		Width:                 width, Height: height, SamplesPerPixel: side * side, Trees: trees}
	save := func() {
		b.Helper()
		data, err := json.MarshalIndent(record, "", "  ")
		if err != nil {
			b.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(out, "measurement.json"), append(data, '\n'), 0644); err != nil {
			b.Fatal(err)
		}
	}
	for trial := range b.N {
		record.Frames = nil
		record.Comparison = nil
		for _, renderer := range []string{"gpu", "gpu-native"} {
			scene := bytes.Replace(authored, []byte("  renderer: gpu\n"), []byte("  renderer: "+renderer+"\n"), 1)
			base := filepath.Join(out, renderer)
			if err := os.WriteFile(base+".yaml", scene, 0644); err != nil {
				b.Fatal(err)
			}
			sceneHash := hashFile(b, base+".yaml")
			cmd := exec.Command(binary, "-set", base+".yaml", "-out", base+".png", "-raw", base+".pfm")
			cmd.Dir = root
			var said, wrong bytes.Buffer
			cmd.Stdout, cmd.Stderr = &said, &wrong
			b.Logf("trial %d: %s begins", trial, renderer)
			started := time.Now()
			runError := cmd.Run()
			elapsed := time.Since(started).Seconds()
			log := append(bytes.Clone(said.Bytes()), wrong.Bytes()...)
			if err := os.WriteFile(base+".log", log, 0644); err != nil {
				b.Fatal(err)
			}
			if runError != nil {
				b.Fatalf("%s failed: %v\n%s", renderer, runError, log)
			}
			if hashFile(b, binary) != executableHash {
				b.Fatal("renderer changed during measurement")
			}
			for path, expected := range map[string]string{
				base + ".yaml":                      sceneHash,
				filepath.Join(root, "sources.yaml"): record.SourceCatalogueSHA256,
				filepath.Join(shelf, ground.File()): record.TerrainSHA256,
			} {
				if hashFile(b, path) != expected {
					b.Fatalf("input changed during rendering: %s", path)
				}
			}
			measured := frameMeasurement{Renderer: renderer, SceneSHA256: sceneHash,
				PNGSHA256: hashFile(b, base+".png"), RawSHA256: hashFile(b, base+".pfm"), WholeSeconds: elapsed}
			linear, err := raw.Read(base + ".pfm")
			if err != nil {
				b.Fatal(err)
			}
			if linear.W != width || linear.H != height {
				b.Fatal("raw dimensions changed")
			}
			for i, p := range linear.Pix {
				for _, v := range []float32{p.R, p.G, p.B} {
					if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
						b.Fatalf("nonfinite raw pixel %d", i)
					}
				}
			}
			if renderer == "gpu-native" {
				measured.Native = readCounts(b, said.String())
			}
			record.Frames = append(record.Frames, measured)
			save()
			b.Logf("%s completed in %.6f seconds", renderer, elapsed)
		}
		comparison := comparePNGs(b, readPNG(b, filepath.Join(out, "gpu.png")), readPNG(b, filepath.Join(out, "gpu-native.png")))
		record.Comparison = &comparison
		save()
		fraction := float64(comparison.BeyondOneJND) / float64(comparison.Pixels)
		b.Logf("%d/%d pixels beyond one JND (%.8f%%), worst %.8f at %v", comparison.BeyondOneJND, comparison.Pixels, 100*fraction, comparison.WorstJND, comparison.WorstAt)
		if comparison.AlphaDifferences != 0 || fraction > .02 {
			b.Fatal("outside the existing two-percent still-image budget")
		}
	}
}

func hashFile(b testing.TB, path string) string {
	b.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		b.Fatal(err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func readCounts(b testing.TB, log string) *nativeCounts {
	b.Helper()
	pattern := regexp.MustCompile("native closest: ([0-9]+) queries, ([0-9]+) candidates, ([0-9]+) refined, ([0-9]+) software fallbacks")
	rows := pattern.FindAllStringSubmatch(log, -1)
	if len(rows) != 1 {
		b.Fatalf("expected one native outcome record: %s", log)
	}
	var values [4]uint64
	for i := range values {
		v, err := strconv.ParseUint(rows[0][i+1], 10, 64)
		if err != nil {
			b.Fatal(err)
		}
		values[i] = v
	}
	counts := &nativeCounts{values[0], values[1], values[2], values[3]}
	if counts.Queries == 0 || counts.Refined == 0 || counts.Refined+counts.Fallbacks != counts.Candidates {
		b.Fatalf("invalid native outcomes: %+v", counts)
	}
	return counts
}

func readPNG(b testing.TB, path string) image.Image {
	b.Helper()
	file, err := os.Open(path)
	if err != nil {
		b.Fatal(err)
	}
	defer file.Close()
	img, err := png.Decode(file)
	if err != nil {
		b.Fatal(err)
	}
	return img
}

// comparePNGs visits every committed display pixel. Its budget is the same
// two-percent rule used for native traversal, without subsampling or masking.
func comparePNGs(b testing.TB, one, two image.Image) imageComparison {
	b.Helper()
	if one.Bounds() != two.Bounds() {
		b.Fatal("PNG dimensions disagree")
	}
	result := imageComparison{Pixels: one.Bounds().Dx() * one.Bounds().Dy()}
	lab := func(img image.Image, x, y int) (ok.OKLab, uint32) {
		r, g, blue, a := img.At(x, y).RGBA()
		return ok.FromSwatch(srgb.RGB8(uint8(r>>8), uint8(g>>8), uint8(blue>>8)).Swatch()), a
	}
	for y := one.Bounds().Min.Y; y < one.Bounds().Max.Y; y++ {
		for x := one.Bounds().Min.X; x < one.Bounds().Max.X; x++ {
			a, alpha := lab(one, x, y)
			c, beta := lab(two, x, y)
			difference := ok.Distance(a, c) / ok.Eye
			if math.IsNaN(difference) || math.IsInf(difference, 0) {
				b.Fatalf("nonfinite image difference at %d,%d", x, y)
			}
			if alpha != beta {
				result.AlphaDifferences++
			}
			if difference > 1 {
				result.BeyondOneJND++
			}
			if difference > result.WorstJND {
				result.WorstJND = difference
				result.WorstAt = [2]int{x, y}
			}
		}
	}
	return result
}
