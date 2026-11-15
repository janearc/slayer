package clean

import "testing"

func TestAdds(t *testing.T) {}

// anti-test of TestAdds
func TestAddsRefusesOverflow(t *testing.T) {}

func TestSubtracts(t *testing.T) {
	// anti-test of TestSubtracts
	t.Log("the near miss")
}
