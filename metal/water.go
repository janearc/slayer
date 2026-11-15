//go:build darwin

package metal

// WaterLibrary is the sea's optics (oceantex) in the Metal shading
// language: the Quan-Fry refractive index, the unpolarised Fresnel
// split, the Cox-Munk slope chance and Smith's hiding. The same
// physics serves water at any scale: an ocean, a pool, a glass.
//
// The library holds no measured numbers of its own. Every sourced
// value - the Quan-Fry coefficients above all - arrives as an
// argument, so oceantex's sources.yaml stays the one home of every
// number and its provenance.
//
// The arithmetic is 32-bit float where oceantex's is 64, and erfc is
// Abramowitz and Stegun's 7.1.26 fit, good to 1.5e-7, where Go has
// the real function. The tests measure how closely the two agree.
const WaterLibrary = `
#include <metal_stdlib>
using namespace metal;

// water_index is the refractive index of seawater against air, by the
// empirical equation of Quan and Fry (1995): the ten coefficients in
// qf, salinity in parts per thousand, temperature in degrees Celsius,
// wavelength in nanometres.
static float water_index(constant float *qf, float salinity,
                         float celsius, float nm) {
	float t2 = celsius * celsius;
	return qf[0] + (qf[1] + qf[2] * celsius + qf[3] * t2) * salinity +
	       qf[4] * t2 +
	       (qf[5] + qf[6] * salinity + qf[7] * celsius) / nm +
	       qf[8] / (nm * nm) + qf[9] / (nm * nm * nm);
}

// water_fresnel is the fraction of unpolarised light a flat air-water
// surface reflects: Fresnel's equations with Snell's law, as
// oceantex.Fresnel.
static float water_fresnel(float cosine, float index) {
	cosine = clamp(cosine, 0.0f, 1.0f);
	float sine2 = (1 - cosine * cosine) / (index * index);
	float inside = sqrt(1 - sine2);
	float across = (cosine - index * inside) / (cosine + index * inside);
	float along = (index * cosine - inside) / (index * cosine + inside);
	return (across * across + along * along) / 2;
}

// slope_chance is the chance of a slope: a Gaussian, the first-order
// form of Cox and Munk (1954), per unit slope along and across the
// wind, as oceantex.SlopeChance.
static float slope_chance(float along, float across, float var_along,
                          float var_across) {
	return exp(-(along * along / var_along +
	             across * across / var_across) / 2) /
	       (2 * M_PI_F * sqrt(var_along * var_across));
}

// erfc_as is the complementary error function by Abramowitz and
// Stegun 7.1.26, within 1.5e-7 everywhere, for smith_hidden: metal
// has no erfc of its own.
static float erfc_as(float x) {
	float t = 1 / (1 + 0.3275911f * fabs(x));
	float poly = t * (0.254829592f + t * (-0.284496736f +
	             t * (1.421413741f + t * (-1.453152027f +
	             t * 1.061405429f))));
	float value = poly * exp(-x * x);
	return x < 0 ? 2 - value : value;
}

// smith_hidden is Smith's function: how much of a rough surface is
// hidden from a direction, as oceantex.Hidden, from the cosine of the
// direction's angle to the vertical and the slope variance along it.
static float smith_hidden(float cosine, float variance) {
	if (cosine <= 0) {
		return INFINITY;
	}
	if (cosine >= 1 || variance <= 0) {
		return 0;
	}
	float cot = cosine / sqrt(1 - cosine * cosine);
	float v = cot / sqrt(2 * variance);
	return (exp(-v * v) / (v * sqrt(M_PI_F)) - erfc_as(v)) / 2;
}
`
