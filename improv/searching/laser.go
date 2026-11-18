package searching

// guideBeamDiameterMetres is the nominal 30 cm diameter published by ESO
// for 4LGSF (sources.yaml: eso-4lgsf). The authored Gaussian model takes
// that diameter at its 1/e^2 intensity contour; ESO does not specify the
// contour. Radius follows by halving the diameter at both construction sites.
const guideBeamDiameterMetres = 0.30

// guideBeamPeakGain preserves the integrated light of the earlier authored
// 0.76 m diameter at this diameter. It is a visual strength choice, rather
// than a calibration to the published laser's total power.
const guideBeamPeakGain = (0.76 / guideBeamDiameterMetres) *
	(0.76 / guideBeamDiameterMetres)
