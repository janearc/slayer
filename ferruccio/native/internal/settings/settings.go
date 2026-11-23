// Package settings is where Ferruccio's values that a person sets live, and
// the one place that says what each is. A value comes from a flag, then the
// environment, then the settings file, then it is unset. The file is
// ~/.config/ferruccio/config, in game's format, read by puzzlemouth's config
// package. The launch path and the render path both resolve through Resolve.
package settings

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/janearc/puzzlemouth/config"
)

// Name is one setting.
type Name string

// The settings a person sets from the app.
const (
	Cache      Name = "cache"
	Shelf      Name = "shelf"
	Slayer     Name = "slayer"
	Examples   Name = "examples"
	Theme      Name = "theme"
	Renderer   Name = "renderer"
	Processors Name = "processors"
)

// Kind is how a person chooses a setting's value: a folder or an executable
// in the system's own chooser, or one of a list.
type Kind int

// The kinds of setting.
const (
	Folder Kind = iota + 1
	Executable
	Choice
)

// Key is what is known of one setting.
type Key struct {
	Name  Name
	Label string
	Kind  Kind
	// Flag and Variable are the flag and the environment variable that
	// override the file; either may be empty where there is none.
	Flag, Variable string
	// Choices are the values a Choice takes. Where it is nil the caller
	// knows them, as the theme list is Chroma's.
	Choices []string
	// Default is what an unset setting means; it is empty where unset means
	// nothing.
	Default string
	// Why says in plain words what the setting is for.
	Why string
}

// Renderer values, and the processor counts and word the sheet offers.
const (
	RendererCPU  = "cpu"
	RendererGPU  = "gpu"
	ProcessorAll = "all"
)

// Keys are the settings, in the order the sheet shows them. The defaults are
// what Ferruccio did before there were settings: the CPU renderer on two
// processors.
var Keys = []Key{
	{Name: Cache, Label: "Cache folder", Kind: Folder, Flag: "cache", Variable: "SLAYER_CACHE", Why: "where Slayer keeps what it can work out again; it will not draw without one"},
	{Name: Shelf, Label: "Source shelf", Kind: Folder, Flag: "shelf", Variable: "SLAYER_SHELF", Why: "the folder of source papers and data some scenes cite; optional"},
	{Name: Slayer, Label: "Slayer", Kind: Executable, Flag: "slayer", Why: "the Slayer program that draws your pictures"},
	{Name: Examples, Label: "Examples folder", Kind: Folder, Flag: "examples", Why: "where the starting scenes are found"},
	{Name: Theme, Label: "Theme", Kind: Choice, Flag: "theme", Why: "the colours of the window"},
	{Name: Renderer, Label: "Renderer", Kind: Choice, Flag: "renderer", Choices: []string{RendererCPU, RendererGPU}, Default: RendererCPU, Why: "cpu draws every scene and is the reference; gpu is quicker but refuses scenes it cannot draw"},
	{Name: Processors, Label: "Processors", Kind: Choice, Flag: "processors", Choices: []string{"2", "4", "8", ProcessorAll}, Default: "2", Why: "how many processors a full render may use; more is quicker and slows the rest of the machine"},
}

// KeyOf is the key of a setting.
func KeyOf(name Name) (Key, bool) {
	for _, key := range Keys {
		if key.Name == name {
			return key, true
		}
	}
	return Key{}, false
}

// Source says where a value came from.
type Source string

// The places a value comes from.
const (
	FromFlag        Source = "flag"
	FromEnvironment Source = "environment"
	FromFile        Source = "file"
	Unset           Source = "unset"
)

// Entry is one setting as it resolves.
type Entry struct {
	Key Key
	// Value is the value, empty when Source is Unset.
	Value  string
	Source Source
	// Origin names the flag or variable the value came from, or the file;
	// it is empty when the value is unset.
	Origin string
}

// Locked reports whether a person cannot change the entry from the sheet:
// a flag or the environment overrides the file, so a saved value would not
// take effect.
func (entry Entry) Locked() bool {
	return entry.Source == FromFlag || entry.Source == FromEnvironment
}

// Effective is the value in force: the value, or the default when unset.
func (entry Entry) Effective() string {
	if entry.Value == "" {
		return entry.Key.Default
	}
	return entry.Value
}

// Resolved is every setting as it resolved, and the file they were read from.
type Resolved struct {
	Path string
	// Problem is why the file could not be read, in the words of the
	// library that refused it. A file with a problem gives no values.
	Problem string
	Entries []Entry
}

// Get is the entry for a setting.
func (resolved Resolved) Get(name Name) Entry {
	for _, entry := range resolved.Entries {
		if entry.Key.Name == name {
			return entry
		}
	}
	return Entry{}
}

// Value is the value of a setting, empty when it is unset.
func (resolved Resolved) Value(name Name) string { return resolved.Get(name).Value }

// Lookup reads a flag or a variable by name and gives "" for none.
type Lookup func(name string) string

// tail is a line's tail, kept whole, for a key that takes no words.
type tail string

// SetLine keeps the tail of the line.
func (value *tail) SetLine(line config.Line) error {
	*value = tail(line.Tail)
	return nil
}

// fileValues is what the settings file may say.
type fileValues struct {
	Cache      tail `config:"cache,0,tail" help:"the folder Slayer keeps what it can work out again in"`
	Shelf      tail `config:"shelf,0,tail" help:"the folder of source papers and data"`
	Slayer     tail `config:"slayer,0,tail" help:"the Slayer executable"`
	Examples   tail `config:"examples,0,tail" help:"the folder of starting scenes"`
	Theme      tail `config:"theme,0,tail" help:"the colour theme"`
	Renderer   tail `config:"renderer,0,tail" help:"cpu or gpu"`
	Processors tail `config:"processors,0,tail" help:"2, 4, 8 or all"`
}

// schema is made once from fileValues.
var schema, schemaError = config.NewSchema[fileValues]()

// valueOf is the file's value for a setting.
func (values fileValues) valueOf(name Name) string {
	switch name {
	case Cache:
		return string(values.Cache)
	case Shelf:
		return string(values.Shelf)
	case Slayer:
		return string(values.Slayer)
	case Examples:
		return string(values.Examples)
	case Theme:
		return string(values.Theme)
	case Renderer:
		return string(values.Renderer)
	case Processors:
		return string(values.Processors)
	}
	return ""
}

// Path is the settings file under home: home/.config/ferruccio/config. The
// caller says home, so a test can point it at a folder of its own.
func Path(home string) string {
	return filepath.Join(home, ".config", "ferruccio", "config")
}

// Resolve reads the settings file at path and gives each setting its value:
// the flag's if it was given, else the environment variable's, else the
// file's, else none. A file that is not there is no file. A file that cannot
// be read leaves Problem set and every file value empty.
func Resolve(path string, flag, environment Lookup) Resolved {
	resolved := Resolved{Path: path}
	file, problem := readFile(path)
	resolved.Problem = problem
	for _, key := range Keys {
		resolved.Entries = append(resolved.Entries, entryOf(key, path, file, flag, environment))
	}
	return resolved
}

// entryOf resolves one setting.
func entryOf(key Key, path string, file fileValues, flag, environment Lookup) Entry {
	if key.Flag != "" && flag != nil {
		if value := flag(key.Flag); value != "" {
			return Entry{Key: key, Value: value, Source: FromFlag, Origin: "--" + key.Flag}
		}
	}
	if key.Variable != "" && environment != nil {
		if value := environment(key.Variable); value != "" {
			return Entry{Key: key, Value: value, Source: FromEnvironment, Origin: key.Variable}
		}
	}
	if value := file.valueOf(key.Name); value != "" {
		return Entry{Key: key, Value: value, Source: FromFile, Origin: path}
	}
	return Entry{Key: key, Source: Unset}
}

// readFile reads the file at path, or says why it cannot.
func readFile(path string) (fileValues, string) {
	var values fileValues
	if schemaError != nil {
		return values, schemaError.Error()
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return values, ""
	}
	if err != nil {
		return values, err.Error()
	}
	if err := schema.Read(path, raw, &values); err != nil {
		return fileValues{}, err.Error()
	}
	return values, ""
}

// Save sets one setting in the file at path, or removes it when value is
// empty. A Choice with its own list takes only a value in it.
func Save(path string, name Name, value string) error {
	key, known := KeyOf(name)
	if !known {
		return fmt.Errorf("there is no setting %q", name)
	}
	if value != "" && key.Choices != nil && !slices.Contains(key.Choices, value) {
		return fmt.Errorf("%s is %s, not %q", key.Label, strings.Join(key.Choices, ", "), value)
	}
	return Update(path, []Change{{Key: string(name), Value: value}})
}
