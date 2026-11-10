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
}

// Reference reports declared source metadata, not a resolved datum transform.
func (raster *Raster) Reference() SourceReference { return raster.reference }

// readReference reads the GeoKey directory into the raster's reference, if
// the file carries one; a file without one is not an error.
func (raster *Raster) readReference(tags map[int]entry) error {
	const directoryTag = 34735 // GeoKeyDirectoryTag, OGC GeoTIFF1.1 section7.1.2.
	raw, present := tags[directoryTag]
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
	wanted := map[uint64]*int{
		1024: &ref.Model, 1025: &ref.RasterType,
		2048: &ref.GeographicCRS, 2054: &ref.AngularUnits,
		3072: &ref.ProjectedCRS, 3076: &ref.LinearUnits,
		4096: &ref.VerticalCRS, 4099: &ref.VerticalUnits,
	}
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
	}
	raster.reference = ref
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
