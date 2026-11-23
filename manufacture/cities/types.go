package cities

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// Detail is how much of an element is drawn: a hull from far away, the
// street's detail up close. Its names and their order are data.
type Detail string

// Triangle is one face of a mesh, wound outward.
type Triangle [3]maths.Vec

// Mesh is a closed set of triangles in its own space. Meshes are shared
// by pointer: two elements with the same *Mesh are the same geometry.
type Mesh struct {
	Name      string
	Triangles []Triangle
}

// Rendered is all a renderer needs for one element: its mesh, where this
// instance of the mesh sits, and its material, resolved.
type Rendered struct {
	Mesh     *Mesh
	Place    maths.Subspace
	Material *thing.Material
	// Lamp, if given, says the part is also a light: it shines from the
	// middle of the part's mesh. The tracer lights by lamps, not by what
	// glows, so a room is lit only if its lamp is one.
	Lamp *Lamp
}

// Lamp is a light a part carries: the heat of its colour in kelvin and its
// luminance at a metre in the kit's relative light, the unit a set's lamp
// is given in, shining every way.
//
// A Tint, if it is not nought, is the colour of the light instead, as
// amounts of light from the three srgb lamps, in linear light: a neon
// tube's glass has a colour that no heat of a black body has. The heat is
// then not read.
//
// Beyond is how far in metres from the middle of the part the light starts,
// so that nothing nearer than that shades what it lights: nought for the
// manufactory's own distance, and more than the radius of a thin shell that
// the light is inside and passes.
type Lamp struct {
	Heat, Luminance, Beyond float64
	Tint                    maths.Light
}

// designElement is the irreducible thing in a city.
type designElement interface {
	Rendered(level Detail) (Rendered, error)
}

// partRole is what a part of a building is for: wall, glazing, trim, roof.
// A role chooses a material from data; it is never a number.
type partRole string

// mass is one closed volume of a building: a podium, a storey, a crown.
type mass struct {
	Name  string
	Form  form
	Place maths.Subspace
	Role  partRole
}

// form is a closed solid that can mesh itself at a detail level.
type form interface {
	Mesh(level Detail) (*Mesh, error)
}

// placedElement is one design element at one place on a building.
type placedElement struct {
	Element designElement
	Place   maths.Subspace
}

// Design is a building as separable design elements, not yet geometry.
type Design struct {
	Name     string
	Massing  []mass
	Elements []placedElement
	// Mounts are places where the design keeps a thing the manufactory
	// makes, which this package knows nothing of.
	Mounts []Mount
}

// Mount is a place for a thing of a kind in the manufactory's catalogue,
// a telescope on a roof: the kind's name, where it stands in the design's
// frame, and how much of its usual size it is drawn at.
type Mount struct {
	Kind  string
	Place maths.Subspace
}

// Lot is the ground a drawn building stands on.
type Lot struct {
	Place        maths.Subspace
	Width, Depth float64
}

// District is the distributions a drawn building is chosen from: the
// masses, elements and materials it may have, and how likely each is.
type District struct {
	Name string
}

// Seed decides every draw a building's design makes, so the same seed at
// the same place always makes the same building.
type Seed uint64
