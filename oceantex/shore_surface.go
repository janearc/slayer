package oceantex

import (
	"errors"
)

// ShoreSurface is an immutable snapshot of the existing shore model and
// wave cascades at one time. It provides height and instantaneous crest foam
// for use by 3D scenes. It does not model overturning or foam persistence.
type ShoreSurface struct {
	waves *surface
}

// NewShoreSurface prepares a copy of shore for the sea at seconds. Wind and
// fetch must be positive so that the breaking model has a nonzero wave RMS.
func NewShoreSurface(shore *Shore, sea Sea, seconds float64, seed uint64) (*ShoreSurface, error) {
	if shore == nil || !finite(sea.Wind) || !finite(sea.Fetch) ||
		!finite(sea.Depth) || !finite(sea.Toward) || sea.Wind <= 0 ||
		sea.Fetch <= 0 || sea.Depth < 0 {
		return nil, errors.New("oceantex: shore surface needs a shore and finite sea with positive wind and fetch")
	}
	if shore.across < 2 || !finite(shore.extent) || shore.extent <= 0 ||
		!finite(shore.step) || shore.step <= 0 ||
		len(shore.height)/shore.across != shore.across || len(shore.height)%shore.across != 0 {
		return nil, errors.New("oceantex: shore surface needs a constructed shore grid")
	}
	cascades, err := WaveCascades(sea, seconds, seed)
	if err != nil {
		return nil, err
	}
	copyOfShore := *shore
	copyOfShore.meet(sea)
	if !finite(copyOfShore.rms) || copyOfShore.rms <= 0 {
		return nil, errors.New("oceantex: shore surface needs finite positive arriving wave height")
	}
	made := &surface{shore: &copyOfShore}
	for _, cascade := range cascades {
		made.layers = append(made.layers, layer{
			field: cascade.Field,
			texel: cascade.Field.Extent / float64(cascadeGrid),
		})
	}
	return &ShoreSurface{waves: made}, nil
}

// Sample returns the displaced surface height and instantaneous white foam
// fraction at east/north coordinates in metres. The boolean is false outside
// the supplied shore or at invalid coordinates.
func (snapshot *ShoreSurface) Sample(east, north float64) (height, foam float64, ok bool) {
	if snapshot == nil || snapshot.waves == nil || !finite(east) || !finite(north) ||
		!snapshot.waves.shore.holds(east, north) {
		return 0, 0, false
	}
	shore := snapshot.waves.shore
	open := snapshot.waves.arriving(east, north, 0, 3)
	height = open * shore.read(shore.gain, east, north)
	foam = shore.foam(east, north, open)
	if !finite(height) || !finite(foam) {
		return 0, 0, false
	}
	return height, foam, true
}

// Wet reports whether the still-water shore grid places this point below
// mean sea level. It is false outside the grid and at invalid coordinates.
// It does not model a moving swash line or foam temporarily stranded on land.
func (snapshot *ShoreSurface) Wet(east, north float64) bool {
	return snapshot != nil && snapshot.waves != nil && finite(east) && finite(north) &&
		snapshot.waves.shore.holds(east, north) && snapshot.waves.shore.Height(east, north) < 0
}
