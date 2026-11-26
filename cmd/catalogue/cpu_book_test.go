package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// cpuBookDrawer is a successful CPU draw without ray tracing, for testing the
// book's behavior independently of the rendering engine.
type cpuBookDrawer struct{ count int }

func (drawing *cpuBookDrawer) draw(_ plan, destination string, view *card, _ *producer) error {
	drawing.count++
	view.Status = "CPU render"
	return os.Mkdir(filepath.Join(destination, view.Folder), directoryMode)
}

func TestCPUCardIsVisibleAndNamedOnMixedBookPage(test *testing.T) {
	made := testBook(twoCards)
	made.Cards[1].Status = "CPU render"
	root := test.TempDir()
	if err := made.write(root); err != nil {
		test.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(root, "index.html"))
	if err != nil {
		test.Fatal(err)
	}
	page := string(raw)
	for _, want := range []string{"kind-0-quick/specimen.png", "kind-1-quick/specimen.png", "Rendered on the CPU", "native GPU"} {
		if !strings.Contains(page, want) {
			test.Errorf("mixed page lacks %q", want)
		}
	}
	if strings.Contains(page, `<div class="pending">CPU render`) {
		test.Fatal("page presents the completed CPU image as pending")
	}
}

func TestMixedBookPDFIncludesBothRenderBackends(test *testing.T) {
	made := testBook(twoCards)
	made.Cards[1].Status = "CPU render"
	root := lay(test, made)
	writeTestPicture(test, filepath.Join(root, made.Cards[1].Folder, "specimen.png"))
	raw := composeDocument(test, root)
	if images := bytes.Count(raw, []byte("/Subtype /Image")); images != twoCards {
		test.Fatalf("composed %d pictures, want %d", images, twoCards)
	}
	if !bytes.Contains(raw, []byte("CPU")) {
		test.Fatal("PDF does not identify the CPU-rendered card")
	}
}

func TestCPUAndGPUCardsResumeWithoutMisattributingBackend(test *testing.T) {
	root := filepath.Join(test.TempDir(), "book")
	gpu := runDrawing(test, drawsEvery(test, oneCard, nil), "render", "webb", "--out", root)
	if gpu.status != successfulExit {
		test.Fatal(gpu.wrong)
	}
	cpuDraw := &cpuBookDrawer{}
	cpu := runDrawing(test, cpuDraw, "render", "fern", "--renderer", "cpu", "--out", root)
	if cpu.status != successfulExit || cpuDraw.count != oneCard {
		test.Fatalf("CPU draw: %+v, %d", cpu, cpuDraw.count)
	}
	held := loadBook(test, root)
	if cardOf(test, held, "webb").Status != renderedStatus || cardOf(test, held, "fern").Status != "CPU render" {
		test.Fatalf("backends lost on mixed-book resume: %+v", held.Cards)
	}
	again := &cpuBookDrawer{}
	resumed := runDrawing(test, again, "render", "fern", "--renderer", "cpu", "--out", root)
	if resumed.status != successfulExit || again.count != noEntries {
		test.Fatalf("CPU card was not kept: %+v, %d", resumed, again.count)
	}
	changed := runDrawing(test, drawsEvery(test, oneCard, nil), "render", "fern", "--out", root)
	if changed.status != successfulExit {
		test.Fatal(changed.wrong)
	}
	if got := cardOf(test, loadBook(test, root), "fern").Status; got != renderedStatus {
		test.Fatalf("selected CPU card was not redrawn on GPU: %q", got)
	}
	backToCPU := &cpuBookDrawer{}
	changed = runDrawing(test, backToCPU, "render", "webb", "--renderer", "cpu", "--out", root)
	if changed.status != successfulExit || backToCPU.count != oneCard {
		test.Fatalf("selected GPU card was not redrawn on CPU: %+v, %d", changed, backToCPU.count)
	}
	held = loadBook(test, root)
	if cardOf(test, held, "webb").Status != "CPU render" || cardOf(test, held, "fern").Status != renderedStatus {
		test.Fatalf("backends after redrawing GPU on CPU: %+v", held.Cards)
	}
}

func TestUnknownCatalogueRendererIsRefusedBeforeOutput(test *testing.T) {
	root := filepath.Join(test.TempDir(), "book")
	refuseInvocation(test, root, "render", "webb", "--renderer", "invented", "--out", root)
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		test.Fatal("invalid renderer created output", err)
	}
}
