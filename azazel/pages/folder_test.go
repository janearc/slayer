package pages

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// largePage is the size of a page written while it is read.
const largePage = 1 << 20 // picked: a megabyte takes several writes to land

// scriptPages is every folder page frames-index.sh wrote from the
// fixture: all but the hand page it kept.
var scriptPages = []string{
	"frames/2026-11-01-studio-sky",
	"frames/2026-11-03-fable-moonlake",
	"frames/2026-11-05-studio-many",
	"frames/2026-11-06-studio-empty",
	"frames/2026-11-08-team-r&d",
	"animations/2026-11-02-team-heard-film",
	"animations/2026-11-07-atelier-chunks",
}

// asScript is azazel's folder page with the script's mark in place of
// its own, the one line the two pages are meant to differ by.
func asScript(page string) string {
	return strings.Replace(page, folderMark, scriptMark, 1)
}

// TestFolderPagesMatchTheScript compares each folder page with what
// frames-index.sh wrote from the same tree, byte for byte but the mark.
func TestFolderPagesMatchTheScript(t *testing.T) {
	root := fixture(t)
	for _, name := range scriptPages {
		found, err := scanFolder(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		page := folderPage(filepath.Base(name), found, nil)
		if asScript(page) != expected(t, "folders/"+name+".html") {
			t.Errorf("%s differs from frames-index.sh's:\n%s", name, page)
		}
	}
}

// TestFolderPageComparisonSeesOnePicture is the anti-test: one picture
// more and the page no longer matches the script's.
func TestFolderPageComparisonSeesOnePicture(t *testing.T) {
	root := fixture(t)
	name := "frames/2026-11-08-team-r&d"
	lay(t, root, fixtureNoon.Unix(), name+"/two.png")
	found, err := scanFolder(filepath.Join(root, name))
	if err != nil {
		t.Fatal(err)
	}
	page := folderPage(filepath.Base(name), found, nil)
	if asScript(page) == expected(t, "folders/"+name+".html") {
		t.Fatal("a changed folder still matched the script's page")
	}
}

// publishOnce publishes one folder's page as a tick does.
func publishOnce(t *testing.T, root string, folder Folder) string {
	t.Helper()
	found, err := scanFolder(folder.path(root))
	if err != nil {
		t.Fatal(err)
	}
	hands := &clock{at: fixtureNoon}
	notice, err := publisherOf(settingsFor(root, nil, hands)).
		publishFolder(folder, found, filmView{})
	if err != nil {
		t.Fatal(err)
	}
	return notice
}

// handFolder is the fixture's folder whose page was made by hand.
var handFolder = Folder{
	Shelf: framesShelf,
	Name:  "2026-11-04-workshop-salt&pepper",
}

// TestHandPageIsSetAsideOnceAndNeverOverwritten moves the hand page aside,
// links it, and then keeps a second hand page where it is.
func TestHandPageIsSetAsideOnceAndNeverOverwritten(t *testing.T) {
	root := fixture(t)
	dir := handFolder.path(root)
	hand := read(t, filepath.Join(dir, pageName))
	if notice := publishOnce(t, root, handFolder); notice != "" {
		t.Fatal(notice)
	}
	page := read(t, filepath.Join(dir, pageName))
	if read(t, filepath.Join(dir, handPageName)) != hand ||
		!strings.Contains(page, folderMark) ||
		!strings.Contains(page, `href="`+handPageName+`"`) {
		t.Fatal("the hand page was not set aside and linked", page)
	}
	second := "<p>a second page by hand</p>\n"
	err := os.WriteFile(filepath.Join(dir, pageName), []byte(second), listing)
	if err != nil {
		t.Fatal(err)
	}
	if notice := publishOnce(t, root, handFolder); notice != keptNotice {
		t.Fatal("no notice for a kept hand page:", notice)
	}
	if read(t, filepath.Join(dir, pageName)) != second ||
		read(t, filepath.Join(dir, handPageName)) != hand {
		t.Fatal("a hand page was written over")
	}
}

// TestOwnPagesAreWrittenOverInPlace is the anti-test: a page the script
// or azazel wrote is replaced, and nothing is set aside.
func TestOwnPagesAreWrittenOverInPlace(t *testing.T) {
	root := fixture(t)
	folder := Folder{Shelf: animationsShelf, Name: "2026-11-02-team-heard-film"}
	publishOnce(t, root, folder)
	publishOnce(t, root, folder)
	dir := folder.path(root)
	if !strings.Contains(read(t, filepath.Join(dir, pageName)), folderMark) {
		t.Fatal("the script's page was not replaced")
	}
	if _, err := os.Lstat(filepath.Join(dir, handPageName)); err == nil {
		t.Fatal("a page of the script's was set aside as a hand page")
	}
}

// TestHandPageLinkedBeforeAFailedWriteIsStillReplaced: a link made just
// before a write that failed is the same file, not another hand page.
func TestHandPageLinkedBeforeAFailedWriteIsStillReplaced(t *testing.T) {
	root := fixture(t)
	dir := handFolder.path(root)
	err := os.Link(filepath.Join(dir, pageName),
		filepath.Join(dir, handPageName))
	if err != nil {
		t.Fatal(err)
	}
	if notice := publishOnce(t, root, handFolder); notice != "" {
		t.Fatal(notice)
	}
	if !strings.Contains(read(t, filepath.Join(dir, pageName)), folderMark) {
		t.Fatal("the folder page was not written")
	}
}

// TestWritesAreWholeForEveryReader rewrites a large page while a reader
// reads it: every read is one whole version, never part of one.
func TestWritesAreWholeForEveryReader(t *testing.T) {
	path := filepath.Join(t.TempDir(), pageName)
	versions := []string{strings.Repeat("a", largePage),
		strings.Repeat("b", largePage)}
	if err := writeBeside(path, []byte(versions[0]), written); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for index := range 50 { // picked: enough writes to meet a read
			if err := writeBeside(path, []byte(versions[index%2]),
				written); err != nil {
				t.Error(err)
			}
		}
	}()
	torn := 0
	for reading := true; reading; {
		select {
		case <-done:
			reading = false
		default:
		}
		data, err := os.ReadFile(path)
		if err != nil || string(data) != versions[0] &&
			string(data) != versions[1] {
			torn++
		}
	}
	if torn > 0 {
		t.Fatalf("a reader saw part of a page %d times", torn)
	}
}

// TestFailedWriteLeavesThePageAndNoPart is the anti-test: a write that
// cannot be made leaves the old page whole and no temporary file.
func TestFailedWriteLeavesThePageAndNoPart(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, pageName)
	if err := writeBeside(path, []byte("old\n"), written); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o555); err != nil { // defined: read-only
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, folders) })
	if writeBeside(path, []byte("new\n"), written) == nil {
		t.Fatal("a write into a read-only folder succeeded")
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 || read(t, path) != "old\n" {
		t.Fatal("the failed write left something behind", entries, err)
	}
	if shown(filepath.Base(path)+partEnding) || isFilm(partEnding) {
		t.Fatal("a temporary name is counted as a picture or a film")
	}
}
