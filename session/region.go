package session

import (
	"fmt"
	"image"
)

// regionRefusal checks a region against the resolved draw and its modes.
func regionRefusal(given asked) error {
	if given.region == nil {
		return nil
	}
	box := *given.region
	if box.Empty() || !box.In(image.Rect(0, 0, given.width, given.height)) {
		return fmt.Errorf("region %v is outside the resolved %d by %d frame", box, given.width, given.height)
	}
	for _, shortcut := range []struct {
		name string
		on   bool
	}{
		{"restir", given.restir},
		{"fastlight", given.fastLighting},
		{"volume-lighting", given.cachedLighting},
		{"sky-looks", given.skyLooks},
	} {
		if shortcut.on {
			return fmt.Errorf("region: %s is not supported", shortcut.name)
		}
	}
	if (given.onGPU && (given.nativeMeshes || given.nativeShadows)) ||
		given.composite != nil || given.wire || given.frame != nil ||
		given.glass != nil || given.lensed != nil || given.optics.aperture != 0 ||
		given.optics.opened != nil || given.optics.closing != nil ||
		len(given.initialMedia) != 0 || given.overlay != nil {
		return fmt.Errorf("region: the requested renderer, optics, shortcut or initial media are not supported")
	}
	return nil
}
