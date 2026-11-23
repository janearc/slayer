package sources

import (
	"path/filepath"
	"testing"
)

func TestAFontKeepsItsEndingOnTheShelfAndAScriptsPageDoesNot(t *testing.T) {
	list, err := listed(t, "sources:\n"+
		"  - key: a-face\n    from: https://example.org/a/Face-Bold.otf\n"+
		"    licence: open\n"+
		"  - key: a-page\n    from: https://example.org/cgi/page.cfm\n"+
		"    licence: open\n")
	if err != nil {
		t.Fatal(err)
	}
	shelf := NewShelf("/shelf", nil)
	for key, want := range map[string]string{"a-face": "a-face.otf",
		"a-page": "a-page"} {
		source, found := list.Find(key)
		if !found {
			t.Fatalf("no source %s", key)
		}
		if got := shelf.Path(source); got != filepath.Join("/shelf", want) {
			t.Errorf("%s is on the shelf at %s, want %s", key, got, want)
		}
	}
}
