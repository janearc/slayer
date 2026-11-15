package instrument

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var errAlreadyCleaned = errors.New("instrument: plan already cleaned")

type Options struct {
	Executable string
	Args       []string
	Env        []string
}

type Plan struct {
	mu         sync.RWMutex
	executable string
	args       []string
	env        []string
	privateDir string
	timingPath string
	cleaned    bool
}

// Build makes a Plan to run the executable, which must be given as an absolute
// path. It makes a private temporary directory, mode 0700, and picks a timing
// file path inside it without creating the file. Args and Env are copied.
func Build(opts Options) (*Plan, error) {
	if opts.Executable == "" {
		return nil, fmt.Errorf("instrument: executable is required")
	}
	if !filepath.IsAbs(opts.Executable) {
		return nil, fmt.Errorf("instrument: executable must be absolute")
	}
	dir, err := os.MkdirTemp("", "oraclebench-instrument-")
	if err != nil {
		return nil, err
	}
	chmodErr := os.Chmod(dir, 0o700)
	if chmodErr != nil {
		removeErr := os.Remove(dir)
		return nil, errors.Join(chmodErr, removeErr)
	}
	plan := &Plan{
		executable: opts.Executable,
		args:       append([]string(nil), opts.Args...),
		env:        append([]string(nil), opts.Env...),
		privateDir: dir,
		timingPath: filepath.Join(dir, "timing"),
	}
	return plan, nil
}

// Executable is the program path the plan runs, or "" for a nil or cleaned
// plan.
func (p *Plan) Executable() string {
	if p == nil {
		return ""
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.cleaned {
		return ""
	}
	return p.executable
}

// Args is a copy of the arguments given to Build, or nil for a nil or cleaned
// plan.
func (p *Plan) Args() []string {
	if p == nil {
		return nil
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.cleaned {
		return nil
	}
	return append([]string(nil), p.args...)
}

// Env is a copy of the environment given to Build, or nil for a nil or
// cleaned plan.
func (p *Plan) Env() []string {
	if p == nil {
		return nil
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.cleaned {
		return nil
	}
	return append([]string(nil), p.env...)
}

// TrustedCommand returns the argv, a copy of the environment and the timing
// path. The argv runs the executable under /usr/bin/time -l -o with the timing
// path, so the timing output goes to that file. It is all empty for a nil or
// cleaned plan.
func (p *Plan) TrustedCommand() ([]string, []string, string) {
	if p == nil {
		return nil, nil, ""
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.cleaned {
		return nil, nil, ""
	}
	argv := make([]string, 0, 4+1+len(p.args))
	argv = append(argv, "/usr/bin/time", "-l", "-o", p.timingPath, p.executable)
	argv = append(argv, p.args...)
	return argv, append([]string(nil), p.env...), p.timingPath
}

// TimingPath is the file where /usr/bin/time writes its report for this plan,
// or "" for a nil or cleaned plan.
func (p *Plan) TimingPath() string {
	if p == nil {
		return ""
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.cleaned {
		return ""
	}
	return p.timingPath
}

// Cleanup removes the timing file, if it is there, and the private directory.
// It refuses a nil plan, one already cleaned, and paths that are not absolute
// or where the timing file is not named "timing" directly inside the
// directory. The plan is marked cleaned only if both removals worked, so a
// failed Cleanup can be tried again.
func (p *Plan) Cleanup() error {
	if p == nil {
		return errors.New("instrument: nil plan")
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.cleaned {
		return errAlreadyCleaned
	}
	if p.privateDir == "" || p.timingPath == "" {
		return errors.New("instrument: plan has no owned private dir")
	}
	dir := filepath.Clean(p.privateDir)
	timing := filepath.Clean(p.timingPath)
	if !filepath.IsAbs(dir) || !filepath.IsAbs(timing) {
		return errors.New("instrument: plan paths must be absolute")
	}
	if !strings.HasPrefix(timing, dir+string(os.PathSeparator)) || filepath.Base(timing) != "timing" {
		return errors.New("instrument: timing path is outside the owned private dir")
	}
	var cleanupErr error
	if err := os.Remove(timing); err != nil && !os.IsNotExist(err) {
		cleanupErr = errors.Join(cleanupErr, fmt.Errorf("instrument: remove timing file: %w", err))
	}
	if err := os.Remove(dir); err != nil {
		cleanupErr = errors.Join(cleanupErr, fmt.Errorf("instrument: remove private dir: %w", err))
	}
	if cleanupErr != nil {
		return cleanupErr
	}
	p.cleaned = true
	return nil
}
