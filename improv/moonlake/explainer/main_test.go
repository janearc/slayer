package main

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// the files of a small show: sixteen beats half a second apart from 2 s,
// and a script whose one short is the second from 2 s to 3 s.
const (
	smallScore = `score:
  beats:
    times: [2.0, 2.5, 3.0, 3.5, 4.0, 4.5, 5.0, 5.5, 6.0, 6.5, 7.0, 7.5,
            8.0, 8.5, 9.0, 9.5]
    measured: [1, 1, 1, 1, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1]
`
	smallCues = `cues:
  - {at: "1.1", strobes: grid8, duty: 0.1}
  - {at: "2.1", rig: 1}
`
	smallScript = `shorts:
  tiny: {from: {at_sec: 2}, until: {at_sec: 3}}
says:
  - {at_sec: 2, says: "A sentence."}
`
)

// room is a directory with the small show's score, cues and script.
func room(t *testing.T) paths {
	t.Helper()
	dir := t.TempDir()
	asked := paths{score: filepath.Join(dir, "score.yaml"),
		cues: filepath.Join(dir, "cues.yaml"), script: filepath.Join(dir, "explain.yaml")}
	for path, text := range map[string]string{asked.score: smallScore,
		asked.cues: smallCues, asked.script: smallScript} {
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return asked
}

// song writes a mono 16-bit wav of that many seconds at 8000 a second
// and is its path.
func song(t *testing.T, seconds int) string {
	t.Helper()
	const rate = 8000
	var data []byte
	for i := 0; i < rate*seconds; i++ {
		data = binary.LittleEndian.AppendUint16(data, uint16(int16((i%80)*200)))
	}
	var out []byte
	out = append(out, "RIFF"...)
	out = binary.LittleEndian.AppendUint32(out, uint32(36+len(data)))
	out = append(out, "WAVEfmt "...)
	out = binary.LittleEndian.AppendUint32(out, 16)
	out = binary.LittleEndian.AppendUint16(out, 1)
	out = binary.LittleEndian.AppendUint16(out, 1)
	out = binary.LittleEndian.AppendUint32(out, rate)
	out = binary.LittleEndian.AppendUint32(out, rate*2)
	out = binary.LittleEndian.AppendUint16(out, 2)
	out = binary.LittleEndian.AppendUint16(out, 16)
	out = append(out, "data"...)
	out = binary.LittleEndian.AppendUint32(out, uint32(len(data)))
	out = append(out, data...)
	path := filepath.Join(t.TempDir(), "song.wav")
	if err := os.WriteFile(path, out, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// explainerFor builds from the three files, and names which one it could
// not read when one is missing.
func TestExplainerForReadsTheThreeFiles(t *testing.T) {
	asked := room(t)
	made, err := explainerFor(asked)
	if err != nil {
		t.Fatal(err)
	}
	from, until, ok := made.Script().Short("tiny")
	if !ok || from != 2 || until != 3 {
		t.Errorf("tiny is %v to %v (%v)", from, until, ok)
	}
	for _, missing := range []string{"score", "cues", "script"} {
		broken := asked
		switch missing {
		case "score":
			broken.score += ".gone"
		case "cues":
			broken.cues += ".gone"
		case "script":
			broken.script += ".gone"
		}
		if _, err := explainerFor(broken); err == nil {
			t.Errorf("explainerFor accepted a missing %s", missing)
		}
	}
}

// songSpan is the song between two moments: a second at 44100 a second,
// two channels, and an error where the song ends early or is missing.
func TestSongSpanCutsTheSong(t *testing.T) {
	path := song(t, 3)
	cut, err := songSpan(path, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if cut.Rate() != 44100 || cut.Length() != 44100 {
		t.Errorf("span is %d samples at %d", cut.Length(), cut.Rate())
	}
	if _, err := songSpan(path, 1, 9); err == nil ||
		!strings.Contains(err.Error(), "ends before") {
		t.Errorf("a span past the end gave %v", err)
	}
	if _, err := songSpan(path+".gone", 1, 2); err == nil {
		t.Error("songSpan accepted a missing song")
	}
}
