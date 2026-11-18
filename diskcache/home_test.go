package diskcache

import (
	"os"
	"strings"
	"testing"
)

func TestHomeIsTheCacheVariable(t *testing.T) {
	home, err := Home(lookupOf(map[string]string{
		homeVariable: "/fast/cache", "SLAYER_RENDERS": "/renders"}))
	if err != nil || home != "/fast/cache" {
		t.Fatalf("home %q, %v; want /fast/cache", home, err)
	}
}

// anti-test of TestHomeIsTheCacheVariable
func TestHomeRefusesARelativeCacheVariable(t *testing.T) {
	_, err := Home(lookupOf(map[string]string{homeVariable: "cache"}))
	if err == nil || !strings.Contains(err.Error(), homeVariable) {
		t.Fatalf("a relative %s was taken: %v", homeVariable, err)
	}
}

// anti-test: with SLAYER_CACHE unset or empty, Home refuses by name,
// whatever the renders home, the user's cache folder and the temporary
// folder say, and nothing appears in any of them.
func TestHomeUnsetRefusesAndWritesNothing(t *testing.T) {
	folders := map[string]string{}
	for _, name := range []string{"SLAYER_RENDERS", "HOME",
		"XDG_CACHE_HOME", "TMPDIR"} {
		folders[name] = t.TempDir()
	}
	for _, said := range []map[string]string{folders, {homeVariable: ""}} {
		for name, folder := range folders {
			said[name] = folder
		}
		got, err := Home(lookupOf(said))
		if err == nil || !strings.Contains(err.Error(), homeVariable) {
			t.Fatalf("home %q, %v; want a refusal naming %s", got, err,
				homeVariable)
		}
	}
	for name, folder := range folders {
		if found, _ := os.ReadDir(folder); len(found) != nothing {
			t.Errorf("%s (%s) was written to: %v", name, folder, found)
		}
	}
}
