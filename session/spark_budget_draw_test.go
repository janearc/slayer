//go:build draw

package session

import (
	"os"
	"strings"
	"testing"
)

func TestFrontRefusesAggregateSparkBudgetFromRealYAML(t *testing.T) {
	given := frontOnFixture(t)
	raw, err := os.ReadFile(given.set)
	if err != nil {
		t.Fatal(err)
	}
	block := `      sparks:
        rate: {value: 1000, unit: s^-1}
        lifetime: {value: 3, unit: s}
        radius: {value: 0.025, unit: m}
        release-height: {value: 3, unit: m}
        launch-speed: {value: 9, unit: m/s}
        spread: {value: 2, unit: m/s}
        updraft: {value: 8, unit: m/s}
        drag-time: {value: 0.8, unit: s}
        gravity: {value: 9.80665, unit: m/s^2}
        temperature: {value: 1650, unit: K}
        end-temperature: {value: 900, unit: K}
`
	text := strings.Replace(string(raw), "      flame:", block+"      flame:", 1)
	if err := os.WriteFile(given.set, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	// Every emitter is within its own 8192 limit. Four overlapping
	// targets plus the existing fire exceed the aggregate scene budget.
	if _, _, _, err := staged(given); err == nil || !strings.Contains(err.Error(), "front a passing front exceeds the budget of 8192 sparks") {
		t.Fatalf("aggregate front budget: %v", err)
	}
}
