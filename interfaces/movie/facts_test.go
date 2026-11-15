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

// wideBox is a box whose size is written in 64 bits after a size of one.
func wideBox(kind string, body int) []byte {
	out := binary.BigEndian.AppendUint32(nil, boxLarger)
	out = append(out, kind...)
	out = binary.BigEndian.AppendUint64(out, uint64(boxHeader+boxWide+body))
	return append(out, make([]byte, body)...)
}

// boxesAt writes boxes to a file and gives its path.
func boxesAt(t *testing.T, boxes ...[]byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), movieName)
	var all []byte
	for _, one := range boxes {
		all = append(all, one...)
	}
	if err := os.WriteFile(path, all, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// the index before the media is a fast start, past boxes of either
// size; a last box that runs to the end of the file is read too.
func TestTheIndexFirstIsAFastStart(t *testing.T) {
	toEnd := binary.BigEndian.AppendUint32(nil, boxToEnd)
	toEnd = append(toEnd, "moov"...)
	files := [][][]byte{
		{box("ftyp", 4), box("moov", 16), box("mdat", 32)},
		{wideBox("ftyp", 4), box("free", 0), box("moov", 1), box("mdat", 1)},
		{box("ftyp", 4), toEnd},
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
	path := boxesAt(t, box("ftyp", 4), box("mdat", 32), box("moov", 16))
	fast, err := fastStart(path)
	if err != nil || fast {
		t.Errorf("media first: fast %v, %v", fast, err)
	}
}

// a file with neither box, a box shorter than its header, a cut-off
// header, or no file at all is refused.
func TestFilesThatAreNotMoviesAreRefused(t *testing.T) {
	short := binary.BigEndian.AppendUint32(nil, boxSize)
	cutWide := binary.BigEndian.AppendUint32(nil, boxLarger)
	refusals := map[string]string{
		boxesAt(t, box("ftyp", 4)):                      "not a finished mp4",
		boxesAt(t, append(short, "ftyp"...)):            "says it is 4 bytes",
		boxesAt(t, append(cutWide, "moov"...)):          "not a finished",
		boxesAt(t, box("ftyp", 4), []byte("moo")):       "EOF",
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
