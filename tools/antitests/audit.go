package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// the marker is `// anti-test of TestX`, alone on its line, nothing after
// it. an old marker is the earlier loose forms (`// anti-test:`,
// `// anti:`, `// Anti-tests:`), any case; one that does not match the
// exact marker names no test and is only counted.
var (
	markerPattern = regexp.MustCompile(`^\s*// anti-test of (Test\w+)\s*$`)
	oldPattern    = regexp.MustCompile(`(?i)^\s*//\s*anti(-tests?)?\s*:`)
)

// place is a file and line, printed as file:line
type place struct {
	file string
	line int
}

// String is the place as file:line.
func (p place) String() string { return fmt.Sprintf("%s:%d", p.file, p.line) }

// marker is one marker line and the test it names
type marker struct {
	at   place
	name string
}

// testFunc is a test function, and whether a marker sits directly above
type testFunc struct {
	at     place
	name   string
	marked bool
}

// report is what one package's test files say about their anti-tests
type report struct {
	dir        string
	tests      int
	antis      int
	positives  int
	named      int
	missing    []marker
	old        []place
	unanswered []place
}

// ratio is positives named over positives; a package with no positives
// has nothing unanswered, so it is one
func (r report) ratio() float64 {
	if r.positives == 0 {
		return 1
	}
	return float64(r.named) / float64(r.positives)
}

// isTestName says whether a function name is a go test: Test, then
// anything but a lower-case letter
func isTestName(name string) bool {
	rest, ok := strings.CutPrefix(name, "Test")
	if !ok {
		return false
	}
	return rest == "" || !(rest[0] >= 'a' && rest[0] <= 'z')
}

// scanFile reads one test file for its test functions, markers and old
// markers
func scanFile(fset *token.FileSet, path string) ([]testFunc, []marker,
	[]place, error) {
	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, nil, nil, err
	}
	var funcs []testFunc
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if ok && fn.Recv == nil && isTestName(fn.Name.Name) {
			funcs = append(funcs, testFunc{
				at:     place{path, fset.Position(fn.Pos()).Line},
				name:   fn.Name.Name,
				marked: docMarked(fn.Doc),
			})
		}
	}
	var marks []marker
	var olds []place
	for _, group := range file.Comments {
		for _, one := range group.List {
			at := place{path, fset.Position(one.Pos()).Line}
			if m := markerPattern.FindStringSubmatch(one.Text); m != nil {
				marks = append(marks, marker{at, m[1]})
			} else if oldPattern.MatchString(one.Text) {
				olds = append(olds, at)
			}
		}
	}
	return funcs, marks, olds, nil
}

// docMarked says whether a function's comment, directly above it, holds
// an exact marker line
func docMarked(doc *ast.CommentGroup) bool {
	if doc == nil {
		return false
	}
	for _, one := range doc.List {
		if markerPattern.MatchString(one.Text) {
			return true
		}
	}
	return false
}

// auditDir counts the anti-tests of the _test.go files in one directory
func auditDir(dir string) (report, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*_test.go"))
	if err != nil {
		return report{}, err
	}
	rep := report{dir: dir}
	fset := token.NewFileSet()
	var funcs []testFunc
	var marks []marker
	for _, path := range files {
		f, m, o, err := scanFile(fset, path)
		if err != nil {
			return report{}, err
		}
		funcs, marks, rep.old = append(funcs, f...), append(marks, m...),
			append(rep.old, o...)
	}
	rep.tally(funcs, marks)
	return rep, nil
}

// tally fills the counts from the functions and markers found
func (r *report) tally(funcs []testFunc, marks []marker) {
	exists := map[string]bool{}
	for _, fn := range funcs {
		exists[fn.name] = true
	}
	named := map[string]bool{}
	for _, m := range marks {
		if exists[m.name] {
			named[m.name] = true
		} else {
			r.missing = append(r.missing, m)
		}
	}
	r.tests = len(funcs)
	for _, fn := range funcs {
		switch {
		case fn.marked:
			r.antis++
		case named[fn.name]:
			r.positives++
			r.named++
		default:
			r.positives++
			r.unanswered = append(r.unanswered, fn.at)
		}
	}
}

// expand turns the arguments into directories that hold test files;
// "dir/..." walks below dir, leaving out testdata, vendor, secrets and
// hidden or underscore directories
func expand(args []string) ([]string, error) {
	var dirs []string
	for _, arg := range args {
		root, walk := strings.CutSuffix(arg, "...")
		if !walk {
			dirs = append(dirs, arg)
			continue
		}
		if root == "" {
			root = "."
		}
		found, err := walkTests(filepath.Clean(root))
		if err != nil {
			return nil, err
		}
		dirs = append(dirs, found...)
	}
	sort.Strings(dirs)
	return dirs, nil
}

// skipped is true for a directory name the walk leaves out: testdata,
// vendor, secrets, or a name of two or more characters that starts with a
// dot or an underscore.
func skipped(name string) bool {
	return name == "testdata" || name == "vendor" || name == "secrets" ||
		(len(name) > 1 && (name[0] == '.' || name[0] == '_'))
}

// walkTests lists the directories at or below root that hold a file ending
// in _test.go. It does not go into a skipped directory, though root itself
// is always walked.
func walkTests(root string) ([]string, error) {
	var dirs []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry,
		err error) error {
		if err != nil || !d.IsDir() {
			return err
		}
		if path != root && skipped(d.Name()) {
			return filepath.SkipDir
		}
		if tests, _ := filepath.Glob(filepath.Join(path,
			"*_test.go")); len(tests) > 0 {
			dirs = append(dirs, path)
		}
		return nil
	})
	return dirs, err
}
