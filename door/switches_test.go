package door

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/janearc/slayer/job"
)

// agentHelp is what the tool says of itself to an agent.
func agentHelp(t *testing.T) string {
	t.Helper()
	env := envIn(t, context.Background(), t.TempDir())
	var said, wrong bytes.Buffer
	if code := Tool(env).Run([]string{"--agent-help"}, &said,
		&wrong); code != nothing {
		t.Fatalf("--agent-help left with %d: %s", code, wrong.String())
	}
	return said.String()
}

// missing is each of the schema's flags and shortcuts a help page does
// not hold by its flag and by the line the schema says of it.
func missing(page string) []string {
	var lost []string
	for _, each := range job.Switches() {
		if !strings.Contains(page, "--"+each.Name()) ||
			!strings.Contains(page, each.Line()) {
			lost = append(lost, each.Name())
		}
	}
	return lost
}

// TestAgentHelpHoldsEveryFlagAndShortcutOfTheSchema reads the tool's
// --agent-help: every flag and shortcut the job's schema declares is
// there, by name and with the schema's own line. A shortcut added to
// the schema and not to the door would fail here.
func TestAgentHelpHoldsEveryFlagAndShortcutOfTheSchema(t *testing.T) {
	if len(job.Switches()) < len([]string{"vibes", "fastlight",
		"lamp-store"}) {
		t.Fatalf("the schema declares %d switches", len(job.Switches()))
	}
	if lost := missing(agentHelp(t)); len(lost) > nothing {
		t.Errorf("--agent-help does not hold %v", lost)
	}
}

// TestAHelpPageWithoutASwitchIsFoundOut is the anti-test: the same
// page with one switch's line taken out, or its flag's name, is found
// to lack that switch and no other.
// anti-test of TestAgentHelpHoldsEveryFlagAndShortcutOfTheSchema
func TestAHelpPageWithoutASwitchIsFoundOut(t *testing.T) {
	page := agentHelp(t)
	for _, each := range job.Switches() {
		for _, gone := range []string{each.Line(), "--" + each.Name()} {
			lost := missing(strings.ReplaceAll(page, gone, ""))
			if len(lost) != 1 || lost[nothing] != each.Name() {
				t.Errorf("without %q the page lacks %v", gone, lost)
			}
		}
	}
}
