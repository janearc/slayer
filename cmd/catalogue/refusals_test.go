package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/interfaces/terminal/mocks"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"go.uber.org/mock/gomock"
)

// altered is the default plan with one line changed, written beside the
// test, and the test stops if the line was not there to change.
func altered(test *testing.T, before, after string) string {
	test.Helper()
	raw := strings.ReplaceAll(string(planRaw), before, after)
	if raw == string(planRaw) {
		test.Fatalf("the default plan has no line %q", before)
	}
	path := filepath.Join(test.TempDir(), "altered.yaml")
	if err := os.WriteFile(path, []byte(raw), artifactMode); err != nil {
		test.Fatal(err)
	}
	return path
}

// oneEntry is a menu entry its schema takes: a kind and the details it
// is drawn at.
const oneEntry = `  - kind: %s
    group: Tests
    about: A specimen for the tests.
    notes: Authored for the tests.
    details: [%s]
`

// A menu is refused that is not a menu, that names a kind the
// manufactory does not make, that names a kind twice, that asks for a
// detail there is none of, and that leaves a kind out; each by what is
// wrong with it. A detail there is none of is refused by the menu's
// schema, at the line it is on, before the menu's own check sees it.
func TestTheMenuRefusesWhatItCannotOffer(test *testing.T) {
	webb := fmt.Sprintf(oneEntry, "webb", "quick")
	menus := map[string]struct{ raw, says string }{
		"not a menu":        {"entries: 7\n", "entries"},
		"a kind never made": {"entries:\n" + fmt.Sprintf(oneEntry, "zebra", "quick"), "zebra"},
		"a kind twice":      {"entries:\n" + webb + webb, `duplicate catalogue kind "webb"`},
		"no such detail":    {"entries:\n" + fmt.Sprintf(oneEntry, "webb", "gigantic"), "refused.yaml:6:"},
		"a kind left out":   {"entries:\n" + webb, "catalogue is missing kind"},
	}
	for name, each := range menus {
		made, err := parseMenu("refused.yaml", []byte(each.raw))
		if err == nil || made != nil || !strings.Contains(err.Error(), each.says) {
			test.Errorf("%s: %v; want it to say %q", name, err, each.says)
		}
	}
}

// With no names a menu offers every entry, in its order, as a copy; a
// name it does not carry is refused with every such name, sorted, and
// the anti-test is a name it does carry.
func TestSelectingEntriesOffersAllOrRefusesTheUnknown(test *testing.T) {
	menu, err := readMenu()
	if err != nil {
		test.Fatal(err)
	}
	all, err := menu.selectEntries(nil)
	if err != nil || len(all) != len(menu.Entries) {
		test.Fatalf("no names offered %d of %d entries, %v", len(all),
			len(menu.Entries), err)
	}
	first := menu.Entries[0].Kind
	all[0].Kind = "changed"
	if menu.Entries[0].Kind != first {
		test.Error("the entries offered are the menu's own, not a copy")
	}
	_, err = menu.selectEntries([]string{"zebra", "webb", "aardvark"})
	if err == nil || !strings.Contains(err.Error(),
		"unknown catalogue kinds: [aardvark zebra]") {
		test.Errorf("unknown names: %v", err)
	}
	one, err := menu.selectEntries([]string{"webb"})
	if err != nil || len(one) != 1 || one[0].Kind != "webb" {
		test.Errorf("webb alone: %v, %v", one, err)
	}
}

// refusing is somewhere nothing can be written.
func refusing(test *testing.T) *mocks.MockWriter {
	writer := mocks.NewMockWriter(gomock.NewController(test))
	writer.EXPECT().Write(gomock.Any()).
		Return(0, errors.New("nothing can be written here")).AnyTimes()
	return writer
}

// list says every entry on a line of its own. The anti-tests: a name
// the menu does not carry stops it, and so does a place it cannot
// write to.
func TestListSaysEveryEntryOrStops(test *testing.T) {
	menu, err := readMenu()
	if err != nil {
		test.Fatal(err)
	}
	var said, wrong bytes.Buffer
	if status := tool(context.Background()).Run([]string{"list"}, &said,
		&wrong); status != successfulExit {
		test.Fatalf("list stopped: %s", wrong.String())
	}
	if lines := strings.Count(said.String(), "\n"); lines != len(menu.Entries) {
		test.Errorf("list said %d lines for %d entries", lines, len(menu.Entries))
	}
	said.Reset()
	if status := tool(context.Background()).Run([]string{"list", "zebra"},
		&said, &wrong); status == successfulExit ||
		!strings.Contains(wrong.String(), "unknown catalogue kinds: [zebra]") {
		test.Errorf("list zebra: %q, %q", said.String(), wrong.String())
	}
	wrong.Reset()
	if status := tool(context.Background()).Run([]string{"list", "webb"},
		refusing(test), &wrong); status == successfulExit {
		test.Errorf("list wrote where nothing can be written: %q", wrong.String())
	}
}

// A plan file that is not there is refused before anything is made, and
// a plan asking for a detail there is none of is refused by name.
func TestAPlanThatIsNotThereOrAsksForNoSuchDetailIsRefused(test *testing.T) {
	root := test.TempDir()
	output := filepath.Join(root, "book")
	refuseInvocation(test, output, "render", "webb", "--plan",
		filepath.Join(root, "no-such-plan.yaml"), "--out", output)
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		test.Error("a plan that is not there created output", err)
	}
	made := *defaultPlan(test)
	made.Details = []string{"gigantic"}
	if err := made.check(); err == nil || !strings.Contains(err.Error(),
		`unknown catalogue detail "gigantic"`) {
		test.Errorf("a plan at detail gigantic: %v", err)
	}
}

// A book is not written where there is no folder, nor where one of its
// three files cannot be: each is tried in turn with a folder standing
// where the file would go. The anti-test is an empty folder, which
// takes all three.
func TestABookIsNotWrittenWhereItsFilesCannotGo(test *testing.T) {
	root := test.TempDir()
	if err := (book{}).write(filepath.Join(root, "no-such-folder")); err == nil {
		test.Error("a book was written into no folder")
	}
	if err := writeIcons(filepath.Join(root, "no-such-folder")); err == nil {
		test.Error("icons were written into no folder")
	}
	for _, blocked := range []string{"index.html", "catalogue.css", "catalogue.json"} {
		folder := filepath.Join(root, "blocked-"+blocked)
		if err := os.MkdirAll(filepath.Join(folder, blocked), directoryMode); err != nil {
			test.Fatal(err)
		}
		if err := (book{}).write(folder); err == nil {
			test.Errorf("a book was written over a folder named %s", blocked)
		}
	}
	whole := filepath.Join(root, "whole")
	if err := os.Mkdir(whole, directoryMode); err != nil {
		test.Fatal(err)
	}
	if err := (book{}).write(whole); err != nil {
		test.Fatal(err)
	}
	if err := writeIcons(whole); err != nil {
		test.Fatal(err)
	}
	for _, name := range []string{"index.html", "catalogue.css",
		"catalogue.json", "icon-light.png", "icon-dark.png"} {
		if info, err := os.Stat(filepath.Join(whole, name)); err != nil ||
			info.Size() == 0 {
			test.Errorf("%s was not written: %v", name, err)
		}
	}
}

// A file's hash is the SHA-256 of its bytes. The anti-tests: a file
// that is not there, and a folder, have none; and a record that cannot
// be said as JSON is not written.
func TestAHashIsOfAFilesBytesAndARecordMustBeSayable(test *testing.T) {
	root := test.TempDir()
	path := filepath.Join(root, "bytes")
	content := []byte("what the file holds")
	if err := os.WriteFile(path, content, artifactMode); err != nil {
		test.Fatal(err)
	}
	got, err := hashFile(path)
	if want := sha256.Sum256(content); err != nil || !bytes.Equal(got, want[:]) {
		test.Errorf("the hash is %x, %v; want %x", got, err, want)
	}
	if _, err := hashFile(filepath.Join(root, "not-there")); err == nil {
		test.Error("a file that is not there has a hash")
	}
	if _, err := hashFile(root); err == nil {
		test.Error("a folder has a hash")
	}
	record := filepath.Join(root, "record.json")
	if err := writeJSON(record, make(chan int)); err == nil {
		test.Error("a channel was written as JSON")
	}
	if _, err := os.Stat(record); !os.IsNotExist(err) {
		test.Error("a record that cannot be said was written", err)
	}
}

// Before anything is made, render refuses a name the menu does not
// carry, an entry that is drawn at none of the plan's details (a fern
// is quick only, and the plan here asks for demo), and a place it
// cannot say its count to.
func TestRenderRefusesBeforeItMakesAnything(test *testing.T) {
	output := filepath.Join(test.TempDir(), "book")
	refuseInvocation(test, output, "render", "zebra", "--out", output)
	demo := altered(test, "details: [quick]", "details: [demo]")
	var said, wrong bytes.Buffer
	status := tool(context.Background()).Run([]string{"render", "fern",
		"--plan", demo, "--out", output}, &said, &wrong)
	if status == successfulExit ||
		!strings.Contains(wrong.String(), "fern supports none of [demo]") {
		test.Errorf("a fern at demo: %q, %q", said.String(), wrong.String())
	}
	wrong.Reset()
	if status := tool(context.Background()).Run([]string{"render", "webb",
		"--dry-run", "--out", output}, refusing(test), &wrong); status == successfulExit {
		test.Error("a dry run said its count where nothing can be written")
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		test.Error("a refused render created output", err)
	}
}

// A book is not started under a file, where no folder can be made. A
// specimen that cannot be built, here for a face limit of one, is
// written into the book as failed with why; the run goes on, and ends
// non-zero with the failure named. The anti-test: the same book at the
// plan's own limit has no failed card.
func TestABookSaysWhichSpecimenFailedAndGoesOn(test *testing.T) {
	root := test.TempDir()
	blocker := filepath.Join(root, "a-file")
	if err := os.WriteFile(blocker, []byte("in the way"), artifactMode); err != nil {
		test.Fatal(err)
	}
	under := filepath.Join(blocker, "deeper", "book")
	refuseInvocation(test, under, "render", "webb", "--out", under)

	output := filepath.Join(root, "book")
	tiny := altered(test, "face-limit: 1500000", "face-limit: 1")
	var said, wrong bytes.Buffer
	status := tool(context.Background()).Run([]string{"render", "webb",
		"fern", "--plan", tiny, "--out", output}, &said, &wrong)
	if status == successfulExit {
		test.Fatal("a specimen over its face limit was drawn")
	}
	for _, folder := range []string{"webb-quick", "fern-quick"} {
		if !strings.Contains(wrong.String(), folder+": ") {
			test.Errorf("the run did not name %s: %q", folder, wrong.String())
		}
		if _, err := os.Stat(filepath.Join(output, folder)); !os.IsNotExist(err) {
			test.Error("a failed specimen has a folder", err)
		}
	}
	record, err := os.ReadFile(filepath.Join(output, "catalogue.json"))
	if err != nil || strings.Count(string(record), `"failed"`) != twoCards {
		test.Errorf("the book does not say both specimens failed: %v", err)
	}
}

// A specimen is not framed with no extent or an endless one, with no
// up, or through a lens of no width; the anti-test is a metre cube
// with an up and the plan's lens, which is.
func TestAViewNeedsAnExtentAnUpAndALens(test *testing.T) {
	plan := defaultPlan(test)
	world := thing.NewWorld()
	cube := maths.Box{Min: maths.V(0, 0, 0), Max: maths.V(1, 1, 1)}
	endless := maths.Box{Min: maths.V(0, 0, 0), Max: maths.V(math.Inf(1), 1, 1)}
	if made, err := plan.view(world, cube, 12, 0, maths.YAxis); err != nil ||
		made == nil || made.faces != 12 {
		test.Fatalf("a metre cube was not framed: %v", err)
	}
	for name, box := range map[string]maths.Box{"none": {}, "endless": endless} {
		if made, err := plan.view(world, box, 0, 0, maths.YAxis); err == nil ||
			made != nil || !strings.Contains(err.Error(), "extent") {
			test.Errorf("an extent of %s was framed: %v", name, err)
		}
	}
	if made, err := plan.view(world, cube, 0, 0, maths.Vec{}); err == nil || made != nil {
		test.Errorf("a view with no up was framed: %v", err)
	}
	plan.Wide = 0
	if made, err := plan.view(world, cube, 0, 0, maths.YAxis); err == nil || made != nil {
		test.Errorf("a lens of no width framed a view: %v", err)
	}
}

// The lamps a specimen carries are lit at the plan's exposure, and an
// exposure that is not above nought lights none and is refused; the
// anti-test is the plan's own exposure, which lights every one of the
// Larson's.
func TestCarriedLampsNeedAnExposureAboveNought(test *testing.T) {
	row, err := manufacture.KindNamed("mcrn-larson")
	if err != nil {
		test.Fatal(err)
	}
	made, err := manufacture.MakeWithComplexity(row, manufacture.Sow(5), "quick")
	if err != nil {
		test.Fatal(err)
	}
	carried, err := manufacture.PlaceLights(made.Descriptions(), maths.Local())
	if err != nil || len(carried) == 0 {
		test.Fatalf("the Larson carries %d lamps, %v", len(carried), err)
	}
	plan := defaultPlan(test)
	lit := thing.NewWorld()
	if err := plan.illuminate(lit, carried); err != nil {
		test.Fatal(err)
	}
	if got, want := len(lit.Lamps), len(plan.Sources)+len(carried); got != want {
		test.Errorf("%d lamps lit, want %d", got, want)
	}
	plan.LightExposure = 0
	if err := plan.illuminate(thing.NewWorld(), carried); err == nil {
		test.Error("carried lamps were lit at an exposure of nought")
	}
}

// Stored faces are a mesh's own count. The anti-test: a shape that is
// no mesh stores none.
func TestStoredFacesAreAMeshsOwn(test *testing.T) {
	two := &shape.Mesh{Vertices: []maths.Vec{maths.V(0, 0, 0), maths.V(1, 0, 0),
		maths.V(1, 1, 0), maths.V(0, 1, 0)}, Faces: [][3]int{{0, 1, 2}, {0, 2, 3}}}
	if got := storedFaces(two); got != 2 {
		test.Errorf("a mesh of two faces stores %d", got)
	}
	if got := storedFaces(nil); got != noEntries {
		test.Errorf("no shape stores %d faces", got)
	}
}

// A picture with light that is not a number, or coverage past all of a
// pixel, is no specimen.
func TestLightAndCoverageMustBeNumbersInRange(test *testing.T) {
	plan := defaultPlan(test)
	for name, spoil := range map[string]func(*frame.Frame){
		"light that is not a number": func(picture *frame.Frame) {
			picture.Pix[0].R = float32(math.NaN())
		},
		"endless light": func(picture *frame.Frame) {
			picture.Pix[0].G = float32(math.Inf(1))
		},
		"coverage past all": func(picture *frame.Frame) {
			picture.Pix[0].Cover = 2
		},
		"coverage below none": func(picture *frame.Frame) {
			picture.Pix[0].Cover = -1
		},
	} {
		picture := frame.New(singlePixel, singlePixel)
		picture.Pix[0].Cover, picture.Pix[0].Depth = fullCoverage, singlePixel
		if err := plan.visible(picture); err != nil {
			test.Fatal("a sound pixel was refused", err)
		}
		spoil(picture)
		if err := plan.visible(picture); err == nil ||
			!strings.Contains(err.Error(), "invalid specimen light or coverage") {
			test.Errorf("%s: %v", name, err)
		}
	}
}
