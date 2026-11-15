package job

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// module is slayer's module path, and root its tree from here.
const (
	module = "github.com/janearc/slayer/"
	root   = ".."
)

// direct is what the job package may import of slayer's own.
var direct = []string{"camera", "maths"}

// renderers are the packages the job package must never reach, however
// far down: the renderers and the readers of scenes.
var renderers = []string{"hillside", "set", "gpu", "trace", "render",
	"metal", "film", "scene"}

// importsOf is slayer's packages a package's go files import, tests left
// out, by their path under the module.
func importsOf(t *testing.T, directory string) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(root, directory))
	if err != nil {
		t.Fatal(err)
	}
	var found []string
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") ||
			strings.HasSuffix(name, "_test.go") {
			continue
		}
		parsed, err := parser.ParseFile(token.NewFileSet(),
			filepath.Join(root, directory, name), nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, spec := range parsed.Imports {
			path, _ := strconv.Unquote(spec.Path.Value)
			inner, ours := strings.CutPrefix(path, module)
			if ours && !slices.Contains(found, inner) {
				found = append(found, inner)
			}
		}
	}
	return found
}

// reached is every package of slayer's that directory reaches, however
// far down.
func reached(t *testing.T, directory string) []string {
	t.Helper()
	seen := []string{}
	waiting := importsOf(t, directory)
	for len(waiting) > nothing {
		next := waiting[nothing]
		waiting = waiting[one:]
		if !slices.Contains(seen, next) {
			seen = append(seen, next)
			waiting = append(waiting, importsOf(t, next)...)
		}
	}
	return seen
}

// forbidden is the first package in packages that is a renderer or under
// one.
func forbidden(packages []string) string {
	for _, each := range packages {
		head, _, _ := strings.Cut(each, "/")
		if slices.Contains(renderers, head) {
			return each
		}
	}
	return ""
}

// TestTheJobPackageReachesNoRenderer holds the job package to camera and
// maths, and everything under them to no renderer.
func TestTheJobPackageReachesNoRenderer(t *testing.T) {
	for _, each := range importsOf(t, "job") {
		if !slices.Contains(direct, each) {
			t.Errorf("job imports %s; it may import only %v", each, direct)
		}
	}
	if found := forbidden(reached(t, "job")); found != "" {
		t.Errorf("job reaches %s", found)
	}
}

// TestTheImportWalkFindsARenderer is the anti-test: the same walk over
// hillside finds the renderers it reaches.
// anti-test of TestTheJobPackageReachesNoRenderer
func TestTheImportWalkFindsARenderer(t *testing.T) {
	if found := forbidden(reached(t, "hillside")); found == "" {
		t.Error("the walk found no renderer under hillside")
	}
}
