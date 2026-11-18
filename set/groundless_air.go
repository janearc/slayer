package set

import yaml "gopkg.in/yaml.v3"

// GroundlessAir is a scene's explicit local Earth coordinate origin,
// atmosphere-column altitude and named approximation choice.
type GroundlessAir struct {
	coordinateOrigin, boundary string
	referenceAltitude          float64
}

// CoordinateOrigin names the datum for east/north/up scene coordinates.
func (air GroundlessAir) CoordinateOrigin() string {
	return air.coordinateOrigin
}

// Boundary names the existing Earth model's fixed land-albedo and
// continental-aerosol approximation.
func (air GroundlessAir) Boundary() string { return air.boundary }

// ReferenceAltitude is the single atmosphere-column altitude, in metres
// above the model's mean-radius datum.
func (air GroundlessAir) ReferenceAltitude() float64 {
	return air.referenceAltitude
}

// earthAirTop is the upper boundary in airs/earth.yaml, in metres above
// the same mean-radius datum.
const earthAirTop = 86000.0

// groundlessAir reads a deliberately explicit coordinate and atmosphere
// choice. It does not imply sea level, a geodetic datum, or a site model.
func (reading reader) groundlessAir(_, block *yaml.Node) error {
	air := GroundlessAir{}
	altitude := func(key, value *yaml.Node) error {
		if err := reading.quantity(&air.referenceAltitude, "m", 0,
			earthAirTop)(key, value); err != nil {
			return err
		}
		if air.referenceAltitude >= earthAirTop {
			return reading.refuse(value, "reference altitude must be below the Earth air model top at %g m", earthAirTop)
		}
		return nil
	}
	err := reading.keys(block, map[string]step{
		"coordinate-origin":  reading.only("earth-mean-radius"),
		"reference-altitude": altitude,
		"boundary":           reading.only("earth-standard-land"),
	}, nil)
	if err != nil {
		return err
	}
	if err := reading.all(block, "groundless-air", "coordinate-origin",
		"reference-altitude", "boundary"); err != nil {
		return err
	}
	air.coordinateOrigin = valueOf(block, "coordinate-origin")
	air.boundary = valueOf(block, "boundary")
	reading.set.groundlessAir = &air
	return nil
}
