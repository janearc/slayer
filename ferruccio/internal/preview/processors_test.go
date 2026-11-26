package preview

import (
	"runtime"
	"strconv"
	"testing"
)

func TestAProcessorCountThatIsNotANumberOrAllIsTheDefaultNotTheChilds(t *testing.T) {
	t.Setenv("GOMAXPROCS", "")
	for _, bad := range []string{"banana", "0", "-4", "2.5"} {
		if got, _ := limitIn(environment(Tools{Processors: bad})); got != "2" {
			t.Fatalf("%q gave the child a limit of %q, not the default two", bad, got)
		}
	}
	if got, _ := limitIn(environment(Tools{Processors: "3"})); got != "3" {
		t.Fatalf("a count the sheet does not offer but that is a number was %q", got)
	}
}

func TestQueuedRenderCarriesChosenProcessors(t *testing.T) {
	t.Setenv("SLAYER_CPU_WORKERS", "99")
	for chosen, want := range map[string]string{"": "2", "4": "4", "all": strconv.Itoa(min(runtime.NumCPU(), 1024)), "invalid": "2"} {
		if got := queueEnvironment(Tools{Processors: chosen})["SLAYER_CPU_WORKERS"]; got != want {
			t.Fatalf("processors %q became %q, want %q", chosen, got, want)
		}
	}
}

func TestQueuedRenderCarriesConfiguredCache(t *testing.T) {
	t.Setenv("SLAYER_CACHE", "/inherited")
	if got := queueEnvironment(Tools{Cache: "/chosen/cache"})["SLAYER_CACHE"]; got != "/chosen/cache" {
		t.Fatalf("queue cache = %q", got)
	}
}

func TestRenderCarriesConfiguredSceneData(t *testing.T) {
	t.Setenv("SLAYER_DATA", "/inherited")
	tools := Tools{Data: "/chosen/data"}
	if got := queueEnvironment(tools)["SLAYER_DATA"]; got != "/chosen/data" {
		t.Fatalf("queue scene data = %q", got)
	}
	var found bool
	for _, value := range environment(tools) {
		found = found || value == "SLAYER_DATA=/chosen/data"
		if value == "SLAYER_DATA=/inherited" {
			t.Fatal("inherited data overrode the chosen path")
		}
	}
	if !found {
		t.Fatal("direct render did not receive scene data")
	}
}
