package job

import (
	_ "embed"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/janearc/puzzlemouth/yamlfile"
	yaml "gopkg.in/yaml.v3"
)

//go:embed job.schema.json
var schemaRaw []byte

// schema is the compiled schema every job is read through, whether it
// came from a file, a sequence plan or a command line.
var schema, schemaError = yamlfile.NewSchema(schemaName, schemaRaw)

// ErrRefused is what every refusal of a job wraps, so a caller can tell
// a job that was wrong from a file that could not be read.
var ErrRefused = errors.New("job refused")

// refusal is a job refused: what was wrong and where, as file, line and
// column, or as the file alone where the schema itself said.
type refusal struct{ err error }

// Error is the refusal's own words.
func (refused refusal) Error() string { return refused.err.Error() }

// Unwrap is what was wrong, and ErrRefused.
func (refused refusal) Unwrap() []error {
	return []error{refused.err, ErrRefused}
}

// source is where a job's words came from: name for its messages, base
// for the paths it writes, and file, the job file, or nothing for a job
// lifted from a plan.
type source struct {
	name, base, file string
	inputs           *digested
	imported         *importSnapshot
	outputForms      *outputForms
}

// admitted is a job checked and built from its words: the schema's
// checks, its paths resolved, its preset applied, the checks a schema
// cannot make, and its camera built. Every door into a job comes here.
func admitted(from source, raw []byte) (Job, error) {
	if err := errors.Join(schemaError, presetsError); err != nil {
		return Job{}, err
	}
	node, err := schema.Read(from.name, raw)
	if err != nil {
		return Job{}, refusal{err}
	}
	var wire document
	if err = node.Decode(&wire); err != nil {
		return Job{}, fmt.Errorf("%s: %w", from.name, err)
	}
	if from.outputForms == nil {
		forms := formsOf(wire.Outputs)
		from.outputForms = &forms
	}
	wire.resolve(from.base)
	wire.preset()
	root := node.Content[nothing]
	for _, check := range checks {
		if where, why := check(wire); why != "" {
			return Job{}, refuse(from.name, root, where, why)
		}
	}
	return built(from, root, wire)
}

// refuse is a refusal at the key the path names, or as near to it as the
// document has.
func refuse(name string, root *yaml.Node, where []string,
	why string) error {
	at := keyAt(root, where)
	return refusal{fmt.Errorf("%s:%d:%d: %s", name, at.Line, at.Column,
		why)}
}

// keyAt is the key node the path names, or the deepest node on the way
// to it that the document has. A piece is a mapping's key or a
// sequence's index.
func keyAt(node *yaml.Node, path []string) *yaml.Node {
	for i, piece := range path {
		key, value := childOf(node, piece)
		if value == nil {
			return node
		}
		if i == len(path)-one {
			return key
		}
		node = value
	}
	return node
}

// childOf is the key and value under piece, or nils where there is none.
// A sequence's item is its own key.
func childOf(node *yaml.Node, piece string) (key, value *yaml.Node) {
	if node.Kind == yaml.SequenceNode {
		index, err := strconv.Atoi(piece)
		if err != nil || index < nothing || index >= len(node.Content) {
			return nil, nil
		}
		return node.Content[index], node.Content[index]
	}
	for i := 0; i+1 < len(node.Content); i += halves {
		if node.Content[i].Value == piece {
			return node.Content[i], node.Content[i+one]
		}
	}
	return nil, nil
}

// resolve makes every path the job writes absolute, against base. A
// tune or a voicing written @name is a built-in and is left as written.
func (wire *document) resolve(base string) {
	at := func(path *string) {
		if *path != "" && !filepath.IsAbs(*path) {
			*path = filepath.Join(base, *path)
		}
		*path = cleaned(*path)
	}
	builtInOr := func(path *string) {
		if len(*path) == nothing || (*path)[:one] != builtIn {
			at(path)
		}
	}
	at(&wire.Scene.Set)
	at(&wire.Scene.Play)
	at(&wire.Scene.Frames)
	if wire.Titles != nil {
		at(&wire.Titles.Theme)
	}
	if wire.Sound != nil {
		builtInOr(&wire.Sound.Tune)
		builtInOr(&wire.Sound.Voicing)
		at(&wire.Sound.File)
		at(&wire.Sound.Score)
	}
	if outputs := wire.Outputs; outputs != nil {
		for _, path := range []*string{&outputs.Picture, &outputs.Raw,
			&outputs.Frames, &outputs.Movie} {
			at(path)
		}
	}
}

// cleaned is a path made plain, and an unsaid path left unsaid.
func cleaned(path string) string {
	if path == "" {
		return ""
	}
	return filepath.Clean(path)
}
