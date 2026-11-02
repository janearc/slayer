package maths

// Light is an amount of red, green and blue light. It is linear: twice the
// light is twice the number. There is no ceiling. One type serves for
// a lamp's strength. It is also the share of light a surface gives back,
// and what a ray brings to the eye.
//
// It is here because a renderer does arithmetic on light: add, scale and
// multiply. What light looks like is libtheme's business. When libtheme
// has bands, Light becomes an amount per band.
//
//ninja:vow practice a library: its callers are in other modules
type Light struct {
	//ninja:vow gate a value, whose fields are what it is
	R, G, B float64 //ninja:vow name a colour is r, g, b at every call site
}

// RGB is a Light from its three parts.
//
//ninja:vow practice a library: its callers are in other modules
func RGB(red, green, blue float64) Light { return Light{red, green, blue} }

// Grey is the same amount of all three.
//
//ninja:vow practice a library: its callers are in other modules
func Grey(amount float64) Light { return Light{amount, amount, amount} }

// Add is the two lights together, as when two lamps light one point.
//
//ninja:vow practice a library: its callers are in other modules
func (light Light) Add(other Light) Light {
	return Light{light.R + other.R, light.G + other.G, light.B + other.B}
}

// Scale is the light multiplied by factor.
//
//ninja:vow practice a library: its callers are in other modules
func (light Light) Scale(factor float64) Light {
	return Light{light.R * factor, light.G * factor, light.B * factor}
}

// Times is the two multiplied part by part: a lamp's light on a surface
// that gives back only some of each colour.
//
//ninja:vow practice a library: its callers are in other modules
func (light Light) Times(other Light) Light {
	return Light{light.R * other.R, light.G * other.G, light.B * other.B}
}

// Luminance is how bright the light looks. It is one number, weighted as
// the eye weights the three parts. Blue counts for about 7%.
//
//ninja:vow practice a library: its callers are in other modules
func (light Light) Luminance() float64 {
	return redWeight*light.R + greenWeight*light.G + blueWeight*light.B
}

// What Luminance weighs each part by. ITU-R BT.709 defines them: it is
// the standard for the red, green and blue of high definition pictures.
// The three add to one.
const (
	redWeight   = 0.2126 // defined: BT.709's weight of red
	greenWeight = 0.7152 // defined: BT.709's weight of green
	blueWeight  = 0.0722 // defined: BT.709's weight of blue
)
