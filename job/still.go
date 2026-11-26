package job

import (
	"encoding/json"
	"errors"
	"path/filepath"

	"github.com/janearc/slayer/set"
)

// StillOf is the least job of a set file: a still of it, with every
// value left to the set and the scene's defaults, as `slayer draw` makes
// of a set file given in place of a job. It names no job file, so File
// says nothing, and its outputs are the verb's to choose. It is checked
// as a job read from a file is.
func StillOf(set string) (Job, error) {
	if set == "" {
		return Job{}, refusal{errors.New("a still of a set needs " +
			"the set file; none was given")}
	}
	absolute, err := filepath.Abs(set)
	if err != nil {
		return Job{}, err
	}
	wire := document{Schema: schemaName,
		Scene: sceneWire{Set: absolute}}
	// the marshalling cannot fail: the document is two strings
	raw, _ := json.Marshal(wire)
	from := source{name: set + ", as a still",
		base: filepath.Dir(absolute)}
	return admitted(from, raw)
}

// StillFrom is a still of the set bytes at path. Relative references are
// resolved from path, and the parsed set and its decoded source digests are
// retained; the file itself is never written.
func StillFrom(path string, raw []byte) (Job, error) {
	if path == "" {
		return Job{}, refusal{errors.New("a still of a set needs the set file; none was given")}
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return Job{}, err
	}
	read, err := set.Read(absolute, raw)
	if err != nil {
		return Job{}, err
	}
	inputs := &digested{paths: map[string]string{inputSet: absolute}}
	retained, err := freezeSet(read, raw, inputs)
	if err != nil {
		return Job{}, err
	}
	wire := document{Schema: schemaName, Scene: sceneWire{Set: absolute}}
	encoded, _ := json.Marshal(wire)
	from := source{name: path + ", as a still", base: filepath.Dir(absolute), inputs: inputs, imported: retained}
	return admitted(from, encoded)
}
