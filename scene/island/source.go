package island

import (
	"context"
	"fmt"
	"strings"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/terrain"
)

// RasterHeights borrows a geographic raster with an explicit height policy.
// It interprets that source's angles on the authored sphere, adds a declared
// offset, and calls the result radial height. This is a scenery approximation,
// not a horizontal or vertical datum transformation.
type RasterHeights struct {
	raster *terrain.Raster
	crs    int
	datum  string
	offset float64
	index  *terrain.HeightIndex
}

// Indexed returns a new source description with complete native height bounds.
// Construction reads the borrowed raster serially; earlier descriptions are
// unchanged. The immutable index survives later decoded-tile cache eviction.
func (source *RasterHeights) Indexed(ctx context.Context, block int, limitBytes int64) (*RasterHeights, error) {
	if source == nil || source.raster == nil {
		return nil, fmt.Errorf("island: no source for a height index")
	}
	index, err := source.raster.BuildHeightIndex(ctx, block, limitBytes)
	if err != nil {
		return nil, err
	}
	copy := *source
	copy.index = index
	return &copy, nil
}

// HeightRange bounds source heights, including the same declared radial offset
// used by Height. An unindexed description leaves the caller's global envelope
// in force; it does not estimate a bound from a few sampled corners.
func (source *RasterHeights) HeightRange(west, south, east, north float64) (float64, float64, bool, error) {
	if source == nil || source.index == nil {
		return 0, 0, false, nil
	}
	low, high, has, err := source.index.Range(west, south, east, north)
	if err != nil || !has {
		return 0, 0, false, err
	}
	return low + source.offset, high + source.offset, true, nil
}

// NewRasterHeights checks the declared horizontal CRS and degree registration.
// The caller must read the source's metre-height metadata and name its datum;
// a constant offset cannot turn NAVD88 into MSL or an ellipsoid height. The
// source remains owned by its caller and must stay open for preparation.
func NewRasterHeights(raster *terrain.Raster, expectedCRS int, datum string,
	heightOffset measure.Length,
) (*RasterHeights, error) {
	if raster == nil || expectedCRS <= 0 || strings.TrimSpace(datum) == "" || !finite(heightOffset.InMetres()) {
		return nil, fmt.Errorf("island: need a geographic source, CRS and named height datum")
	}
	ref := raster.Reference()
	if ref.Model != 2 || ref.GeographicCRS != expectedCRS || ref.AngularUnits != 9102 ||
		(ref.RasterType != 1 && ref.RasterType != 2) ||
		(ref.VerticalUnits != 0 && ref.VerticalUnits != 9001) {
		return nil, fmt.Errorf("island: source does not match its declared degree/metre reference")
	}
	if _, _, err := raster.SampleLocation(0, 0); err != nil {
		return nil, err
	}
	return &RasterHeights{raster: raster, crs: expectedCRS, datum: datum, offset: heightOffset.InMetres()}, nil
}

// Reference reports the source interpretation, including an explicitly zero
// offset. Keeping the original datum name prevents an output record from
// presenting a declared radial approximation as a geodetic conversion.
func (source *RasterHeights) Reference() (crs int, datum string, offset measure.Length) {
	return source.crs, source.datum, measure.Metres(source.offset)
}

// Height retains absence and errors while applying the declared radial offset.
func (source *RasterHeights) Height(longitude, latitude float64) (float64, bool, error) {
	if source == nil || source.raster == nil {
		return 0, false, fmt.Errorf("island: no geographic height source")
	}
	height, has, err := source.raster.GeographicElevation(longitude, latitude)
	if err != nil || !has {
		return 0, false, err
	}
	height += source.offset
	if !finite(height) {
		return 0, false, fmt.Errorf("island: radial height offset overflow")
	}
	return height, true, nil
}

// Layers retains an ordered source preference. Coarser measured topobathymetry
// can cover a detailed land source's absence without inventing flat ground.
type Layers struct{ sources []Heights }

// NewLayers copies the source list; it neither opens nor closes its sources.
func NewLayers(sources ...Heights) (*Layers, error) {
	if len(sources) == 0 {
		return nil, fmt.Errorf("island: need at least one height layer")
	}
	for _, source := range sources {
		if source == nil {
			return nil, fmt.Errorf("island: absent height layer")
		}
	}
	return &Layers{sources: append([]Heights(nil), sources...)}, nil
}

// Height falls back only for absence. Corruption and exhausted source budgets
// stay errors; they must not quietly change which ground the scene renders.
func (layers *Layers) Height(longitude, latitude float64) (float64, bool, error) {
	if layers == nil || len(layers.sources) == 0 {
		return 0, false, fmt.Errorf("island: no height layers")
	}
	for _, source := range layers.sources {
		height, has, err := source.Height(longitude, latitude)
		if err != nil || has {
			return height, has, err
		}
	}
	return 0, false, nil
}

// HeightRange includes every layer's possible values, regardless of source
// preference. A layer without a known bound keeps the global envelope in use.
func (layers *Layers) HeightRange(west, south, east, north float64) (float64, float64, bool, error) {
	if layers == nil || len(layers.sources) == 0 {
		return 0, 0, false, nil
	}
	var minimum, maximum float64
	for index, source := range layers.sources {
		bounded, ok := source.(interface {
			HeightRange(float64, float64, float64, float64) (float64, float64, bool, error)
		})
		if !ok {
			return 0, 0, false, nil
		}
		low, high, has, err := bounded.HeightRange(west, south, east, north)
		if err != nil || !has {
			return 0, 0, false, err
		}
		if index == 0 {
			minimum, maximum = low, high
		} else {
			minimum, maximum = min(minimum, low), max(maximum, high)
		}
	}
	return minimum, maximum, true, nil
}
