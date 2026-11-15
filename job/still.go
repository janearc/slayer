package job

import (
	"encoding/json"
	"errors"
	"path/filepath"
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
