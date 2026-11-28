// Package generate makes bodies from a handful of numbers rather than by
// name: a planet like Earth or like Jupiter, not Earth or Jupiter. Its
// design is doc/system-generators-design.md.
//
// A Planet is what a caller gives: mass, what it is made of, how it turns,
// where it goes round its star, how old it is, and a seed. Derive works
// out what follows from those numbers, and Make hangs the parts on a
// bodies.Body. The named templates in scene/bodies/template are the check:
// given Earth's numbers, Make should come out near the Earth template.
//
// Every relation Derive uses is cited in the code, from a source on the
// shelf, before it is relied on.
package generate
