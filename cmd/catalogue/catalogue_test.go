package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/render/frame"
)

//go:embed testdata/faults.yaml
var faultsRaw []byte

type planFault struct {
	Before, After string
}

// successfulExit is defined by the process success status.
const successfulExit = 0

// singleArtifact is defined by the one pre-existing file in the refusal test.
const singleArtifact = 1

func defaultPlan(test *testing.T) *plan {
	test.Helper()
	made, err := readPlan("plan.yaml", planRaw)
	if err != nil {
		test.Fatal(err)
	}
	return made
}

func TestCatalogueRefusesInvalidPlansBeforeCreatingOutput(test *testing.T) {
	var faults []planFault
	if err := yamlfile.Read("faults.yaml", faultsRaw, &faults); err != nil {
		test.Fatal(err)
	}
	for index, fault := range faults {
		test.Run(fmt.Sprint(index), fault.refused)
	}
}

func (fault planFault) refused(test *testing.T) {
	raw := strings.ReplaceAll(string(planRaw), fault.Before, fault.After)
	if raw == string(planRaw) {
		test.Fatal("ineffective plan fault", fault)
	}
	root := test.TempDir()
	path := filepath.Join(root, "bad.yaml")
	if err := os.WriteFile(path, []byte(raw), artifactMode); err != nil {
		test.Fatal(err)
	}
	output := filepath.Join(root, "book")
	refuseInvocation(test, output,
		"render", "webb", "--plan", path, "--out", output)
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		test.Fatal("bad plan created output", err)
	}
}

func refuseInvocation(test *testing.T, output string, args ...string) {
	test.Helper()
	var said, wrong bytes.Buffer
	status := tool(context.Background()).Run(args, &said, &wrong)
	if status == successfulExit || wrong.Len() == noEntries {
		test.Fatal("invalid invocation succeeded", output, said.String(), wrong.String())
	}
}

// A directory that is not a book is refused and left as it was. The
// anti-test is a book, which is resumed, not refused: see
// TestASecondRunDrawsOnlyWhatIsNotRendered.
func TestCatalogueNeverOverwritesADirectoryThatIsNotABook(test *testing.T) {
	root := test.TempDir()
	marker := filepath.Join(root, "keep.txt")
	expected := []byte("keep this book")
	if err := os.WriteFile(marker, expected, artifactMode); err != nil {
		test.Fatal(err)
	}
	refuseInvocation(test, root, "render", "webb", "--out", root)
	actual, err := os.ReadFile(marker)
	if err != nil || !bytes.Equal(actual, expected) {
		test.Fatal("existing book changed", err)
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != singleArtifact {
		test.Fatal("existing book gained output", entries, err)
	}
}

func TestCatalogueProvenanceHashesTheExecutingBytes(test *testing.T) {
	got, err := compiledProducer()
	if err != nil {
		test.Fatal(err)
	}
	path, err := os.Executable()
	if err != nil {
		test.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		test.Fatal(err)
	}
	expected := fmt.Sprintf("%x", sha256.Sum256(raw))
	if got.BinarySHA256 != expected || got.Revision == "" || got.Modified == "" {
		test.Fatal("producer identity is missing or hashes other bytes", got)
	}
}

// singlePixel is defined as the smallest nonempty image.
const singlePixel = 1

func TestBrightBackgroundDoesNotProveASpecimenWasVisible(test *testing.T) {
	plan := defaultPlan(test)
	picture := frame.New(singlePixel, singlePixel)
	for index := range picture.Pix {
		picture.Pix[index].R = fullCoverage
	}
	if err := plan.visible(picture); err == nil {
		test.Fatal("bright background passed the surface-coverage gate")
	}
}

func TestCoveredPixelsNeedFinitePositiveDepth(test *testing.T) {
	plan := defaultPlan(test)
	picture := frame.New(singlePixel, singlePixel)
	for index := range picture.Pix {
		picture.Pix[index].Cover = fullCoverage
		picture.Pix[index].Depth = singlePixel
	}
	if err := plan.visible(picture); err != nil {
		test.Fatal("valid covered pixel was refused", err)
	}
	for index := range picture.Pix {
		picture.Pix[index].Depth = float32(math.NaN())
	}
	if err := plan.visible(picture); err == nil {
		test.Fatal("nonfinite covered depth passed the specimen gate")
	}
}

func TestCancelledCatalogueCreatesNoBook(test *testing.T) {
	ctx, stop := context.WithCancel(context.Background())
	stop()
	output := filepath.Join(test.TempDir(), "cancelled")
	var said, wrong bytes.Buffer
	status := tool(ctx).Run(
		[]string{"render", "webb", "--out", output}, &said, &wrong)
	if status == successfulExit {
		test.Fatal("cancelled render succeeded")
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		test.Fatal("cancelled invocation created a book", err)
	}
}

func Example_list() {
	tool(context.Background()).Run([]string{"list", "webb"}, os.Stdout, os.Stderr)
	// Output:
	// webb: A gold segmented primary on a tower above five aluminium sunshield layers, its bus and solar panels on the sun's side. [hull clumps quick demo]
}

func Example_render() {
	tool(context.Background()).Run([]string{"render", "webb", "--dry-run"}, os.Stdout, os.Stderr)
	// Output:
	// 1 views -> out/catalogue
}
