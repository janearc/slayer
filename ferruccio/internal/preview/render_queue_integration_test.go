package preview

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestRenderOptionsThroughAzazel needs an owner-dispatched worker. Its outputs
// and Ferruccio receipts stay in the caller's evidence directory.
// Supply optics.yaml (24 by 16, two rays per side), region.yaml (a 12 by 10
// crop of optics.yaml), and stock.yaml (a 24-pixel-wide Super 8 set with
// two rays per side, two glimpses, and soot development).
func TestRenderOptionsThroughAzazel(t *testing.T) {
	root := os.Getenv("FERRUCCIO_RENDER_EVIDENCE")
	if root == "" {
		t.Skip("set the shared evidence directory and configure an Azazel worker")
	}
	tools := Tools{
		Azazel: os.Getenv("FERRUCCIO_EVIDENCE_AZAZEL"), Queue: os.Getenv("FERRUCCIO_EVIDENCE_QUEUE"),
		Slayer: os.Getenv("FERRUCCIO_EVIDENCE_SLAYER"), Agent: "ferruccio-render-check",
		Cache: os.Getenv("SLAYER_CACHE"), Shelf: os.Getenv("SLAYER_SHELF"), Processors: "2",
	}
	type renderCase struct {
		name, source            string
		gpu                     bool
		native                  bool
		samples                 int
		perPixel                int
		glimpses                int
		width, height           int
		preset                  string
		frameWidth, frameHeight int
	}
	cases := []renderCase{
		{name: "legacy-cpu", source: "optics.yaml", perPixel: 4, glimpses: 1, width: 24, height: 16},
		{name: "legacy-native-gpu", source: "optics.yaml", native: true, perPixel: 4, glimpses: 1, width: 24, height: 16},
		{name: "super8-stock-cpu", source: "stock.yaml", perPixel: 4, glimpses: 2, width: 24, height: 17},
		{name: "half-portrait-region-cpu", source: "region.yaml", perPixel: 4, glimpses: 1,
			width: 12, height: 10, preset: "half-portrait", frameWidth: 1080, frameHeight: 960},
		{name: "imax-region-gpu", source: "region.yaml", gpu: true, perPixel: 4, glimpses: 1,
			width: 12, height: 10, preset: "imax", frameWidth: 4096, frameHeight: 3002},
	}
	for _, backend := range []string{"cpu", "gpu", "native-gpu"} {
		for _, samples := range []int{1, 4, 7, 16, 25, 64} {
			cases = append(cases, renderCase{
				name: fmt.Sprintf("exact-%s-%d", backend, samples), source: "optics.yaml",
				gpu: backend == "gpu", native: backend == "native-gpu", samples: samples, perPixel: samples, glimpses: 1, width: 24, height: 16,
			})
		}
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(root, test.source)
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			profile := Profile{Full: true, CPURenderer: !test.gpu && !test.native,
				GPU: test.gpu, NativeGPU: test.native, Samples: test.samples,
				SizePreset: test.preset, Width: test.frameWidth, Height: test.frameHeight}
			plan, err := Prepare(Request{Path: path, Text: string(source), Profile: profile,
				OutputRoot: root, DirectoryPrefix: test.name + "-"})
			if err != nil {
				t.Fatal(err)
			}
			defer plan.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
			defer cancel()
			var acknowledged bool
			if err := RunObserved(ctx, tools, plan, os.Stderr, func(update Update) {
				if update.Sequence > 0 && !acknowledged {
					t.Logf("Azazel job %d in %s", update.Sequence, plan.Directory)
					acknowledged = true
				}
			}); err != nil {
				t.Fatalf("render %s: %v", plan.Directory, err)
			}
			raw, err := os.ReadFile(filepath.Join(plan.Directory, "receipt.json"))
			if err != nil {
				t.Fatal(err)
			}
			var receipt struct {
				Width     int               `json:"width"`
				Height    int               `json:"height"`
				Sequence  uint64            `json:"queue_sequence"`
				Renderer  string            `json:"renderer_override"`
				PerPixel  int               `json:"samples_per_pixel"`
				Rays      *int              `json:"rays_per_axis"`
				Requested int               `json:"primary_samples_requested"`
				Rig       *physicalRig      `json:"physical_rig"`
				Stock     *developmentStock `json:"development_stock"`
				Lane      string            `json:"lane"`
				Preset    string            `json:"size_preset"`
				Region    map[string]int    `json:"region"`
			}
			if err := json.Unmarshal(raw, &receipt); err != nil {
				t.Fatal(err)
			}
			renderer := "cpu"
			lane := "tests"
			if test.gpu {
				renderer = "gpu"
				lane = "gpu"
			}
			if test.native {
				renderer = "gpu-native"
				lane = "gpu"
			}
			if receipt.Sequence == 0 || receipt.Width != test.width || receipt.Height != test.height ||
				receipt.Renderer != renderer || receipt.Lane != lane ||
				receipt.PerPixel != test.perPixel || receipt.Requested != test.width*test.height*test.perPixel*test.glimpses {
				t.Fatalf("unexpected receipt %s", raw)
			}
			if test.samples > 0 && receipt.Rays != nil || test.samples == 0 && (receipt.Rays == nil || *receipt.Rays != 2) {
				t.Fatalf("incorrect sampling mode %s", raw)
			}
			if test.glimpses > 1 && (receipt.Rig == nil || receipt.Rig.Glimpses != test.glimpses ||
				receipt.Rig.WidthMM != 5.79 || receipt.Rig.HeightMM != 4.01 ||
				receipt.Stock == nil || receipt.Stock.Name != "soot") {
				t.Fatalf("physical settings missing from receipt %s", raw)
			}
			if test.preset != "" && (receipt.Preset != test.preset ||
				receipt.Region["frame_width"] != test.frameWidth || receipt.Region["frame_height"] != test.frameHeight) {
				t.Fatalf("preset or region missing from receipt %s", raw)
			}
			t.Logf("accepted Azazel job %d in %s", receipt.Sequence, plan.Directory)
		})
	}
}
