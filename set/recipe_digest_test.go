package set

import (
	"strings"
	"testing"
)

// digestSource is a space set with three recipes, two of them alike.
const digestSource = `$schema: https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: Recipe digests
  setting: space
  light: none
  recipes:
    low: {kind: street, wetness: 0.5, length: {value: 50, unit: m}}
    twin: {kind: street, wetness: 0.5, length: {value: 50, unit: m}}
    long: {kind: street, wetness: 0.5, length: {value: 60, unit: m}}
  set:
    things:
      one: {recipe: low, seed: 1, east: 0, north: 0, height: {value: 0, unit: m}}
  film: {width: 640, height: 400, wide: 40, rays: 1, exposure: 1}
  cameras:
    view:
      location: {at: [0, -60, 20]}
      orientation: {look_at: [0, 0, 0]}
`

// digestOfRecipe reads a source and gives the digest of a named recipe.
func digestOfRecipe(t *testing.T, source, name string) string {
	t.Helper()
	read, err := Read("digest.yaml", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	recipe, found := read.Recipe(name)
	if !found {
		t.Fatalf("no recipe %q", name)
	}
	return recipe.Digest()
}

func TestRecipeDigestIsStableAndNamesTheBlock(t *testing.T) {
	first := digestOfRecipe(t, digestSource, "low")
	if len(first) != 64 {
		t.Fatalf("digest %q is not a SHA-256 in hexadecimal", first)
	}
	if first != digestOfRecipe(t, digestSource, "low") {
		t.Fatal("one source gave two digests")
	}
	if first != digestOfRecipe(t, digestSource, "twin") {
		t.Fatal("two recipes written alike have two digests")
	}
}

func TestRecipeDigestChangesWithAnEdit(t *testing.T) {
	if digestOfRecipe(t, digestSource, "low") == digestOfRecipe(t, digestSource, "long") {
		t.Fatal("a longer street has the digest of a shorter")
	}
	edited := strings.Replace(digestSource, "low: {kind: street, wetness: 0.5", "low: {kind: street, wetness: 0.6", 1)
	if digestOfRecipe(t, digestSource, "low") == digestOfRecipe(t, edited, "low") {
		t.Fatal("an edited recipe kept its digest")
	}
}

func TestRecipeDigestIgnoresWhereAndCommentsAreWritten(t *testing.T) {
	moved := strings.Replace(digestSource, "    low:", "    # a comment\n\n    low:", 1)
	if digestOfRecipe(t, digestSource, "low") != digestOfRecipe(t, moved, "low") {
		t.Fatal("a comment or a blank line changed a digest")
	}
}

func TestRecipeDigestIsUnaffectedByAnotherRecipe(t *testing.T) {
	edited := strings.Replace(digestSource, "length: {value: 60, unit: m}", "length: {value: 70, unit: m}", 1)
	if digestOfRecipe(t, digestSource, "low") != digestOfRecipe(t, edited, "low") {
		t.Fatal("editing one recipe changed another's digest")
	}
}

func TestRecipeWithoutAFileHasNoDigest(t *testing.T) {
	if (Recipe{}).Digest() != "" {
		t.Fatal("a recipe that was never read has a digest")
	}
}
