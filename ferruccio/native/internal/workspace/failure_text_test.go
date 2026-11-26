package workspace

import (
	"errors"
	"strings"
	"testing"
)

func TestFailureDiagnosticLeadsWithTheChildsLastLine(t *testing.T) {
	transcript := "loading the set\nslayer draw: diskcache: SLAYER_CACHE is not set\n"
	got := failureDiagnostic(errors.New("shared Slayer preview failed: exit status 1"), transcript)
	if !strings.HasPrefix(got, "slayer draw: diskcache: SLAYER_CACHE is not set\n") {
		t.Fatalf("the child's sentence does not come first: %q", got)
	}
	if !strings.Contains(got, "exit status 1") || !strings.Contains(got, "loading the set") {
		t.Fatalf("the error or the rest of the output was dropped: %q", got)
	}
}

func TestFailureDiagnosticSurvivesTheInlineCut(t *testing.T) {
	transcript := strings.Repeat("lane chatter that goes on and on\n", 40) + "slayer: disk full\n"
	got := summarizeDiagnostic(failureDiagnostic(errors.New("failed"), transcript), 16)
	if !strings.HasPrefix(got, "slayer: disk full") {
		t.Fatalf("the reason was cut away: %q", got)
	}
}

func TestLongFailureDiagnosticIsCutWithinTheInlineLimit(t *testing.T) {
	transcript := "slayer draw: " + strings.Repeat("a", 40) + "\n"
	got := summarizeDiagnostic(failureDiagnostic(errors.New("failed"), transcript), 16)
	if !strings.HasPrefix(got, "slayer draw: ") || !strings.HasSuffix(got, "…") || len([]rune(got)) != messageSummaryRunes(16) {
		t.Fatalf("long reason was not bounded with its beginning intact: %q", got)
	}
}

func TestFailureDiagnosticWithNoOutputIsTheError(t *testing.T) {
	got := failureDiagnostic(errors.New("start shared Slayer preview: no such file"), "")
	if !strings.HasPrefix(got, "start shared Slayer preview") {
		t.Fatalf("got %q", got)
	}
}

func TestLastLineSkipsTrailingBlankLinesAndIsEmptyForNothing(t *testing.T) {
	if got := lastLine("a\nb  \n\n  \n"); got != "b" {
		t.Fatalf("got %q", got)
	}
	if got := lastLine(" \n "); got != "" {
		t.Fatalf("a blank transcript had the line %q", got)
	}
}
