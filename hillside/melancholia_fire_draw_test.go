//go:build draw

package hillside

import (
	"os"
	"strings"
	"testing"
)

// melancholiaFires is how many fires Melancholia burns: the bonfire, four
// named trees, and ten trees of the east rim. Counted by hand from the set.
const melancholiaFires = 15

// stagedMelancholia stages the set at path and gives its burning count,
// its retained fires, and the reason a sequence would refuse it for fire.
func stagedMelancholia(t *testing.T, path string) (int, int, string) {
	t.Helper()
	given := mustParse(t)
	given.set = path
	_, made, prepared, err := staged(given)
	if err != nil {
		t.Fatal(err)
	}
	if prepared.live == nil {
		t.Fatal("Melancholia staged with no retained scene")
	}
	defer prepared.live.closeBeds()
	return made.burning, len(prepared.live.fires), prepared.live.unsupported
}

// Melancholia's fire is fifteen recipes and no legacy alight: nothing in
// it makes a sequence refuse for its lamps. A copy with one tree set alight
// in the legacy way is refused, so the check can fail.
func TestMelancholiaFireIsRecipesOnly(t *testing.T) {
	if os.Getenv("SLAYER_SHELF") == "" || testing.Short() {
		t.Skip("no shelf named by SLAYER_SHELF, or short")
	}
	const path = "../sets/melancholia.yaml"
	_, fires, why := stagedMelancholia(t, path)
	if fires != melancholiaFires || why != "" {
		t.Fatalf("Melancholia retains %d fires, refused for %q; want %d and no refusal",
			fires, why, melancholiaFires)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "alight:") {
		t.Fatal("Melancholia's set says alight: use a fire recipe")
	}
	again := strings.Replace(string(raw), "      rim b:\n", "      rim b:\n        alight: true\n", 1)
	if again == string(raw) {
		t.Fatal("the copy did not change")
	}
	// the copy sits beside the set so that it finds the shelf's sources.yaml
	copied := "../sets/zz-melancholia-alight.yaml"
	defer os.Remove(copied)
	if err = os.WriteFile(copied, []byte(again), 0o600); err != nil {
		t.Fatal(err)
	}
	_, _, why = stagedMelancholia(t, copied)
	if !strings.Contains(why, "use fire recipes") {
		t.Fatalf("a legacy alight tree was not refused: %q", why)
	}
}
