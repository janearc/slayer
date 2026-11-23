package cheeptune

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
)

// decodeStrict reads a document into v the way miami reads a scene: // line
// comments are allowed and removed first, and a field the type does not know
// is an error, never skipped. A document an agent wrote with a typo in a key
// otherwise renders a tune that is quietly missing the thing it asked for.
func decodeStrict(b []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(stripComments(b)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	if dec.More() {
		return errors.New("there is more after the document ends")
	}
	return nil
}

// stripComments removes // to the end of the line, outside strings. The line
// itself stays, so an error's line number still points at the right line.
func stripComments(b []byte) []byte {
	out := make([]byte, 0, len(b))
	in, esc := false, false
	for i := 0; i < len(b); i++ {
		c := b[i]
		switch {
		case in:
			out = append(out, c)
			switch {
			case esc:
				esc = false
			case c == '\\':
				esc = true
			case c == '"':
				in = false
			}
		case c == '"':
			in = true
			out = append(out, c)
		case c == '/' && i+1 < len(b) && b[i+1] == '/':
			for i < len(b) && b[i] != '\n' {
				i++
			}
			if i < len(b) {
				out = append(out, '\n')
			}
		default:
			out = append(out, c)
		}
	}
	return out
}

// kind says which document a file holds, from its top-level keys and nothing
// else. Searching the text for a word sends a score whose comment mentions
// channels to the voicing reader; a key is what the document is.
func kind(b []byte) (string, error) {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(stripComments(b), &top); err != nil {
		return "", fmt.Errorf("not a json object: %w", err)
	}
	_, parts := top["parts"]
	_, chans := top["channels"]
	switch {
	case parts && chans:
		return "", errors.New("holds both parts and channels")
	case parts:
		return "score", nil
	case chans:
		return "voicing", nil
	}
	keys := make([]string, 0, len(top))
	for k := range top {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return "", fmt.Errorf(
		"neither a score (parts) nor a voicing (channels); keys are %v",
		keys,
	)
}

// readFile is os.ReadFile, named for the one place documents come in.
func readFile(path string) ([]byte, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return b, nil
}
