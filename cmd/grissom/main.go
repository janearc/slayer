// Command grissom is the survey viewer. It lays images onto the sky's
// triangles and draws them on Metal, with the earth as hexes beside them.
//
// Three views. The sky, seen from inside: drag to look, scroll to zoom.
// The earth, seen from outside: e flips to it and back, drag turns it,
// scroll comes nearer, p places the observer at the crosshair.
//
// Standing at that place: f flips to it and back, drag turns you round
// the horizon and up, scroll zooms, comma and period step the clock an hour.
//
// Keys 1 to 4 turn the sky's layers on and off; o cycles the outlines
// through on, faded and off; q quits. For low vision each of these is the
// viewer's to set: = and - move the lightness, c the colourfulness, b the
// background, [ and ] the outline's width.
//
// All colour is libtheme's. Each sky layer's hue is its wavelength's false
// colour. The earth's land and sea are measured reflectances from the
// USGS Spectral Library under libtheme's D65 daylight, and where the land
// lies is Natural Earth's; both come through celestial-data.
package main

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa -framework Metal -framework MetalKit -framework QuartzCore
#include <stdlib.h>
#include "viewer.h"
*/
import "C"

import (
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"log"
	"maps"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/janearc/celestial-data/naturalearth"
	"github.com/janearc/celestial-data/usgs"
	"github.com/janearc/celestial/fits"
	"github.com/janearc/celestial/grissom/composite"
	"github.com/janearc/celestial/grissom/layer"
	"github.com/janearc/celestial/grissom/mesh"
	"github.com/janearc/celestial/measure"
	"github.com/janearc/celestial/radiation"
	"github.com/janearc/celestial/research"
	"github.com/janearc/celestial/sky"
	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/libtheme-css/spaces/ok"
	"github.com/janearc/libtheme-css/spaces/srgb"
	"github.com/janearc/slayer/hud"
	"github.com/janearc/slayer/observer"
)

// Cocoa wants its run loop on the main thread.
func init() { runtime.LockOSThread() }

// The views, as goView names them to the window.
const (
	skyView   = 0
	earthView = 1
	placeView = 2
)

// source is one image to lay, the wavelength it was taken at in
// nanometres (none for a layer that only occludes), and the shares of its
// cells' values its stretch runs between (picked).
type source struct {
	name, path  string
	nm          float64
	floor, ceil float64
	occludes    []string
	strength    float64
	logarithmic bool
}

// view is what the window shows, changed by the mouse and keys.
var view struct {
	sync.Mutex
	mode int
	// the sky from inside: where the middle of the window looks, and how
	// wide it is, in degrees.
	ra, dec, fov float64
	// the earth from outside: the place under the middle of the window,
	// and the camera's distance in earth radii.
	lon, lat, distance float64
	// standing on the earth: where, which way, how wide and when.
	placed               bool
	placeLon, placeLat   float64
	azimuth, altitude    float64
	placeFov             float64
	clock                time.Time
	width                int
	outline              int
	lightness            float64
	chroma, ground, line int
	stack                *composite.Stack
	sources              []source
	on                   []bool
	cells                []mesh.Cell
	skyColours           []byte
	backCount            int
	centreAlt            float64
	hudSize              int
	hudSaid              string
	theme                int
	wide                 *composite.Stack
	wideSources          []source
	wideOn               []bool
	backCells            []mesh.Cell
	wideDepth            int
	backColours          []byte
	earth                earthMesh
}

// earthMesh is the earth's triangles, the hex each corner is, and the
// measured light of each hex's ground.
type earthMesh struct {
	corners   []float32
	hexOf     []int
	hexes     [][3]float64
	onLand    []bool
	colours   []byte
	land, sea swatch.Swatch
	map_      *naturalearth.Map
	marked    int
	// the cells' triangles by key, and the depth they are at, to find the
	// hex nearest a place.
	triangleOf map[uint64]int
	depth      int
	// NASA's images laid on the hexes, those on this machine, and which
	// is shown: one past the last is the measured land and sea.
	images  []earthImage
	showing int
}

// outlines are the outline's strengths the o key cycles through.
var outlines = []float64{0.35, 0.12, 0}

// main loads the surveys and opens the window or writes offscreen views.
func main() {
	data := flag.String("data", "", "the sgra data directory")
	earthData := flag.String("earth", "", "the directory holding celestial-data's earth files")
	depth := flag.Int("depth", 13, "the mesh depth the sky's layers are laid at")
	earthDepth := flag.Int("earth-depth", 7, "the mesh depth the earth's hexes are at")
	shot := flag.String("shot", "", "draw one frame offscreen to this png and stop")
	shotView := flag.Int("view", skyView, "the view a shot draws: 0 sky, 1 earth, 2 standing")
	startGround := flag.Int("ground", 0, "the background to start at, as b steps")
	startChroma := flag.Int("chroma", 0, "the colourfulness to start at, as c steps")
	skyData := flag.String("sky", "", "the directory holding celestial-data's whole-sky maps")
	wideDepth := flag.Int("wide-depth", 8, "the mesh depth the whole-sky maps are laid at")
	startFov := flag.Float64("fov", 1, "the sky view's width to start at, in degrees")
	look := flag.String("look", "", "lon,lat the earth view and the observer start over")
	colourways := flag.String("colourways", "colourways", "the directory of the hud's colourways")
	startTheme := flag.String("colourway", "twilight", "the hud colourway to start in")
	startEarth := flag.Int("earth-image", 0, "the earth to start on, as m steps")
	flag.Parse()
	readThemes(*colourways, *startTheme)

	view.sources = []source{
		{"8 um glow (GLIMPSE I4)", *data + "/data-irac/GLM_00000+0000_mosaic_I4.fits",
			irac4, 0.45, 0.997, nil, 1, false},
		{"3.6 um stars (GLIMPSE I1)", *data + "/data-irac/GLM_00000+0000_mosaic_I1.fits",
			irac1, 0.70, 0.999, nil, 1, false},
		{"radio 1.28 GHz (MeerKAT)", *data + "/data-meerkat/MeerKAT_Galactic_Centre_1284MHz-StokesI.fits",
			radiation.HertzToNm(meerkatHz), 0.75, 0.998, nil, 0.8, false},
		{"dust and gas column (PPMAP, Herschel)", *data + "/data-ppmap-l000_cdens/l000_cdens.fits",
			0, 0.20, 0.99, []string{"3.6 um stars (GLIMPSE I1)", "8 um glow (GLIMPSE I4)"}, 0.7, false},
	}
	view.lightness = 0.75
	view.outline = 1 // faded: at whole-sky widths full outlines shimmer
	view.ground, view.chroma = *startGround, *startChroma
	view.ra, view.dec, view.fov = 266.4168371, -29.0078106, *startFov
	view.lon, view.lat, view.distance = -100, 30, 3.2
	view.placeLon, view.placeLat = -122.4194, 37.7749
	if *look != "" {
		_, err := fmt.Sscanf(*look, "%f,%f", &view.lon, &view.lat)
		must(err)
		view.placeLon, view.placeLat = view.lon, view.lat
	}
	view.azimuth, view.altitude, view.placeFov = 180, 20, 60
	view.clock = time.Now()

	skyCorners := buildSky(*depth)
	backCorners := buildBack(*skyData, *wideDepth)
	buildEarth(*earthData, *earthDepth)
	view.earth.marked = -1
	view.earth.showing = *startEarth
	recolour()
	bg, _ := srgb.FromSwatch(ground())
	C.viewer_background(C.double(bg.R), C.double(bg.G), C.double(bg.B))
	log.Printf("%d sky triangles, %d earth triangles", len(view.cells), len(view.earth.hexOf)/3)
	log.Print("sky: drag, scroll; e earth; p place at the crosshair; f stand there; , . clock; 1-4 layers; o outlines; = - lightness; c colour; b background; [ ] outline width; m cycles Blue Marble, Black Marble, measured earth; q quits")

	meshes := [3]C.viewer_mesh{
		meshOf(skyCorners, view.skyColours),
		meshOf(backCorners, view.backColours),
		meshOf(view.earth.corners, view.earth.colours),
	}
	if *shot != "" {
		view.placed = true
		faceCentre()
		mark()
		for _, each := range []int{skyView, earthView, placeView} {
			if *shotView >= 0 && each != *shotView {
				continue
			}
			view.mode = each
			recolourEarth()
			name := []string{"sky", "earth", "standing"}[each]
			writeShot(*shot+"-"+name+".png", meshes)
		}
		return
	}
	if err := installIcons(); err != nil {
		log.Fatal(err)
	}
	title := C.CString("grissom")
	defer C.free(unsafe.Pointer(title))
	var why [1024]C.char
	if C.viewer_run(meshes[0], meshes[1], meshes[2], title, &why[0], C.int(len(why))) != 0 {
		log.Fatal(C.GoString(&why[0]))
	}
}

// meshOf is triangles' corners and colours as the window takes them.
func meshOf(corners []float32, colours []byte) C.viewer_mesh {
	return C.viewer_mesh{
		corners: (*C.float)(unsafe.Pointer(&corners[0])),
		colours: (*C.uint8_t)(unsafe.Pointer(&colours[0])),
		count:   C.int(len(corners) / 9),
	}
}

// buildSky lays the layers, stacks them and makes their triangles.
func buildSky(depth int) []float32 {
	layers := make([]*layer.Layer, len(view.sources))
	var group sync.WaitGroup
	for i, each := range view.sources {
		group.Add(1)
		go func() {
			defer group.Done()
			start := time.Now()
			layers[i] = lay(each, depth)
			log.Printf("laid %s: %d cells in %v", each.name, len(layers[i].Cells()), time.Since(start).Round(time.Second))
		}()
	}
	group.Wait()
	view.stack = composite.NewStack()
	keys := make(map[uint64]bool)
	for i, each := range view.sources {
		low, high := stretch(layers[i], each.floor, each.ceil)
		must(view.stack.Add(layers[i], low, high))
		must(view.stack.Strength(each.name, each.strength))
		view.on = append(view.on, true)
		for _, key := range layers[i].Cells() {
			keys[key] = true
		}
	}
	for _, each := range view.sources {
		for _, other := range each.occludes {
			must(view.stack.Relate(each.name, other, composite.Occludes))
		}
	}
	var corners []float32
	for _, key := range slices.Sorted(maps.Keys(keys)) {
		cell, err := mesh.FromKey(key)
		must(err)
		view.cells = append(view.cells, *cell)
		corners = appendCorners(corners, *cell)
	}
	view.skyColours = make([]byte, len(view.cells)*12)
	return corners
}

// cellsAt is every cell of the mesh at a depth.
func cellsAt(depth int) []mesh.Cell {
	cells := mesh.Faces()
	for range depth {
		var next []*mesh.Cell
		for _, cell := range cells {
			split, err := cell.Children()
			must(err)
			next = append(next, split...)
		}
		cells = next
	}
	all := make([]mesh.Cell, len(cells))
	for i, cell := range cells {
		all[i] = *cell
	}
	return all
}

// appendCorners is corners with a cell's three added, on the unit sphere.
func appendCorners(corners []float32, cell mesh.Cell) []float32 {
	one, two, three := cell.Corners()
	for _, corner := range []mesh.Point{one, two, three} {
		x, y, z := unit(corner.Lon(), corner.Lat())
		corners = append(corners, float32(x), float32(y), float32(z))
	}
	return corners
}

// buildEarth makes the earth's triangles at a depth, finds the hex each
// corner is, and asks the land of each hex. Land and sea take their
// measured reflectance under libtheme's D65.
func buildEarth(dir string, depth int) {
	file, err := os.Open(dir + "/ne_50m_land.geojson")
	must(err)
	land, err := naturalearth.Read(file)
	must(err)
	file.Close()
	view.earth.map_ = land
	library, err := os.Open(dir + "/ASCIIdata_splib07a.zip")
	must(err)
	info, err := library.Stat()
	must(err)
	view.earth.sea = reflected(library, info.Size(), "Seawater_Open_Ocean_SW2_lwch_BECKa_AREF")
	view.earth.land = reflected(library, info.Size(), "Grass_dry.5+.5green_AMX28_BECKa_AREF")
	library.Close()

	start := time.Now()
	ids := make(map[[3]int64]int)
	view.earth.triangleOf = make(map[uint64]int)
	view.earth.depth = depth
	for _, cell := range cellsAt(depth) {
		view.earth.triangleOf[cell.Key()] = len(view.earth.corners) / 9
		before := len(view.earth.corners)
		view.earth.corners = appendCorners(view.earth.corners, cell)
		for k := before; k < len(view.earth.corners); k += 3 {
			x, y, z := view.earth.corners[k], view.earth.corners[k+1], view.earth.corners[k+2]
			key := [3]int64{int64(math.Round(float64(x) * 1e6)), int64(math.Round(float64(y) * 1e6)), int64(math.Round(float64(z) * 1e6))}
			id, seen := ids[key]
			if !seen {
				id = len(view.earth.hexes)
				ids[key] = id
				view.earth.hexes = append(view.earth.hexes, [3]float64{float64(x), float64(y), float64(z)})
			}
			view.earth.hexOf = append(view.earth.hexOf, id)
		}
	}
	view.earth.onLand = make([]bool, len(view.earth.hexes))
	workers := 4
	var group sync.WaitGroup
	for w := range workers {
		group.Add(1)
		go func() {
			defer group.Done()
			for i := w; i < len(view.earth.hexes); i += workers {
				lon, lat := lonLat(view.earth.hexes[i])
				view.earth.onLand[i] = land.Holds(lon, lat)
			}
		}()
	}
	group.Wait()
	view.earth.colours = make([]byte, len(view.earth.hexOf)*4)
	log.Printf("earth: %d hexes in %v", len(view.earth.hexes), time.Since(start).Round(time.Millisecond))
	for _, each := range []earthImage{{name: "Blue Marble, July 2004", file: "world.200407.3x5400x2700.jpg"},
		{name: "Black Marble, 2016, at night", file: "BlackMarble_2016_01deg.jpg"}} {
		if _, err := os.Stat(dir + "/" + each.file); err != nil {
			continue
		}
		each.light = layMarble(dir + "/" + each.file)
		view.earth.images = append(view.earth.images, each)
	}
}

// layMarble lays Blue Marble onto the hexes: each hex is the mean of the
// pixels nearest its centre, taken as light through libtheme's sRGB, not
// as bytes. The image stays on this machine; nothing made from it is
// shipped.
func layMarble(path string) []swatch.Swatch {
	start := time.Now()
	file, err := os.Open(path)
	must(err)
	picture, err := jpeg.Decode(file)
	must(err)
	file.Close()
	// sRGB is linear in light once decoded, so a pixel's light is the sum
	// of its three channels' lights; libtheme gives each.
	var channels [3][256]swatch.Swatch
	for v := range 256 {
		channels[0][v] = srgb.RGB8(uint8(v), 0, 0).Swatch()
		channels[1][v] = srgb.RGB8(0, uint8(v), 0).Swatch()
		channels[2][v] = srgb.RGB8(0, 0, uint8(v)).Swatch()
	}
	bounds := picture.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	workers := 4
	sums := make([][][4]float64, workers)
	var group sync.WaitGroup
	for w := range workers {
		group.Add(1)
		go func() {
			defer group.Done()
			sum := make([][4]float64, len(view.earth.hexes))
			for y := w; y < height; y += workers {
				lat := 90 - (float64(y)+0.5)*180/float64(height)
				for x := range width {
					lon := -180 + (float64(x)+0.5)*360/float64(width)
					hex := nearestHex(lon, lat)
					r, g, b, _ := picture.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
					light := [3]swatch.Swatch{channels[0][r>>8], channels[1][g>>8], channels[2][b>>8]}
					for _, part := range light {
						px, py, pz := part.XYZ()
						sum[hex][0] += px
						sum[hex][1] += py
						sum[hex][2] += pz
					}
					sum[hex][3]++
				}
			}
			sums[w] = sum
		}()
	}
	group.Wait()
	light := make([]swatch.Swatch, len(view.earth.hexes))
	for hex := range view.earth.hexes {
		var total [4]float64
		for w := range workers {
			for k := range 4 {
				total[k] += sums[w][hex][k]
			}
		}
		if total[3] > 0 {
			light[hex] = swatch.FromXYZ(total[0]/total[3], total[1]/total[3], total[2]/total[3])
		}
	}
	log.Printf("%s laid onto %d hexes from %dx%d pixels in %v", path, len(view.earth.hexes), width, height, time.Since(start).Round(time.Millisecond))
	return light
}

// earthImage is one of NASA's images of the earth, laid on its hexes.
type earthImage struct {
	name, file string
	light      []swatch.Swatch
}

// nearestHex is the hex whose centre is nearest a place: the nearest
// corner of the triangle the place falls in.
func nearestHex(lon, lat float64) int {
	place, err := mesh.NewPoint(lon, lat)
	must(err)
	cell, err := mesh.Locate(place, view.earth.depth)
	must(err)
	triangle := view.earth.triangleOf[cell.Key()]
	x, y, z := unit(lon, lat)
	best, nearest := 0, -2.0
	for corner := range 3 {
		hex := view.earth.hexOf[triangle*3+corner]
		at := view.earth.hexes[hex]
		if facing := at[0]*x + at[1]*y + at[2]*z; facing > nearest {
			best, nearest = hex, facing
		}
	}
	return best
}

// reflected is a measured surface's light under D65, relative to white:
// its reflectance times libtheme's D65 at each nanometre the eye sees.
func reflected(library *os.File, size int64, name string) swatch.Swatch {
	measured, err := usgs.Read(library, size, name)
	must(err)
	// libtheme's Illuminant scales every light to a luminance of one, so
	// the sum is taken over its observer's own response at each
	// nanometre, Monochrome, shortest first; the daylight's own sum is the
	// white it is measured against.
	daylight := swatch.D65()
	lo, hi := swatch.Observed()
	var x, y, z, white float64
	for nm := lo; nm <= hi; nm++ {
		power, lit := daylight[nm]
		if !lit {
			continue
		}
		mx, my, mz := swatch.Monochrome(nm).XYZ()
		white += power * my
		if share, ok := measured.At(float64(nm)); ok {
			x, y, z = x+power*share*mx, y+power*share*my, z+power*share*mz
		}
	}
	log.Printf("%s reflects %.3f of daylight's luminance", name, y/white)
	return swatch.FromXYZ(x/white, y/white, z/white)
}

// faceCentre turns the standing observer toward Sgr A*, whatever the
// clock.
func faceCentre() {
	zenith, north, east := horizon()
	centre := vec(unit(266.4168371, -29.0078106))
	view.centreAlt = math.Asin(dot(centre, zenith)) * 180 / math.Pi
	view.azimuth = math.Atan2(dot(centre, east), dot(centre, north)) * 180 / math.Pi
	// below the horizon, face its way with the horizon in view, so the
	// ground does not fill the window.
	view.altitude = math.Max(view.centreAlt, horizonView)
	log.Printf("Sgr A* at azimuth %.1f, altitude %.1f, %s", view.azimuth, view.centreAlt, view.clock.Format(time.Kitchen))
}

// horizonView is the lowest a flip faces, in degrees: the horizon a third
// of the way up a 60 degree window.
const horizonView = 10

// lonLat is a direction's longitude and latitude in degrees.
func lonLat(at [3]float64) (float64, float64) {
	lon := math.Atan2(at[1], at[0]) * 180 / math.Pi
	lat := math.Atan2(at[2], math.Hypot(at[0], at[1])) * 180 / math.Pi
	return lon, lat
}

// The wavelengths the layers were taken at. IRAC's headers give only the
// channel's number, and celestial holds no IRAC bands yet, so these are
// the channels' nominal names, recalled: 3.6 and 8.0 micrometres. MeerKAT's
// is read from its header's CRVAL3, in hertz.
const (
	irac1     = 3600
	irac4     = 8000
	meerkatHz = 1283895507.8125
)

// recolour paints every triangle, all colour through libtheme.
//
// Each sky layer's hue is slayer's false colour for its wavelength: the
// layers' range, widened by a tenth of its span at each end in the
// logarithm so none sits on the eye's dark edges, laid across the eye's by
// celestial's profile.
//
// Its lightness and chroma are the viewer's, and
// libtheme's Fit brings it into sRGB. Each layer's light, scaled by its
// level, is added to the background's in XYZ.
//
// The earth's land and sea keep their measured hue and chroma, moved by
// the viewer's lightness and colourfulness. The marked hex is the
// outline's colour.
func recolour() {
	short, long := math.Inf(1), math.Inf(-1)
	for _, each := range slices.Concat(view.sources, view.wideSources) {
		if each.nm > 0 {
			short, long = math.Min(short, each.nm), math.Max(long, each.nm)
		}
	}
	margin := math.Log(long/short) / 10
	span, err := radiation.NewRange(short*math.Exp(-margin), long*math.Exp(margin))
	must(err)
	shades := make(map[string]swatch.Swatch)
	for _, each := range slices.Concat(view.sources, view.wideSources) {
		if each.nm <= 0 {
			continue
		}
		x, y, z := observer.FalseColour(*span, each.nm).XYZ()
		hue := ok.FromSwatch(swatch.FromXYZ(x/y, 1, z/y)).Polar().H
		chosen := ok.OKLCH{L: view.lightness, C: chromas[view.chroma], H: hue}
		fitted, _ := ok.Fit(chosen, srgb.In)
		shades[each.name] = fitted.Rect().Swatch()
	}
	// the layers of a stack share one white: each lit one's light is
	// that stack's share, so the sky reaches white only where all are at
	// their brightest.
	for _, group := range []struct {
		sources []source
		on      []bool
	}{{view.sources, view.on}, {view.wideSources, view.wideOn}} {
		lit := 0
		for i, each := range group.sources {
			if group.on[i] && each.nm > 0 {
				lit++
			}
		}
		for _, each := range group.sources {
			if shade, shown := shades[each.name]; shown && lit > 0 {
				x, y, z := shade.XYZ()
				share := 1 / float64(lit)
				shades[each.name] = swatch.FromXYZ(x*share, y*share, z*share)
			}
		}
	}
	gx, gy, gz := ground().XYZ()
	parallel(len(view.cells), func(i int) {
		x, y, z := wideLight(shades, view.cells[i], gx, gy, gz)
		for _, lit := range view.stack.At(view.cells[i]) {
			shade, shown := shades[lit.Name()]
			if !shown {
				continue
			}
			sx, sy, sz := shade.XYZ()
			x, y, z = x+sx*lit.Level(), y+sy*lit.Level(), z+sz*lit.Level()
		}
		r, g, b := bytesOf(swatch.FromXYZ(x, y, z))
		for k := range 3 {
			copy(view.skyColours[i*12+k*4:], []byte{r, g, b, 255})
		}
	})
	parallel(len(view.backCells), func(i int) {
		r, g, b := bytesOf(swatch.FromXYZ(wideLight(shades, view.backCells[i], gx, gy, gz)))
		for k := range 3 {
			copy(view.backColours[i*12+k*4:], []byte{r, g, b, 255})
		}
	})
	recolourEarth()
}

// recolourEarth paints the earth: Blue Marble or the measured land and
// sea, as the viewer sets them, and the observer's hex marked only when
// the earth is seen from outside.
func recolourEarth() {
	land, sea := seen(view.earth.land), seen(view.earth.sea)
	lr, lg, lb := lineColour()
	mark := [3]byte{byteOf(lr), byteOf(lg), byteOf(lb)}
	hexColours := make([][3]byte, len(view.earth.hexes))
	parallel(len(view.earth.hexes), func(hex int) {
		switch {
		case hex == view.earth.marked && view.mode == earthView:
			hexColours[hex] = mark
		case view.earth.showing < len(view.earth.images):
			hexColours[hex] = seen(view.earth.images[view.earth.showing].light[hex])
		case view.earth.onLand[hex]:
			hexColours[hex] = land
		default:
			hexColours[hex] = sea
		}
	})
	for k, hex := range view.earth.hexOf {
		colour := hexColours[hex]
		copy(view.earth.colours[k*4:], []byte{colour[0], colour[1], colour[2], 255})
	}
}

// seen is a measured surface's colour as the viewer sets it: its
// lightness moved by theirs, its chroma scaled by theirs, fitted to sRGB.
func seen(surface swatch.Swatch) [3]byte {
	shade := ok.FromSwatch(surface).Polar()
	shade.L = math.Max(0, math.Min(1, shade.L+view.lightness-0.75))
	shade.C *= chromas[view.chroma] / chromas[0]
	fitted, _ := ok.Fit(shade, srgb.In)
	r, g, b := bytesOf(fitted.Rect().Swatch())
	return [3]byte{r, g, b}
}

// bytesOf is a swatch as sRGB bytes, by libtheme.
func bytesOf(light swatch.Swatch) (byte, byte, byte) {
	lamps, _ := srgb.FromSwatch(light)
	return lamps.Bytes()
}

// byteOf is a lamp from none to one as a byte.
func byteOf(lamp float64) byte { return byte(math.Round(math.Max(0, math.Min(1, lamp)) * 255)) }

// parallel runs work over n items on four workers.
func parallel(n int, work func(int)) {
	workers := 4
	var group sync.WaitGroup
	for w := range workers {
		group.Add(1)
		go func() {
			defer group.Done()
			for i := w; i < n; i += workers {
				work(i)
			}
		}()
	}
	group.Wait()
}

// ground is the background's light: a grey of the viewer's lightness.
func ground() swatch.Swatch { return ok.Grey(grounds[view.ground]).Swatch() }

// The settings the viewer moves for their eyes. Each starts at the first
// and is moved by a key; none is a judgement of what anyone sees.
var (
	// chromas are how colourful the layers are, from as much as fits to
	// none; c steps through them.
	chromas = []float64{0.4, 0.2, 0.1, 0}
	// grounds are the background's lightness, in OKLab; b steps through
	// them, dark to light.
	grounds = []float64{0.12, 0, 0.3, 0.5, 0.75, 0.95}
	// widths are the outline's width in pixels; [ and ] step through
	// them.
	widths = []float64{1.5, 1, 2.5, 4, 6}
)

// lineColour is the outline's colour: white over a dark background, black
// over a light one, through libtheme.
func lineColour() (float64, float64, float64) {
	far := 1.0
	if grounds[view.ground] > 0.5 {
		far = 0
	}
	lamps, _ := srgb.FromSwatch(ok.Grey(far).Swatch())
	return lamps.R, lamps.G, lamps.B
}

// refresh recolours everything and hands it to the window.
func refresh() {
	recolour()
	C.viewer_colours(C.viewer_sky, (*C.uint8_t)(unsafe.Pointer(&view.skyColours[0])), C.int(len(view.cells)))
	C.viewer_colours(C.viewer_back, (*C.uint8_t)(unsafe.Pointer(&view.backColours[0])), C.int(view.backCount))
	C.viewer_colours(C.viewer_earth, (*C.uint8_t)(unsafe.Pointer(&view.earth.colours[0])), C.int(len(view.earth.hexOf)/3))
	lamps, _ := srgb.FromSwatch(ground())
	C.viewer_background(C.double(lamps.R), C.double(lamps.G), C.double(lamps.B))
	log.Printf("lightness %.2f, chroma %.2f, background %.2f, width %.1f",
		view.lightness, chromas[view.chroma], grounds[view.ground], widths[view.line])
}

// mark finds the hex under the placed observer and marks it.
func mark() {
	x, y, z := unit(view.placeLon, view.placeLat)
	best, nearest := -1, -2.0
	for i, hex := range view.earth.hexes {
		if facing := hex[0]*x + hex[1]*y + hex[2]*z; facing > nearest {
			best, nearest = i, facing
		}
	}
	view.earth.marked = best
	recolour()
}

// goDrag turns the active view using Cocoa's drag distance in points.
//
//export goDrag
func goDrag(dx, dy C.double) {
	view.Lock()
	defer view.Unlock()
	pixels := float64(max(view.width, 1)) / 2 // retina: two pixels a point
	switch view.mode {
	case skyView:
		step := view.fov / pixels
		view.ra += float64(dx) * step / math.Max(math.Cos(view.dec*math.Pi/180), 0.05)
		view.dec = math.Max(-89.9, math.Min(89.9, view.dec+float64(dy)*step))
	case earthView:
		step := 60 / pixels * (view.distance - 1)
		view.lon -= float64(dx) * step / math.Max(math.Cos(view.lat*math.Pi/180), 0.05)
		view.lat = math.Max(-89.9, math.Min(89.9, view.lat+float64(dy)*step))
	case placeView:
		step := view.placeFov / pixels
		view.azimuth -= float64(dx) * step
		view.altitude = math.Max(-89.9, math.Min(89.9, view.altitude+float64(dy)*step))
	}
}

// goScroll changes sky field of view or distance from the earth.
//
//export goScroll
func goScroll(dy C.double) {
	view.Lock()
	defer view.Unlock()
	scale := math.Exp(-float64(dy) * 0.01)
	switch view.mode {
	case skyView:
		view.fov = math.Max(0.02, math.Min(120, view.fov*scale))
	case earthView:
		view.distance = math.Max(1.02, math.Min(12, 1+(view.distance-1)*scale))
	case placeView:
		view.placeFov = math.Max(0.5, math.Min(120, view.placeFov*scale))
	}
}

// goKey handles the viewer's mode, layer and appearance keys.
//
//export goKey
func goKey(key C.int) {
	switch {
	case key == 'q':
		os.Exit(0)
	case key == 'e':
		view.Lock()
		view.mode = map[bool]int{true: skyView, false: earthView}[view.mode == earthView]
		view.Unlock()
		refreshEarth()
	case key == 'p' && view.mode == earthView:
		view.Lock()
		view.placeLon, view.placeLat, view.placed = view.lon, view.lat, true
		view.Unlock()
		mark()
		refresh()
		log.Printf("observer placed at %.3f, %.3f", view.placeLat, view.placeLon)
	case key == 'f' && view.placed:
		view.Lock()
		view.mode = map[bool]int{true: earthView, false: placeView}[view.mode == placeView]
		if view.mode == placeView {
			faceCentre()
		}
		view.Unlock()
		refreshEarth()
	case key == ',' || key == '.':
		view.Lock()
		step := time.Hour
		if key == ',' {
			step = -step
		}
		view.clock = view.clock.Add(step)
		view.Unlock()
		log.Printf("clock %s", view.clock.Format(time.RFC1123))
	case key == 'h':
		view.Lock()
		view.hudSize = (view.hudSize + 1) % len(hudRows)
		view.hudSaid = ""
		view.Unlock()
	case key == 't':
		view.Lock()
		view.theme = (view.theme + 1) % len(hudThemes)
		view.Unlock()
	case key == 'o':
		view.Lock()
		view.outline = (view.outline + 1) % len(outlines)
		view.Unlock()
	case key == '=' || key == '+' || key == '-':
		step := 0.05
		if key == '-' {
			step = -step
		}
		view.lightness = math.Max(0.1, math.Min(1, view.lightness+step))
		refresh()
	case key == 'm':
		view.earth.showing = (view.earth.showing + 1) % (len(view.earth.images) + 1)
		refresh()
	case key == 'c':
		view.chroma = (view.chroma + 1) % len(chromas)
		refresh()
	case key == 'b':
		view.ground = (view.ground + 1) % len(grounds)
		refresh()
	case key == '[' || key == ']':
		view.Lock()
		if key == ']' {
			view.line = min(view.line+1, len(widths)-1)
		} else {
			view.line = max(view.line-1, 0)
		}
		view.Unlock()
	case key >= '5' && int(key-'5') < len(view.wideSources):
		i := int(key - '5')
		view.wideOn[i] = !view.wideOn[i]
		must(view.wide.Turn(view.wideSources[i].name, view.wideOn[i]))
		refresh()
	case key >= '1' && key <= '4' && int(key-'1') < len(view.sources):
		i := int(key - '1')
		view.on[i] = !view.on[i]
		must(view.stack.Turn(view.sources[i].name, view.on[i]))
		refresh()
	}
}

// goView writes the two shader uniform blocks for the active view.
//
//export goView
func goView(out *C.float, width, height C.int) C.int {
	view.Lock()
	defer view.Unlock()
	view.width = int(width)
	both := unsafe.Slice((*float32)(unsafe.Pointer(out)), 72)
	u, ground := both[:36], both[36:]
	pole := [3]float64{0, 0, 1}
	var matrix [16]float64
	var earthMatrix [16]float64
	switch view.mode {
	case skyView:
		forward := vec(unit(view.ra, view.dec))
		matrix = camera(forward, pole, [3]float64{}, view.fov, width, height)
	case earthView:
		toward := vec(unit(view.lon, view.lat))
		from := scaled(toward, view.distance)
		matrix = camera(scaled(toward, -1), pole, from, earthFov, width, height)
		u[32], u[33], u[34] = float32(width)/2, float32(height)/2, 1
	case placeView:
		zenith, north, east := horizon()
		az, alt := view.azimuth*math.Pi/180, view.altitude*math.Pi/180
		level := plus(scaled(north, math.Cos(az)), scaled(east, math.Sin(az)))
		forward := plus(scaled(level, math.Cos(alt)), scaled(zenith, math.Sin(alt)))
		matrix = camera(forward, zenith, [3]float64{}, view.placeFov, width, height)
		// the earth from the eye: the camera at the placed point, so
		// high above the ground, and the earth turned to the sky by the
		// sidereal time.
		eye := scaled(zenith, 1+eyeHeight/research.Value("earth-mean-radius-m"))
		from := camera(forward, zenith, eye, view.placeFov, width, height)
		earthMatrix = times(from, turnedBy(sidereal()))
	}
	for k := range 16 {
		u[k] = float32(matrix[k])
	}
	u[16] = float32(outlines[view.outline])
	u[17] = float32(widths[view.line])
	u[18] = map[bool]float32{true: 1, false: 0}[view.mode == earthView]
	r, g, b := lineColour()
	u[20], u[21], u[22], u[23] = float32(r), float32(g), float32(b), 1
	copy(ground, u)
	for k := range 16 {
		ground[k] = float32(earthMatrix[k])
	}
	ground[18], ground[34] = 1, 0
	showHud(int(width), int(height))
	return C.int(view.mode)
}

// eyeHeight is how high the standing observer's eye is, in metres:
// picked, high enough that the hexes reach a horizon some 360 km off.
const eyeHeight = 10000

// turnedBy is the turn about the pole by an angle in degrees, as a view
// matrix column by column: the earth's frame into the sky's.
func turnedBy(angle float64) [16]float64 {
	sin, cos := math.Sincos(angle * math.Pi / 180)
	return [16]float64{cos, sin, 0, 0, -sin, cos, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}
}

// times is one matrix after another, each column by column.
func times(a, b [16]float64) [16]float64 {
	var product [16]float64
	for col := range 4 {
		for row := range 4 {
			for k := range 4 {
				product[col*4+row] += a[k*4+row] * b[col*4+k]
			}
		}
	}
	return product
}

// sidereal is Greenwich mean sidereal time at the clock, in degrees, by
// celestial, with UT1 taken as UTC.
func sidereal() float64 {
	seconds := float64(view.clock.UnixNano()) / 1e9
	julian := seconds/86400 + 2440587.5
	whole := math.Floor(julian)
	moment, err := sky.FromJulian(whole, julian-whole, sky.UTC())
	must(err)
	turning, err := moment.UT1(measure.Seconds(0))
	must(err)
	angle, err := turning.MeanSidereal()
	must(err)
	return angle.InDegrees()
}

// earthFov is the earth view's width in degrees: the earth's disc fills
// most of it from 3.2 radii away, and a patch of it from close by.
const earthFov = 40

// The uniforms, as the shader's struct lays them out in floats: the view
// matrix at 0 to 15; the outline's strength at 16 and width at 17; hex at
// 18 and ground at 19; the line's colour at 20 to 23; up at 24 to 27; the
// ground's colour at 28 to 31.
//
// The crosshair's centre is at 32 and 33, and
// whether to draw it at 34.

// horizon is the zenith, the north point and the east point of the placed
// observer at the clock's time, as directions in the equatorial frame. The
// sky turns by celestial's mean sidereal time, with UT1 taken as UTC.
func horizon() ([3]float64, [3]float64, [3]float64) {
	local := sidereal() + view.placeLon
	zenith := vec(unit(local, view.placeLat))
	north := vec(unit(local+180, 90-view.placeLat))
	east := cross(north, zenith)
	return zenith, north, east
}

// camera is the view matrix, column by column, for a camera at a place
// looking along forward with an up toward upward. The window's width
// spans fov degrees.
//
// East is on the right when facing north from the
// ground, and on the left when looking up at the sky from its centre:
// both are right of forward across upward.
func camera(forward, upward, at [3]float64, fov float64, width, height C.int) [16]float64 {
	forward = norm(forward)
	right := norm(cross(forward, upward))
	up := cross(right, forward)
	s := 1 / math.Tan(fov*math.Pi/180/2)
	aspect := float64(width) / float64(height)
	rows := [4][4]float64{
		{right[0] * s, right[1] * s, right[2] * s, -dot(right, at) * s},
		{up[0] * s * aspect, up[1] * s * aspect, up[2] * s * aspect, -dot(up, at) * s * aspect},
		{forward[0] * 0.5, forward[1] * 0.5, forward[2] * 0.5, -dot(forward, at) * 0.5},
		{forward[0], forward[1], forward[2], -dot(forward, at)},
	}
	var matrix [16]float64
	for col := range 4 {
		for row := range 4 {
			matrix[col*4+row] = rows[row][col]
		}
	}
	return matrix
}

// vec collects the three coordinates used by the viewer's matrices.
func vec(x, y, z float64) [3]float64 { return [3]float64{x, y, z} }

// plus adds two vectors component by component.
func plus(a, b [3]float64) [3]float64 {
	return [3]float64{a[0] + b[0], a[1] + b[1], a[2] + b[2]}
}

// scaled multiplies a vector by k.
func scaled(a [3]float64, k float64) [3]float64 { return [3]float64{a[0] * k, a[1] * k, a[2] * k} }

// dot measures the alignment and magnitudes of two vectors.
func dot(a, b [3]float64) float64 { return a[0]*b[0] + a[1]*b[1] + a[2]*b[2] }

// cross gives the perpendicular in the viewer's right-handed coordinates.
func cross(a, b [3]float64) [3]float64 {
	return [3]float64{a[1]*b[2] - a[2]*b[1], a[2]*b[0] - a[0]*b[2], a[0]*b[1] - a[1]*b[0]}
}

// norm gives a nonzero vector unit length; its callers supply the basis.
func norm(a [3]float64) [3]float64 { return scaled(a, 1/math.Sqrt(dot(a, a))) }

// unit is the direction to a longitude and latitude, in degrees.
func unit(lon, lat float64) (float64, float64, float64) {
	sinLon, cosLon := math.Sincos(lon * math.Pi / 180)
	sinLat, cosLat := math.Sincos(lat * math.Pi / 180)
	return cosLat * cosLon, cosLat * sinLon, sinLat
}

// lay opens an image and lays it at a depth.
func lay(each source, depth int) *layer.Layer {
	file, err := os.Open(each.path)
	must(err)
	img, err := fits.Open(file)
	must(err)
	laid, err := layer.Lay(each.name, img, depth, layer.NewSource(each.path, "", time.Time{}, 0, ""))
	must(err)
	return laid
}

// stretch is the cell values at two shares of the way through them.
func stretch(laid *layer.Layer, floor, ceil float64) (float64, float64) {
	var values []float64
	for _, key := range laid.Cells() {
		cell, err := mesh.FromKey(key)
		must(err)
		value, _ := laid.At(*cell)
		values = append(values, value.Mean())
	}
	slices.Sort(values)
	at := func(share float64) float64 { return values[int(share*float64(len(values)-1))] }
	return at(floor), at(ceil)
}

// writeShot draws one frame offscreen through the window's pipeline and
// writes it as a png.
func writeShot(path string, meshes [3]C.viewer_mesh) {
	width, height := 1400, 1000
	pixels := make([]byte, width*height*4)
	var why [1024]C.char
	if C.viewer_shot(meshes[0], meshes[1], meshes[2], C.int(width), C.int(height),
		(*C.uint8_t)(unsafe.Pointer(&pixels[0])), &why[0], C.int(len(why))) != 0 {
		log.Fatal(C.GoString(&why[0]))
	}
	for i := 0; i < len(pixels); i += 4 {
		pixels[i], pixels[i+2] = pixels[i+2], pixels[i]
	}
	picture := &image.RGBA{Pix: pixels, Stride: width * 4, Rect: image.Rect(0, 0, width, height)}
	file, err := os.Create(path)
	must(err)
	must(png.Encode(file, picture))
	must(file.Close())
	log.Printf("wrote %s", path)
}

// must reports a startup or viewer failure and stops the process.
func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// buildBack is the sphere behind everything: every cell at the whole-sky
// depth, lit by whatever whole-sky maps are on this machine, each on a
// logarithmic stretch.
//
// Their wavelengths: Mellinger's green, its filter's, recalled as 540 nm;
// H-alpha's line, recalled as 656.3 nm;
// IRIS at its band's name, 100 micrometres; 408 MHz by celestial's
// HertzToNm.
func buildBack(dir string, depth int) []float32 {
	view.wideDepth = depth
	candidates := []source{
		{name: "optical (Mellinger green)", path: dir + "/skyview-Mellinger-G-3600.fits", nm: 540, floor: 0.5, ceil: 0.9995, strength: 0.6, logarithmic: true},
		{name: "H-alpha", path: dir + "/skyview-H-Alpha-Comp-3600.fits", nm: 656.3, floor: 0.5, ceil: 0.999, strength: 0.6, logarithmic: true},
		{name: "dust 100 um (IRIS)", path: dir + "/skyview-IRIS-100-1800.fits", nm: 100000, floor: 0.5, ceil: 0.999, strength: 0.6, logarithmic: true},
		{name: "radio 408 MHz (Haslam)", path: dir + "/skyview-408MHz-3600.fits", nm: radiation.HertzToNm(408e6), floor: 0.5, ceil: 0.999, strength: 0.6, logarithmic: true},
	}
	for _, each := range candidates {
		if _, err := os.Stat(each.path); err == nil {
			view.wideSources = append(view.wideSources, each)
		}
	}
	layers := make([]*layer.Layer, len(view.wideSources))
	var group sync.WaitGroup
	for i, each := range view.wideSources {
		group.Add(1)
		go func() {
			defer group.Done()
			start := time.Now()
			layers[i] = lay(each, depth)
			log.Printf("laid %s: %d cells in %v", each.name, len(layers[i].Cells()), time.Since(start).Round(time.Second))
		}()
	}
	group.Wait()
	view.wide = composite.NewStack()
	for i, each := range view.wideSources {
		low, high := stretch(layers[i], each.floor, each.ceil)
		must(view.wide.Add(layers[i], low, high))
		must(view.wide.Shape(each.name, composite.Logarithmic))
		view.wideOn = append(view.wideOn, true)
	}
	view.backCells = cellsAt(depth)
	var corners []float32
	for _, cell := range view.backCells {
		corners = appendCorners(corners, cell)
	}
	view.backCount = len(view.backCells)
	view.backColours = make([]byte, view.backCount*12)
	return corners
}

// wideLight is the background's light with the whole-sky maps' added, at
// the cell at the whole-sky depth that holds a cell.
func wideLight(shades map[string]swatch.Swatch, cell mesh.Cell, x, y, z float64) (float64, float64, float64) {
	for cell.Depth() > view.wideDepth {
		parent, err := cell.Parent()
		must(err)
		cell = *parent
	}
	for _, lit := range view.wide.At(cell) {
		shade, shown := shades[lit.Name()]
		if !shown {
			continue
		}
		sx, sy, sz := shade.XYZ()
		x, y, z = x+sx*lit.Level(), y+sy*lit.Level(), z+sz*lit.Level()
	}
	return x, y, z
}

// refreshEarth recolours the earth alone and hands it to the window.
func refreshEarth() {
	recolourEarth()
	C.viewer_colours(C.viewer_earth, (*C.uint8_t)(unsafe.Pointer(&view.earth.colours[0])), C.int(len(view.earth.hexOf)/3))
}

// hudRows are the hud's sizes, as rows of cells down the window: h steps
// through them, larger text and then none.
var hudRows = []int{52, 34, 0}

// hudThemes are the hud's colourways: the author's, from chroma's colourways,
// each read once as a libtheme sheet of roles. t steps through them.
var hudThemes []namedTheme

// namedTheme is a colourway and its name.
type namedTheme struct {
	name  string
	theme hud.Theme
}

// readThemes reads every colourway in a directory, sorted, leaving out
// any libtheme cannot read, and starts at one by name.
func readThemes(dir, start string) {
	paths, err := filepath.Glob(dir + "/*.css")
	must(err)
	slices.Sort(paths)
	for _, path := range paths {
		theme, err := hud.ReadTheme(path)
		if err != nil {
			log.Printf("colourway %s left out: %v", path, err)
			continue
		}
		name := strings.TrimSuffix(filepath.Base(path), ".css")
		if name == start {
			view.theme = len(hudThemes)
		}
		hudThemes = append(hudThemes, namedTheme{name: name, theme: theme})
	}
	if len(hudThemes) == 0 {
		log.Fatalf("no colourway in %s", dir)
	}
}

// showHud lays the hud over the window when its words or size change:
// slayer's hud, drawn into a picture at half the window's pixels.
func showHud(width, height int) {
	rows := hudRows[view.hudSize]
	if rows == 0 {
		if view.hudSaid != "off" {
			C.viewer_hud(nil, 0, 0)
			view.hudSaid = "off"
		}
		return
	}
	lines, help := hudWords()
	said := fmt.Sprint(width, height, rows, lines, help)
	if said == view.hudSaid {
		return
	}
	view.hudSaid = said
	var made hud.Hud
	row := 1 / float64(rows)
	for i, line := range lines {
		if line == "" {
			continue
		}
		made.Elements = append(made.Elements, hud.Text{Words: hud.Words(" " + line + " "),
			At: hud.Point{Across: row / 2, Down: row * float64(i+1)}, Ink: "ink", Ground: "ground"})
	}
	for i, line := range help {
		made.Elements = append(made.Elements, hud.Text{Words: hud.Words(" " + line + " "),
			At: hud.Point{Across: row / 2, Down: 1 - row*float64(len(help)-i+1)}, Ink: "ink", Ground: "ground"})
	}
	picture := image.NewRGBA(image.Rect(0, 0, width/2, height/2))
	must(made.Picture(picture, rows, 0, hudThemes[view.theme].theme, hud.View{}))
	C.viewer_hud((*C.uint8_t)(unsafe.Pointer(&picture.Pix[0])), C.int(width/2), C.int(height/2))
}

// hudWords is what the hud says: the view and what it looks at, the
// layers and the settings at the top; the keys at the foot.
func hudWords() ([]string, []string) {
	var lines []string
	switch view.mode {
	case skyView:
		lines = append(lines, fmt.Sprintf("sky, from inside   looking at RA %.2f, Dec %.2f   %.1f degrees across", math.Mod(math.Mod(view.ra, 360)+360, 360), view.dec, view.fov))
	case earthView:
		lines = append(lines, fmt.Sprintf("earth, from %.1f radii out   over %s   observer %s", view.distance, latLon(view.lat, view.lon), placed()))
	case placeView:
		facing := fmt.Sprintf("facing azimuth %.0f, altitude %.0f", math.Mod(math.Mod(view.azimuth, 360)+360, 360), view.altitude)
		lines = append(lines, fmt.Sprintf("standing at %s, %.0f km up   %s   %.0f degrees across", latLon(view.placeLat, view.placeLon), eyeHeight/1000.0, facing, view.placeFov))
		where := fmt.Sprintf("Sgr A* is %.0f degrees above the horizon", view.centreAlt)
		if view.centreAlt < 0 {
			where = fmt.Sprintf("Sgr A* is below the horizon, by %.0f degrees", -view.centreAlt)
		}
		lines = append(lines, view.clock.Format("Mon 2 Jan 2006, 15:04 MST")+"   "+where)
	}
	lines = append(lines, "")
	for i, each := range view.sources {
		lines = append(lines, fmt.Sprintf("%d %s  %s   Sgr A*, depth 13", i+1, onOff(view.on[i]), each.name))
	}
	for i, each := range view.wideSources {
		lines = append(lines, fmt.Sprintf("%d %s  %s   whole sky, depth %d", i+5, onOff(view.wideOn[i]), each.name, view.wideDepth))
	}
	earth := "measured land and sea (USGS)"
	if view.earth.showing < len(view.earth.images) {
		earth = view.earth.images[view.earth.showing].name + " (NASA, local only)"
	}
	lines = append(lines, "m    earth: "+earth, "",
		fmt.Sprintf("lightness %.2f   colour %.2f   background %.2f   outline %s, %.1f px",
			view.lightness, chromas[view.chroma], grounds[view.ground], []string{"on", "faded", "off"}[view.outline], widths[view.line]),
		"t    hud colourway: "+hudThemes[view.theme].name)
	help := []string{
		"drag look   scroll zoom   e earth or sky   p place at the crosshair   f stand there   comma period clock",
		"1-8 layers   o outlines   = - lightness   c colour   b background   [ ] outline width   m earth   h hud size   t hud colours   q quit",
	}
	return lines, help
}

// onOff is a layer's state as the hud says it.
func onOff(on bool) string { return map[bool]string{true: "on ", false: "off"}[on] }

// placed is where the observer stands, or that they have not been placed.
func placed() string {
	if !view.placed {
		return "not placed: p places at the crosshair"
	}
	return latLon(view.placeLat, view.placeLon)
}

// latLon is a place as degrees north or south and east or west.
func latLon(lat, lon float64) string {
	lon = math.Mod(math.Mod(lon+180, 360)+360, 360) - 180
	ns, ew := "N", "E"
	if lat < 0 {
		ns, lat = "S", -lat
	}
	if lon < 0 {
		ew, lon = "W", -lon
	}
	return fmt.Sprintf("%.2f %s, %.2f %s", lat, ns, lon, ew)
}
