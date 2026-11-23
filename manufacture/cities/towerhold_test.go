package cities

import (
	"fmt"
	"hash/fnv"
	"testing"
)

// heldPins are the towers whose hull and quick drawings are held: one of
// each skin the detail work touches, plain, and one under the cyberpunk
// layer, as the close-up study has them.
func heldPins() map[string]Pins {
	held := map[string]Pins{}
	for _, skin := range []Skin{skinCurtain, skinMirror, skinGranite,
		skinConcrete, skinGothic, skinCantilever} {
		held[string(skin)] = plainPins(skin)
	}
	cyber := plainPins(skinCurtain)
	cyber.Cyber = &yes
	held["cyber"] = cyber
	return held
}

// heldKey is a number that says what a drawing is, part by part: each
// part's name, where it stands, how it is sized and what it is made of.
// Two drawings with one key are, for every purpose here, one.
func heldKey(drawn []Rendered) string {
	sum := fnv.New64a()
	for _, one := range drawn {
		fmt.Fprintf(sum, "%s %d %s %s\n", partKey(one),
			len(one.Mesh.Triangles), one.Material.Name, lampKey(one.Lamp))
	}
	return fmt.Sprintf("%d parts %x", len(drawn), sum.Sum64())
}

// lampKey is a lamp as the held keys were taken, by its heat and its
// luminance: the lamp type has since gained a tint and a reach, which its
// printed form would carry into the key and move every lamp's with no part
// of the drawing changed. A lamp with a tint or a reach is not one of the
// held towers', whose lamps have none.
func lampKey(lamp *Lamp) string {
	if lamp == nil {
		return "<nil>"
	}
	return fmt.Sprintf("&{%v %v}", lamp.Heat, lamp.Luminance)
}

// heldDrawings are the keys of the held towers at the hull and
// quick levels, taken before the demo level was given its own parts. A
// change here is a change to what every scene already drawn shows.
var heldDrawings = map[string]string{
	"cantilever hull":  "31 parts 5b864ee748542943",
	"cantilever quick": "2361 parts eef800a93bd734b9",
	"cyber hull":       "35 parts d6853e2cd637496",
	"cyber quick":      "1346 parts 15fac61a92602489",
	"curtain hull":     "31 parts 5b864ee748542943",
	"curtain quick":    "1031 parts 96fda84e31de7f9d",
	"mirror hull":      "31 parts a53e551a83cb460",
	"mirror quick":     "1537 parts 73970b43b5454806",
	"granite hull":     "31 parts 7f46ae9624009433",
	"granite quick":    "1093 parts 72a520641544e6eb",
	"concrete hull":    "31 parts ba497a9a3c3c0903",
	"concrete quick":   "1873 parts 165891d9774f75a7",
	"gothic hull":      "31 parts 4df9df814391acf7",
	"gothic quick":     "1995 parts 6e40817c26da3444",
}

func TestTheHullAndQuickLevelsAreHeldWhateverTheDemoLevelDraws(t *testing.T) {
	for name, pins := range heldPins() {
		for _, level := range []Detail{"hull", "quick"} {
			drawn := drawnTower(t, towerOf(t, 3, 0, pins), level, 0)
			key := name + " " + string(level)
			if got := heldKey(drawn); got != heldDrawings[key] {
				t.Errorf("%s: drawn as %q, held as %q", key, got,
					heldDrawings[key])
			}
		}
	}
}
