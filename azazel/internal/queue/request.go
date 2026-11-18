package queue

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// validate checks only execution metadata; the tool validates its own arguments.
func (r Request) validate() error {
	if r.Agent == "" || r.Key == "" || len(r.Agent) > 128 || len(r.Key) > 128 ||
		!filepath.IsAbs(r.Command) || !filepath.IsAbs(r.Directory) ||
		r.TimeoutSeconds <= 0 || r.TimeoutSeconds > 7*24*3600 ||
		r.Lane != "gpu" && r.Lane != "tests" && r.Lane != "mutants" ||
		len(r.Args) > 256 || len(r.Inputs) > 128 {
		return fmt.Errorf("request needs agent, key, absolute command and directory, lane and timeout")
	}
	if r.Pictures != nil && *r.Pictures <= 0 {
		return fmt.Errorf("pictures, when given, must be a positive count")
	}
	info, err := os.Stat(r.Directory)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("working directory is not a directory")
	}
	for _, arg := range append([]string{r.Agent, r.Key}, r.Args...) {
		if len(arg) > 8192 || strings.ContainsRune(arg, 0) {
			return fmt.Errorf("invalid command argument")
		}
	}
	for key, value := range r.Environment {
		if !strings.HasPrefix(key, "SLAYER_") || strings.ContainsAny(key, "=\x00") || strings.ContainsRune(value, 0) {
			return fmt.Errorf("unsupported environment key %q", key)
		}
	}
	if value, found := r.Environment["SLAYER_CPU_WORKERS"]; found {
		workers, err := strconv.Atoi(value)
		if err != nil || workers < 1 || workers > 1024 {
			return fmt.Errorf("SLAYER_CPU_WORKERS must be a whole number between 1 and 1024")
		}
	}
	return nil
}

// fingerprint streams bytes so large executables do not inflate the queue.
func fingerprint(path string, executable bool) (Fingerprint, error) {
	var out Fingerprint
	if !filepath.IsAbs(path) {
		return out, fmt.Errorf("input path must be absolute: %q", path)
	}
	file, err := os.Open(path)
	if err != nil {
		return out, err
	}
	defer file.Close()
	return digest(file, path, executable)
}

// verify rechecks submitted executable and input identities.
func (j Job) verify() error {
	if err := j.Request.validate(); err != nil {
		return fmt.Errorf("%w: %w", ErrUnsafe, err)
	}
	executables := len(j.Files) - len(j.Request.Inputs)
	if executables < 1 || executables > 2 || j.Files[0].Path != j.Request.Command {
		return ErrUnsafe
	}
	for i, want := range j.Files {
		if i >= executables && want.Path != j.Request.Inputs[i-executables] {
			return ErrUnsafe
		}
		got, err := fingerprint(want.Path, i < executables)
		if err != nil {
			return err
		}
		if got != want {
			return fmt.Errorf("submitted bytes changed: %s", want.Path)
		}
	}
	return nil
}
