package medium

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

const (
	// stormsPerHemisphere is how many whorls a sky has in each hemisphere's
	// storm track.
	stormsPerHemisphere = 3
	// stormCount is the whorls in the whole sky.
	stormCount = 2 * stormsPerHemisphere
	// stormStream is the offset from a sky's seed that places its storms,
	// apart from the patches' noise and the billow's.
	stormStream = 2
	// stormLatitudeSpread is how far, in radians, a storm may sit either
	// side of the storm track's latitude.
	stormLatitudeSpread = 0.12
	// stormLongitudeSlack is the share of its slot round the track a storm
	// may wander in, so that no two of a hemisphere's storms meet.
	stormLongitudeSlack = 0.6
	// stormReach is how many storm radii out a storm stops turning the air.
	stormReach = 3.0
	// windReversal is how fast the winds change direction with latitude, in
	// radians to the radian: easterly to thirty degrees, westerly to sixty,
	// easterly again beyond, so the shift goes as sin(6 latitude).
	windReversal = 6.0
	// weatherNumbers is how many numbers lead a weather's table, before the
	// storms; see CloudWeather.Table.
	weatherNumbers = 19
	// armReach is how many storm radii out a storm's arms are banded.
	armReach = 1.5
	// beadOctaves is the octaves of the noise that breaks the equator's
	// belt into clusters, and filamentOctaves that of the filaments.
	beadOctaves, filamentOctaves = 2, 3
	// beadStream and filamentStream are the offsets from a sky's seed of
	// those noises, apart from the patches', the billow's and the storms'.
	beadStream, filamentStream = 4, 3
	// stormNumbers is the numbers a storm takes in the table: its centre
	// and its sign.
	stormNumbers = 4
	// lockedNumbers is the numbers that follow the storms in the table: a
	// locked sky's flag, its way to its sun, and its deck, terminator and
	// night, see CloudWeather.Table.
	lockedNumbers = 21
	// whorlLowest and whorlHighest are the least and the most share of its
	// height a whorl leaves a cloud: a lane is cloud as low as the first and
	// an arm a ridge as high as the second times the cloud's own. Picked by
	// eye 2026-11-27.
	whorlLowest, whorlHighest = 0.5, 2.0
	// brokenStream is the offset from a sky's seed of the noise that
	// breaks the cloud at a locked world's terminator.
	brokenStream = 5
	// towerStream is the offset from a sky's seed of the noise that gives a
	// locked world's convective towers their different heights, and
	// towerOctaves its octaves.
	towerStream, towerOctaves = 6, 3
	// towerFrom and towerTo are the tower noise's values between which a
	// cloud goes from the low sheet to a tall tower. A long step, over half
	// of the noise's range, gives towers that are hills and not mesas,
	// rounded tops whose sides are as soft as a cloud's; a step of a fifth
	// of it, which round three had, drew them as rock. Picked by eye
	// 2026-11-27.
	towerFrom, towerTo = -0.2, 0.3
	// billowRise is how many times finer than the towers the billows that
	// round their tops are, and billowOctaves their octaves. Picked by eye
	// 2026-11-27.
	billowRise, billowOctaves = 5.0, 3
	// lockedStormSpread is how far, in radians, a locked sky's whorl may sit
	// either side of its ring, picked by eye 2026-11-27 so that the whorls
	// north and south of the terminator are not mirror images.
	lockedStormSpread = 0.3
)

// Storm is one whorl in the clouds: a centre on the unit sphere and which
// way it turns, one for counter-clockwise seen from above and minus one
// for clockwise. In a locked sky the sign's size is the whorl's strength,
// which makes it that many times as wide and as turned as the weather's
// Size and Turn say; its sign alone says which way it turns.
type Storm struct {
	Centre maths.Vec
	Sign   float64
}

// CloudWeather is how Density lays weather over its patches, for a
// renderer that asks the same density itself. Equator is how much more
// cover the equator's belt has, Dry how much less the dry belts have at
// DryAt, and Track how much more the storm tracks have at TrackAt, each
// belt Width wide, all latitudes in radians. Stretch is how many times
// longer along the wind than across a cloud is drawn, and Jet how far, in
// radians, the winds slide the sky round the axis. A storm turns the sky
// by Turn radians at its heart, over Size radians of the sphere, and
// clouds its heart Core more, bar an eye, Eye radii wide, that it clears
// by EyeClear. Its Arms arms are banded by ArmCover, clearer between,
// thinning outward. Filament is how much of a noise finer Across times
// across the wind breaks a streak into filaments. Cluster is how far the
// equator's belt is broken, in Beads clusters round the body. Storms are
// where the seed put them.
//
// A sky that is Locked has one face to its sun, toward the unit vector
// Toward in the medium's frame, and its weather is laid about that point
// and not by latitude (Yang, Cowan and Abbot 2013, from memory, to
// shelve): a thick convective Deck of cover at the substellar point, Deck
// wide in radians of arc; a turbulent belt of Terminator more cover about
// the terminator, TerminatorWidth wide, broken by finer noise to the
// extent of Broken; Night less cover from NightFrom radians out, coming
// on over NightWidth, and a negative Night is more, the air that rose at
// the substellar point carrying its cloud over the rest of the planet; and
// Far less again from FarFrom, coming on over FarWidth, where the night's
// cloud thins. Its whorls turn round the terminator, each the way the
// rotation turns it in its hemisphere, and as wide and as strong as
// the seed says; the winds' sliding and stretching are Jet and Stretch,
// which a locked sky has none of, and Filament is none. Its clouds are
// Towers: see Top.
type CloudWeather struct {
	Equator, Dry, Track float64
	Width, DryAt        float64
	TrackAt             float64
	Stretch, Jet        float64
	Size, Turn, Core    float64
	Filament, Across    float64
	Cluster, Beads      float64
	Eye, EyeClear       float64
	Arms, ArmCover      float64
	Seed                uint64
	Storms              [stormCount]Storm

	Locked                       bool
	Toward                       maths.Vec
	Deck, DeckWidth              float64
	Terminator, TerminatorWidth  float64
	Night, NightFrom, NightWidth float64
	Broken, LockedStorm          float64
	LockedStrength               float64
	TowerBase, TowerWidth        float64
	Towers, TowerFrequency       float64
	Far, FarFrom, FarWidth       float64
	TowerBillow, WhorlThin       float64
}

// Weather is the weather Density lays, with this sky's storms placed by
// its seed. The axis the latitudes are measured from is the body's own,
// up in the medium's frame, which turns with the body.
func (clouds Clouds) Weather() CloudWeather {
	weather := CloudWeather{Equator: chosen.cloudEquator,
		Dry: chosen.cloudDry, Track: chosen.cloudTrack,
		Width: chosen.cloudWidth, DryAt: chosen.cloudDryAt,
		TrackAt: chosen.cloudTrackAt, Stretch: chosen.cloudStretch,
		Jet: chosen.cloudJet, Size: chosen.cloudWhorlSize,
		Turn: chosen.cloudWhorlTurn, Core: chosen.cloudWhorlCore,
		Filament: chosen.cloudFilament, Across: chosen.cloudAcross,
		Cluster: chosen.cloudCluster, Beads: chosen.cloudBeads,
		Eye: chosen.cloudEye, EyeClear: chosen.cloudEyeClear,
		Arms: chosen.cloudArms, ArmCover: chosen.cloudArmCover,
		Seed: clouds.Seed, Locked: clouds.Locked,
		Toward: clouds.Toward.Unit(), Deck: chosen.cloudDeck,
		DeckWidth: chosen.cloudDeckWidth, Terminator: chosen.cloudTerminator,
		TerminatorWidth: chosen.cloudTerminatorWidth,
		Night:           chosen.cloudNight, NightFrom: chosen.cloudNightFrom,
		NightWidth: chosen.cloudNightWidth, Broken: chosen.cloudBroken,
		LockedStorm: chosen.cloudLockedStorm}
	if weather.Locked {
		weather.Stretch = chosen.cloudLockedStretch
		weather.Filament = chosen.cloudLockedFilament
		weather.Jet = chosen.cloudLockedJet
		weather.LockedStrength = chosen.cloudLockedStrength
		weather.TowerBase = chosen.cloudTowerBase
		weather.TowerWidth = chosen.cloudTowerWidth
		weather.Towers = chosen.cloudTowers
		weather.TowerFrequency = chosen.cloudTowerFrequency
		weather.Far = chosen.cloudFar
		weather.FarFrom = chosen.cloudFarFrom
		weather.FarWidth = chosen.cloudFarWidth
		weather.TowerBillow = chosen.cloudTowerBillow
		weather.EyeClear = chosen.cloudLockedEyeClear
		weather.ArmCover = chosen.cloudLockedArmCover
		weather.Turn = chosen.cloudLockedTurn
		weather.WhorlThin = chosen.cloudLockedWhorlThin
		weather.Core = chosen.cloudLockedCore
		weather.Size = chosen.cloudLockedSize
	}
	if clouds.HasTurn {
		weather.Turn = clouds.Turn
	}
	for index := range weather.Storms {
		weather.Storms[index] = weather.place(clouds.Seed, index)
	}
	return weather
}

// place is the storm at the index: the first half north, turning
// counter-clockwise, the rest south, turning clockwise, spread round the
// storm track by the seed.
func (weather CloudWeather) place(seed uint64, index int) Storm {
	if weather.Locked {
		return weather.ring(seed, index)
	}
	sign := float64(unity)
	if index >= stormsPerHemisphere {
		sign = -sign
	}
	slot := index % stormsPerHemisphere
	stream := seed + stormStream
	across := noise.Hash(stream, int64(2*index))*2 - unity
	along := noise.Hash(stream, int64(2*index+1))
	latitude := sign * (weather.TrackAt + across*stormLatitudeSpread)
	slack := stormLongitudeSlack
	longitude := 2 * math.Pi * (float64(slot) + (unity-slack)/2 +
		slack*along) / stormsPerHemisphere
	flat := math.Cos(latitude)
	return Storm{Centre: maths.V(flat*math.Cos(longitude),
		math.Sin(latitude), flat*math.Sin(longitude)), Sign: sign}
}

// frame is two unit vectors square to each other and to Toward, which
// the whorls of a locked sky are spread round.
func (weather CloudWeather) frame() (maths.Vec, maths.Vec) {
	first := maths.YAxis.Cross(weather.Toward)
	if first.Len() < frameLeast {
		first = maths.XAxis.Cross(weather.Toward)
	}
	first = first.Unit()
	return first, weather.Toward.Cross(first)
}

// frameLeast is how short the spin axis's cross with Toward may be before
// the axis is taken to be along it, and another stands in: defined as a
// thousandth, a look along the axis to a tenth of a degree.
const frameLeast = 1e-3

// ring is the whorl at the index in a locked sky: the six spread round
// the terminator, each LockedStorm radians from the substellar point, a
// little either way by the seed, and each turning the way the rotation
// turns the air of its hemisphere: counter-clockwise north, clockwise
// south.
func (weather CloudWeather) ring(seed uint64, index int) Storm {
	stream := seed + stormStream
	across := noise.Hash(stream, int64(2*index))*2 - unity
	along := noise.Hash(stream, int64(2*index+1))
	size := noise.Hash(stream, int64(stormNumbers*stormCount+index))*2 - unity
	slack := stormLongitudeSlack
	azimuth := 2 * math.Pi * (float64(index) + (unity-slack)/2 +
		slack*along) / stormCount
	away := weather.LockedStorm + across*lockedStormSpread
	first, second := weather.frame()
	round := first.Scale(math.Cos(azimuth)).Add(second.Scale(
		math.Sin(azimuth)))
	centre := weather.Toward.Scale(math.Cos(away)).Add(round.Scale(
		math.Sin(away)))
	sign := unity + weather.LockedStrength*size
	if centre.Y < nought {
		sign = -sign
	}
	return Storm{Centre: centre, Sign: sign}
}

// Locking is the cover a locked sky adds at a direction, and how far into
// the terminator's belt the direction is, from nought to one: the deck at
// the substellar point, the belt about the terminator, and the clearing
// of the night side.
func (weather CloudWeather) Locking(direction maths.Vec) (cover, belt float64) {
	away := math.Acos(math.Max(-unity, math.Min(unity,
		direction.Dot(weather.Toward))))
	deck := weather.Deck * latitudeBell(away, nought, weather.DeckWidth)
	belt = latitudeBell(away, math.Pi/2, weather.TerminatorWidth)
	night := weather.Night * maths.Smooth(weather.NightFrom,
		weather.NightFrom+weather.NightWidth, away)
	far := weather.Far * maths.Smooth(weather.FarFrom,
		weather.FarFrom+weather.FarWidth, away)
	return deck + weather.Terminator*belt - night - far, belt
}

// Cover is the cover the weather adds at a direction, by its latitude: the
// belts of a sky that turns with its pole, or a locked sky's deck, belt and
// clearing. The second number is how far into a locked sky's terminator
// the place is, and nought where the sky is not locked.
func (weather CloudWeather) Cover(direction maths.Vec,
	latitude float64) (float64, float64) {
	if weather.Locked {
		return weather.Locking(direction)
	}
	return weather.Beaded(latitude, direction), nought
}

// Table is the weather as numbers for the kernel: the nineteen of Equator,
// Dry, Track, Width, DryAt, TrackAt, Stretch, Jet, Size, Turn, Core,
// Filament, Across, Cluster, Beads, Eye, EyeClear, Arms and ArmCover,
// then each storm's centre and sign, then twenty-one for a locked sky: one
// where it is locked and nought where it is not, Toward's three, Deck,
// DeckWidth, Terminator, TerminatorWidth, Night, NightFrom, NightWidth,
// Broken, TowerBase, TowerWidth, Towers, TowerFrequency, Far, FarFrom,
// FarWidth, TowerBillow and WhorlThin.
func (weather CloudWeather) Table() []float64 {
	table := make([]float64, 0, weatherNumbers+stormCount*stormNumbers+
		lockedNumbers)
	table = append(table, weather.Equator, weather.Dry, weather.Track,
		weather.Width, weather.DryAt, weather.TrackAt, weather.Stretch,
		weather.Jet, weather.Size, weather.Turn, weather.Core,
		weather.Filament, weather.Across, weather.Cluster, weather.Beads,
		weather.Eye, weather.EyeClear, weather.Arms, weather.ArmCover)
	for _, storm := range weather.Storms {
		table = append(table, storm.Centre.X, storm.Centre.Y, storm.Centre.Z,
			storm.Sign)
	}
	locked := float64(nought)
	if weather.Locked {
		locked = unity
	}
	return append(table, locked, weather.Toward.X, weather.Toward.Y,
		weather.Toward.Z, weather.Deck, weather.DeckWidth,
		weather.Terminator, weather.TerminatorWidth, weather.Night,
		weather.NightFrom, weather.NightWidth, weather.Broken,
		weather.TowerBase, weather.TowerWidth, weather.Towers,
		weather.TowerFrequency, weather.Far, weather.FarFrom,
		weather.FarWidth, weather.TowerBillow, weather.WhorlThin)
}

// Relief is the share of its height a cloud at a direction keeps under the
// whorls: one in a sky that turns, and in a locked sky one plus WhorlThin
// times what the arms and eyes add to the cover there, between whorlLowest
// and whorlHighest. An arm is a ridge of taller cloud and a lane lower
// cloud; the cover stays full, and a low sun throws shadow along them.
func (weather CloudWeather) Relief(direction maths.Vec) float64 {
	if !weather.Locked {
		return unity
	}
	bare := weather
	bare.Core = nought
	_, arms := bare.Swirl(direction)
	return math.Max(whorlLowest, math.Min(whorlHighest,
		unity+weather.WhorlThin*arms))
}

// Top is how much of the layer's height a cloud at a direction reaches, from
// nought to one, which a renderer multiplies the cloud's reach by. A sky
// that turns has every cloud as tall as its patch says, and it is one. A
// locked sky has convective towers: tallest at the substellar point, where
// the air rises, falling to TowerBase of that toward the terminator
// TowerWidth radians out, and from tower to tower by noise, the least of
// them Towers lower than the greatest (Yang, Cowan and Abbot 2013, from
// memory, to shelve, for the tallest cloud being at the substellar point).
// Finer noise, TowerBillow of it, billows the tops, so that a tower is
// cauliflower and not a hill.
func (weather CloudWeather) Top(direction maths.Vec) float64 {
	if !weather.Locked {
		return unity
	}
	away := math.Acos(math.Max(-unity, math.Min(unity,
		direction.Dot(weather.Toward))))
	tall := weather.TowerBase + (unity-weather.TowerBase)*
		latitudeBell(away, nought, weather.TowerWidth)
	fine := direction.Scale(weather.TowerFrequency)
	rise := maths.Smooth(towerFrom, towerTo, noise.Fractal(
		weather.Seed+towerStream, fine.X, fine.Y, fine.Z, towerOctaves))
	finer := fine.Scale(billowRise)
	billow := halfway + noise.Fractal(weather.Seed+towerStream+1, finer.X,
		finer.Y, finer.Z, billowOctaves)
	return math.Min(unity, tall*(unity-weather.Towers*(unity-rise))*
		(unity-weather.TowerBillow*(halfway-billow)))
}

// latitudeBell is one at the centre and gone a width away, in latitude.
func latitudeBell(latitude, centre, width float64) float64 {
	away := (latitude - centre) / width
	return math.Exp(-away * away)
}

// Belts is the cover added to the base at a latitude, in radians north or
// south: a thick belt at the equator where the trade winds meet, clear
// dry belts about twenty five degrees out, and storm tracks about fifty.
func (weather CloudWeather) Belts(latitude float64) float64 {
	reach := math.Abs(latitude)
	return weather.Equator*latitudeBell(reach, nought, weather.Width) -
		weather.Dry*latitudeBell(reach, weather.DryAt, weather.Width) +
		weather.Track*latitudeBell(reach, weather.TrackAt, weather.Width)
}

// Beaded is Belts with the equator's belt broken into clusters strung
// along it, by noise round the axis: where the noise is low the belt is
// gone, so the equator is not one clump.
func (weather CloudWeather) Beaded(latitude float64, direction maths.Vec) float64 {
	beads := weather.Beads
	round := noise.Fractal(weather.Seed+beadStream, direction.X*beads, 0,
		direction.Z*beads, beadOctaves)
	strung := math.Max(nought, unity+2*weather.Cluster*round)
	equator := weather.Equator * latitudeBell(math.Abs(latitude), nought,
		weather.Width)
	return weather.Belts(latitude) + equator*(strung-unity)
}

// Filaments is the grain added to the noise at a direction already
// slid and stretched by the wind: finer noise across the wind, so a
// streak is broken into filaments of varied width with clear air between.
func (weather CloudWeather) Filaments(read maths.Vec, frequency float64) float64 {
	across := frequency * weather.Across
	return weather.Filament * noise.Fractal(weather.Seed+filamentStream,
		read.X*frequency, read.Y*across, read.Z*frequency, filamentOctaves)
}

// Wind is the direction as the noise is to read it: slid round the axis by
// the winds of its latitude, then drawn out along the latitude line by the
// stretch, so clouds streak east and west and not in blobs.
func (weather CloudWeather) Wind(direction maths.Vec, latitude float64) maths.Vec {
	slide := -weather.Jet * math.Sin(windReversal*math.Abs(latitude))
	cos, sin := math.Cos(slide), math.Sin(slide)
	east := (direction.X*cos - direction.Z*sin) / weather.Stretch
	north := (direction.X*sin + direction.Z*cos) / weather.Stretch
	return maths.V(east, direction.Y, north)
}

// turn is the direction turned by the angle about the storm's centre,
// positive counter-clockwise seen from above the centre.
func (storm Storm) turn(direction maths.Vec, angle float64) maths.Vec {
	cos, sin := math.Cos(angle), math.Sin(angle)
	along := storm.Centre.Dot(direction) * (unity - cos)
	return direction.Scale(cos).Add(storm.Centre.Cross(direction).Scale(sin)).
		Add(storm.Centre.Scale(along))
}

// Swirl is where the noise is to be read for a direction once every storm
// has pulled the sky round its heart, the turn unwinding with distance,
// and how much cloud the storms add at the place. A storm shows the sky
// turned the way it turns: the noise read at the place it came from.
func (weather CloudWeather) Swirl(direction maths.Vec) (maths.Vec, float64) {
	var added float64
	for _, storm := range weather.Storms {
		away := math.Acos(math.Max(-unity, math.Min(unity,
			storm.Centre.Dot(direction)))) / (weather.Size *
			math.Abs(storm.Sign))
		if away > stormReach {
			continue
		}
		pull := math.Exp(-away * away)
		added += weather.heart(storm, direction, away, pull)
		direction = storm.turn(direction, -storm.Sign*weather.Turn*pull)
	}
	return direction, added
}

// heart is the cover a storm adds at a direction away radii from its
// centre, its pull there: more at the heart but for the clear eye, and
// arms banded by the turn itself, so that the bands wind as the sky does,
// fade outward, and fade in over the eye's width, where the azimuth they
// are banded by has no meaning and a pinch stood.
func (weather CloudWeather) heart(storm Storm, direction maths.Vec,
	away, pull float64) float64 {
	first := maths.YAxis.Cross(storm.Centre).Unit()
	second := storm.Centre.Cross(first)
	azimuth := math.Copysign(unity, storm.Sign) * math.Atan2(
		direction.Dot(second), direction.Dot(first))
	phase := weather.Arms * (azimuth - weather.Turn*pull)
	reach := away / armReach
	eye := away / weather.Eye
	if weather.Locked {
		return weather.Core*pull - weather.EyeClear*math.Exp(-eye*eye) +
			weather.ArmCover*math.Exp(-reach*reach)*math.Cos(phase)*
				maths.Smooth(nought, weather.Eye, away)
	}
	return weather.Core*pull - weather.EyeClear*math.Exp(-eye*eye) +
		weather.ArmCover*math.Exp(-reach*reach)*math.Cos(phase)
}
