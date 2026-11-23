// Package mocks holds the mocks the tests of slayer's tracer make of what
// it reads from its scene packages, one package for each, made by uber's
// mockgen from generate.sh.
//
// They are for tests that need a part of those to answer what no fake
// would: a NaN, an edge, a call the tracer must not make.
package mocks

//go:generate sh generate.sh
