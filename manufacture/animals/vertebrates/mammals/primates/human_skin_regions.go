package primates

import (
	"math"

	"github.com/janearc/slayer/maths"
)

type skinAbsorption struct{ melanin, blood float64 }

type facialSkin struct {
	nose, cheek, forehead, chin, temple, underEye, ear float64
}

func facialSkinAt(rest maths.Vec) facialSkin {
	if rest.Y < 1.42 {
		return facialSkin{}
	}
	x, y, z := math.Abs(rest.X), rest.Y, rest.Z
	head := maths.Smooth(1.42, 1.48, y)
	front := head * maths.Smooth(.031, .073, z)
	cheekHeight := 1.536 + .002*math.Tanh(40*rest.X)
	return facialSkin{
		nose:     front * bell(x/.020) * bell((y-1.548)/.024),
		cheek:    front * bell((x-.043)/.028) * bell((y-cheekHeight)/.024),
		forehead: front * bell(x/.061) * bell((y-1.607)/.037),
		chin:     front * bell(x/.029) * bell((y-1.483)/.018),
		temple:   head * maths.Smooth(.005, .049, z) * bell((x-.062)/.021) * bell((y-1.588)/.033),
		underEye: front * bell((x-eyeHalfSpacing)/.019) * bell((y-(eyeHeight-.010))/.010),
		ear:      head * bell((x-.077)/.019) * bell((y-1.548)/.037) * bell((z+.017)/.025),
	}
}

// adultSkinRegions follows the native chart in canonical metres. The broad
// patches overlap smoothly across the same skin used by every pose.
func adultSkinRegions(rest maths.Vec, face facialSkin, complexion float64) (skinAbsorption, float64) {
	x, y, z := math.Abs(rest.X), rest.Y, rest.Z
	side := humanSides[0]
	elbow, wrist, knee := side.elbow(), side.wrist(), side.knee()
	arms := maths.Smooth(.162, .198, x) * maths.Smooth(.760, .913, y) *
		(1 - maths.Smooth(1.275, 1.375, y))
	axis := limbAxisAt(y, side.shoulder(), elbow, wrist)
	outer := arms * maths.Smooth(-.005, .029, x-axis.X)
	inner := arms * (1 - maths.Smooth(-.024, .001, x-axis.X))
	elbowTip := arms * bell((y-elbow.Y)/.034) * (1 - maths.Smooth(-.054, -.030, z))
	kneecap := bell((y-knee.Y)/.058) * bell((x-knee.X)/.060) * maths.Smooth(.012, .041, z)

	hand := maths.Smooth(.197, .228, x) * (1 - maths.Smooth(.789, .831, y))
	palm := hand * (1 - maths.Smooth(.247, .260, x)) * maths.Smooth(.615, .664, y)
	knuckles := hand * maths.Smooth(.256, .276, x) * bell((y-.749)/.026)
	arch := .006 * bell((z-.055)/.040)
	sole := (1 - maths.Smooth(.010+arch, .028+arch, y)) *
		maths.Smooth(.023, .057, x) * (1 - maths.Smooth(.141, .164, x))
	glabrous := max(palm, sole)

	// Picked regional shifts in optical density. Reduced palmar and plantar
	// absorption depends on the overall complexion.
	lightening := .026 + .34*complexion + .20*complexion*complexion
	sunExposure := .050 + .042*complexion
	facialBlood := .29*face.cheek + .19*face.nose + .18*face.ear +
		.12*face.chin + .12*face.underEye - .036*face.temple
	return skinAbsorption{
		melanin: sunExposure*(face.forehead+.56*face.temple+.36*face.underEye+outer) - .040*inner +
			.063*elbowTip + .054*kneecap + .044*knuckles - lightening*glabrous,
		blood: (1-.28*complexion)*facialBlood + .05*knuckles + .08*glabrous,
	}, glabrous
}
