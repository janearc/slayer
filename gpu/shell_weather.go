package gpu

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
)

// shellVolume sends a radial field directly to the kernel, without a box grid.
// Existing volume lanes carry shell radii, patch controls and advection phase.
func (encoded *scene) shellVolume(space maths.Subspace, shell medium.ShellParameters, material int) {
	weather := shell.Weather
	shift := weather.Wind.Scale(shell.Seconds)
	given := placedAt(space)
	given["puff"] = []float64{shell.Radius + shell.High, weather.Extinction, weather.PatchScale, weather.Coverage}
	given["info"] = wholes(int(uint32(weather.Seed)), int(uint32(weather.Seed>>32)), shellKind, material)
	given["baked"] = wholes(shell.NoiseOctaves, 0)
	given["low"] = []float64{shell.Radius, shell.Low, shell.High}
	given["high"] = []float64{shift.X, shift.Y, shift.Z, shell.ThresholdShoulder}
	given["grid"], given["shone"] = nil, wholes(-1, -1)
	given["lighting"], given["tint"] = wholes(0, 0, 0, -1), []float64{1, 1, 1}
	noAir(given)
	encoded.volumes.row(volumeStruct, given)
	encoded.counts.volumes++
	encoded.counts.shells++
}
