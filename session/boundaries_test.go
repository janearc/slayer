package session

import (
	"bytes"
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/fermat"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/raw"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

func TestProofAndRawOutputFailuresPreserveTheFilesystemCause(t *testing.T) {
	dir := t.TempDir()
	blocked := filepath.Join(dir, "blocked")
	if err := os.WriteFile(blocked, []byte("keep me"), 0600); err != nil {
		t.Fatal(err)
	}
	picture := frame.New(2, 2)
	if err := keep(picture, filepath.Join(blocked, "raw.pfm")); err == nil || !strings.Contains(err.Error(), blocked) {
		t.Fatalf("raw parent failure: %v", err)
	}
	proof := fermat.Proof{Raw: filepath.Join(dir, "absent.pfm")}
	if err := sheet(proof, filepath.Join(blocked, "proof")); err == nil || !strings.Contains(err.Error(), blocked) {
		t.Fatalf("proof parent failure: %v", err)
	}
	if err := sheet(proof, filepath.Join(dir, "proof")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("raw read cause: %v", err)
	}
	path := filepath.Join(dir, "small.pfm")
	if err := raw.Write(path, picture); err != nil {
		t.Fatal(err)
	}
	given := mustParse(t)
	given.width, given.height, given.samples = 3, 2, 1
	given.prove, given.proof, given.many = path, filepath.Join(dir, "explicit-proof"), 1
	var wrong bytes.Buffer
	if code := proving(thing.NewWorld(), given, 1, nil, io.Discard, &wrong); code != 1 || !strings.Contains(wrong.String(), "2 by 2") {
		t.Fatalf("wrong proof size: %d %s", code, wrong.String())
	}
	if sheetFor(given) != given.proof {
		t.Fatal("explicit proof folder ignored")
	}
}

func TestStockAndLegacyStagingRefuseInvalidInputs(t *testing.T) {
	given := mustParse(t)
	if _, _, err := (stock{}).made(given, "absent kind", 0); err == nil || !strings.Contains(err.Error(), "absent kind") {
		t.Fatalf("stock kind refusal: %v", err)
	}
	if _, err := manufacturedRecipe(given, "can", 0, "demo", nil); err == nil || !strings.Contains(err.Error(), "no demo") {
		t.Fatalf("stock representation refusal: %v", err)
	}
	optical := manufacture.Optical{}
	if _, err := manufacturedRecipe(given, "optical-observatory", 0, "quick", &optical); err == nil {
		t.Fatal("invalid observatory accepted")
	}
	one, err := manufactured(given, "spotlight", 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := one.lamps(maths.Subspace{Scale: math.NaN()}); err == nil || !strings.Contains(err.Error(), "invalid light placement") {
		t.Fatalf("lamp placement: %v", err)
	}
	given.exposure = math.NaN()
	one, err = manufactured(given, "spotlight", 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := one.lamps(maths.Local()); err == nil || !strings.Contains(err.Error(), "exposure") {
		t.Fatalf("lamp exposure: %v", err)
	}
	if got := facesIn(&shape.Mesh{}); got != 0 {
		t.Fatalf("empty legacy face count: %d", got)
	}
	if facesIn(shape.Sphere{}) != 0 {
		t.Fatal("non-mesh counted as a face")
	}
	given = mustParse(t)
	given.apart = -1
	if _, _, err := hillside(given); err == nil {
		t.Fatal("invalid placement spacing accepted")
	}
	given = mustParse(t)
	given.above = true
	if eyeFor(given, 10) != aboveAt(10) {
		t.Fatal("above view dispatch changed")
	}
}

func TestLegacyFireRefusalsAndImplicitUnitScale(t *testing.T) {
	given := mustParse(t)
	one, err := manufactured(given, "spruce", 0)
	if err != nil {
		t.Fatal(err)
	}
	unitR, unitH := reach(one.things, 1)
	if r, h := reach(one.things, 0); r != unitR || h != unitH {
		t.Fatal("implicit unit scale changed reach")
	}
	if _, err := alight(thing.NewWorld(), kept{}, maths.Local(), 1, given); err == nil {
		t.Fatal("empty burning geometry accepted")
	}
	given.floor = 1e9
	if _, err := alight(thing.NewWorld(), one, maths.Local(), 1, given); err == nil {
		t.Fatal("out-of-atmosphere fire accepted")
	}
	if err := smoke(thing.NewWorld(), maths.Local(), -1, 0, 1); err == nil {
		t.Fatal("negative smoke radius accepted")
	}
	if err := lamps(thing.NewWorld(), asked{earth: true}, -1); err == nil {
		t.Fatal("negative distant light accepted")
	}
}
