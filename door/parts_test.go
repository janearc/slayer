package door

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// a note line is the folder, one tab and the words, appended with one
// write every time, a folder's later line after its earlier one, since
// the page takes a folder's last line; a line break goes before it where
// the file does not end in one.
func TestNoteAlwaysAppends(t *testing.T) {
	notes := filepath.Join(t.TempDir(), "team", notesName)
	first, err := newEntry("frames/lake", "first")
	if err != nil {
		t.Fatal(err)
	}
	if err := first.into(notes); err != nil {
		t.Fatal(err)
	}
	written(t, filepath.Dir(notes), notesName,
		"frames/lake\tfirst\nframes/hill\thill")
	second, _ := newEntry("frames/lake", "second")
	if err := second.into(notes); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(notes)
	want := "frames/lake\tfirst\nframes/hill\thill\nframes/lake\tsecond\n"
	if err != nil || string(raw) != want {
		t.Errorf("the notes hold %q, not %q: %v", raw, want, err)
	}
}

// the notes file is never replaced, so a line another
// writer appends while a note is written is kept, and an appender
// holding the file open still writes into the file the page reads.
// anti-test of TestNoteAlwaysAppends
func TestNoteNeverRewritesTheFile(t *testing.T) {
	notes := written(t, t.TempDir(), notesName, "frames/lake\tfirst\n")
	before, err := os.Stat(notes)
	if err != nil {
		t.Fatal(err)
	}
	other, err := os.OpenFile(notes, os.O_APPEND|os.O_WRONLY, fileMode)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	again, _ := newEntry("frames/lake", "second")
	if err := again.into(notes); err != nil {
		t.Fatal(err)
	}
	if _, err := other.WriteString("frames/sea\tsea\n"); err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(notes)
	if err != nil || !os.SameFile(before, after) {
		t.Fatalf("the notes file was replaced: %v", err)
	}
	raw, _ := os.ReadFile(notes)
	want := "frames/lake\tfirst\nframes/lake\tsecond\nframes/sea\tsea\n"
	if string(raw) != want {
		t.Errorf("the notes hold %q, not %q", raw, want)
	}
}

// a note or a folder with a tab or a line break, or none,
// is refused.
// anti-test of TestNoteAlwaysAppends
func TestNoteRefusesWhatIsNotOneLine(t *testing.T) {
	for _, each := range [][2]string{{"a\tb", "words"},
		{"folder", "two\nlines"}, {"folder", "a\ttab"}, {"", "words"},
		{"folder", ""}} {
		if _, err := newEntry(each[0], each[1]); err == nil {
			t.Errorf("the note %q was taken", each)
		}
	}
}

// a ledger of another key is stale: the frames it lists are removed and
// a new ledger is written; a file it does not list, or a name that is
// not a frame's, is never removed.
func TestLedgerDiscardsStaleFramesOnly(t *testing.T) {
	parent := t.TempDir()
	folder := filepath.Join(parent, "film")
	if err := os.MkdirAll(folder, folderMode); err != nil {
		t.Fatal(err)
	}
	frame := fmt.Sprintf(frameName, nothing)
	written(t, folder, frame, "a stale frame")
	written(t, folder, "keep.txt", "not the door's")
	escape := written(t, parent, "escape.png", "outside the folder")
	written(t, folder, manifestName, `{"key": "theirs", "frames": {"`+
		frame+`": "x", "../escape.png": "x", "keep.txt": "x"}}`)
	book, stale, err := openLedger(folder, "ours")
	if err != nil || !stale {
		t.Fatalf("a stale ledger gave %v, stale %v", err, stale)
	}
	if _, err := os.Stat(filepath.Join(folder, frame)); err == nil {
		t.Error("a stale frame was kept")
	}
	for _, left := range []string{filepath.Join(folder, "keep.txt"),
		escape} {
		if _, err := os.Stat(left); err != nil {
			t.Errorf("%s, which is no frame of the door's, was removed",
				left)
		}
	}
	if book.record.Key != "ours" || len(book.record.Frames) != nothing {
		t.Errorf("the new ledger is %+v", book.record)
	}
}

// a stale ledger in a folder holding a picture it does
// not list is refused by that picture's name, and nothing is removed.
// anti-test of TestLedgerDiscardsStaleFramesOnly
func TestLedgerRefusesAStrangersPicture(t *testing.T) {
	folder := t.TempDir()
	frame := fmt.Sprintf(frameName, nothing)
	written(t, folder, frame, "a stale frame")
	written(t, folder, "holiday.png", "someone's")
	written(t, folder, manifestName, `{"key": "theirs", "frames": {"`+
		frame+`": "x"}}`)
	if _, _, err := openLedger(folder, "ours"); err == nil ||
		!strings.Contains(err.Error(), "holiday.png") {
		t.Errorf("a stranger's picture gave %v", err)
	}
	for _, name := range []string{frame, "holiday.png", manifestName} {
		if _, err := os.Stat(filepath.Join(folder, name)); err != nil {
			t.Errorf("%s was removed", name)
		}
	}
}

// a ledger that does not read is refused; a frame it never recorded is
// not kept.
// anti-test of TestLedgerIsWrittenBeforeAnyFrame
func TestLedgerRefusesWhatDoesNotRead(t *testing.T) {
	folder := t.TempDir()
	written(t, folder, manifestName, "{")
	if _, _, err := openLedger(folder, "ours"); err == nil ||
		!strings.Contains(err.Error(), "empty the folder") {
		t.Errorf("a broken ledger gave %v", err)
	}
	written(t, folder, manifestName, `{"key": "ours"}`)
	book, _, err := openLedger(folder, "ours")
	if err != nil {
		t.Fatal(err)
	}
	if book.kept(fmt.Sprintf(frameName, nothing)) {
		t.Error("a frame never recorded was kept")
	}
}

// the advice a broken ledger gives works: a folder emptied of it and its
// frames is drawn into again; following it only half, moving the ledger
// aside and leaving the frames, is refused by name.
func TestLedgerAdviceWorks(t *testing.T) {
	folder := t.TempDir()
	written(t, folder, manifestName, "{")
	written(t, folder, fmt.Sprintf(frameName, nothing), "a frame")
	// anti-test of TestLedgerAdviceWorks
	if err := os.Rename(filepath.Join(folder, manifestName),
		filepath.Join(folder, "aside.json")); err != nil {
		t.Fatal(err)
	}
	if _, _, err := openLedger(folder, "ours"); err == nil ||
		!strings.Contains(err.Error(), "holds pictures") {
		t.Errorf("a ledger moved aside gave %v", err)
	}
	if err := os.RemoveAll(folder); err != nil {
		t.Fatal(err)
	}
	if _, _, err := openLedger(folder, "ours"); err != nil {
		t.Errorf("an emptied folder was refused: %v", err)
	}
}

// a new folder's ledger is written when the folder is made, before any
// frame, so a run killed during its first frame leaves a folder the next
// run resumes; a frame written and not yet recorded is drawn again.
func TestLedgerIsWrittenBeforeAnyFrame(t *testing.T) {
	folder := filepath.Join(t.TempDir(), "film")
	if _, _, err := openLedger(folder, "ours"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(folder, manifestName)); err != nil {
		t.Fatalf("a new folder has no ledger: %v", err)
	}
	name := fmt.Sprintf(frameName, nothing)
	written(t, folder, name, "a frame the kill left unrecorded")
	book, _, err := openLedger(folder, "ours")
	if err != nil {
		t.Fatalf("a folder killed in its first frame was refused: %v", err)
	}
	if book.kept(name) {
		t.Error("a frame never recorded was kept")
	}
}

// copying a file onto itself leaves it alone; onto another path, the
// path holds its bytes.
func TestCopyFile(t *testing.T) {
	directory := t.TempDir()
	from := written(t, directory, "job.yaml", "words")
	if err := copyFile(from, from); err != nil {
		t.Fatal(err)
	}
	to := filepath.Join(directory, "deeper", "job.yaml")
	if err := copyFile(from, to); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(to)
	if err != nil || string(raw) != "words" {
		t.Errorf("the copy holds %q: %v", raw, err)
	}
	if err := copyFile(filepath.Join(directory, "gone"), to); err == nil {
		t.Error("a file not there was copied")
	}
}

// every verb is declared soundly, and its example holds its output and
// passes --dry-run.
func TestTheExamplesHere(t *testing.T) {
	env := envIn(t, context.Background(), t.TempDir())
	if err := Tool(env).Examples(os.DirFS(".")); err != nil {
		t.Error(err)
	}
}

// draw's example: a still of widow, drawn dry, says where its picture
// would go and writes nothing.
func Example_draw() {
	here, _ := os.Getwd()
	env, _ := NewEnv(Given{Context: context.Background(),
		Environment: func(string) (string, bool) { return "", false },
		Now:         time.Now, Directory: here})
	var said bytes.Buffer
	Tool(env).Run([]string{"draw", "../sets/widow.yaml", "--dry-run"},
		&said, &said)
	for _, line := range strings.Split(said.String(), "\n") {
		if strings.HasPrefix(line, "would write") {
			fmt.Println(strings.ReplaceAll(line, here, "."))
		}
	}
	// Output: would write picture ./out/widow.png
}

// manyFrames is a long video's frames, for the ledger's cost. Picked by
// the door slice's author, 2026-11-12: several minutes at 24 a second.
const manyFrames = 6000

// BenchmarkLedgerNoteAtSixThousand is the cost of recording one frame in
// a ledger already holding manyFrames: 3.6 ms on this machine.
func BenchmarkLedgerNoteAtSixThousand(b *testing.B) {
	book, _, err := openLedger(b.TempDir(), "ours")
	if err != nil {
		b.Fatal(err)
	}
	var sum [sha256.Size]byte
	for index := range manyFrames {
		book.record.Frames[fmt.Sprintf(frameName, index)] = "sum"
	}
	for b.Loop() {
		if err := book.note(fmt.Sprintf(frameName, nothing), sum); err != nil {
			b.Fatal(err)
		}
	}
}
