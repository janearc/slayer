package gpu

import (
	"errors"
	"fmt"
	"time"

	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
)

// Parts of a world a Refusal can be about.
const (
	// partWorld is the world as a whole: its ocean, fields, haze, optics,
	// and what is packed after every part is placed.
	partWorld = "world"
	// partThing is one placed thing.
	partThing = "thing"
	// partVolume is one volume.
	partVolume = "volume"
	// partLamp is one lamp.
	partLamp = "lamp"
)

// noIndex is the index of a refusal about no one part of the world.
const noIndex = -1

// Refusal is one reason the GPU path would not draw a world. A placed
// thing carries no name of its own once the world is made, so it is
// found by its place in the world's list, the kind of its shape, and
// the name of its material.
type Refusal struct {
	// Part is "world", "thing", "volume" or "lamp".
	Part string
	// Index is the part's place in its list in the world, or -1 for the
	// world as a whole.
	Index int
	// Kind is the Go type of the thing's shape, the volume's medium or
	// the lamp; empty for the world.
	Kind string
	// Material is the name of the part's material, where it has one.
	Material string
	// Reason is what the GPU path cannot do, as it says it.
	Reason string
}

// Refusals is every reason the GPU path would refuse to draw world at
// time at, found without drawing and without stopping at the first:
// the world as a whole, each placed thing, each volume and each lamp,
// through the same encoder that draws. The drawing path still stops at
// the first refusal; this keeps going. It is empty when the encoder
// would take the world.
//
// It covers what the encoder refuses and not what is refused after
// encoding, when a frame is drawn: the kernel's own limits.
func Refusals(world *thing.World, at time.Duration) []Refusal {
	encoded, found := stagedRefusals(world, at)
	found = append(found, encoded.thingRefusals(world, at)...)
	found = append(found, encoded.volumeRefusals(world, at)...)
	found = append(found, lampRefusals(world)...)
	found = append(found, crowdRefusals(world)...)
	return append(found, encoded.packedRefusals(world)...)
}

// reasonOf is what an error says the GPU path cannot do: the words of an
// unsupported one without the sentence round them, and any other whole.
func reasonOf(err error) string {
	var refused unsupported
	if errors.As(err, &refused) {
		return refused.what
	}
	return err.Error()
}

// ofWorld is a refusal of the world as a whole.
func ofWorld(err error) Refusal {
	return Refusal{Part: partWorld, Index: noIndex, Reason: reasonOf(err)}
}

// stagedRefusals is the scene set up as the encoder sets it up for the
// world, and every refusal of the world as a whole that the encoder
// meets on the way: what it looks at before it places a thing.
func stagedRefusals(world *thing.World, at time.Duration) (*scene, []Refusal) {
	var found []Refusal
	if trace.HasClearCloudOverlay(world) &&
		(world.OceanSurface == nil || !world.OceanSurface.DirectMoonMeanPlane) {
		found = append(found, ofWorld(unsupported{"named clear-water " +
			"transport with local cloud volumes outside finite lunar route"}))
	}
	hasNamedOptics, err := trace.PrepareClearState(world, nil)
	if err != nil {
		found = append(found, ofWorld(err))
	}
	moon, err := trace.PlanOceanMoon(world, at)
	if err != nil {
		found = append(found, ofWorld(unsupported{err.Error()}))
	}
	for _, refused := range worldRefusals(world) {
		found = append(found, ofWorld(refused))
	}
	encoded := newScene()
	encoded.oceanMoon = moon
	encoded.opticalState = hasNamedOptics
	encoded.oceanSpotScattering = world.OceanSurface != nil &&
		world.OceanSurface.SpotInscatter
	encoded.gravity = len(world.Fields) == maxGravityFields
	encoded.at = at
	encoded.volumeSky = world.VolumeIllumination.Samples()
	encoded.spread = trace.Spread(world, at)
	encoded.glowing, _ = world.Glow.(glowLattice)
	if hasNamedOptics {
		if err := encoded.prepareOpticalState(world, nil); err != nil {
			found = append(found, ofWorld(err))
		}
	}
	return encoded, found
}

// materialName is the name of a thing's or volume's material, or nothing
// where it has none.
func materialName(material *thing.Material) string {
	if material == nil {
		return ""
	}
	return material.Name
}

// thingRefusals is the refusal of each placed thing the encoder would
// place, each tried by itself so that one does not hide the next.
func (encoded *scene) thingRefusals(world *thing.World,
	at time.Duration) []Refusal {
	var found []Refusal
	for index, placed := range world.Things {
		if placed.Hidden() {
			continue
		}
		if err := encoded.place(placed, at); err != nil {
			found = append(found, Refusal{Part: partThing, Index: index,
				Kind:     fmt.Sprintf("%T", placed.Shape),
				Material: materialName(placed.Material),
				Reason:   reasonOf(err)})
		}
	}
	return found
}

// volumeRefusals is the refusal of each volume the encoder would place,
// and of there being more of them than it will.
func (encoded *scene) volumeRefusals(world *thing.World,
	at time.Duration) []Refusal {
	var found []Refusal
	if len(world.Volumes) > mostVolumes {
		found = append(found, ofWorld(unsupported{fmt.Sprintf(
			"more than %d volumes", mostVolumes)}))
	}
	for index, volume := range world.Volumes {
		if volume.Hidden() || volume.Medium == nil {
			continue
		}
		if err := encoded.volumeAt(volume, at); err != nil {
			found = append(found, Refusal{Part: partVolume, Index: index,
				Kind:     fmt.Sprintf("%T", volume.Medium),
				Material: materialName(volume.Material),
				Reason:   reasonOf(err)})
		}
	}
	return found
}

// lampRefusals is the refusal of each lamp.
func lampRefusals(world *thing.World) []Refusal {
	var found []Refusal
	for index, lamp := range world.Lamps {
		if err := refuseLamp(lamp, len(world.Lamps)); err != nil {
			found = append(found, Refusal{Part: partLamp, Index: index,
				Kind: fmt.Sprintf("%T", lamp), Reason: reasonOf(err)})
		}
	}
	return found
}

// crowdRefusals is the refusal of a world whose lamps the kernel cannot
// ask, from the crowd they make as the encoder makes it: the world's own,
// as it is no one lamp's.
func crowdRefusals(world *thing.World) []Refusal {
	if err := lampCrowdOf(world.Lamps).refuse(world.Lamps); err != nil {
		return []Refusal{ofWorld(err)}
	}
	return nil
}

// packedRefusals is the refusal of what the encoder does once every
// part is placed: packing the faces and the standings, the grid of
// light the world's volumes are lit by, and the world's glow.
func (encoded *scene) packedRefusals(world *thing.World) []Refusal {
	var found []Refusal
	for _, step := range []func() error{encoded.packPool,
		encoded.instancesOf, func() error {
			return encoded.reserveIncidentGrid(world.VolumeIllumination)
		}, encoded.glowOf} {
		if err := step(); err != nil {
			found = append(found, ofWorld(err))
		}
	}
	return found
}
