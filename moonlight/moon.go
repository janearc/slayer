package moonlight

import "github.com/janearc/celestial/measure"

// The moon's volumetric mean radius in m, its orbit's semimajor axis in
// m and its geometric albedo: nasa's moon fact sheet (sources.yaml's
// nasa-moon-fact-sheet), 1737.4 km, 0.3844 million km and 0.12. Here
// until celestial holds the moon.
const (
	moonRadiusMetres   = 1737.4e3
	moonDistanceMetres = 0.3844e9
	moonAlbedo         = 0.12
)

// Moon is the moon as nasa's fact sheet gives it: its radius, its mean
// distance and its geometric albedo.
func Moon() Satellite {
	return Satellite{radius: measure.Metres(moonRadiusMetres),
		distance: measure.Metres(moonDistanceMetres),
		albedo:   moonAlbedo}
}

// Satellite is a moon's measures: how big, how far, how bright.
type Satellite struct {
	radius, distance measure.Length
	albedo           float64
}

// Radius is the moon's mean radius.
func (moon Satellite) Radius() measure.Length { return moon.radius }

// Distance is the moon's mean distance, its orbit's semimajor axis.
func (moon Satellite) Distance() measure.Length { return moon.distance }

// Albedo is the moon's geometric albedo: how bright it is full, against
// a white disc as big that sends all its light back.
func (moon Satellite) Albedo() float64 { return moon.albedo }
