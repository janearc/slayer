package door

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/slayer/job"
)

// schemaHelp is schema's page.
const schemaHelp = `# schema

slayer schema prints the job schema (job-v1): every key of a job file,
with what it says, as the editor shows it on hover.

slayer schema --where prints the path of a copy of it kept under the
user's cache directory, written there first if it is absent or not this
build's. A job file's first line names that path for the YAML language
server: # yaml-language-server: $schema=PATH
`

// The cache the editor's files are kept in, under the user's cache
// directory, as os.UserCacheDir finds it, read through the door's
// environment so a test can give another.
const (
	userCacheFolder = "slayer"
	schemaName      = "job-v1.schema.json"
	luaName         = "slayer.lua"
	xdgCache        = "XDG_CACHE_HOME"
	homeName        = "HOME"
	darwinCache     = "Library/Caches" // defined by macOS
	plainCache      = ".cache"         // defined by the XDG base directory spec
	darwin          = "darwin"
)

// schemaing is the schema verb.
type schemaing struct{ env Env }

// schema prints the schema, or with --where the path of its copy.
func (verb schemaing) schema(call pzm.Call) error {
	if len(call.Args()) != nothing {
		return errors.New("schema takes no file: slayer schema [--where]")
	}
	if !call.On("where") {
		_, err := call.Out().Write(job.Schema())
		return err
	}
	if call.DryRun() {
		where, err := verb.env.cached(schemaName)
		if err == nil {
			_, err = fmt.Fprintf(call.Out(), "would write %s\n", where)
		}
		return err
	}
	where, err := verb.env.schemaCopy()
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(call.Out(), where)
	return err
}

// cacheDirectory is the user's cache directory, as os.UserCacheDir
// finds it, from the door's environment.
func (env Env) cacheDirectory() (string, error) {
	if said, set := env.lookup(xdgCache); set && runtime.GOOS != darwin {
		return said, nil
	}
	home, set := env.lookup(homeName)
	if !set {
		return "", errors.New("no cache directory: HOME is not set; set " +
			"it, or XDG_CACHE_HOME")
	}
	if runtime.GOOS == darwin {
		return filepath.Join(home, darwinCache), nil
	}
	return filepath.Join(home, plainCache), nil
}

// cached is the path of a file of slayer's in the cache.
func (env Env) cached(name string) (string, error) {
	directory, err := env.cacheDirectory()
	if err != nil {
		return "", err
	}
	return filepath.Join(directory, userCacheFolder, name), nil
}

// schemaCopy is the path of the schema's copy in the cache, written
// there first when it is absent or not this build's.
func (env Env) schemaCopy() (string, error) {
	return env.keep(schemaName, job.Schema())
}

// keep is the path of a file of slayer's in the cache, holding what it
// is given: written beside itself and renamed into place when absent or
// stale, and left alone when it already holds it.
func (env Env) keep(name string, holds []byte) (string, error) {
	where, err := env.cached(name)
	if err != nil {
		return "", err
	}
	if kept, err := os.ReadFile(where); err == nil &&
		bytes.Equal(kept, holds) {
		return where, nil
	}
	if err := os.MkdirAll(filepath.Dir(where), folderMode); err != nil {
		return "", err
	}
	return where, replaced(where, holds)
}

// described is one key of the schema and what it says.
type described struct {
	Description string               `json:"description"`
	Properties  map[string]described `json:"properties"`
	Ref         string               `json:"$ref"`
}

// keyLines is the schema's keys and what each says, a line each, the
// keys of an object under it, for the agent's guide.
func keyLines() ([]string, error) {
	var schema struct {
		Properties map[string]described `json:"properties"`
		Defs       map[string]described `json:"$defs"`
	}
	if err := json.Unmarshal(job.Schema(), &schema); err != nil {
		return nil, err
	}
	return linesOf(schema.Properties, schema.Defs, ""), nil
}

// linesOf is the lines of some keys, sorted, each indented under its
// parent, a key that refers to a definition said as the definition.
func linesOf(keys, defs map[string]described, indent string) []string {
	names := make([]string, nothing, len(keys))
	for name := range keys {
		names = append(names, name)
	}
	sort.Strings(names)
	var lines []string
	for _, name := range names {
		each := keys[name]
		if each.Description == "" && each.Ref != "" {
			each = defs[strings.TrimPrefix(each.Ref, defsPrefix)]
		}
		lines = append(lines, fmt.Sprintf("%s- `%s`: %s", indent, name,
			oneLine(each.Description)))
		lines = append(lines, linesOf(each.Properties, defs,
			indent+guideIndent)...)
	}
	return lines
}

// defsPrefix is how a schema refers to one of its definitions. Defined
// by JSON Schema.
const defsPrefix = "#/$defs/"

// guideIndent is how far a key is indented under its parent in the
// guide. Defined by Markdown: two spaces nest a list.
const guideIndent = "  "

// oneLine is text on one line.
func oneLine(text string) string {
	return strings.Join(strings.Fields(text), " ")
}
