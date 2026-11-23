package set_test

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

// aCanal is a small world with one channel, as written. The channel runs
// north for 200 m, 12 m between walls half a metre thick, its bed 3 m under
// ground at 0.5 m.
const aCanal = `scene:
  name: a canal
  film:
    width: 64
    height: 36
    rays: 1
    wide: 60
    exposure: 1
    print:
      knee: 0.7
      top: 8
      source: eye
  world:
    unit: 1
    materials:
      brick:
        colour: {srgb_linear: [0.3, 0.1, 0.05]}
        source: eye
      stone:
        colour: {srgb_linear: [0.3, 0.3, 0.3]}
        source: eye
      paving:
        colour: {srgb_linear: [0.1, 0.1, 0.1]}
        source: eye
      silt:
        colour: {srgb_linear: [0.02, 0.02, 0.01]}
        source: eye
    channels:
      canal:
        line: [[0, -100], [0, 100]]
        width: {value: 12, unit: m}
        depth: {value: 3, unit: m}
        top: {value: 0.5, unit: m}
        reach: {value: 20, unit: m}
        ground: paving
        bed: silt
        wall: {material: brick, thickness: {value: 0.5, unit: m}}
        coping:
          material: stone
          height: {value: 0.2, unit: m}
          overhang: {value: 0.1, unit: m}
        source: eye
`

// canalRead is the canal's world, as written or changed by each pair of
// was and now, or the refusal it meets.
func canalRead(t *testing.T, changes ...string) (set.World, error) {
	t.Helper()
	written := aCanal
	for index := 0; index+1 < len(changes); index += 2 {
		if !strings.Contains(written, changes[index]) {
			t.Fatalf("%q is not in the canal", changes[index])
		}
		written = strings.Replace(written, changes[index],
			changes[index+1], 1)
	}
	made, err := set.Read("canal.yaml", []byte(written))
	if err != nil {
		return set.World{}, err
	}
	world, _ := made.World()
	return world, nil
}

// a channel is each part as it was written, in metres.
func TestAChannelIsWhatItSays(t *testing.T) {
	world, err := canalRead(t)
	if err != nil {
		t.Fatal(err)
	}
	channels := world.Channels()
	if len(channels) != 1 {
		t.Fatalf("channels %+v", channels)
	}
	canal := channels[0]
	line := canal.Line()
	if canal.Name() != "canal" || len(line) != 2 ||
		line[0] != [2]float64{0, -100} || line[1] != [2]float64{0, 100} {
		t.Errorf("the canal is %q along %v", canal.Name(), line)
	}
	if canal.Width() != 12 || canal.Depth() != 3 || canal.Top() != 0.5 ||
		canal.Reach() != 20 || canal.WallThickness() != 0.5 ||
		canal.CopingHeight() != 0.2 || canal.CopingOverhang() != 0.1 {
		t.Errorf("the canal measures %+v", canal)
	}
	if canal.Ground() != "paving" || canal.Bed() != "silt" ||
		canal.Wall() != "brick" || canal.Coping() != "stone" {
		t.Errorf("the canal is made of %+v", canal)
	}
}

// a channel's line is its own: changing the copy changes nothing.
func TestAChannelsLineIsACopy(t *testing.T) {
	world, err := canalRead(t)
	if err != nil {
		t.Fatal(err)
	}
	world.Channels()[0].Line()[0][0] = 99
	if got := world.Channels()[0].Line()[0][0]; got != 0 {
		t.Errorf("the line's first point moved to %g", got)
	}
}

// a channel that leaves out a part, or says it badly, is refused by what
// is wrong with it, where it was written.
func TestAChannelThatIsWrongIsRefused(t *testing.T) {
	cases := []struct{ name, was, now, said string }{
		{"no bed", "        bed: silt\n", "", "bed"},
		{"one point", "[[0, -100], [0, 100]]", "[[0, -100]]", "two"},
		{"a point of three numbers", "[[0, -100], [0, 100]]",
			"[[0, -100], [0, 100, 1]]", "east, north"},
		{"a line that stays", "[[0, -100], [0, 100]]",
			"[[0, 100], [0, 100]]", "no length"},
		{"feet", "width: {value: 12, unit: m}",
			"width: {value: 12, unit: ft}", "unit"},
		{"no width", "width: {value: 12, unit: m}",
			"width: {value: 0, unit: m}", "value"},
		{"a coping the wall's height", "height: {value: 0.2, unit: m}",
			"height: {value: 3, unit: m}", "no height"},
		{"a coping past the middle", "overhang: {value: 0.1, unit: m}",
			"overhang: {value: 6, unit: m}", "past its middle"},
		{"a wall of a material nobody holds", "material: brick",
			"material: granite", "granite"},
		{"a key it does not take", "reach: {value: 20, unit: m}",
			"reach: {value: 20, unit: m}\n        water: 1", "water"},
	}
	for _, each := range cases {
		_, err := canalRead(t, each.was, each.now)
		if err == nil || !strings.Contains(err.Error(), each.said) {
			t.Errorf("%s: read as %v, wanted a refusal naming %q",
				each.name, err, each.said)
		}
	}
}

// a channel's key is in the world's order with the others, the unit the
// canal writes first among them.
func TestAChannelKeepsItsPlaceInTheWorldsOrder(t *testing.T) {
	world, err := canalRead(t)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(world.Order(), " ")
	if got != "unit materials channels" {
		t.Errorf("the world's order is %q", got)
	}
}
