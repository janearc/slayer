package cities

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/texture"
)

// sampleSide is how far across, in metres, a surface is sampled, and
// sampleStep how far apart the samples are: enough of a wall to hold
// several panels, boards and patches, at a spacing finer than the joints.
const (
	sampleSide = 6.0
	sampleStep = 0.01
)

// variation is how much a texture varies over a wall of it, as the
// standard deviation of the light it gives back over its mean, so that a
// dark surface and a pale one are measured alike.
func variation(surface texture.Texture) float64 {
	var sum, squares, count float64
	for x := 0.0; x < sampleSide; x += sampleStep {
		for y := 0.0; y < sampleSide; y += sampleStep {
			light := surface.At(maths.V(x, y, 0))
			level := (light.R + light.G + light.B) / 3
			sum, squares, count = sum+level, squares+level*level, count+1
		}
	}
	mean := sum / count
	return math.Sqrt(math.Max(0, squares/count-mean*mean)) / mean
}

// rowHouseCladding is the brick of a row house at the demo level, which
// is the least surface a tower may have: bricks that differ by a tone,
// with a mortar between them.
func rowHouseCladding(t *testing.T) texture.Texture {
	t.Helper()
	drawing := drawerOf(t, "portofino", 1)
	tone := drawing.fixed("mortar-tone")
	return brick{Brick: maths.RGB(0.45, 0.25, 0.2),
		Mortar: maths.RGB(tone, tone, tone),
		Length: drawing.fixed("brick-length"),
		Height: drawing.fixed("brick-height"),
		Joint:  drawing.fixed("mortar-joint"),
		Tone:   drawing.fixed("brick-tone")}
}

// mainSurface is the most varied texture among the parts of a tower of a
// skin, drawn at the demo level, for the skin's walls are its main parts.
func mainSurface(t *testing.T, skin Skin, names []string) float64 {
	t.Helper()
	pins := plainPins(skin)
	pins.Style = stylePunched
	most := 0.0
	for _, name := range names {
		demoTower := towerOf(t, 3, 0, pins)
		for _, one := range named(drawnTower(t, demoTower,
			"demo", 0), name) {
			if one.Material.Texture != nil {
				most = math.Max(most, variation(one.Material.Texture))
				break
			}
		}
	}
	return most
}

// wallParts are the parts that make the wall of a skin, by skin.
var wallParts = map[Skin][]string{skinGranite: {"pier", "apron"},
	skinConcrete: {"pier", "slab edge"}, skinGothic: {"buttress", "jamb"},
	skinCantilever: {"floor plate"}, skinCurtain: {"mullion", "plant"}}

func TestNoTowerSkinButMirrorIsSmootherThanARowHousesBrick(t *testing.T) {
	least := variation(rowHouseCladding(t))
	for skin, names := range wallParts {
		if got := mainSurface(t, skin, names); !(got >= least) {
			t.Errorf("%s varies by %v over a wall, a row house's brick by %v",
				skin, got, least)
		}
	}
}

func TestOnlyTheMirrorIsLeftSmoothAndItsMetalIsSeamed(t *testing.T) {
	pins := plainPins(skinMirror)
	drawn := drawnTower(t, towerOf(t, 3, 0, pins), "demo", 0)
	for _, one := range named(drawn, "mullion cap") {
		if one.Material.Texture == nil {
			t.Errorf("a mullion cap has no seams or finish")
		}
	}
	for _, one := range named(drawn, "panel") {
		if one.Material.Texture != nil {
			t.Errorf("the mirror glass has a texture: it is a true mirror")
		}
	}
}

func TestStoneHasJointsAGrainAndAReliefAndConcreteHasBoardsAndPatches(
	t *testing.T) {
	granite := plainPins(skinGranite)
	granite.Style = stylePunched
	for name, pins := range map[string]Pins{"granite": granite,
		"concrete": plainPins(skinConcrete)} {
		drawn := drawnTower(t, towerOf(t, 3, 0, pins), "demo", 0)
		piers := named(drawn, "pier")
		if len(piers) == 0 {
			t.Fatalf("%s: no piers", name)
		}
		look := piers[0].Material
		if _, product := look.Texture.(product); !product ||
			look.Crinkle == nil || !(look.Crinkle.Slope > 0) {
			t.Errorf("%s: a pier is %T with relief %v", name, look.Texture,
				look.Crinkle)
		}
	}
}

func TestEverySurfaceFadesToItsAverageWhenFarOffAndIsItselfUpClose(
	t *testing.T) {
	look := ashlar{brick{Brick: maths.Grey(0.5), Mortar: maths.Grey(0.1),
		Length: 1, Height: 0.5, Joint: 0.02, Tone: 0}}
	for _, surface := range []texture.Texture{look,
		coursed{lap{Board: maths.Grey(0.5), Height: 0.14, Shadow: 0.5,
			Share: 0.14, Tone: 0}}} {
		near := texture.Sample(surface, maths.V(0.5, 0.25, 0), 0)
		far := texture.Sample(surface, maths.V(0.5, 0.25, 0), 10)
		if near == far && surface.At(maths.V(0.5, 0.25, 0)) != near {
			t.Errorf("%T: near and far are alike", surface)
		}
		if far.R >= 0.5 || far.R <= 0.1 {
			t.Errorf("%T: far off is %v, not between joint and face", surface,
				far)
		}
	}
}

func TestAProductMultipliesItsSurfaceByItsGrain(t *testing.T) {
	product := product{Surface: uniform{Colour: maths.RGB(0.8, 0.4, 0.2)},
		Grain: uniform{Colour: maths.Grey(0.5)}}
	want := maths.RGB(0.4, 0.2, 0.1)
	if got := product.At(maths.Vec{}); got != want {
		t.Errorf("At is %v, wanted %v", got, want)
	}
	if got := product.Within(maths.Vec{}, 1); got != want {
		t.Errorf("Within is %v, wanted %v", got, want)
	}
}

func TestCyberSurfacesAreGrimedAndWetAndOthersAreNot(t *testing.T) {
	plain := plainPins(skinConcrete)
	cyber := plainPins(skinConcrete)
	cyber.Cyber = &yes
	wet := towerNumber(t, "cyber-wet-gloss")
	plainTower := towerOf(t, 3, 0, plain)
	// Poured concrete is not quite dull (the culture gives it a gloss of
	// 0.02); what a dry pier is not is wet, which is near a mirror.
	for _, one := range named(drawnTower(t, plainTower, "demo",
		0), "pier") {
		if one.Material.Gloss >= wet-tolerance {
			t.Errorf("a dry pier shines like a wet one: %v", one.Material.Gloss)
		}
	}
	cyberTower := towerOf(t, 3, 0, cyber)
	piers := named(drawnTower(t, cyberTower, "demo", 0), "pier")
	if len(piers) == 0 || piers[0].Material.Gloss <=
		towerNumber(t, "cyber-wet-gloss")-tolerance {
		t.Fatalf("a wet pier does not shine")
	}
	if _, product := piers[0].Material.Texture.(product); !product {
		t.Errorf("a wet pier is %T, not grime over its surface",
			piers[0].Material.Texture)
	}
}
