package gpu

import "math"

// A valid double HG parameter may round to an invalid float32 endpoint.
// Refuse rather than clamp its authored law; likewise do not flush a tiny
// nonzero asymmetry to isotropic under the kernel's fast-math behavior.
func metalCloudAsymmetry(g float64) error {
	packed := float32(g)
	if math.IsNaN(g) || math.IsInf(g, 0) || !(packed > -1 && packed < 1) ||
		(g != 0 && math.Abs(float64(packed)) < float64(math.Float32frombits(0x00800000))) {
		return unsupported{"cloud HG asymmetry outside the normal float32 open interval (-1,1)"}
	}
	return nil
}

// scatter.w is this volume's phase; air.y explicitly selects it, so authored
// zero cannot be mistaken for omitted inheritance. Water optics are untouched.
func (encoded *scene) cloudPhaseLast(g float64) {
	encoded.putLast("scatter", floatWords(0, 0, 0, g))
	encoded.putLast("air", []uint32{0, 1})
}
