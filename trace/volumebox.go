package trace

import "github.com/janearc/slayer/maths"

// A haze of forty volumes, three mists, a cloud deck and three dozen
// plumes, was visited whole by every shadow ray and every ray through
// it: each volume took the ray into its own space and clipped it to its
// box before it could say the ray had missed. A volume placed now keeps
// its box in the world (boxOut, as a thing does), and a ray asks that
// box first.

// meets is whether a ray of the world may cross the volume before far:
// false only when it misses the volume's world box, and true for a
// volume of no box.
func (volume *placedVolume) meets(ray maths.Ray, far float64) bool {
	if !volume.walled {
		return true
	}
	_, _, met := volume.world.Hit(ray, 0, far)
	return met
}
