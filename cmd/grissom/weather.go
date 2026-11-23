package main

// A weather layer for the earth: a gridded variable read through
// celestial-data's netCDF adapter, whole, once. Each time step that holds
// data is asked at every triangle's centre and kept as one cover value
// per triangle, then mixed over the ground as cloud.

import (
	"fmt"
	"log"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/janearc/celestial-data/netcdf"
	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/libtheme-css/spaces/ok"
	"github.com/janearc/libtheme-css/spaces/srgb"
)

const (
	// firstWithData is the --weather-time that was not given: the first
	// step that holds data.
	firstWithData = -1
	// defaultWeatherVariable is the variable read when --weather-var is
	// not given: ERA-40's total cloud cover.
	defaultWeatherVariable = "tcc"
	// cornersPerTriangle and axesPerCorner lay out a triangle's nine
	// numbers in the earth's corners.
	cornersPerTriangle = 3
	axesPerCorner      = 3
	// bytesPerCorner is a corner's colour in the earth's colours: red,
	// green, blue and alpha.
	bytesPerCorner = 4
	// picked: the share of daylight a cloud top returns, set by eye to
	// read as white without glaring beside the measured land and sea. It
	// is not derived from a measured cloud.
	cloudReflectance = 0.8
	// dateLayout is how a step's date is written in the log.
	dateLayout = "2006-01-02 15:04"
)

// weatherStep is one time step that holds data: where it was in the file,
// when it is, and its cover at each of the earth's triangles. A triangle
// the file has no data for is NaN.
type weatherStep struct {
	index int
	when  time.Time
	cover []float64
}

// weatherFile is a variable read whole: the steps that hold data, which
// of them a still shows, and how many steps the file has in all.
type weatherFile struct {
	path, name string
	kept       []*netcdf.Slice
	chosen     int
	total      int
}

// openWeather reads the whole variable through the adapter, leaves out
// the steps that hold no data and picks the step a still shows. It gives
// nil when no file was asked for. The adapter's errors pass on as they
// are, for they name the file and the variable.
func openWeather(settings options) (*weatherFile, error) {
	if settings.weather == "" {
		return nil, nil
	}
	file, err := netcdf.Open(settings.weather)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	field, err := file.Field(settings.weatherVar)
	if err != nil {
		return nil, err
	}
	all, err := field.All()
	if err != nil {
		return nil, err
	}
	found := &weatherFile{path: settings.weather, name: settings.weatherVar,
		total: len(all)}
	found.kept = slices.DeleteFunc(all, func(step *netcdf.Slice) bool {
		return !step.Holds()
	})
	if len(found.kept) == 0 {
		return nil, fmt.Errorf("%s: variable %q holds no data at any of "+
			"its %d time steps", found.path, found.name, found.total)
	}
	found.chosen, err = found.choose(settings.weatherTime)
	return found, err
}

// choose is the position among the kept steps of the step a --weather-time
// names, or of the first when none was given. A step with no data, or none
// at all, is refused, naming the nearest steps that have data.
func (found *weatherFile) choose(wanted int) (int, error) {
	if wanted == firstWithData {
		return 0, nil
	}
	for position, step := range found.kept {
		if step.Index() == wanted {
			return position, nil
		}
	}
	return 0, fmt.Errorf("%s: variable %q has no data at time step %d, "+
		"which is one of %d; the nearest steps that have data are %s",
		found.path, found.name, wanted, found.total, found.nearest(wanted))
}

// nearest says the kept steps either side of a step, as the file counts
// them: the last before it and the first after it, or the one there is.
func (found *weatherFile) nearest(wanted int) string {
	var near []string
	before, after := -1, -1
	for _, step := range found.kept {
		if step.Index() < wanted {
			before = step.Index()
		}
		if step.Index() > wanted && after < 0 {
			after = step.Index()
		}
	}
	for _, each := range []int{before, after} {
		if each >= 0 {
			near = append(near, fmt.Sprint(each))
		}
	}
	return strings.Join(near, " and ")
}

// covers computes, once, the cover at each triangle for each kept step,
// and logs the dates of the steps kept.
func (found *weatherFile) covers(corners []float32) []weatherStep {
	steps := make([]weatherStep, len(found.kept))
	log.Printf("weather %s from %s: %d of %d time steps hold data",
		found.name, found.path, len(found.kept), found.total)
	for position, step := range found.kept {
		steps[position] = weatherStep{index: step.Index(), when: step.Time(),
			cover: trianglesCover(corners, step)}
		log.Printf("weather step %d, %s", step.Index(),
			step.Time().Format(dateLayout))
	}
	return steps
}

// lay computes the covers and shows the chosen step's, painting the earth
// again with it.
func (found *weatherFile) lay() {
	view.earth.weather = found.covers(view.earth.corners)
	view.earth.cover = view.earth.weather[found.chosen].cover
	recolourEarth()
}

// triangleCentre is the normalised mean of a triangle's three corners.
func triangleCentre(corners []float32, triangle int) [3]float64 {
	var sum [3]float64
	for corner := range cornersPerTriangle {
		for axis := range axesPerCorner {
			at := triangle*cornersPerTriangle*axesPerCorner +
				corner*axesPerCorner + axis
			sum[axis] += float64(corners[at])
		}
	}
	return norm(sum)
}

// trianglesCover is a step's value at each triangle's centre, one for
// each; NaN where the step has no data there.
func trianglesCover(corners []float32, step *netcdf.Slice) []float64 {
	cover := make([]float64, len(corners)/(cornersPerTriangle*axesPerCorner))
	parallel(len(cover), func(triangle int) {
		lon, lat := lonLat(triangleCentre(corners, triangle))
		value, have := step.ValueAt(lon, lat)
		if !have {
			value = math.NaN()
		}
		cover[triangle] = value
	})
	return cover
}

// lightMix is two lights mixed in XYZ, share of the way from one to
// the other. Light adds; bytes do not.
func lightMix(from, to swatch.Swatch, share float64) swatch.Swatch {
	fx, fy, fz := from.XYZ()
	tx, ty, tz := to.XYZ()
	return swatch.FromXYZ(fx+(tx-fx)*share, fy+(ty-fy)*share, fz+(tz-fz)*share)
}

// cloudLight is the light a cloud top returns: daylight's white times
// the cloud's reflectance.
func cloudLight() swatch.Swatch {
	x, y, z := ok.Grey(1).Swatch().XYZ()
	return swatch.FromXYZ(x*cloudReflectance, y*cloudReflectance,
		z*cloudReflectance)
}

// mixClouds mixes each triangle's three corner colours toward cloud by
// its cover, a fraction from none to all and clamped; a triangle with
// no data or no cover keeps its colours. The mix is in light, taken
// through libtheme's sRGB, not in bytes.
func mixClouds(colours []byte, cover []float64, cloud swatch.Swatch) {
	parallel(len(cover), func(triangle int) {
		share := math.Min(cover[triangle], 1)
		if math.IsNaN(share) || share <= 0 {
			return
		}
		for corner := range cornersPerTriangle {
			at := (triangle*cornersPerTriangle + corner) * bytesPerCorner
			ground := srgb.RGB8(colours[at], colours[at+1],
				colours[at+2]).Swatch()
			colours[at], colours[at+1], colours[at+2] =
				bytesOf(lightMix(ground, cloud, share))
		}
	})
}

// coverEarth lays the weather layer over the earth's colours when one is
// loaded, with the cloud seen as the viewer's lightness and colour set.
func coverEarth() {
	if view.earth.cover == nil {
		return
	}
	shown := seen(cloudLight())
	mixClouds(view.earth.colours, view.earth.cover,
		srgb.RGB8(shown[0], shown[1], shown[2]).Swatch())
}
