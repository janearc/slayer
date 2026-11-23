package door

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/janearc/slayer/session"
)

// Given is what the edge hands the door: everything a verb reads from
// the process, so a test can hand it something else.
type Given struct {
	// Context is what the verbs run under; cancelled, they stop between
	// frames.
	Context context.Context
	// Build, Built, Who and Sessions are the build's stamp, as the
	// linker sets them.
	Build, Built, Who, Sessions string
	// Environment looks a variable up, as os.LookupEnv does.
	Environment func(name string) (string, bool)
	// Now is the clock.
	Now func() time.Time
	// Directory is the working directory, absolute: out/ is under it.
	Directory string
	// Progress is where a still's draw says how far it is, a line at a
	// time; nil is the process's standard error.
	Progress io.Writer
}

// Env is the door's surroundings, checked: made by NewEnv, read by the
// verbs, never changed.
type Env struct {
	ctx         context.Context
	build       stamp
	environment func(string) (string, bool)
	now         func() time.Time
	directory   string
	progress    io.Writer
}

// stamp is the four words that say which build is running.
type stamp struct{ build, built, who, sessions string }

// NewEnv is the surroundings given, checked: a context, an environment,
// a clock and an absolute working directory are each required, by name.
func NewEnv(given Given) (Env, error) {
	var missing []error
	if given.Context == nil {
		missing = append(missing, errors.New("door: no context given"))
	}
	if given.Environment == nil {
		missing = append(missing,
			errors.New("door: no environment given; pass os.LookupEnv"))
	}
	if given.Now == nil {
		missing = append(missing,
			errors.New("door: no clock given; pass time.Now"))
	}
	if !filepath.IsAbs(given.Directory) {
		missing = append(missing, errors.New("door: the working "+
			"directory is not absolute: "+given.Directory))
	}
	if err := errors.Join(missing...); err != nil {
		return Env{}, err
	}
	progress := given.Progress
	if progress == nil {
		progress = os.Stderr
	}
	return Env{ctx: given.Context, environment: given.Environment,
		now: given.Now, directory: given.Directory, progress: progress,
		build: stamp{given.Build, given.Built, given.Who,
			given.Sessions}}, nil
}

// hillside is the stamp as hillside takes it.
func (made stamp) hillside() session.Stamp {
	return session.Stamped(made.build, made.built, made.who,
		made.sessions)
}

// lookup is a variable of the environment, and whether it is set and
// says something.
func (env Env) lookup(name string) (string, bool) {
	value, set := env.environment(name)
	return value, set && value != ""
}

// drawnBy is what names the build that draws, for the records a film
// keeps: its version and when it was built, or, for a build the linker
// stamped nothing on, the sha256 of the running program itself.
func (made stamp) drawnBy() string {
	if made.build != "" || made.built != "" {
		return made.build + " " + made.built
	}
	return unstamped()
}

// unstamped is the sha256 of the running program, worked out once, or
// the reason it could not be, which still differs from any digest.
var unstamped = sync.OnceValue(func() string {
	path, err := os.Executable()
	if err != nil {
		return "unknown program: " + err.Error()
	}
	file, err := os.Open(path)
	if err != nil {
		return "unreadable program: " + err.Error()
	}
	defer file.Close()
	sum := sha256.New()
	if _, err := io.Copy(sum, file); err != nil {
		return "unreadable program: " + err.Error()
	}
	return "program " + hex.EncodeToString(sum.Sum(nil))
})
