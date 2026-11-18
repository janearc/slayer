package manufacture

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// TestPublicFunctionsHaveDocumentation keeps the public construction API
// discoverable through go doc as new recipes and helpers are added.
func TestPublicFunctionsHaveDocumentation(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") ||
			strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), entry.Name(), nil, parser.ParseComments)
		if err != nil {
			t.Fatalf("%s: %v", entry.Name(), err)
		}
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Recv != nil || !function.Name.IsExported() {
				continue
			}
			found++
			if function.Doc == nil || !strings.HasPrefix(function.Doc.Text(), function.Name.Name+" ") {
				t.Errorf("%s: public function %s needs a Go documentation comment",
					entry.Name(), function.Name.Name)
			}
		}
	}
	if found == 0 {
		t.Fatal("no public manufacture functions were inspected")
	}
}

// TestDirectSceneConstructorsHaveDocumentation covers the public functions
// used when a scene builds its own geometry and media.
func TestDirectSceneConstructorsHaveDocumentation(t *testing.T) {
	for _, directory := range []string{"../mesh", "../scene/shape", "../scene/medium", "../scene/thing"} {
		t.Run(directory, func(t *testing.T) {
			entries, err := os.ReadDir(directory)
			if err != nil {
				t.Fatal(err)
			}
			found := 0
			for _, entry := range entries {
				if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") ||
					strings.HasSuffix(entry.Name(), "_test.go") {
					continue
				}
				path := directory + "/" + entry.Name()
				file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ParseComments)
				if err != nil {
					t.Fatalf("%s: %v", path, err)
				}
				for _, declaration := range file.Decls {
					function, ok := declaration.(*ast.FuncDecl)
					if !ok || function.Recv != nil ||
						!function.Name.IsExported() {
						continue
					}
					found++
					if function.Doc == nil || !strings.HasPrefix(function.Doc.Text(), function.Name.Name+" ") {
						t.Errorf("%s: constructor %s needs a Go documentation comment",
							path, function.Name.Name)
					}
				}
			}
			if found == 0 {
				t.Fatal("no direct scene constructors were inspected")
			}
		})
	}
}
