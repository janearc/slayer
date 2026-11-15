// Command breakdown says what slayer is made of: every file git tracks,
// sorted into kinds, with how many files and how many lines each kind
// has, and each kind's share of the lines.
//
// usage, from slayer's root:
//
//	breakdown
//
// a file's kind is told by its name alone: a go file whose name ends in
// _test.go is a test, and any name no kind claims is "everything else".
// a line is a newline, and a last line with no newline counts as one. a
// file with a nought byte in it is not text (a picture, a table of
// floats) and is counted as a file with no lines.
//
// it reads what git tracks and not what is in the folder, so what was
// built, fetched or scratched there is not counted. it changes nothing.
package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// A kind is one sort of file, by a name to print and how a file's name
// shows it is one.
type kind struct {
	name string
	is   func(path string) bool
}

// ending is whether a path ends in one of some endings.
func ending(endings ...string) func(string) bool {
	return func(path string) bool {
		for _, each := range endings {
			if strings.HasSuffix(path, each) {
				return true
			}
		}
		return false
	}
}

// goTest is the ending of a go file that is a test.
const goTest = "_test.go"

// theRest is the kind of a file no kind claims.
const theRest = "everything else"

// kinds are the kinds, in the order they are printed. No two claim one
// file: a test's name ends in .go too, so go is a go file that is not a
// test.
var kinds = []kind{
	{"go", func(path string) bool {
		return ending(".go")(path) && !ending(goTest)(path)
	}},
	{"go tests", ending(goTest)},
	{"metal", ending(".metal")},
	{"objective-c", ending(".m")},
	{"c", ending(".c")},
	{"headers", ending(".h")},
	{"markdown", ending(".md")},
	{"yaml", ending(".yaml", ".yml")},
	{"shell", ending(".sh")},
	{"gamefile", func(path string) bool {
		return filepath.Base(path) == ".game"
	}},
}

// kindOf is the name of the kind a file is.
func kindOf(path string) string {
	for _, each := range kinds {
		if each.is(path) {
			return each.name
		}
	}
	return theRest
}

// linesOf is how many lines a file's contents hold, and whether they
// are text. Contents with a nought byte are not, and have no lines.
func linesOf(contents []byte) (lines int, text bool) {
	if bytes.IndexByte(contents, 0) >= 0 {
		return 0, false
	}
	lines = bytes.Count(contents, []byte{'\n'})
	if len(contents) > 0 && contents[len(contents)-1] != '\n' {
		lines++
	}
	return lines, true
}

// A count is what one kind adds up to: its files, and its lines.
type count struct {
	kind         string
	files, lines int
}

// counted is every kind's count over some files, in the kinds' order
// with everything else last, each file read by read. A kind with no
// file is left out. A file that cannot be read is an error that names
// it: a count that left a file out would be a wrong count.
func counted(paths []string,
	read func(string) ([]byte, error)) ([]count, error) {
	found := map[string]*count{}
	for _, path := range paths {
		contents, err := read(path)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		name := kindOf(path)
		if found[name] == nil {
			found[name] = &count{kind: name}
		}
		lines, _ := linesOf(contents)
		found[name].files++
		found[name].lines += lines
	}
	var counts []count
	for _, each := range append(names(), theRest) {
		if found[each] != nil {
			counts = append(counts, *found[each])
		}
	}
	return counts, nil
}

// names are the kinds' names, in order.
func names() []string {
	named := make([]string, len(kinds))
	for index, each := range kinds {
		named[index] = each.name
	}
	return named
}

// hundred turns a share into a percentage.
const hundred = 100

// table is the counts as lines to print: a heading, a row a kind with
// its files, its lines and its share of all the lines, and the total.
// With no lines at all every share is nought.
func table(counts []count) string {
	total := count{kind: "total"}
	for _, each := range counts {
		total.files += each.files
		total.lines += each.lines
	}
	row := func(of count) string {
		share := 0.0
		if total.lines > 0 {
			share = hundred * float64(of.lines) / float64(total.lines)
		}
		return fmt.Sprintf("%-16s %7d %9d %6.1f%%\n", of.kind, of.files,
			of.lines, share)
	}
	said := strings.Builder{}
	fmt.Fprintf(&said, "%-16s %7s %9s %7s\n", "kind", "files", "lines",
		"share")
	for _, each := range counts {
		said.WriteString(row(each))
	}
	said.WriteString(row(total))
	return said.String()
}

// tracked is every file git tracks under the folder this is run in.
func tracked() ([]string, error) {
	listed, err := exec.Command("git", "ls-files", "-z").Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-files: %w", err)
	}
	return strings.FieldsFunc(string(listed), func(mark rune) bool {
		return mark == 0
	}), nil
}

// main prints the breakdown of what git tracks here, or says why it
// cannot and exits 1.
func main() {
	paths, err := tracked()
	if err == nil {
		var counts []count
		if counts, err = counted(paths, os.ReadFile); err == nil {
			fmt.Print(table(counts))
			return
		}
	}
	fmt.Fprintln(os.Stderr, "breakdown:", err)
	os.Exit(1)
}
