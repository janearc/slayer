// Package airs is the air of a planet as the solver of light in air
// takes it: shells from the ground up, each with the gas's scattering
// and absorption and the particles' extinction, albedo and asymmetry,
// every one per metre, at one wavelength; and the ground's albedo.
//
// Venus is the first air and Earth the second. Everything each is made
// of is in venus.yaml and earth.yaml, and every number there says
// whether it was read from a source, and where, or picked, and why.
//
// Venus's gas is worked out by Rayleigh's cross section for carbon
// dioxide and nitrogen, and its cloud and hazes by Mie's scattering
// (package mie) of the sulphuric acid droplets Titov and others 2018
// gather from the probes.
//
// The cloud's optical depth is picked at 630 nm, where the sources give
// it, and the same scale is held at every other wavelength, so the way
// the cloud changes with colour is Mie's and not a pick.
//
// Earth's gas is celestial's 1976 standard atmosphere, by height, with
// celestial's Rayleigh cross section for dry air. Its ozone, which
// absorbs in the Chappuis band and keeps the twilight zenith blue, is
// the Nimbus-7 TOMS standard profile with the cross sections of
// Serdyuchenko and others 2014.
//
// Earth's aerosol is a clear day's haze: an exponential layer from the
// ground, with the albedo and asymmetry of Levy and others 2007's
// continental model. Earth works from 300 to 1000 nm, the range of the
// ozone's table.
//
// Sun reads the sun's spectrum. Band gives it at one astronomical unit,
// which is Earth's, and AtVenus puts it at the distance of Venus.
package airs
