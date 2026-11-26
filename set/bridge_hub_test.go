package set_test

import "testing"

// A bridge recipe that is a hub, and says no depth, is read.
func TestAHubBridgeRecipeIsRead(t *testing.T) {
	if err := bridgeSetWith(t, "      hub: true\n"); err != nil {
		t.Error(err)
	}
}

// Anti: a hub that is not true or false is refused by its name.
// anti-test of TestAHubBridgeRecipeIsRead
func TestAHubThatIsNotATruthIsRefused(t *testing.T) {
	mustHold(t, "hub not a truth", bridgeSetWith(t, "      hub: maybe\n"),
		"hub")
}
