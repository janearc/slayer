package cheeptune

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// How a game gets music, in two calls.
//
//	p, err := cheeptune.Open("@crypt", "@crypt-dmg")
//	wav, err := p.Cook(cacheDir)
//	stop, err := stage.Score(wav, p, p.LoopPeriod())
//
// Open reads the two documents and runs the machine; Cook writes the samples
// where a player can reach them, once. The piece is the playhead the scene
// binds to, so the same object that made the sound says where it is.
//
// Open costs a render: about four tenths of a second for twenty seconds of
// music on this machine. A game that will not spend that at startup cooks its
// music as a build step, by writing the WAV ahead of time, and ships the WAV.

// renderVersion changes when the machine's output changes. It is part of a
// cooked file's name, so a piece cooked by an older tune is not mistaken for
// one this tune would render.
const renderVersion = 1

// Open reads a score and a voicing, each either a path or a built-in named
// with "@", and renders the piece they make.
//
// A score where a voicing belongs is refused by name rather than decoded into
// nonsense, because the two are easy to swap and the error otherwise arrives
// as a puzzle.
func Open(score, voicing string) (*Piece, error) {
	sb, err := readDoc(score)
	if err != nil {
		return nil, err
	}
	vb, err := readDoc(voicing)
	if err != nil {
		return nil, err
	}
	if err := isKind(score, sb, "score"); err != nil {
		return nil, err
	}
	if err := isKind(voicing, vb, "voicing"); err != nil {
		return nil, err
	}
	s, err := ParseScore(sb)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", score, err)
	}
	v, err := ParseVoicing(vb)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", voicing, err)
	}
	p, err := Render(s, v)
	if err != nil {
		return nil, err
	}
	p.name = stem(score) + "-" + stem(voicing)
	p.key = keyOf(sb, vb)
	return p, nil
}

// isKind refuses a document that is not the kind the caller asked for.
func isKind(name string, b []byte, want string) error {
	got, err := kind(b)
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	if got != want {
		return fmt.Errorf("%s is a %s, and a %s belongs here", name,
			got, want)
	}
	return nil
}

// Cook writes the piece into dir as a WAV and says where it put it.
//
// The name holds a hash of the two documents and the machine's version, so a
// piece that has not changed is not rendered to disk twice, and a piece that
// has changed cannot be served from an old file. A game may keep the
// directory between runs.
func (p *Piece) Cook(dir string) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	name := p.name
	if name == "" {
		name = p.Score.Name + "-" + p.Voicing.Name
	}
	key := p.key
	if key == "" {
		key = keyOf(pcmBytes(p.Left), pcmBytes(p.Right))
	}
	path := filepath.Join(dir, fmt.Sprintf("%s-%s.wav", name, key))
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}
	if err := p.WriteWAV(path); err != nil {
		return "", err
	}
	return path, nil
}

// keyOf is the short hash a cooked file is named by.
func keyOf(parts ...[]byte) string {
	h := sha256.New()
	fmt.Fprintf(h, "tune v%d\n", renderVersion)
	for _, b := range parts {
		fmt.Fprintf(h, "%d\n", len(b))
		h.Write(b)
	}
	return hex.EncodeToString(h.Sum(nil))[:12]
}

// pcmBytes is samples as bytes, for hashing a piece that was rendered rather
// than opened.
func pcmBytes(x []float32) []byte {
	out := make([]byte, 0, len(x)*2)
	for _, v := range x {
		s := pcm16(v)
		out = append(out, byte(s), byte(s>>8))
	}
	return out
}

// stem is a document's name without its directory, extension or "@".
func stem(name string) string {
	name = strings.TrimPrefix(name, "@")
	name = filepath.Base(name)
	return strings.TrimSuffix(name, filepath.Ext(name))
}
