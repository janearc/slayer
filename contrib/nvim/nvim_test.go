package nvim

import (
	"bytes"
	"os"
	"testing"
)

// Script is slayer.lua as it is on disk, and names what it sets up.
func TestScriptIsTheFile(t *testing.T) {
	raw, err := os.ReadFile("slayer.lua")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(Script(), raw) {
		t.Error("Script is not slayer.lua")
	}
	for _, word := range []string{"job%-v1", "autoread",
		"yaml-language-server", "slayer draw %"} {
		if !bytes.Contains(raw, []byte(word)) {
			t.Errorf("slayer.lua does not say %s", word)
		}
	}
}

// anti-test: it assumes no plugin manager, and a caller that changes
// what it was given does not change the next caller's.
func TestScriptAssumesNoPluginManager(t *testing.T) {
	for _, manager := range []string{"lazy", "packer", "lspconfig",
		"Plug"} {
		if bytes.Contains(Script(), []byte("require('"+manager)) {
			t.Errorf("slayer.lua requires %s", manager)
		}
	}
	given := Script()
	given[0] = 'X'
	if Script()[0] == 'X' {
		t.Error("Script gave its own bytes to be changed")
	}
}
