package movie

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// box is the bytes of one MPEG-4 box of that kind holding body bytes.
func box(kind string, body int) []byte {
	out := binary.BigEndian.AppendUint32(nil, uint32(boxHeader+body))
	return append(append(out, kind...), make([]byte, body)...)
}

// ftypBox is a file-type box of the size the tests pick.
func ftypBox() []byte { return box("ftyp", picked("ftyp-body")) }

// moovBox is an index box of the size the tests pick.
func moovBox() []byte { return box("moov", picked("moov-body")) }

// mdatBox is a media box of the size the tests pick.
func mdatBox() []byte { return box("mdat", picked("mdat-body")) }

// wideBox is a box whose size is written in 64 bits after a size of one.
func wideBox(kind string, body int) []byte {
	out := binary.BigEndian.AppendUint32(nil, boxLarger)
	out = append(out, kind...)
	out = binary.BigEndian.AppendUint64(out, uint64(boxHeader+boxWide+body))
	return append(out, make([]byte, body)...)
}

// toEndBox is the start of a box of that kind whose size says it runs to
// the end of the file.
func toEndBox(kind string) []byte {
	out := binary.BigEndian.AppendUint32(nil, boxToEnd)
	return append(out, kind...)
}

// boxesAt writes boxes to a file and gives its path.
func boxesAt(t *testing.T, boxes ...[]byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), movieName)
	var all []byte
	for _, each := range boxes {
		all = append(all, each...)
	}
	if err := os.WriteFile(path, all, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// the index before the media is a fast start, past boxes of either
// size; a last box that runs to the end of the file is read too.
func TestTheIndexFirstIsAFastStart(t *testing.T) {
	files := [][][]byte{
		{ftypBox(), moovBox(), mdatBox()},
		{wideBox("ftyp", picked("ftyp-body")), box("free", zero),
			box("moov", one), box("mdat", one)},
		{ftypBox(), toEndBox("moov")},
	}
	for place, boxes := range files {
		fast, err := fastStart(boxesAt(t, boxes...))
		if err != nil || !fast {
			t.Errorf("file %d: fast %v, %v", place, fast, err)
		}
	}
}

// the anti-test: the media before the index is not a fast start.
func TestTheMediaFirstIsNotAFastStart(t *testing.T) {
	path := boxesAt(t, ftypBox(), mdatBox(), moovBox())
	fast, err := fastStart(path)
	if err != nil || fast {
		t.Errorf("media first: fast %v, %v", fast, err)
	}
}

// a box that runs to the end of the file ends the search: what is in it
// is not read as more boxes, even when it holds what looks like an index.
func TestABoxToTheEndIsNotReadInside(t *testing.T) {
	path := boxesAt(t, ftypBox(), toEndBox("free"), moovBox())
	fast, err := fastStart(path)
	refused := err != nil &&
		strings.Contains(err.Error(), "not a finished mp4")
	if fast || !refused {
		t.Errorf("an index inside a box to the end: fast %v, %v", fast, err)
	}
}

// a file with neither box, a box shorter than its header, a cut-off
// header, or no file at all is refused.
func TestFilesThatAreNotMoviesAreRefused(t *testing.T) {
	short := binary.BigEndian.AppendUint32(nil, boxSize)
	cutWide := binary.BigEndian.AppendUint32(nil, boxLarger)
	refusals := map[string]string{
		boxesAt(t, ftypBox()):                           "not a finished mp4",
		boxesAt(t, append(short, "ftyp"...)):            "says it is 4 bytes",
		boxesAt(t, append(cutWide, "moov"...)):          "not a finished",
		boxesAt(t, ftypBox(), []byte("moo")):            "EOF",
		filepath.Join(t.TempDir(), "absent", movieName): "no such file",
	}
	for path, says := range refusals {
		_, err := fastStart(path)
		if err == nil || !strings.Contains(err.Error(), says) {
			t.Errorf("%s: %v, want %q", path, err, says)
		}
	}
}

// codecs are named by their common names, and unknown ones by their
// four letters as written.
func TestCodecsAreNamed(t *testing.T) {
	names := map[string]string{"avc1": "h264", "aac ": "aac",
		"hvc1": "hvc1", "": ""}
	for letters, want := range names {
		if got := codecName(letters); got != want {
			t.Errorf("%q named %q, want %q", letters, got, want)
		}
	}
}
