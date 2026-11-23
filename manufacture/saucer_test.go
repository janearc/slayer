package manufacture

import (
	_ "embed"
	"math"
	"testing"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/maths"
)

//go:embed testdata/saucer.yaml
var saucerContractRaw []byte

// the saucer's cabin light is found by its name, sits inside the dome's
// glass, above the deck, and is a red line every way.
func TestTheLarsonCarriesARedCabinLightInsideItsDome(t *testing.T) {
	var want struct {
		Name    string
		RedFrom float64 `yaml:"red-from"`
		RedTo   float64 `yaml:"red-to"`
	}
	if err := yamlfile.Read("saucer.yaml", saucerContractRaw, &want); err != nil {
		t.Fatal(err)
	}
	made := mustMake(t, "mcrn-larson", 5)
	lights, err := made.Lights(maths.Local())
	if err != nil {
		t.Fatal(err)
	}
	var cabin *lit
	for index := range lights {
		if lights[index].name == want.Name {
			cabin = &lights[index]
		}
	}
	if cabin == nil {
		t.Fatalf("no light named %q among %d", want.Name, len(lights))
	}
	size := saucerShape()
	at := cabin.At()
	inner := size.dome - size.wall
	deckPlace := maths.V(0, size.deck(), 0)
	if !(at.Y > size.deck()) || !(at.Sub(deckPlace).Len() < inner) {
		t.Errorf("the cabin light is at %v, not under the dome's glass", at)
	}
	colour := cabin.Light().colour
	wave := colour.wave.InMetres()
	if !colour.lined() || !(wave >= want.RedFrom && wave <= want.RedTo) {
		t.Errorf("the cabin light is %v m, not one red line", wave)
	}
	half, soft := cabin.Light().Angles()
	if !(math.Abs(half.InRadians()-math.Pi) < 1e-12) || soft.InRadians() != 0 {
		t.Errorf("the cabin light is a cone of %v and %v, not every way", half, soft)
	}
}

// a light with no name is still a light, and says it has none.
func TestALightWithNoNameSaysSo(t *testing.T) {
	made := mustMake(t, "campfire", 5)
	lights, err := made.Lights(maths.Local())
	if err != nil {
		t.Fatal(err)
	}
	if len(lights) == 0 || lights[0].name != "" {
		t.Errorf("a campfire's light is named %q", lights[0].name)
	}
}
