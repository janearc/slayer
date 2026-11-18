package instrument

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestANilPlanSaysNothingAndRefusesCleanup checks every accessor of a nil
// plan, and that cleaning it is an error.
func TestANilPlanSaysNothingAndRefusesCleanup(t *testing.T) {
	var plan *Plan
	if plan.Executable() != "" || plan.Args() != nil || plan.Env() != nil || plan.TimingPath() != "" {
		t.Error("a nil plan gave a value")
	}
	if argv, env, timing := plan.TrustedCommand(); argv != nil || env != nil || timing != "" {
		t.Errorf("a nil plan gave a command: %#v %#v %q", argv, env, timing)
	}
	if err := plan.Cleanup(); err == nil || !strings.Contains(err.Error(), "nil plan") {
		t.Errorf("cleaning a nil plan: %v", err)
	}
}

// TestBuildReportsAPrivateDirectoryItCannotMake checks that a temporary
// directory that does not exist is an error and no plan.
func TestBuildReportsAPrivateDirectoryItCannotMake(t *testing.T) {
	t.Setenv("TMPDIR", filepath.Join(t.TempDir(), "missing"))
	plan, err := Build(Options{Executable: "/bin/true"})
	if err == nil || plan != nil {
		t.Errorf("build without a temporary directory: %v, %v", plan, err)
	}
}

// TestCleanupRefusesPathsItDoesNotOwn checks the three refusals that stop
// it removing anything: paths that are not absolute, a timing file outside
// the directory and a timing file with the wrong name.
func TestCleanupRefusesPathsItDoesNotOwn(t *testing.T) {
	dir := t.TempDir()
	cases := map[string]struct {
		plan func() *Plan
		want string
	}{
		"relative paths":  {func() *Plan { return &Plan{privateDir: "private", timingPath: "private/timing"} }, "must be absolute"},
		"timing outside":  {func() *Plan { return &Plan{privateDir: dir, timingPath: filepath.Join(filepath.Dir(dir), "timing")} }, "outside the owned private dir"},
		"timing misnamed": {func() *Plan { return &Plan{privateDir: dir, timingPath: filepath.Join(dir, "report")} }, "outside the owned private dir"},
	}
	for name, each := range cases {
		plan := each.plan()
		err := plan.Cleanup()
		if err == nil || !strings.Contains(err.Error(), each.want) {
			t.Errorf("%s: %v, want %q", name, err, each.want)
		}
		if plan.cleaned {
			t.Errorf("%s: the plan was marked cleaned", name)
		}
	}
	if _, err := os.Stat(dir); err != nil {
		t.Errorf("a refused cleanup removed the directory: %v", err)
	}
}

// TestCleanupReportsATimingFileItCannotRemove checks that a timing path
// which is a directory with something in it is an error that names it, the
// plan is not marked cleaned, and a later cleanup works once it is emptied.
func TestCleanupReportsATimingFileItCannotRemove(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	plan, err := Build(Options{Executable: "/bin/true"})
	if err != nil {
		t.Fatal(err)
	}
	timing := plan.TimingPath()
	if err := os.MkdirAll(filepath.Join(timing, "inner"), 0o700); err != nil {
		t.Fatal(err)
	}
	err = plan.Cleanup()
	if err == nil || !strings.Contains(err.Error(), "remove timing file") || !strings.Contains(err.Error(), "remove private dir") {
		t.Fatalf("cleaning a timing path that is a full directory: %v", err)
	}
	if plan.Executable() != "/bin/true" {
		t.Error("a failed cleanup marked the plan cleaned")
	}
	if err := os.RemoveAll(timing); err != nil {
		t.Fatal(err)
	}
	if err := plan.Cleanup(); err != nil {
		t.Errorf("cleaning again after emptying it: %v", err)
	}
	if plan.Executable() != "" {
		t.Error("a successful cleanup left the plan live")
	}
}

// TestCleanupRefusesATimingFileBelowASubdirectory checks the documented rule
// that the timing file is named timing directly inside the private directory.
func TestCleanupRefusesATimingFileBelowASubdirectory(t *testing.T) {
	dir := t.TempDir()
	plan := &Plan{privateDir: dir, timingPath: filepath.Join(dir, "sub", "timing")}
	if err := plan.Cleanup(); err == nil || plan.cleaned {
		t.Errorf("a timing path below a subdirectory was accepted: %v", err)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Errorf("the directory was removed: %v", err)
	}
}
