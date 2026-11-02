package swatches

import (
	"go/ast"
	"go/parser"
	"go/token"
	"slices"
	"strings"
	"testing"

	"github.com/janearc/libtheme-css/primitives/swatch"
)

// every paint and lamp swatches.go makes one at a time has a name: each
// var it gives a value of its own, and not Terran's thirteen, is in
// named, under its own name in lower case, as itself.
func TestEverySwatchHasItsName(t *testing.T) {
	parsed, err := parser.ParseFile(token.NewFileSet(), "swatches.go", nil,
		0)
	if err != nil {
		t.Fatal(err)
	}
	var made []string
	for _, each := range parsed.Decls {
		block, ok := each.(*ast.GenDecl)
		if !ok || block.Tok != token.VAR {
			continue
		}
		for _, spec := range block.Specs {
			value := spec.(*ast.ValueSpec)
			if _, many := value.Values[0].(*ast.CompositeLit); many {
				continue
			}
			for _, name := range value.Names {
				made = append(made, strings.ToLower(name.Name))
			}
		}
	}
	slices.Sort(made)
	if len(made) != 18 || !slices.Equal(made, Names()) {
		t.Errorf("swatches.go makes %v, and the names are %v", made, Names())
	}
	for name, want := range map[string]swatch.Swatch{"rust": Rust,
		"tungsten": Tungsten, "daylight": Daylight, "ice": Ice} {
		if got, found := Named(name); !found || got != want {
			t.Errorf("%s is %v, %v", name, got, found)
		}
	}
}

// a name that is no swatch's finds nothing.
func TestANameThatIsNoSwatchFindsNothing(t *testing.T) {
	for _, name := range []string{"Rust", "terran", "", "mauve"} {
		if got, found := Named(name); found || got != (swatch.Swatch{}) {
			t.Errorf("%q: %v, %v", name, got, found)
		}
	}
}
