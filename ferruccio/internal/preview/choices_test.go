package preview

import (
	"strings"
	"testing"
)

// limitIn is the GOMAXPROCS value in an environment, and whether it is there.
func limitIn(environment []string) (string, bool) {
	found, value := false, ""
	for _, entry := range environment {
		if got, ok := strings.CutPrefix(entry, "GOMAXPROCS="); ok {
			found, value = true, got
		}
	}
	return value, found
}

func TestChildIsGivenTwoProcessorsUnlessTheyChooseOtherwise(t *testing.T) {
	t.Setenv("GOMAXPROCS", "")
	if got, _ := limitIn(environment(Tools{})); got != "2" {
		t.Fatalf("the default limit is %q, not two", got)
	}
	for _, chosen := range []string{"2", "4", "8"} {
		if got, _ := limitIn(environment(Tools{Processors: chosen})); got != chosen {
			t.Fatalf("chose %s, the child got %q", chosen, got)
		}
	}
}

func TestAllProcessorsGivesTheChildNoLimitEvenWhenOneWasInherited(t *testing.T) {
	t.Setenv("GOMAXPROCS", "3")
	if got, found := limitIn(environment(Tools{Processors: ProcessorsAll})); found {
		t.Fatalf("the child was given a limit of %q", got)
	}
	if got, _ := limitIn(environment(Tools{})); got != "2" {
		t.Fatalf("an inherited limit beat the default: %q", got)
	}
}

func TestEnvironmentStillGivesTheChildItsCacheAndShelf(t *testing.T) {
	joined := strings.Join(environment(Tools{Cache: "/c", Shelf: "/s", Processors: ProcessorsAll}), "\n")
	if !strings.Contains(joined, "SLAYER_CACHE=/c") || !strings.Contains(joined, "SLAYER_SHELF=/s") {
		t.Fatal("the cache or shelf was dropped")
	}
}

func TestGPUChoiceAsksSlayerForTheGPURenderer(t *testing.T) {
	plan := &Plan{Kind: "set", Input: "in.yaml", Output: "out.png", Profile: Profile{Width: 16, Height: 12, GPU: true}}
	argv, err := plan.arguments(Tools{Slayer: "slayer"})
	if err != nil {
		t.Fatal(err)
	}
	joined := " " + strings.Join(argv, " ") + " "
	if !strings.Contains(joined, " --renderer gpu ") || strings.Contains(joined, " cpu ") {
		t.Fatalf("argv %v", argv)
	}
}

func TestCPUChoiceStaysTheDefaultsArgvAndTheTwoNeverMeet(t *testing.T) {
	plan := &Plan{Kind: "set", Input: "in.yaml", Output: "out.png", Profile: Profile{Width: 16, Height: 12, CPURenderer: true}}
	argv, err := plan.arguments(Tools{Slayer: "slayer"})
	if err != nil {
		t.Fatal(err)
	}
	joined := " " + strings.Join(argv, " ") + " "
	if !strings.Contains(joined, " --renderer cpu ") || strings.Contains(joined, " gpu ") {
		t.Fatalf("argv %v", argv)
	}
}

func TestACatalogueIsNeverGivenARendererChoice(t *testing.T) {
	plan := &Plan{Kind: "catalogue", Specimen: "subaru", Input: "in.yaml", Directory: "/d", Profile: Profile{GPU: true}}
	argv, err := plan.arguments(Tools{Catalogue: "catalogue"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.Join(argv, " "), "renderer") {
		t.Fatalf("a catalogue render was given a renderer: %v", argv)
	}
}

func TestNoCacheLineNamesSettingsTheFlagAndTheVariable(t *testing.T) {
	line := ErrNoCache.Error()
	for _, want := range []string{"Settings", "--cache", CacheVariable} {
		if !strings.Contains(line, want) {
			t.Fatalf("the line does not name %s: %s", want, line)
		}
	}
	if strings.Contains(line, "\n") {
		t.Fatal("the line is not one line")
	}
}
