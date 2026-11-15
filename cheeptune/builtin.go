package cheeptune

import (
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"
)

// Scores and voicings that travel with the library.
//
// A game that depends on tune should not also depend on tune's working
// directory. Either it copies documents into its own tree and the copies
// drift, or it has no music without a checkout of this one.
//
// So the useful ones are compiled in, the way miami compiles in its scenes. A
// name beginning with "@" is a built-in:
//
//	cheeptune.Open("@crypt", "@crypt-dmg")   a floor you move on
//	cheeptune.Open("mine.json", "@dmg-echo") your tune, our sound
//	cheeptune.Open("mine.json", "mine.css")  neither of them ours
//
// Built in rather than fetched, because music that needs the network is music
// a game cannot start with.

//go:embed builtin/*.json
var builtin embed.FS

// Builtin lists what is compiled in, scores first, then voicings, so a caller
// can say what it may ask for without reading this directory.
func Builtin() (scores, voicings []string) {
	entries, err := fs.ReadDir(builtin, "builtin")
	if err != nil {
		return nil, nil
	}
	for _, e := range entries {
		b, err := builtin.ReadFile("builtin/" + e.Name())
		if err != nil {
			continue
		}
		name := "@" + strings.TrimSuffix(e.Name(), ".json")
		switch kind, _ := Kind(b); kind {
		case "score":
			scores = append(scores, name)
		case "voicing":
			voicings = append(voicings, name)
		}
	}
	sort.Strings(scores)
	sort.Strings(voicings)
	return scores, voicings
}

// ReadDoc is a document's bytes, from the embedded set when it is named with
// an "@" and from the filesystem otherwise.
//
// For a tool that wants to know what a name holds before deciding what to do
// with it: pair it with Kind.
func ReadDoc(name string) ([]byte, error) { return readDoc(name) }

// readDoc is a document from the embedded set when it is named with an "@",
// and from the filesystem otherwise.
func readDoc(name string) ([]byte, error) {
	if !strings.HasPrefix(name, "@") {
		return readFile(name)
	}
	b, err := builtin.ReadFile("builtin/" + name[1:] + ".json")
	if err == nil {
		return b, nil
	}
	if lb, lerr := libraryDoc(name[1:]); lerr == nil {
		return lb, nil
	}
	scores, voicings := Builtin()
	return nil, fmt.Errorf("no built-in %q; there is %s", name,
		strings.Join(append(scores, voicings...), " "))
}
