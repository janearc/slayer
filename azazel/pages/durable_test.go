package pages

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"go.uber.org/mock/gomock"
)

// laySilently makes an empty file for a sweep test, with its folder.
func laySilently(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), folders); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("x"), listing); err != nil {
		t.Fatal(err)
	}
}

// namesIn lists the names a folder holds, sorted.
func namesIn(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names
}

// The temporaries this package makes are removed, and nothing else is.
func TestLeftoverTemporariesAreRemovedAndNothingElse(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{".index.html.123.part", ".partial.mp4.9.part",
		".partial.json.1.part", ".render-progress.txt.4.part",
		".latest.html.8.part", ".set-yaml.txt.5.part"} {
		laySilently(t, filepath.Join(dir, name))
	}
	kept := []string{".frame.png.7.part", ".index.html.123", "frame-000000.png",
		"index.html", "partial.mp4"}
	for _, name := range kept {
		laySilently(t, filepath.Join(dir, name))
	}
	if err := removeLeftovers(dir); err != nil {
		t.Fatal(err)
	}
	if left := namesIn(t, dir); !slices.Equal(left, kept) {
		t.Fatal("the sweep left", left)
	}
}

// Anti-test: a folder that is not there has nothing left over, and a
// folder that cannot be read is an error.
func TestSweepOfAMissingFolderIsQuietAndOfAFileFails(t *testing.T) {
	absent := filepath.Join(t.TempDir(), "absent")
	if err := removeLeftovers(absent); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "plain")
	laySilently(t, file)
	if err := removeLeftovers(file); err == nil {
		t.Fatal("a file was swept as a folder")
	}
}

// A coordinator sweeps the gallery's folders when it starts and a render
// folder the first time it meets it, and not again while its job runs.
func TestCoordinatorSweepsAtStartAndOncePerFolder(t *testing.T) {
	root, hands := renders(t), &clock{at: fixtureNoon}
	stills(t, root, pair[0], 0, 1)
	folder := pair[0].path(root)
	rootLeft := filepath.Join(root, ".latest.html.3.part")
	shelfLeft := filepath.Join(root, framesShelf, ".index.html.3.part")
	firstLeft := filepath.Join(folder, ".index.html.1.part")
	for _, path := range []string{rootLeft, shelfLeft, firstLeft} {
		laySilently(t, path)
	}
	var log bytes.Buffer
	coordinator := New(settingsFor(root, nil, hands), &log)
	for _, path := range []string{rootLeft, shelfLeft} {
		if _, err := os.Lstat(path); !errors.Is(err, fs.ErrNotExist) {
			t.Fatal("the gallery's leftover was kept", path)
		}
	}
	coordinator.Start(jobFor(1, root, pair[0], 4))
	if _, err := os.Lstat(firstLeft); !errors.Is(err, fs.ErrNotExist) {
		t.Fatal("the folder's leftover was kept")
	}
	second := filepath.Join(folder, ".index.html.2.part")
	laySilently(t, second)
	coordinator.Start(jobFor(2, root, pair[0], 4))
	if _, err := os.Lstat(second); err != nil {
		t.Fatal("a live temporary was swept by a second job", err)
	}
}

// A new partial film and its record agree: after an encode the record is
// the count the film was made from.
func TestEncodeLeavesTheNewFilmWithItsOwnCount(t *testing.T) {
	root, folder := filmFolder(t, upTo(6), "6")
	dir := folder.path(root)
	laySilently(t, filepath.Join(dir, partialName))
	if err := os.WriteFile(filepath.Join(dir, partialRecord),
		[]byte(`{"frames":3,"rate":6,"seconds":0.5}`), listing); err != nil {
		t.Fatal(err)
	}
	encoder := NewMockEncoder(gomock.NewController(t))
	expectFilm(encoder, root, folder, 6)
	made := publisherOf(Settings{Renders: root, Encoder: encoder})
	if err := made.encode(t.Context(), dir, viewOf(t, root, folder, nil)); err != nil {
		t.Fatal(err)
	}
	if got := readPartial(dir); got.Frames != 6 || read(t, filepath.Join(dir, partialName)) != "film" {
		t.Fatal("the film and its record disagree", got)
	}
}

// Anti-test: when the old record cannot be withdrawn the old film is left
// as it was, so a record never sits beside a film it was not made from.
func TestEncodeKeepsTheOldFilmWhenTheRecordCannotBeWithdrawn(t *testing.T) {
	root, folder := filmFolder(t, upTo(6), "6")
	dir := folder.path(root)
	if err := os.WriteFile(filepath.Join(dir, partialName), []byte("old"), listing); err != nil {
		t.Fatal(err)
	}
	// A record that is a folder with something in it cannot be removed.
	laySilently(t, filepath.Join(dir, partialRecord, "held"))
	encoder := NewMockEncoder(gomock.NewController(t))
	expectFilm(encoder, root, folder, 6)
	made := publisherOf(Settings{Renders: root, Encoder: encoder})
	if err := made.encode(t.Context(), dir, viewOf(t, root, folder, nil)); err == nil {
		t.Fatal("an encode went on with a record it could not withdraw")
	}
	if read(t, filepath.Join(dir, partialName)) != "old" {
		t.Fatal("the new film replaced the old beside its old record")
	}
	for _, name := range namesIn(t, dir) {
		if leftover(name) {
			t.Fatal("a temporary was left", name)
		}
	}
}

// A written file is whole and carries its mode; syncing a path that is not
// there is an error rather than silence.
func TestWriteBesideIsWholeAndSyncReportsAMissingPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "index.html")
	if err := writeBeside(path, []byte("page"), written); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != written || read(t, path) != "page" {
		t.Fatal(info, err)
	}
	if left := namesIn(t, dir); !slices.Equal(left, []string{"index.html"}) {
		t.Fatal(left)
	}
	if err := syncPath(filepath.Join(dir, "absent")); err == nil {
		t.Fatal("a missing path was synced")
	}
	if err := syncPath(dir); err != nil {
		t.Fatal(err)
	}
}
