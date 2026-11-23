// Package cities makes buildings, one at a time or a city at once, from
// a design: what a building is, said as separable design elements, not
// as geometry. A design comes from a set file, written by hand, or is
// drawn from a district's distributions by a seed. Both are made the
// same way after that.
//
// A design has masses, the closed volumes a building stands in, and
// placed elements, the things on it: a window, a cornice, a door, a
// roof's plant. Each is drawn on its own, so the kinds of building a
// city can hold are the product of its elements, not a list.
//
// Every element can say, at a detail level, all a renderer needs to
// draw it: a mesh, where this instance of it sits, and what it is made
// of. Two elements that share a mesh return the same one, so a window
// placed a thousand times is one mesh placed a thousand times.
//
// The package knows nothing of the manufactory. The manufactory takes
// what an element renders and makes its own parts from it; that is the
// only place a material becomes a slot. Every number a design chooses
// comes from data with a name; the package holds no palette, no family
// list and no bare number that is a choice.
package cities
