// Package mocks holds the mocks of everything slayer's tracer reads from
// its scene and film packages, one package for each, made by uber's
// mockgen from generate.sh.
//
// They are for tests that need a part of those to answer what no fake
// would: a NaN, an edge, a call the tracer must not make.
package mocks

//go:generate sh generate.sh
