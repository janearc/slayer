package pages

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

// galleryNames pairs each gallery page with the scripts' page for it.
var galleryNames = map[string]string{
	latestName:                       "latest.html",
	framesShelf + "/" + pageName:     "frames-index.html",
	animationsShelf + "/" + pageName: "animations-index.html",
}

// publishFixtureGallery publishes the fixture's gallery and returns its
// root.
func publishFixtureGallery(t *testing.T) string {
	t.Helper()
	root := fixture(t)
	hands := &clock{at: fixtureNoon}
	if err := publisherOf(settingsFor(root, nil, hands)).
		publishGallery(); err != nil {
		t.Fatal(err)
	}
	return root
}

// TestGalleryMatchesTheScripts compares latest.html and both indexes
// with what latest.sh and indexes.sh wrote from the same tree, byte for
// byte but for the time each was made.
func TestGalleryMatchesTheScripts(t *testing.T) {
	root := publishFixtureGallery(t)
	for name, script := range galleryNames {
		got := withoutMade(read(t, filepath.Join(root, name)))
		if want := withoutMade(expected(t, script)); got != want {
			t.Errorf("%s differs from %s:\n%s", name, script, got)
		}
	}
	hands := &clock{at: fixtureNoon}
	found := gallery{root: root, when: hands.now(), zone: time.UTC}
	var err error
	found.files, err = walkGallery(root)
	if err == nil {
		found.notes, err = readNotes(root)
	}
	page, wrong := found.html(galleryPage{name: latestName,
		title: "Latest renders", most: 3})
	if err != nil || wrong != nil ||
		withoutMade(page) != withoutMade(expected(t, "latest-3.html")) {
		t.Fatal("three cards differ from latest.sh 3", err, wrong, page)
	}
}

// TestGalleryComparisonSeesOneSecond is the anti-test: one picture a
// second newer changes which picture its card shows, and the page no
// longer matches the script's.
func TestGalleryComparisonSeesOneSecond(t *testing.T) {
	root := fixture(t)
	later := time.Unix(1794000251, 0) // picked: one second after a.png
	picture := filepath.Join(root, "frames/2026-11-01-studio-sky/B.png")
	if err := os.Chtimes(picture, later, later); err != nil {
		t.Fatal(err)
	}
	hands := &clock{at: fixtureNoon}
	if err := publisherOf(settingsFor(root, nil, hands)).
		publishGallery(); err != nil {
		t.Fatal(err)
	}
	got := withoutMade(read(t, filepath.Join(root, latestName)))
	if got == withoutMade(expected(t, "latest.html")) {
		t.Fatal("a changed tree still matched the script's page")
	}
}

// TestSetFileCopiesAreWrittenBesideThem checks the NAME-yaml.txt copies a
// card's button opens.
func TestSetFileCopiesAreWrittenBesideThem(t *testing.T) {
	root := publishFixtureGallery(t)
	for _, set := range []string{
		"frames/2026-11-01-studio-sky/sky.yaml",
		"frames/2026-11-03-fable-moonlake/moon.yaml",
		"frames/2026-11-03-fable-moonlake/shorts/one/one.yml",
	} {
		path := filepath.Join(root, set)
		copied := path[:len(path)-len(filepath.Ext(path))] + "-yaml.txt"
		if read(t, copied) != read(t, path) {
			t.Errorf("%s was not copied", set)
		}
	}
}

// TestSetFileCopyNewerThanItsSetIsKept is the anti-test: a copy newer
// than its set file is left as it is, as cp under -nt leaves it.
func TestSetFileCopyNewerThanItsSetIsKept(t *testing.T) {
	root := fixture(t)
	copied := filepath.Join(root, "frames/2026-11-01-studio-sky/sky-yaml.txt")
	if err := os.WriteFile(copied, []byte("kept\n"), listing); err != nil {
		t.Fatal(err)
	}
	hands := &clock{at: fixtureNoon}
	if err := publisherOf(settingsFor(root, nil, hands)).
		publishGallery(); err != nil {
		t.Fatal(err)
	}
	if read(t, copied) != "kept\n" {
		t.Fatal("a newer copy was written over")
	}
}

// measuredOrder is the Mac sort's order of these names, as measured.
var measuredOrder = []string{"a b", "a_b", "a-b", "a.b", "a/b", "a0", "a9",
	"ab", "aB", "Ab", "AB", "az"}

// TestCollationIsTheMacSortOrder sorts a list as the scripts' sort did.
func TestCollationIsTheMacSortOrder(t *testing.T) {
	names := slices.Clone(measuredOrder)
	slices.Reverse(names)
	slices.SortFunc(names, collate)
	if !slices.Equal(names, measuredOrder) {
		t.Fatal(names)
	}
	if collate("Aa", "aB") >= 0 || collate("a-c", "ab") >= 0 {
		t.Fatal("case is not set aside before it is weighed")
	}
}

// TestByteOrderIsNotTheMacSortOrder is the anti-test: the order of the
// bytes is another order, so the collation is what the match rests on.
func TestByteOrderIsNotTheMacSortOrder(t *testing.T) {
	names := slices.Clone(measuredOrder)
	slices.Sort(names)
	if slices.Equal(names, measuredOrder) {
		t.Fatal("bytes and the Mac sort agree on this list")
	}
}
