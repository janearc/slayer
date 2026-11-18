package manufacture

import (
	"errors"
	"math"
	"math/rand"
)

const (
	cavitationMaxCount           = 512
	cavitationMaxSpeed           = 100.0
	cavitationMaxTrailSeconds    = 10.0
	cavitationMaxTrailLength     = 500.0
	cavitationMaxRiseSpeed       = 10.0
	cavitationMaxRise            = 100.0
	cavitationMaxSpread          = 50.0
	cavitationMaxRadius          = 5.0
	cavitationMaxScrewOffset     = 5.0
	cavitationMaxLength          = 200.0
	cavitationMinLength          = 20.0
	cavitationMaxWorldHorizontal = 1_000_000.0
	cavitationMaxWorldVertical   = 10_000.0
	cavitationHorizontalEnvelope = 755.0
	cavitationVerticalEnvelope   = 200.0
)

// CavitationConfig controls a bounded authored gas-sphere trail. ScrewAftM is
// measured aft from the bow; ScrewOffsetM is the authored gap from the screw
// to the first sphere, and ScrewUpM is its chosen height over the keel. This
// is a visual proxy, not a cavitation solver.
type CavitationConfig struct {
	Count        int
	Seed         int64
	SpeedMPS     float64
	TrailSeconds float64
	RiseMPS      float64
	LengthM      float64
	ScrewAftM    float64
	ScrewOffsetM float64
	ScrewUpM     float64
	SpreadM      float64
	RadiusMinM   float64
	RadiusMaxM   float64
}

// CavitationPose locates the craft in map coordinates. Heading is clockwise
// degrees from north, matching set placement.
type CavitationPose struct {
	EastM, NorthM, KeelUpM float64
	HeadingDegrees         float64
}

// CavitationBubble is a map-space placement for one analytic gas sphere.
type CavitationBubble struct {
	EastM, NorthM, UpM, RadiusM float64
}

// NewCavitationCloud returns a seeded and spatially bounded trail behind the
// main screw. Trail length is speed times authored trail time, while rise is
// rise speed times particle age. The spheres are authored geometry; their
// distribution is not a fluid or cavitation simulation.
func NewCavitationCloud(config CavitationConfig, pose CavitationPose) ([]CavitationBubble, error) {
	if err := config.check(); err != nil {
		return nil, err
	}
	if err := pose.check(); err != nil {
		return nil, err
	}
	cloud := make([]CavitationBubble, config.Count)
	if config.Count == 0 {
		return cloud, nil
	}
	random := rand.New(rand.NewSource(config.Seed))
	trail := config.SpeedMPS * config.TrailSeconds
	angle := -pose.HeadingDegrees * math.Pi / 180
	cosine, sine := math.Cos(angle), math.Sin(angle)
	for index := range cloud {
		share := (float64(index) + random.Float64()) / float64(config.Count)
		age := config.TrailSeconds * share
		spread := config.SpreadM * (0.2 + 0.8*share)
		starboard := clamp(random.NormFloat64()*spread*0.38,
			-config.SpreadM, config.SpreadM)
		verticalJitter := clamp(random.NormFloat64()*spread*0.18,
			-config.SpreadM/2, config.SpreadM/2)
		localUp := config.ScrewUpM + config.RiseMPS*age + verticalJitter
		localAft := config.ScrewAftM + config.ScrewOffsetM + trail*share - config.LengthM/2
		radius := config.RadiusMinM +
			(config.RadiusMaxM-config.RadiusMinM)*random.Float64()
		cloud[index] = CavitationBubble{
			EastM:   pose.EastM + cosine*starboard + sine*localAft,
			NorthM:  pose.NorthM + sine*starboard - cosine*localAft,
			UpM:     pose.KeelUpM + localUp,
			RadiusM: radius,
		}
		if !cavitationFinite(cloud[index].EastM, cloud[index].NorthM,
			cloud[index].UpM, cloud[index].RadiusM) ||
			math.Abs(cloud[index].EastM) > cavitationMaxWorldHorizontal ||
			math.Abs(cloud[index].NorthM) > cavitationMaxWorldHorizontal ||
			math.Abs(cloud[index].UpM) > cavitationMaxWorldVertical {
			return nil, errors.New("cavitation cloud placement exceeds world bounds")
		}
	}
	return cloud, nil
}

// check reports an error when the count, speeds, lengths or radii are
// outside their bounds or are not finite.
func (config CavitationConfig) check() error {
	if config.Count < 0 || config.Count > cavitationMaxCount {
		return errors.New("cavitation cloud count must be between 0 and 512")
	}
	values := []float64{config.SpeedMPS, config.TrailSeconds, config.RiseMPS,
		config.LengthM, config.ScrewAftM, config.ScrewOffsetM, config.ScrewUpM,
		config.SpreadM,
		config.RadiusMinM, config.RadiusMaxM}
	if !cavitationFinite(values...) {
		return errors.New("cavitation cloud values must be finite")
	}
	if config.SpeedMPS < 0 || config.SpeedMPS > cavitationMaxSpeed ||
		config.TrailSeconds < 0 || config.TrailSeconds > cavitationMaxTrailSeconds ||
		config.SpeedMPS*config.TrailSeconds > cavitationMaxTrailLength {
		return errors.New("cavitation trail exceeds its speed, duration, or 500 m extent bound")
	}
	if config.RiseMPS < 0 || config.RiseMPS > cavitationMaxRiseSpeed ||
		config.RiseMPS*config.TrailSeconds > cavitationMaxRise {
		return errors.New("cavitation rise exceeds its speed or 100 m extent bound")
	}
	if config.SpreadM < 0 || config.SpreadM > cavitationMaxSpread ||
		config.RadiusMinM <= 0 || config.RadiusMaxM < config.RadiusMinM ||
		config.RadiusMaxM > cavitationMaxRadius {
		return errors.New("cavitation spread or sphere radius exceeds its bounds")
	}
	if config.LengthM < cavitationMinLength || config.LengthM > cavitationMaxLength ||
		config.ScrewAftM < 0 || config.ScrewAftM > config.LengthM ||
		config.ScrewOffsetM < 0 || config.ScrewOffsetM > cavitationMaxScrewOffset ||
		config.ScrewUpM < 0 || config.ScrewUpM > cavitationMaxSpread {
		return errors.New("cavitation screw and vessel dimensions exceed their bounds")
	}
	return nil
}

// check reports an error when the pose is not finite or lies outside the
// world-coordinate bounds.
func (pose CavitationPose) check() error {
	if !cavitationFinite(pose.EastM, pose.NorthM, pose.KeelUpM, pose.HeadingDegrees) {
		return errors.New("cavitation pose values must be finite")
	}
	if math.Abs(pose.EastM) > cavitationMaxWorldHorizontal-cavitationHorizontalEnvelope ||
		math.Abs(pose.NorthM) > cavitationMaxWorldHorizontal-cavitationHorizontalEnvelope ||
		math.Abs(pose.KeelUpM) > cavitationMaxWorldVertical-cavitationVerticalEnvelope ||
		math.Abs(pose.HeadingDegrees) > 360_000 {
		return errors.New("cavitation pose exceeds its world-coordinate bounds")
	}
	return nil
}

// cavitationFinite reports whether every value is a finite number.
func cavitationFinite(values ...float64) bool {
	for _, value := range values {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return false
		}
	}
	return true
}

// clamp limits value to the range from low to high.
func clamp(value, low, high float64) float64 {
	return math.Max(low, math.Min(high, value))
}
