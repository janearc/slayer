package hillside

import (
	"fmt"

	"github.com/janearc/slayer/set"
)

// placeGroundlessAir stages an explicitly referenced Earth atmosphere
// without terrain. Its east/north positions are local tangent offsets and
// its up coordinates use earth-mean-radius as the origin.
func (stage *staging) placeGroundlessAir(read set.Set,
	reference set.GroundlessAir) error {
	if read.Air() != "earth" {
		return fmt.Errorf("%s: groundless-air requires air: earth",
			stage.given.set)
	}
	if reference.CoordinateOrigin() != "earth-mean-radius" {
		return fmt.Errorf("%s: groundless-air requires coordinate-origin earth-mean-radius",
			stage.given.set)
	}
	if reference.Boundary() != "earth-standard-land" {
		return fmt.Errorf("%s: unsupported groundless Earth boundary %q",
			stage.given.set, reference.Boundary())
	}
	if stage.given.lake > nothing {
		return fmt.Errorf("%s: -lake needs terrain and cannot be used with groundless-air",
			stage.given.set)
	}
	for _, each := range read.Things() {
		if _, has := each.Position(); !has {
			return fmt.Errorf("%s: thing %q needs an absolute position with groundless-air",
				stage.given.set, each.Name())
		}
	}
	if len(read.Stands()) > 0 || len(read.Texts()) > 0 ||
		len(read.Ecosystems()) > 0 || len(read.FireFronts()) > 0 ||
		len(read.FuelBeds()) > 0 || len(read.Lasers()) > 0 ||
		len(read.AtmosphericLayers()) > 0 {
		return fmt.Errorf("%s: groundless-air does not support terrain-dependent stands, text, ecosystems, fire, lasers or local atmosphere layers",
			stage.given.set)
	}
	stage.groundlessAir = true
	stage.airAltitude = reference.ReferenceAltitude()
	stage.lowest = 0
	if err := stage.light(read); err != nil {
		return err
	}
	if err := stage.stand(read); err != nil {
		return err
	}
	if err := stage.atmosphere(read); err != nil {
		return err
	}
	if err := stage.prepareFronts(read); err != nil {
		return err
	}
	return stage.prepareBeds(read)
}
