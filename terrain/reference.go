package terrain

import "fmt"

// SourceReference retains GeoTIFF registry codes without guessing a projection
// or height transformation. Zero means unspecified. These fields are SHORT
// GeoKeys defined in OGC GeoTIFF1.1, sections7.2-7.5 and AnnexB.2.2.
type SourceReference struct {
	Model, RasterType           int
	GeographicCRS, ProjectedCRS int
	AngularUnits, LinearUnits   int
	VerticalCRS, VerticalUnits  int
	// Transform is the projection method, ProjCoordTrans, key 3075.
	Transform int
}

// The GeoKey ids, directory tags and codes below are defined by OGC
// GeoTIFF1.1, document 19-008r4, sections 7.1.2 and 7.5 and Annex B.
const (
	geoKeyDirectory    = 34735 // defined: GeoKeyDirectoryTag
	geoKeyDoubles      = 34736 // defined: GeoDoubleParamsTag
	keySemiMajor       = 2057  // defined: GeogSemiMajorAxisGeoKey
	keySemiMinor       = 2058  // defined: GeogSemiMinorAxisGeoKey
	keyTransform       = 3075  // defined: ProjCoordTransGeoKey
	keyParallel        = 3078  // defined: ProjStdParallel1GeoKey
	keyFalseEast       = 3082  // defined: ProjFalseEastingGeoKey
	keyFalseNorth      = 3083  // defined: ProjFalseNorthingGeoKey
	keyCentreLongitude = 3088  // defined: ProjCenterLongGeoKey
	keyCentreLatitude  = 3089  // defined: ProjCenterLatGeoKey
)

// sphereKeys are the GeoKeys, held as doubles, that an equirectangular
// projection on a sphere needs. An absent key is zero, as GeoTIFF defaults it.
type sphereKeys struct {
	semiMajor, semiMinor            float64
	parallel                        float64
	falseEast, falseNorth           float64
	centreLongitude, centreLatitude float64
}

// Reference reports declared source metadata, not a resolved datum transform.
func (raster *Raster) Reference() SourceReference { return raster.reference }

// readReference reads the GeoKey directory into the raster's reference, if
// the file carries one; a file without one is not an error.
func (raster *Raster) readReference(tags map[int]entry) error {
	raw, present := tags[geoKeyDirectory]
	if !present {
		return nil
	}
	if raw.kind != 3 || raw.count < 4 {
		return fmt.Errorf("terrain: invalid GeoKey directory")
	}
	keys := raster.numbers(raw)
	if keys[0] != 1 || keys[1] != 1 || keys[2] > 1 || int(keys[3]) > (len(keys)-4)/4 {
		return fmt.Errorf("terrain: unsupported or truncated GeoKey directory")
	}
	ref := SourceReference{}
	measures := sphereKeys{}
	wanted := map[uint64]*int{
		1024: &ref.Model, 1025: &ref.RasterType,
		2048: &ref.GeographicCRS, 2054: &ref.AngularUnits,
		3072: &ref.ProjectedCRS, 3076: &ref.LinearUnits,
		4096: &ref.VerticalCRS, 4099: &ref.VerticalUnits,
		keyTransform: &ref.Transform,
	}
	measured := map[uint64]*float64{
		keySemiMajor: &measures.semiMajor, keySemiMinor: &measures.semiMinor,
		keyParallel: &measures.parallel, keyFalseEast: &measures.falseEast,
		keyFalseNorth:      &measures.falseNorth,
		keyCentreLongitude: &measures.centreLongitude,
		keyCentreLatitude:  &measures.centreLatitude,
	}
	doubles := raster.reals(tags[geoKeyDoubles])
	seen := map[uint64]bool{}
	for i := 0; i < int(keys[3]); i++ {
		row := keys[4+4*i : 8+4*i]
		if seen[row[0]] {
			return fmt.Errorf("terrain: repeated GeoKey %d", row[0])
		}
		seen[row[0]] = true
		if target, ok := wanted[row[0]]; ok {
			if row[1] != 0 || row[2] != 1 {
				return fmt.Errorf("terrain: non-scalar GeoKey %d", row[0])
			}
			*target = int(row[3])
		}
		if target, ok := measured[row[0]]; ok {
			if row[1] != geoKeyDoubles || row[2] != 1 || row[3] >= uint64(len(doubles)) {
				return fmt.Errorf("terrain: GeoKey %d is not one value of the doubles tag", row[0])
			}
			*target = doubles[row[3]]
		}
	}
	raster.reference, raster.sphere = ref, measures
	return nil
}

// SampleLocation gives a native sample's declared source coordinates.
// PixelIsArea uses cell centres; PixelIsPoint uses integer lattice points.
// Unknown registration is refused here; older projected window calls retain
// their legacy behavior. No data is decoded by this metadata-only operation.
func (raster *Raster) SampleLocation(column, row int) (float64, float64, error) {
	if raster == nil || raster.closed || !raster.placed ||
		column < 0 || row < 0 || column >= raster.width || row >= raster.height {
		return 0, 0, fmt.Errorf("terrain: invalid sample location")
	}
	offset := 0.0
	switch raster.reference.RasterType {
	case 1:
		offset = 0.5
	case 2:
	default:
		return 0, 0, fmt.Errorf("terrain: unspecified pixel registration")
	}
	return raster.west + (float64(column)+offset)*raster.spacing,
		raster.north - (float64(row)+offset)*raster.rowSpacing(), nil
}

// projectedCoordinates protects legacy metre-based window requests from
// geographic, geocentric or explicitly non-metre sources. An untagged source
// retains the old assumed-metres contract; this is not a CRS lookup service.
func (raster *Raster) projectedCoordinates() error {
	ref := raster.reference
	if (ref.Model != 0 && ref.Model != 1) ||
		(ref.Model == 0 && ref.GeographicCRS != 0 && ref.ProjectedCRS == 0) {
		return fmt.Errorf("terrain: source requires a geographic world adapter")
	}
	// EPSG unit9001 is metre; OGC GeoTIFF1.1 AnnexF examples.
	if ref.LinearUnits != 0 && ref.LinearUnits != 9001 {
		return fmt.Errorf("terrain: source coordinates are not declared metres")
	}
	return nil
}
