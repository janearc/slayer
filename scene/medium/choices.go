package medium

import (
	_ "embed"
	"fmt"

	"github.com/janearc/puzzlemouth/numbers"
)

//go:embed choices.yaml
var choicesRaw []byte

// shapeChoices holds the numbers the media's shapes are drawn with, which are
// chosen for the picture, after Puzzlemouth reads them. Density queries do
// not parse data or search the numeric catalogue.
type shapeChoices struct {
	skirtWidths, streamBell, streamEighth       float64
	jetLowerSide, jetKnotSpacing, jetKnotWeight float64
	alongTheAxis                                float64
	jetKnotOctaves, probesLeast, probesMost     int

	cloudFrequency, cloudEdge, cloudWhole, cloudTop float64
	cloudOctaves                                    int
	cloudBillow, cloudDetail, cloudReach            float64
	cloudDetailOctaves                              int

	cloudEquator, cloudDry, cloudTrack, cloudWidth float64
	cloudDryAt, cloudTrackAt, cloudStretch         float64
	cloudJet, cloudWhorlSize, cloudWhorlTurn       float64
	cloudWhorlCore                                 float64

	cloudFilament, cloudAcross, cloudCluster, cloudBeads float64
	cloudEye, cloudEyeClear, cloudArms, cloudArmCover    float64

	cloudDeck, cloudDeckWidth, cloudTerminator float64
	cloudTerminatorWidth, cloudNight           float64
	cloudNightFrom, cloudNightWidth            float64
	cloudBroken, cloudLockedStorm              float64
	cloudLockedStretch, cloudLockedFilament    float64
	cloudLockedJet, cloudLockedStrength        float64
	cloudTowerBase, cloudTowerWidth            float64
	cloudTowers, cloudTowerFrequency           float64
	cloudFar, cloudFarFrom, cloudFarWidth      float64
	cloudTowerBillow, cloudLockedEyeClear      float64
	cloudLockedArmCover, cloudLockedTurn       float64
	cloudLockedWhorlThin, cloudLockedCore      float64
	cloudLockedSize                            float64

	auroraFloor, auroraFall, auroraSoft, auroraRays float64
	auroraFolds, auroraStretch, auroraRayWeight     float64
	auroraOctaves                                   int
	auroraFoldSeed                                  uint64

	plumeCore, plumeThin, plumeSkin, plumePeak float64
	plumeFoot, plumeRise, plumeSkirt           float64

	fizzGrain, fizzSpread          float64
	fizzOctaves, wispyOctaves      int
	wakeFade, wakeSwept, wakeRidge float64
	wakeBank, wakeLift             float64
	exponentMost, clearCloudWidth  float64
	wavelengthRed, wavelengthGreen float64
	wavelengthBlue                 float64
}

// chosen is the shapes' choices, read once when the package starts.
var chosen = readShapeChoices()

// choiceFile is choices.yaml, read once when the package starts. A package
// variable cannot return an error, so a file or row the loader refuses stops
// the program at start with words that name the file and the row.
var choiceFile = readChoiceFile()

// readChoiceFile reads the embedded file; every row it refuses is named.
func readChoiceFile() numbers.Numbers {
	read, err := numbers.Read("choices.yaml", choicesRaw)
	if err != nil {
		panic(fmt.Errorf("medium: choices.yaml refused, row by row: %w", err))
	}
	return read
}

// choice is the named choice. A name the file does not hold is refused in
// words that name the row.
func choice(name string) float64 {
	found, err := choiceFile.Get(name)
	if err != nil {
		panic(fmt.Errorf("medium: choices.yaml has no usable row %q: %w",
			name, err))
	}
	return found.Value
}

// choiceSeed is the named choice, which is a whole number added to a seed.
func choiceSeed(name string) uint64 { return uint64(choice(name)) }

// choiceCount is the named choice, which is a whole number.
func choiceCount(name string) int { return int(choice(name)) }

// readShapeChoices reads the shapes' choices.
func readShapeChoices() shapeChoices {
	pick, count := choice, choiceCount
	return shapeChoices{
		skirtWidths: pick("skirt-widths"), streamBell: pick("stream-bell"),
		streamEighth: pick("stream-eighth"), jetLowerSide: pick("jet-lower-side"),
		jetKnotSpacing: pick("jet-knot-spacing"),
		jetKnotWeight:  pick("jet-knot-weight"),
		alongTheAxis:   pick("along-the-axis"),
		jetKnotOctaves: count("jet-knot-octaves"),
		probesLeast:    count("probes-least"), probesMost: count("probes-most"),
		cloudFrequency: pick("cloud-frequency"), cloudEdge: pick("cloud-edge"),
		cloudWhole: pick("cloud-whole"), cloudTop: pick("cloud-top"),
		cloudOctaves: count("cloud-octaves"),
		cloudBillow:  pick("cloud-billow"), cloudDetail: pick("cloud-detail"),
		cloudReach:         pick("cloud-reach"),
		cloudDetailOctaves: count("cloud-detail-octaves"),
		cloudEquator:       pick("cloud-equator"), cloudDry: pick("cloud-dry"),
		cloudTrack: pick("cloud-track"), cloudWidth: pick("cloud-width"),
		cloudDryAt: pick("cloud-dry-at"), cloudTrackAt: pick("cloud-track-at"),
		cloudStretch: pick("cloud-stretch"), cloudJet: pick("cloud-jet"),
		cloudWhorlSize: pick("cloud-whorl-size"),
		cloudWhorlTurn: pick("cloud-whorl-turn"),
		cloudWhorlCore: pick("cloud-whorl-core"),
		cloudFilament:  pick("cloud-filament"),
		cloudAcross:    pick("cloud-filament-across"),
		cloudCluster:   pick("cloud-cluster"), cloudBeads: pick("cloud-beads"),
		cloudEye:      pick("cloud-eye"),
		cloudEyeClear: pick("cloud-eye-clear"),
		cloudArms:     pick("cloud-arms"),
		cloudArmCover: pick("cloud-arm-cover"),
		cloudDeck:     pick("cloud-deck"), cloudDeckWidth: pick("cloud-deck-width"),
		cloudTerminator:      pick("cloud-terminator"),
		cloudTerminatorWidth: pick("cloud-terminator-width"),
		cloudNight:           pick("cloud-night"),
		cloudNightFrom:       pick("cloud-night-from"),
		cloudNightWidth:      pick("cloud-night-width"),
		cloudBroken:          pick("cloud-broken"),
		cloudLockedStorm:     pick("cloud-locked-storm"),
		cloudLockedStretch:   pick("cloud-locked-stretch"),
		cloudLockedFilament:  pick("cloud-locked-filament"),
		cloudLockedJet:       pick("cloud-locked-jet"),
		cloudLockedStrength:  pick("cloud-locked-strength"),
		cloudTowerBase:       pick("cloud-tower-base"),
		cloudTowerWidth:      pick("cloud-tower-width"),
		cloudTowers:          pick("cloud-towers"),
		cloudTowerFrequency:  pick("cloud-tower-frequency"),
		cloudFar:             pick("cloud-far"),
		cloudFarFrom:         pick("cloud-far-from"),
		cloudFarWidth:        pick("cloud-far-width"),
		cloudTowerBillow:     pick("cloud-tower-billow"),
		cloudLockedEyeClear:  pick("cloud-locked-eye-clear"),
		cloudLockedArmCover:  pick("cloud-locked-arm-cover"),
		cloudLockedTurn:      pick("cloud-locked-turn"),
		cloudLockedCore:      pick("cloud-locked-core"),
		cloudLockedSize:      pick("cloud-locked-size"),
		cloudLockedWhorlThin: pick("cloud-locked-whorl-thin"),
		auroraFloor:          pick("aurora-floor"), auroraFall: pick("aurora-fall"),
		auroraSoft: pick("aurora-soft"), auroraRays: pick("aurora-rays"),
		auroraFolds: pick("aurora-folds"), auroraStretch: pick("aurora-stretch"),
		auroraRayWeight: pick("aurora-ray-weight"),
		auroraOctaves:   count("aurora-octaves"),
		auroraFoldSeed:  uint64(count("aurora-fold-seed")),
		plumeCore:       pick("plume-core"), plumeThin: pick("plume-thin"),
		plumeSkin: pick("plume-skin"), plumePeak: pick("plume-peak"),
		plumeFoot: pick("plume-foot"), plumeRise: pick("plume-rise"),
		plumeSkirt: pick("plume-skirt"), fizzGrain: pick("fizz-grain"),
		fizzSpread:  pick("fizz-spread"),
		fizzOctaves: count("fizz-octaves"), wispyOctaves: count("wispy-octaves"),
		wakeFade: pick("wake-fade"), wakeSwept: pick("wake-swept"),
		wakeRidge: pick("wake-ridge"), wakeBank: pick("wake-bank"),
		wakeLift: pick("wake-lift"), exponentMost: pick("exponent-most"),
		clearCloudWidth: pick("clear-cloud-width"),
		wavelengthRed:   pick("wavelength-red"),
		wavelengthGreen: pick("wavelength-green"),
		wavelengthBlue:  pick("wavelength-blue"),
	}
}
