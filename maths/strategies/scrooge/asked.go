package scrooge

import "os"

// Variable names the environment switch that asks for scrooge while it
// is on trial: set to 1, a draw makes no more of its set than its
// picture can show. Unset, a draw makes everything, as before, and is
// the reference a scrooge draw is compared with.
const Variable = "SLAYER_SCROOGE"

// Asked is whether the environment asks for scrooge.
func Asked() bool { return os.Getenv(Variable) == "1" }

// Pixel is the footprint of one pixel of a picture so many pixels across,
// through a lens so wide in radians, from where the camera stands: a
// cone that starts at nothing and widens by a pixel's angle a metre.
func Pixel(from [3]float64, wide float64, across int) Footprint {
	cone := Footprint{}
	cone.From.X, cone.From.Y, cone.From.Z = from[0], from[1], from[2]
	if across > 0 {
		cone.Spread = wide / float64(across)
	}
	return cone
}
