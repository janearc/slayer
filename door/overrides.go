package door

import (
	"fmt"
	"path/filepath"
	"strconv"
	"time"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/puzzlemouth/verb"
	"github.com/janearc/slayer/job"
)

// override is one flag every job-taking verb shares: its name, its help
// line, what its value is in words, and how its value becomes the job's
// override.
type override struct {
	name  verb.Name
	help  verb.Help
	takes string
	read  reader
}

// reader makes an override of a flag's value, a path in it resolved
// against the working directory, and says whether the value could be
// read.
type reader func(value, directory string) (job.Override, bool)

// overrides is every shared flag, in the order help shows them: the
// plain ones, then the job's flags and shortcuts.
var overrides = append(plainOverrides, switchOverrides()...)

// The words a flag or a shortcut takes on the command line, as in a
// job.
const switchOn, switchOff = "on", "off"

// switchOverrides is a flag for each flag and shortcut the job's schema
// declares, taking on or off, its help the schema's own first
// sentence: so the editor's hover, --help and --agent-help agree, and a
// shortcut added to the schema is on the command line without a line
// here.
func switchOverrides() []override {
	var made []override
	for _, each := range job.Switches() {
		name := each.Name()
		made = append(made, override{verb.Name(name),
			verb.Help(each.Line()), "on or off",
			func(value, _ string) (job.Override, bool) {
				return job.Flagged(name, value == switchOn),
					value == switchOn || value == switchOff
			}})
	}
	return made
}

// plainOverrides is the shared flags that are not the job's flags.
var plainOverrides = []override{
	{"out", "where the main output goes, over the job's", "a path",
		func(value, directory string) (job.Override, bool) {
			if !filepath.IsAbs(value) {
				value = filepath.Join(directory, value)
			}
			return job.Output(value), value != directory
		}},
	{"width", "the picture's width in pixels, over the job's",
		"a whole number", whole(job.Width)},
	{"height", "the picture's height in pixels, over the job's",
		"a whole number", whole(job.Height)},
	{"rays", "rays along each side of a pixel, over the job's",
		"a whole number", whole(job.Rays)},
	{"at", "a still's scene time, as 20s, over the job's",
		"a time such as 20s or 1m30s", func(value, _ string) (
			job.Override, bool) {
			when, err := time.ParseDuration(value)
			return job.At(when), err == nil
		}},
	{"camera", "a camera of the scene by name, over the job's",
		"a camera's name", named(job.CameraName)},
	{"renderer", "cpu, gpu, gpu-native-shadows or gpu-native",
		"a renderer's name", named(job.Renderer)},
	{"quality", "exact or preview, over the job's",
		"exact or preview", named(job.Quality)},
}

// whole reads a value as a whole number for an override.
func whole(made func(int) job.Override) reader {
	return func(value, _ string) (job.Override, bool) {
		number, err := strconv.Atoi(value)
		return made(number), err == nil
	}
}

// named reads a value as a name for an override; an empty one is no
// name.
func named(made func(string) job.Override) reader {
	return func(value, _ string) (job.Override, bool) {
		return made(value), value != ""
	}
}

// jobFlags is the flags every job-taking verb shares, declared once.
func jobFlags() []pzm.Flag {
	flags := make([]pzm.Flag, nothing, len(overrides))
	for _, each := range overrides {
		flags = append(flags, pzm.Value(each.name, verb.NoShort,
			each.help))
	}
	return flags
}

// overridesOf is the overrides a call's shared flags say, in the order
// declared, a relative --out under directory; a flag not given is the
// job's. A value a flag cannot take is refused in one wording, by the
// flag's name. The job checks the numbers' bounds when laid over it.
func overridesOf(call pzm.Call, directory string) ([]job.Override,
	error) {
	var laid []job.Override
	for _, each := range overrides {
		if !call.Given(string(each.name)) {
			continue
		}
		value := call.Value(string(each.name))
		made, read := each.read(value, directory)
		if !read {
			return nil, fmt.Errorf("--%s takes %s, not %q", each.name,
				each.takes, value)
		}
		laid = append(laid, made)
	}
	return laid, nil
}

// switchNames is the names of the job's flags and shortcuts, as the
// command line takes them.
func switchNames() []string {
	var names []string
	for _, each := range job.Switches() {
		names = append(names, each.Name())
	}
	return names
}
