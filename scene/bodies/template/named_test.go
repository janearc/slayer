package template

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/janearc/slayer/scene/bodies"
)

// every template the package makes has a name: each exported function
// of no arguments that hands back a bodies.Body is in named, so a file
// can call any of them. this is the rule, checked against the source.
func TestEveryTemplateHasAName(t *testing.T) {
	files, _ := filepath.Glob("*.go")
	var made []string
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, each := range parsed.Decls {
			fn, ok := each.(*ast.FuncDecl)
			if ok && fn.Recv == nil && fn.Name.IsExported() &&
				fn.Type.Params.NumFields() == 0 &&
				fn.Type.Results.NumFields() == 1 &&
				types(fn.Type.Results.List[0].Type) == "bodies.Body" {
				made = append(made, fn.Name.Name)
			}
		}
	}
	var called []string
	for _, make := range named {
		full := runtime.FuncForPC(reflect.ValueOf(make).Pointer()).Name()
		called = append(called, full[strings.LastIndex(full, ".")+1:])
	}
	slices.Sort(made)
	slices.Sort(called)
	if len(made) < 17 || !slices.Equal(made, called) {
		t.Errorf("the templates made are %v, and the ones named %v", made,
			called)
	}
}

// types is a result's type as written, package and name.
func types(written ast.Expr) string {
	selected, ok := written.(*ast.SelectorExpr)
	if !ok {
		return ""
	}
	return selected.X.(*ast.Ident).Name + "." + selected.Sel.Name
}

// a name finds its template: the body is the one its function makes,
// for a word and for two words joined by a dash.
func TestANameFindsItsTemplate(t *testing.T) {
	for name, want := range map[string]func() bodies.Body{
		"saturn": Saturn, "hot-jupiter": HotJupiter, "hale-bopp": HaleBopp,
	} {
		got, found := Named(name)
		if !found {
			t.Errorf("%s: not found", name)
		}
		for _, a := range apartWithColours(got, want()) {
			t.Errorf("%s: differs from its template: %s", name, a)
		}
	}
}

// what Named hands back is its own copy: changing it leaves the next
// one as the template makes it.
func TestNamedHandsBackItsOwnCopy(t *testing.T) {
	mine, _ := Named("saturn")
	scribble(reflect.ValueOf(&mine).Elem())
	if len(apartWithColours(mine, Saturn())) == 0 {
		t.Fatal("scribbling changed nothing, so the check below proves " +
			"nothing")
	}
	again, _ := Named("saturn")
	for _, a := range apartWithColours(again, Saturn()) {
		t.Errorf("after changing one, the next differs: %s", a)
	}
}

// a name that is not a template's finds nothing, and hands back the
// empty body: a capital, a space for the dash, nothing at all.
func TestANameThatIsNotATemplateFindsNothing(t *testing.T) {
	for _, name := range []string{"Saturn", "hot jupiter", "", "vulcan"} {
		got, found := Named(name)
		if found || !reflect.ValueOf(got).IsZero() {
			t.Errorf("%q: found %v, %+v", name, found, got)
		}
	}
}

// the names come in order, every one of them.
func TestNamesAreInOrder(t *testing.T) {
	names := Names()
	if len(names) != len(named) || !slices.IsSorted(names) ||
		names[0] != "charon" || names[len(names)-1] != "venus" {
		t.Errorf("names: %v", names)
	}
}
