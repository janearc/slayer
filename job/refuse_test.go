package job

import (
	"strings"
	"testing"
)

// a miss is a near miss of a key: the job that says it, and what the
// refusal must say, its place among them.
type miss struct {
	key   string
	text  string
	words []string
}

// sceneMisses are near misses of the scene's keys and the document's.
var sceneMisses = []miss{
	{"$schema", "$schema: https://github.com/janearc/slayer/schemas/" +
		"job-v2.json\nscene: {set: a.yaml}\n",
		[]string{"job.yaml:1:", "/$schema"}},
	{"$schema missing", "scene: {set: a.yaml}\n",
		[]string{"job.yaml:1:", "$schema"}},
	{"scene missing", heading, []string{"job.yaml:1:", "scene"}},
	{"scene misspelt", heading + "scene: {sett: a.yaml}\n",
		[]string{"job.yaml:2:", "sett"}},
	{"scene.set type", heading + "scene: {set: 3}\n",
		[]string{"job.yaml:2:", "/scene/set"}},
	{"scene.smoke bound", heading + "scene: {smoke: 0}\n",
		[]string{"job.yaml:2:", "/scene/smoke"}},
	{"scene.named empty", heading + "scene: {named: ''}\n",
		[]string{"job.yaml:2:", "/scene/named"}},
	{"scene.with alone", heading + "scene: {set: a.yaml, with: {a: 1}}\n",
		[]string{"job.yaml:2:", "with", "named"}},
	{"scene.play alone", heading + "scene: {set: a.yaml, play: c.yaml}\n",
		[]string{"job.yaml:2:", "play", "named"}},
	{"scene.play with no tune", heading +
		"scene: {named: zappy, play: c.yaml}\n",
		[]string{"job.yaml:2:", "play", "tune"}},
	{"scene.play type", heading + "scene: {named: zappy, play: 3}\n",
		[]string{"job.yaml:2:", "/scene/play"}},
	{"scene.frames type", heading + "scene: {frames: [a]}\n",
		[]string{"job.yaml:2:", "/scene/frames"}},
	{"scene none", heading + "scene: {}\n",
		[]string{"job.yaml:2:", "set, smoke, named or frames"}},
	{"scene two forms", heading + "scene: {set: a.yaml, smoke: 3}\n",
		[]string{"job.yaml:2:", "set and smoke"}},
	{"top-level misspelt", still + "fsp: 24\n",
		[]string{"job.yaml:", "fsp"}},
}

// filmMisses are near misses of the film's keys.
var filmMisses = []miss{
	{"film.preset", still + "film: {preset: 8k}\n",
		[]string{"job.yaml:3:", "/film/preset"}},
	{"film.width bound", still + "film: {width: 9000}\n",
		[]string{"job.yaml:3:", "/film/width"}},
	{"film.height bound", still + "film: {height: 0}\n",
		[]string{"job.yaml:3:", "/film/height"}},
	{"film.rays bound", still + "film: {rays: 65}\n",
		[]string{"job.yaml:3:", "/film/rays"}},
	{"film.wide bound", still + "film: {wide: 180}\n",
		[]string{"job.yaml:3:", "/film/wide"}},
	{"film.tall bound", still + "film: {tall: 0}\n",
		[]string{"job.yaml:3:", "/film/tall"}},
	{"film.exposure bound", still + "film: {exposure: 0}\n",
		[]string{"job.yaml:3:", "/film/exposure"}},
	{"film wide and tall", still + "film: {wide: 55, tall: 60}\n",
		[]string{"job.yaml:3:", "wide and tall"}},
	{"film.aperture bound", still + "film: {aperture: 0, focus: 12}\n",
		[]string{"job.yaml:3:", "/film/aperture"}},
	{"film.focus bound", still + "film: {aperture: 0.1, focus: 0}\n",
		[]string{"job.yaml:3:", "/film/focus"}},
	{"film aperture alone", still + "film: {aperture: 0.1}\n",
		[]string{"job.yaml:3:", "an aperture and no focus"}},
	{"film focus alone", still + "film: {focus: 12}\n",
		[]string{"job.yaml:3:", "a focus and no aperture"}},
	{"film aperture past focus", still + "film: {aperture: 12, focus: 12}\n",
		[]string{"job.yaml:3:", "as wide as its focus is far"}},
	{"film misspelt", still + "film: {widht: 640}\n",
		[]string{"job.yaml:3:", "widht"}},
	{"renderer", still + "renderer: vulkan\n",
		[]string{"job.yaml:3:", "/renderer"}},
	{"quality", still + "quality: draft\n",
		[]string{"job.yaml:3:", "/quality"}},
	{"vibes", still + "vibes: true\n",
		[]string{"job.yaml:3:", "/vibes"}},
	{"fastlight", still + "fastlight: cascades\n",
		[]string{"job.yaml:3:", "/fastlight"}},
	{"shortcut unknown", still + "shortcuts: {cascades: on}\n",
		[]string{"job.yaml:3:", "cascades"}},
	{"shortcut neither on nor off", still + "shortcuts: {lamp-store: fast}\n",
		[]string{"job.yaml:3:", "/shortcuts/lamp-store"}},
}

// cameraMisses are near misses of the camera's keys.
var cameraMisses = []miss{
	{"camera.name empty", still + "camera: {name: ''}\n",
		[]string{"job.yaml:3:", "/camera/name"}},
	{"camera.unit", still + "camera: {name: rim, unit: ''}\n",
		[]string{"job.yaml:3:", "/camera/unit"}},
	{"camera name and path", still + pathCamera + "  name: rim\n",
		[]string{"job.yaml:", "name and a path"}},
	{"camera neither", still + "camera:\n" + zoom,
		[]string{"job.yaml:3:", "neither a name nor a path"}},
	{"camera path half said", still + "camera: {unit: m}\n",
		[]string{"job.yaml:3:", "unit, look-at and moves"}},
	{"camera.look-at half a point", still + "camera: {unit: m, " +
		"look-at: {x: 0, y: 0}, moves: []}\n",
		[]string{"job.yaml:3:", "/camera/look-at"}},
	{"camera.moves empty", still + "camera: {unit: m, " +
		"look-at: {x: 0, y: 0, z: 0}, moves: []}\n",
		[]string{"job.yaml:3:", "/camera/moves"}},
	{"camera.up type", still + pathCamera + "  up: up\n",
		[]string{"job.yaml:", "/camera/up"}},
	{"camera.zoom bound", still + "camera:\n  name: rim\n  zoom: {from: " +
		"{value: 180, unit: deg}, to: {value: 20, unit: deg}, duration: " +
		"{value: 8, unit: s}}\n", []string{"job.yaml:5:", "/camera/zoom"}},
	{"camera smooth speed", still + pathCamera + "      ease: smooth\n",
		[]string{"job.yaml:7:", "smooth easing needs duration"}},
	{"camera path refused", still + strings.Replace(pathCamera,
		"z: 2}", "z: -2}", one), []string{"job.yaml:6:",
		"reaches its look target"}},
	{"camera zoom refused", still + "camera:\n  name: rim\n  zoom: {from: " +
		"{value: 5e-324, unit: deg}, to: {value: 20, unit: deg}, " +
		"duration: {value: 8, unit: s}}\n",
		[]string{"job.yaml:5:", "zoom needs finite fields"}},
}

// clockMisses are near misses of the time's keys.
var clockMisses = []miss{
	{"at bound", still + "at: {value: -1, unit: s}\n",
		[]string{"job.yaml:3:", "/at"}},
	{"at unit", still + "at: {value: 1, unit: min}\n",
		[]string{"job.yaml:3:", "/at/unit"}},
	{"at beside fps", film + "at: {value: 1, unit: s}\n",
		[]string{"job.yaml:5:", "at is a still's time"}},
	{"fps bound", still + "fps: 121\nduration: {value: 1, unit: s}\n",
		[]string{"job.yaml:3:", "/fps"}},
	{"duration bound", still + "fps: 24\nduration: {value: 0, unit: s}\n",
		[]string{"job.yaml:4:", "/duration"}},
	{"duration missing", still + "fps: 24\n",
		[]string{"job.yaml:3:", "says its duration"}},
	{"duration missing beside start", still + "start: {value: 1, unit: s}\n",
		[]string{"job.yaml:3:", "says its duration"}},
	{"duration missing beside stretch", still + "scene-duration: {value: " +
		"1, unit: s}\n", []string{"job.yaml:3:", "says its duration"}},
	{"rate missing", still + "duration: {value: 1, unit: s}\n",
		[]string{"job.yaml:3:", "needs a rate"}},
	{"rate missing from preset", still + "duration: {value: 1, unit: s}\n" +
		"film: {preset: 720}\n", []string{"job.yaml:4:", "needs a rate"}},
	{"start bound", film + "start: {value: 90000, unit: s}\n",
		[]string{"job.yaml:5:", "/start"}},
	{"scene-duration bound", film + "scene-duration: {value: 0, unit: s}\n",
		[]string{"job.yaml:5:", "/scene-duration"}},
}

// developMisses are near misses of the shutter's, print's and stock's.
var developMisses = []miss{
	{"shutter.angle bound", film + "shutter: {angle: {value: 361, unit: " +
		"deg}, glimpses: 4}\n", []string{"job.yaml:5:", "/shutter/angle"}},
	{"shutter.glimpses bound", film + "shutter: {angle: {value: 180, " +
		"unit: deg}, glimpses: 0}\n",
		[]string{"job.yaml:5:", "/shutter/glimpses"}},
	{"shutter on a still", still + "shutter: {angle: {value: 180, unit: " +
		"deg}, glimpses: 4}\n", []string{"job.yaml:3:", "a still has none"}},
	{"print.knee bound", still + "print: {knee: 1, top: 8}\n",
		[]string{"job.yaml:3:", "/print/knee"}},
	{"print.top bound", still + "print: {knee: 0.5, top: 1}\n",
		[]string{"job.yaml:3:", "/print/top"}},
	{"print.bloom bound", still + "print: {knee: 0.5, top: 8, bloom: 21}\n",
		[]string{"job.yaml:3:", "/print/bloom"}},
	{"print.stops bound", still + "print: {knee: 0.5, top: 8, stops: 41}\n",
		[]string{"job.yaml:3:", "/print/stops"}},
	{"print beside stock", still + "print: {knee: 0.5, top: 8}\n" +
		"stock: {name: dusk}\n", []string{"job.yaml:4:", "print and stock"}},
	{"stock.name off the shelf", still + "stock: {name: kodachrome}\n",
		[]string{"job.yaml:3:", "/stock/name"}},
	{"stock.seed bound", still + "stock: {name: ash, seed: -1}\n",
		[]string{"job.yaml:3:", "/stock/seed"}},
	{"stock.rated bound", still + "stock: {name: ash, rated: 7}\n",
		[]string{"job.yaml:3:", "/stock/rated"}},
}

// TestEveryNearMissIsRefusedByName holds that each near miss of a key is
// refused, saying where and what.
// anti-test of TestEveryKeyReads
// anti-test of TestEveryObjectIsClosed
// anti-test of TestTheStocksAreTheShelf
// anti-test of TestThePresetsAreOneTable
func TestEveryNearMissIsRefusedByName(t *testing.T) {
	var all []miss
	for _, group := range [][]miss{sceneMisses, filmMisses, cameraMisses,
		clockMisses, developMisses, titleMisses, soundMisses,
		outputMisses} {
		all = append(all, group...)
	}
	for _, each := range all {
		t.Run(each.key, func(t *testing.T) {
			refused(t, each.text, each.words...)
		})
	}
}
