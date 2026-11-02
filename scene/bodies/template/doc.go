// Package template provides named bodies: planets and moons, Pluto and Charon,
// the Sun, a quasar, pulsar, magnetar, hot Jupiter, and two comets. Each
// function returns a fresh bodies.Body that callers can customise.
//
// Templates use one file per body. Their physical values were entered from
// memory and remain unverified. The 2026-10-18 review also relied on memory, so
// it did not establish source verification. See scene/bodies for the recording
// policy. Moon phases are composition choices, not ephemerides.
package template
