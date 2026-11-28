package placement

import "github.com/janearc/slayer/scene/thing"

// MaterialLimits records which surface effects the opaque view leaves out.
type MaterialLimits uint8

const (
	surfaceTexture MaterialLimits = 1 << iota
	subsurfaceScattering
	fibreScattering
	thinTissueTransmission
)

func materialLimits(material *thing.Material) MaterialLimits {
	if material == nil {
		return 0
	}
	var limits MaterialLimits
	if material.Texture != nil {
		limits |= surfaceTexture
	}
	if material.Subsurface.Enabled() {
		limits |= subsurfaceScattering
	}
	if material.Fibre != nil {
		limits |= fibreScattering
	}
	if material.ThinTissue.Enabled() {
		limits |= thinTissueTransmission
	}
	return limits
}

// Notices describes the retained surfaces' material limits for View details.
func (limits MaterialLimits) Notices() []string {
	var notices []string
	if limits&surfaceTexture != 0 {
		notices = append(notices, "Surface textures are omitted in the opaque working view")
	}
	if limits&subsurfaceScattering != 0 {
		notices = append(notices, "Subsurface scattering is omitted in the working view")
	}
	if limits&fibreScattering != 0 {
		notices = append(notices, "Fibre meshes use opaque working shading; fibre scattering is omitted")
	}
	if limits&thinTissueTransmission != 0 {
		notices = append(notices, "Thin tissue transmission is omitted in the working view")
	}
	return notices
}

// CachedMaterialNotices reports limitations of meshes in the local cache.
func (scene *DisplayScene) CachedMaterialNotices() []string {
	var limits MaterialLimits
	for _, object := range scene.Objects {
		if object.Mesh != nil {
			limits |= object.Mesh.materialLimits
		}
	}
	return limits.Notices()
}
