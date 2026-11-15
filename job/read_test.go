package job

import (
	"fmt"
	"path/filepath"
	"testing"
)

// a reads is one key read: the job that says it, what the accessor gives,
// and what it should give, written as fmt writes it.
type reads struct {
	key  string
	text string
	got  func(job Job) (any, bool)
	want string
}

// base is a path's last element, and whether it was said.
func base(path string, has bool) (any, bool) {
	return filepath.Base(path), has
}

// pathCamera is a camera path of one move, as a sequence plan writes one.
const pathCamera = "camera:\n  unit: m\n  look-at: {x: 0, y: 0, z: 0}\n" +
	"  moves:\n    - from: {x: 0, y: 0, z: 6}\n      to: {x: 0, y: 0, z: 2}\n" +
	"      speed: {value: 2, unit: m/s}\n"

// zoom is a zoom from sixty degrees to twenty over eight seconds.
const zoom = "  zoom: {from: {value: 60, unit: deg}, to: {value: 20, unit: " +
	"deg}, duration: {value: 8, unit: s}}\n"

// sceneReads are the scene's keys.
var sceneReads = []reads{
	{"$schema", still, func(job Job) (any, bool) {
		return job.Scene().Form(), true
	}, "set"},
	{"scene.set", still, func(job Job) (any, bool) {
		return base(job.Scene().Set())
	}, "moonlake.yaml"},
	{"scene.smoke", heading + "scene: {smoke: 3}\n",
		func(job Job) (any, bool) { return job.Scene().Smoke() }, "3"},
	{"scene.named", heading + "scene: {named: fire-night}\n",
		func(job Job) (any, bool) { return job.Scene().Named() },
		"fire-night"},
	{"scene.with", heading + "scene: {named: fire, with: {seed: 7}}\n",
		func(job Job) (any, bool) {
			return job.Scene().Parameters(), true
		}, "map[seed:7]"},
	{"scene.play", heading + "scene: {named: zappy, play: cues.yaml}\n" +
		"fps: 24\nduration: {value: 2, unit: s}\n" +
		"sound: {tune: '@moon', voicing: '@plain'}\n",
		func(job Job) (any, bool) { return base(job.Scene().Play()) },
		"cues.yaml"},
	{"scene.frames", heading + "scene: {frames: drawn}\n",
		func(job Job) (any, bool) { return base(job.Scene().Frames()) },
		"drawn"},
}

// filmReads are the film's keys.
var filmReads = []reads{
	{"film.preset", still + "film: {preset: portrait}\n",
		func(job Job) (any, bool) { return job.Film().Preset() },
		"portrait"},
	{"film.preset width", still + "film: {preset: portrait}\n",
		func(job Job) (any, bool) { return job.Film().Width() }, "1080"},
	{"film.preset height", still + "film: {preset: portrait}\n",
		func(job Job) (any, bool) { return job.Film().Height() }, "1920"},
	{"film.preset bare", still + "film: {preset: 720}\n",
		func(job Job) (any, bool) { return job.Film().Width() }, "1280"},
	{"film.width", still + "film: {width: 640}\n",
		func(job Job) (any, bool) { return job.Film().Width() }, "640"},
	{"film.height", still + "film: {height: 480}\n",
		func(job Job) (any, bool) { return job.Film().Height() }, "480"},
	{"film.width over preset", still + "film: {preset: 4k, width: 2000}\n",
		func(job Job) (any, bool) { return job.Film().Width() }, "2000"},
	{"film.rays", still + "film: {rays: 4}\n",
		func(job Job) (any, bool) { return job.Film().Rays() }, "4"},
	{"film.wide", still + "film: {wide: 55}\n",
		func(job Job) (any, bool) { return job.Film().Wide() }, "55"},
	{"film.tall", still + "film: {tall: 60}\n",
		func(job Job) (any, bool) { return job.Film().Tall() }, "60"},
	{"film.exposure", still + "film: {exposure: 0.5}\n",
		func(job Job) (any, bool) { return job.Film().Exposure() }, "0.5"},
	{"film.aperture", still + "film: {aperture: 0.1, focus: 12}\n",
		func(job Job) (any, bool) { return job.Film().Aperture() }, "0.1"},
	{"film.focus", still + "film: {aperture: 0.1, focus: 12}\n",
		func(job Job) (any, bool) { return job.Film().Focus() }, "12"},
	{"renderer", still + "renderer: gpu-native\n",
		func(job Job) (any, bool) { return job.Renderer() }, "gpu-native"},
	{"quality", still + "quality: preview\n",
		func(job Job) (any, bool) { return job.Quality() }, "preview"},
}

// cameraReads are the camera's keys.
var cameraReads = []reads{
	{"camera.name", still + "camera: {name: rim}\n",
		func(job Job) (any, bool) {
			lens, _ := job.Camera()
			return lens.Name()
		}, "rim"},
	{"camera.moves", still + pathCamera, func(job Job) (any, bool) {
		lens, _ := job.Camera()
		path, has := lens.Path()
		return path.Duration(), has
	}, "2s"},
	{"camera.zoom", still + "camera:\n  name: rim\n" + zoom,
		func(job Job) (any, bool) {
			lens, _ := job.Camera()
			zoomed, has := lens.Zoom()
			return zoomed != nil, has
		}, "true"},
}

// clockReads are the time's keys.
var clockReads = []reads{
	{"at", still + "at: {value: 20, unit: s}\n",
		func(job Job) (any, bool) { return job.Clock().At() }, "20s"},
	{"fps", film, func(job Job) (any, bool) { return job.Clock().Rate() },
		"24"},
	{"fps from preset", still + "film: {preset: video-preview}\n" +
		"duration: {value: 2, unit: s}\n",
		func(job Job) (any, bool) { return job.Clock().Rate() }, "6"},
	{"fps over preset", film + "film: {preset: video-preview}\n",
		func(job Job) (any, bool) { return job.Clock().Rate() }, "24"},
	{"duration", film, func(job Job) (any, bool) {
		return job.Clock().Duration(), !job.Clock().Still()
	}, "2s"},
	{"start", film + "start: {value: 40, unit: s}\n",
		func(job Job) (any, bool) { return job.Clock().Start() }, "40s"},
	{"scene-duration", film + "scene-duration: {value: 3600, unit: s}\n",
		func(job Job) (any, bool) {
			return job.Clock().SceneDuration()
		}, "1h0m0s"},
}

// developReads are the shutter's, the print's and the stock's keys.
var developReads = []reads{
	{"shutter.angle", film + "shutter: {angle: {value: 180, unit: deg}, " +
		"glimpses: 4}\n", func(job Job) (any, bool) {
		open, has := job.Shutter()
		return open.Angle(), has
	}, "180"},
	{"shutter.glimpses", film + "shutter: {angle: {value: 90, unit: " +
		"deg}, glimpses: 4}\n", func(job Job) (any, bool) {
		open, has := job.Shutter()
		return open.Glimpses(), has
	}, "4"},
	{"print.knee", still + "print: {knee: 0.7, top: 8}\n",
		func(job Job) (any, bool) {
			curve, has := job.Print()
			return curve.Knee(), has
		}, "0.7"},
	{"print.top", still + "print: {knee: 0.7, top: 8}\n",
		func(job Job) (any, bool) {
			curve, has := job.Print()
			return curve.Top(), has
		}, "8"},
	{"print.bloom", still + "print: {knee: 0.7, top: 8, bloom: 2}\n",
		func(job Job) (any, bool) {
			curve, _ := job.Print()
			return curve.Bloom()
		}, "2"},
	{"print.stops", still + "print: {knee: 0.7, top: 8, stops: -1}\n",
		func(job Job) (any, bool) {
			curve, _ := job.Print()
			return curve.Stops()
		}, "-1"},
	{"stock.name", still + "stock: {name: dusk}\n",
		func(job Job) (any, bool) {
			developed, has := job.Stock()
			return developed.Name(), has
		}, "dusk"},
	{"stock.seed", still + "stock: {name: dusk, seed: 7}\n",
		func(job Job) (any, bool) {
			developed, _ := job.Stock()
			return developed.Seed()
		}, "7"},
	{"stock.rated", still + "stock: {name: dusk, rated: 1}\n",
		func(job Job) (any, bool) {
			developed, _ := job.Stock()
			return developed.Rated()
		}, "1"},
}

// TestEveryKeyReads holds that every key reads, and that its accessor
// says it was said.
func TestEveryKeyReads(t *testing.T) {
	var all []reads
	for _, group := range [][]reads{sceneReads, filmReads, cameraReads,
		clockReads, developReads, titleReads, soundReads, outputReads} {
		all = append(all, group...)
	}
	for _, each := range all {
		t.Run(each.key, func(t *testing.T) {
			value, has := each.got(read(t, each.text))
			if !has || fmt.Sprint(value) != each.want {
				t.Errorf("%s read as %v (said %v), want %s", each.key,
					value, has, each.want)
			}
		})
	}
}
