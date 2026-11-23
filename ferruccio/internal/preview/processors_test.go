package preview

import "testing"

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
