package film

// IdentifiedSensor optionally identifies all state that affects its rays.
// Equal nonempty keys promise identical rays for equal arguments. Changing
// aperture, projection, focus or other ray state must change the key.
// An empty key disables caching; ordinary Sensor implementations are uncached.
type IdentifiedSensor interface {
	Sensor
	RayKey() string
}
