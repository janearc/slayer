package manufacture

import (
	"fmt"
	"math"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// ComponentForm names one reusable closed shape shared by building and
// spacecraft recipes.
type ComponentForm string

const (
	ComponentBox      ComponentForm = "box"
	ComponentDrum     ComponentForm = "drum"
	ComponentDome     ComponentForm = "dome"
	ComponentSlitDome ComponentForm = "slit-dome"
	ComponentRod      ComponentForm = "rod"
	ComponentDish     ComponentForm = "dish"
)

// BuildingForm is retained as the architectural name for ComponentForm.
type BuildingForm = ComponentForm

const (
	BuildingBox      = ComponentBox
	BuildingDrum     = ComponentDrum
	BuildingDome     = ComponentDome
	BuildingSlitDome = ComponentSlitDome
)

// ComponentSurface is a named material role local to a recipe. Buildings use
// walls/roof/accent; spacecraft can name housing/optics/solar/foil roles.
type ComponentSurface string

// BuildingSurface is retained as the architectural name for ComponentSurface.
type BuildingSurface = ComponentSurface

const (
	BuildingWalls  BuildingSurface = "walls"
	BuildingRoof   BuildingSurface = "roof"
	BuildingAccent BuildingSurface = "accent"
)

// ComponentMaterial gives one named role its response. Look optionally keeps
// the complete scene material; otherwise the plain colour/finish fields apply.
type ComponentMaterial struct {
	Surface ComponentSurface
	Name    string
	Colour  maths.Light
	Gloss   float64
	Index   float64
	Look    *thing.Material
}

// BuildingMaterial retains the architectural name for ComponentMaterial.
type BuildingMaterial = ComponentMaterial

// ComponentProfilePoint is a radius and height in a revolved component
// profile, measured from its local Y axis and bottom.
type ComponentProfilePoint struct {
	Radius, Height measure.Length
}

// ComponentSpec is one ordered closed component. Box position is its minimum
// corner; upright round shapes use a bottom centre. Frame places the local
// component in a parent space (zero is identity).
//
// Drum Profile can preserve an authored radial wall; Smooth enables round
// normals on drums and dishes. Cuts overrides the current detail for this
// part; DomeCutsPerLevel refines a dome profile. Rod endpoints are in local
// coordinates. Required fields and form-specific restrictions are documented
// in [building.md].
type ComponentSpec struct {
	Form             ComponentForm
	Position         maths.Vec
	Frame            maths.Subspace
	Width            measure.Length
	Height           measure.Length
	Depth            measure.Length
	Radius           measure.Length
	InnerRadius      measure.Length
	Thickness        measure.Length
	Opening          measure.Angle
	DomeCutsPerLevel int
	Cuts             int
	Profile          []ComponentProfilePoint
	From, To         maths.Vec
	Smooth           bool
	Segmented        bool
	Surface          ComponentSurface
	exact            *componentMetrics
}

// componentMetrics keeps existing float-authored templates bit-exact while
// their public descriptor carries physical units.
type componentMetrics struct {
	width, height, depth, radius, innerRadius, thickness float64
	profile                                              [][coordinatePair]float64
}

// BuildingPart retains the architectural name for ComponentSpec.
type BuildingPart = ComponentSpec

// BuildingSpec describes a finite, upright building in metres. Bounds are
// centered on the X/Z origin, extend from ground level to Height, and must
// contain every listed component. Parts are emitted in slice order.
type BuildingSpec struct {
	Name                 string
	Width, Depth, Height measure.Length
	Materials            []BuildingMaterial
	Parts                []ComponentSpec
}

// SpacecraftSpec assembles a bounded spacecraft from the same components as
// BuildingSpec. It has no randomized choices; its seed only satisfies the
// ordinary factory interface.
type SpacecraftSpec struct {
	Name                 string
	Width, Depth, Height measure.Length
	Materials            []ComponentMaterial
	Parts                []ComponentSpec
}

const maximumBuildingParts = 512
const maximumBuildingMetres = 10_000

const (
	buildingWallSlot slot = iota
	buildingRoofSlot
	buildingAccentSlot
)

// Building checks a reusable component description and returns an ordinary
// factory kind. Its seed is accepted for factory compatibility; this explicit
// spec has no random choices. The input slices are copied before retention.
func Building(spec BuildingSpec) (Kind, error) {
	if err := spec.check(); err != nil {
		return kind{}, err
	}
	spec.Materials = cloneComponentMaterials(spec.Materials)
	spec.Parts = cloneComponentParts(spec.Parts)
	return kind{name: spec.Name, how: buildingMaker{spec}}, nil
}

// check reports an error when the building materials, envelope or parts
// are invalid.
func (spec BuildingSpec) check() error {
	materials, err := checkComponentMaterials(spec.Materials, true)
	if err != nil {
		return err
	}
	return checkComponentEnvelope(spec.Name, spec.Width, spec.Depth, spec.Height,
		spec.Parts, materials, true)
}

// SpacecraftFrom checks a user-composed spacecraft and returns an ordinary
// factory kind. Existing named spacecraft remain available through Spacecraft.
func SpacecraftFrom(spec SpacecraftSpec) (*Kind, error) {
	materials, err := checkComponentMaterials(spec.Materials, false)
	if err != nil {
		return nil, err
	}
	if err := checkComponentEnvelope(spec.Name, spec.Width, spec.Depth,
		spec.Height, spec.Parts, materials, false); err != nil {
		return nil, err
	}
	spec.Materials = cloneComponentMaterials(spec.Materials)
	spec.Parts = cloneComponentParts(spec.Parts)
	return &kind{name: spec.Name, how: spacecraftSpecMaker{spec}}, nil
}

// cloneComponentMaterials copies the materials and their looks, so the copy
// shares no look with the caller.
func cloneComponentMaterials(materials []ComponentMaterial) []ComponentMaterial {
	cloned := append([]ComponentMaterial(nil), materials...)
	for index := range cloned {
		if cloned[index].Look != nil {
			look := *cloned[index].Look
			cloned[index].Look = &look
		}
	}
	return cloned
}

// cloneComponentParts copies the parts and their profiles, so the copy shares
// no profile with the caller.
func cloneComponentParts(parts []ComponentSpec) []ComponentSpec {
	cloned := append([]ComponentSpec(nil), parts...)
	for index := range cloned {
		cloned[index].Profile = append([]ComponentProfilePoint(nil), parts[index].Profile...)
		if parts[index].exact != nil {
			exact := *parts[index].exact
			exact.profile = append([][coordinatePair]float64(nil), parts[index].exact.profile...)
			cloned[index].exact = &exact
		}
	}
	return cloned
}

// checkComponentMaterials checks the count, surface and response of each
// material, and returns the surfaces they declare.
func checkComponentMaterials(materials []ComponentMaterial, building bool) (map[ComponentSurface]bool, error) {
	if len(materials) == 0 || len(materials) > maximumBuildingParts {
		return nil, fmt.Errorf("manufacture: component recipe needs between one and %d materials", maximumBuildingParts)
	}
	known := make(map[ComponentSurface]bool, len(materials))
	for _, material := range materials {
		valid := material.Surface != ""
		if building {
			valid = material.Surface == BuildingWalls || material.Surface == BuildingRoof || material.Surface == BuildingAccent
		}
		if !valid || known[material.Surface] {
			return nil, fmt.Errorf("manufacture: component recipe has an unknown or repeated material role %q", material.Surface)
		}
		name := material.Name
		if material.Look != nil && name == "" {
			name = material.Look.Name
		}
		response := material.Colour
		gloss, index := material.Gloss, material.Index
		if material.Look != nil {
			response = material.Look.Colour
			gloss, index = material.Look.Gloss, material.Look.Index
		}
		if name == "" || !finite(response.R) || !finite(response.G) || !finite(response.B) ||
			response.R < 0 || response.R > 1 || response.G < 0 || response.G > 1 || response.B < 0 || response.B > 1 ||
			!finite(gloss) || gloss < 0 || gloss > 1 || !finite(index) || index < 0 {
			return nil, fmt.Errorf("manufacture: component material %q has invalid response values", name)
		}
		known[material.Surface] = true
	}
	return known, nil
}

// checkComponentEnvelope checks the envelope and the parts. Each part needs a
// declared material and must lie inside the envelope. A building part must be
// upright in the building frame.
func checkComponentEnvelope(name string, widthMeasure, depthMeasure, heightMeasure measure.Length,
	parts []ComponentSpec, materials map[ComponentSurface]bool, building bool,
) error {
	if name == "" {
		return fmt.Errorf("manufacture: component recipe needs a name")
	}
	width, depth, height := widthMeasure.InMetres(), depthMeasure.InMetres(), heightMeasure.InMetres()
	for _, dimension := range []float64{width, depth, height} {
		if !positive(dimension) || dimension > maximumBuildingMetres {
			return fmt.Errorf("manufacture: component envelope needs finite positive metres within the recipe budget")
		}
	}
	if len(parts) == 0 || len(parts) > maximumBuildingParts {
		return fmt.Errorf("manufacture: component recipe needs between one and %d parts", maximumBuildingParts)
	}
	envelope := maths.Box{Min: maths.V(-width/2, 0, -depth/2), Max: maths.V(width/2, height, depth/2)}
	for _, part := range parts {
		if !materials[part.Surface] {
			return fmt.Errorf("manufacture: component has no material for role %q", part.Surface)
		}
		if building && part.Frame != maths.Local() {
			return fmt.Errorf("manufacture: building components must be upright in the building frame")
		}
		bounds, err := part.bounds()
		if err != nil {
			return err
		}
		if bounds.Min.X < envelope.Min.X || bounds.Max.X > envelope.Max.X ||
			bounds.Min.Y < envelope.Min.Y || bounds.Max.Y > envelope.Max.Y ||
			bounds.Min.Z < envelope.Min.Z || bounds.Max.Z > envelope.Max.Z {
			return fmt.Errorf("manufacture: %s component bounds %+v lie outside envelope %+v", part.Form, bounds, envelope)
		}
	}
	return nil
}

// bounds returns the box a part occupies in its parent space. It rejects an
// invalid placement, dimension or profile.
func (part ComponentSpec) bounds() (maths.Box, error) {
	zeroField := func(value measure.Length) bool { return value.InMetres() == 0 }
	if !shape.ValidPlacement(part.Frame) || !finite(part.Position.X) || !finite(part.Position.Y) || !finite(part.Position.Z) {
		return maths.Box{}, fmt.Errorf("manufacture: component placement must be finite and valid")
	}
	if part.Cuts < 0 || part.Cuts > 4096 {
		return maths.Box{}, fmt.Errorf("manufacture: component cut override is outside its budget")
	}
	var local maths.Box
	switch part.Form {
	case ComponentBox:
		w, wok := part.widthMetres(), positive(part.widthMetres()) && part.widthMetres() <= maximumBuildingMetres
		h, hok := part.heightMetres(), positive(part.heightMetres()) && part.heightMetres() <= maximumBuildingMetres
		d, dok := part.depthMetres(), positive(part.depthMetres()) && part.depthMetres() <= maximumBuildingMetres
		if !wok || !hok || !dok || !zeroField(part.Radius) || !zeroField(part.InnerRadius) || !zeroField(part.Thickness) || part.Opening.InRadians() != 0 || part.DomeCutsPerLevel != 0 || part.Smooth || part.Segmented || len(part.Profile) != 0 || !zeroVec(part.From) || !zeroVec(part.To) {
			return maths.Box{}, fmt.Errorf("manufacture: component box needs only positive width, height and depth")
		}
		local = maths.Box{Min: maths.Vec{}, Max: maths.V(w, h, d)}
	case ComponentDrum:
		r, rok := part.radiusMetres(), positive(part.radiusMetres()) && part.radiusMetres() <= maximumBuildingMetres
		i := part.innerRadiusMetres()
		iok := finite(i) && i >= 0 && i <= maximumBuildingMetres
		h, hok := part.heightMetres(), positive(part.heightMetres()) && part.heightMetres() <= maximumBuildingMetres
		if !rok || !iok || !hok || i >= r || !zeroField(part.Width) || !zeroField(part.Depth) || !zeroField(part.Thickness) || part.Opening.InRadians() != 0 || part.DomeCutsPerLevel != 0 || part.Segmented || !zeroVec(part.From) || !zeroVec(part.To) {
			return maths.Box{}, fmt.Errorf("manufacture: component drum needs outer radius above inner radius and positive height")
		}
		local = maths.Box{Min: maths.V(-r, 0, -r), Max: maths.V(r, h, r)}
		if len(part.Profile) > 0 || part.exact != nil && len(part.exact.profile) > 0 {
			if i != 0 {
				return maths.Box{}, fmt.Errorf("manufacture: profiled drum cannot also set inner radius")
			}
			low, high := math.Inf(1), math.Inf(-1)
			for _, point := range part.profileMetres() {
				pr, py := point[0], point[1]
				if !finite(pr) || pr < 0 || pr > r || !finite(py) || py < 0 || py > h {
					return maths.Box{}, fmt.Errorf("manufacture: drum profile point lies outside its declared radius and height")
				}
				low, high = min(low, py), math.Max(high, py)
			}
			if len(part.profileMetres()) < 3 || low != 0 || high != h {
				return maths.Box{}, fmt.Errorf("manufacture: drum profile needs three points spanning its height")
			}
		}
	case ComponentDome:
		r, rok := part.radiusMetres(), positive(part.radiusMetres()) && part.radiusMetres() <= maximumBuildingMetres
		h, hok := part.heightMetres(), positive(part.heightMetres()) && part.heightMetres() <= maximumBuildingMetres
		if !rok || !hok || part.DomeCutsPerLevel < 1 || part.DomeCutsPerLevel > 512 || !zeroField(part.Width) || !zeroField(part.Depth) || !zeroField(part.InnerRadius) || !zeroField(part.Thickness) || part.Opening.InRadians() != 0 || part.Smooth || part.Segmented || len(part.Profile) != 0 || !zeroVec(part.From) || !zeroVec(part.To) {
			return maths.Box{}, fmt.Errorf("manufacture: component dome needs positive radius, height and cut spacing")
		}
		local = maths.Box{Min: maths.V(-r, 0, -r), Max: maths.V(r, h, r)}
	case ComponentSlitDome:
		r, rok := part.radiusMetres(), positive(part.radiusMetres()) && part.radiusMetres() <= maximumBuildingMetres
		t, tok := part.thicknessMetres(), positive(part.thicknessMetres()) && part.thicknessMetres() <= maximumBuildingMetres
		opening := part.Opening.InRadians()
		if !rok || !tok || t >= r || !zeroField(part.Width) || !zeroField(part.Height) || !zeroField(part.Depth) || !zeroField(part.InnerRadius) ||
			!finite(opening) || opening <= 0 || opening >= 2*math.Pi || part.DomeCutsPerLevel != 0 || part.Smooth || part.Segmented || len(part.Profile) != 0 || !zeroVec(part.From) || !zeroVec(part.To) {
			return maths.Box{}, fmt.Errorf("manufacture: slit dome needs a positive radius, thinner wall and opening below a full turn")
		}
		local = maths.Box{Min: maths.V(-r, 0, -r), Max: maths.V(r, r, r)}
	case ComponentRod:
		r, rok := part.radiusMetres(), positive(part.radiusMetres()) && part.radiusMetres() <= maximumBuildingMetres
		if !rok || !zeroField(part.Width) || !zeroField(part.Height) || !zeroField(part.Depth) || !zeroField(part.InnerRadius) || !zeroField(part.Thickness) || part.Opening.InRadians() != 0 || part.DomeCutsPerLevel != 0 || part.Smooth || part.Segmented || len(part.Profile) != 0 || part.From == part.To {
			return maths.Box{}, fmt.Errorf("manufacture: component rod needs distinct endpoints and a positive radius")
		}
		if !finite(part.From.Len()) || !finite(part.To.Len()) {
			return maths.Box{}, fmt.Errorf("manufacture: component rod endpoints must be finite")
		}
		local = maths.Box{Min: maths.V(min(part.From.X, part.To.X)-r, min(part.From.Y, part.To.Y)-r, min(part.From.Z, part.To.Z)-r),
			Max: maths.V(math.Max(part.From.X, part.To.X)+r, math.Max(part.From.Y, part.To.Y)+r, math.Max(part.From.Z, part.To.Z)+r)}
	case ComponentDish:
		r, rok := part.radiusMetres(), positive(part.radiusMetres()) && part.radiusMetres() <= maximumBuildingMetres
		d, dok := part.depthMetres(), positive(part.depthMetres()) && part.depthMetres() <= maximumBuildingMetres
		if !rok || !dok || !zeroField(part.Width) || !zeroField(part.Height) || !zeroField(part.InnerRadius) || !zeroField(part.Thickness) || part.Opening.InRadians() != 0 || part.DomeCutsPerLevel != 0 || len(part.Profile) != 0 || !zeroVec(part.From) || !zeroVec(part.To) || part.Smooth && part.Segmented {
			return maths.Box{}, fmt.Errorf("manufacture: component dish needs a positive radius and depth")
		}
		local = maths.Box{Min: maths.V(-r, -d/5, -r), Max: maths.V(r, d, r)}
	default:
		return maths.Box{}, fmt.Errorf("manufacture: unknown component form %q", part.Form)
	}
	if !finite(part.Position.Len()) {
		return maths.Box{}, fmt.Errorf("manufacture: component position must be finite")
	}
	local.Min, local.Max = local.Min.Add(part.Position), local.Max.Add(part.Position)
	result := maths.Box{Min: part.Frame.Out(local.Min), Max: part.Frame.Out(local.Min)}
	for index := range 8 {
		point := local.Min
		if index&1 != 0 {
			point.X = local.Max.X
		}
		if index&2 != 0 {
			point.Y = local.Max.Y
		}
		if index&4 != 0 {
			point.Z = local.Max.Z
		}
		point = part.Frame.Out(point)
		result.Min = maths.V(min(result.Min.X, point.X), min(result.Min.Y, point.Y), min(result.Min.Z, point.Z))
		result.Max = maths.V(math.Max(result.Max.X, point.X), math.Max(result.Max.Y, point.Y), math.Max(result.Max.Z, point.Z))
	}
	return result, nil
}

// zeroVec reports whether value is the zero vector.
func zeroVec(value maths.Vec) bool { return value == maths.Vec{} }

// widthMetres returns the width in metres, from the exact value when one is
// set.
func (part ComponentSpec) widthMetres() float64 {
	if part.exact != nil {
		return part.exact.width
	}
	return part.Width.InMetres()
}

// heightMetres returns the height in metres, from the exact value when one is
// set.
func (part ComponentSpec) heightMetres() float64 {
	if part.exact != nil {
		return part.exact.height
	}
	return part.Height.InMetres()
}

// depthMetres returns the depth in metres, from the exact value when one is
// set.
func (part ComponentSpec) depthMetres() float64 {
	if part.exact != nil {
		return part.exact.depth
	}
	return part.Depth.InMetres()
}

// radiusMetres returns the radius in metres, from the exact value when one is
// set.
func (part ComponentSpec) radiusMetres() float64 {
	if part.exact != nil {
		return part.exact.radius
	}
	return part.Radius.InMetres()
}

// innerRadiusMetres returns the inner radius in metres, from the exact value
// when one is set.
func (part ComponentSpec) innerRadiusMetres() float64 {
	if part.exact != nil {
		return part.exact.innerRadius
	}
	return part.InnerRadius.InMetres()
}

// thicknessMetres returns the wall thickness in metres, from the exact value
// when one is set.
func (part ComponentSpec) thicknessMetres() float64 {
	if part.exact != nil {
		return part.exact.thickness
	}
	return part.Thickness.InMetres()
}

// profileMetres returns the drum profile in metres, from the exact profile
// when one is set.
func (part ComponentSpec) profileMetres() [][coordinatePair]float64 {
	if part.exact != nil && len(part.exact.profile) > 0 {
		return part.exact.profile
	}
	profile := make([][coordinatePair]float64, len(part.Profile))
	for index, point := range part.Profile {
		profile[index] = [coordinatePair]float64{point.Radius.InMetres(), point.Height.InMetres()}
	}
	return profile
}

type buildingMaker struct{ spec BuildingSpec }

// buildingDome closes a revolved elliptical cap with an exact axial pole.
type buildingDome struct {
	radius, height float64
	cutsPerLevel   int
}

// skin returns the dome surface, with its profile refined to the cut budget
// at each level.
func (shape buildingDome) skin(fine cut) surface {
	levels := max(spaceOne, fine.around/shape.cutsPerLevel)
	profile := [][coordinatePair]float64{
		{spaceZero, spaceZero}, {shape.radius, spaceZero}}
	for level := spaceOne; level < levels; level++ {
		angle := math.Pi * spaceHalf * float64(level) / float64(levels)
		profile = append(profile, [coordinatePair]float64{
			shape.radius * math.Cos(angle), shape.height * math.Sin(angle)})
	}
	profile = append(profile, [coordinatePair]float64{spaceZero, shape.height})
	return (turnedProfile{points: profile}).skin(fine)
}

// made returns the building at the quick detail level.
func (row buildingMaker) made(from seed) (good, error) {
	return row.madeAt(from, "quick")
}

// madeAt returns the building at the named detail level. It returns errGood
// when the seed is not made.
func (row buildingMaker) madeAt(from seed, detail string) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	assembly := &instrumentAssembly{fine: cut{around: int(groundValue("sides-" + detail))},
		object: good{dressed: row.looks()}}
	row.appendTo(assembly, buildingSurfaceSlots())
	width, depth, height := row.spec.Width.InMetres(), row.spec.Depth.InMetres(), row.spec.Height.InMetres()
	bounds := maths.SegmentBox(maths.V(-width/2, 0, -depth/2), maths.V(width/2, height, depth/2), 0)
	object, err := assembly.finish(bounds)
	if err != nil {
		return good{}, err
	}
	return *object, nil
}

// looks returns one look per building material, keyed by surface slot.
func (row buildingMaker) looks() looks {
	palette := make(looks, len(row.spec.Materials))
	for _, material := range row.spec.Materials {
		var role slot
		switch material.Surface {
		case BuildingWalls:
			role = buildingWallSlot
		case BuildingRoof:
			role = buildingRoofSlot
		case BuildingAccent:
			role = buildingAccentSlot
		}
		palette[role] = componentLook(material)
	}
	return palette
}

type spacecraftSpecMaker struct{ spec SpacecraftSpec }

// made returns the spacecraft at the quick detail level.
func (row spacecraftSpecMaker) made(from seed) (good, error) { return row.madeAt(from, "quick") }

// madeAt returns the spacecraft at the named detail level. It returns errGood
// when the seed is not made.
func (row spacecraftSpecMaker) madeAt(from seed, detail string) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	assembly := &instrumentAssembly{fine: cut{around: int(spacecraftValue("sides-" + detail))}}
	materials, roles := componentLooks(row.spec.Materials)
	assembly.object.dressed = materials
	for _, part := range row.spec.Parts {
		part.appendTo(assembly, roles[part.Surface])
	}
	width, depth, height := row.spec.Width.InMetres(), row.spec.Depth.InMetres(), row.spec.Height.InMetres()
	bounds := maths.SegmentBox(maths.V(-width/2, 0, -depth/2), maths.V(width/2, height, depth/2), 0)
	made, err := assembly.finish(bounds)
	if err != nil {
		return good{}, err
	}
	return *made, nil
}

// componentLooks gives each material its own slot. It returns the looks by
// slot and the slot of each surface role.
func componentLooks(materials []ComponentMaterial) (looks, map[ComponentSurface]slot) {
	dressed := make(looks, len(materials))
	roles := make(map[ComponentSurface]slot, len(materials))
	for index, material := range materials {
		role := slot(index)
		roles[material.Surface] = role
		dressed[role] = componentLook(material)
	}
	return dressed, roles
}

// componentLook returns a copy of a material's look. The copy takes the
// material's name when the look has none. A material with no look is built
// from its own colour, gloss and index.
func componentLook(material ComponentMaterial) *thing.Material {
	if material.Look != nil {
		look := *material.Look
		if look.Name == "" {
			look.Name = material.Name
		}
		return &look
	}
	return &thing.Material{Name: material.Name, Colour: material.Colour,
		Gloss: material.Gloss, Index: material.Index}
}

// buildingSurfaceSlots maps each building surface role to its slot.
func buildingSurfaceSlots() map[BuildingSurface]slot {
	return map[BuildingSurface]slot{BuildingWalls: buildingWallSlot,
		BuildingRoof: buildingRoofSlot, BuildingAccent: buildingAccentSlot}
}

// appendTo adds every part of the building to the assembly, each with the slot
// of its surface role.
func (row buildingMaker) appendTo(assembly *instrumentAssembly, surfaces map[BuildingSurface]slot) {
	for _, part := range row.spec.Parts {
		part.appendTo(assembly, surfaces[part.Surface])
	}
}

// appendTo adds the part to the assembly with its material slot. A cut
// override applies only to this part.
func (part ComponentSpec) appendTo(assembly *instrumentAssembly, material slot) {
	previous := assembly.fine
	if part.Cuts > 0 {
		assembly.fine = cut{around: part.Cuts}
	}
	defer func() { assembly.fine = previous }()
	position := part.Position
	space := maths.NewSubspace(position, maths.Identity, spaceOne).Within(part.Frame)
	switch part.Form {
	case ComponentBox:
		centre := position.Add(maths.V(part.widthMetres()*spaceHalf,
			spaceZero, part.depthMetres()*spaceHalf))
		local := maths.NewSubspace(centre, maths.Identity, spaceOne)
		assembly.add(boxShape(part.widthMetres(), part.heightMetres(),
			part.depthMetres()), local.Within(part.Frame), material)
	case ComponentDrum:
		var shape form = groundDrum(part.radiusMetres(), part.innerRadiusMetres(), part.heightMetres())
		if profileValues := part.profileMetres(); len(profileValues) != 0 {
			profile := append([][coordinatePair]float64(nil), profileValues...)
			turned := turnedProfile{points: profile}
			shape = turned
			if part.Smooth {
				shape = roundProfile{turnedProfile: turned}
			}
		} else if part.Smooth {
			profile := groundDrum(part.radiusMetres(), part.innerRadiusMetres(), part.heightMetres()).(turnedProfile)
			shape = roundProfile{turnedProfile: profile}
		}
		assembly.add(shape, space, material)
	case ComponentDome:
		assembly.add(buildingDome{radius: part.radiusMetres(), height: part.heightMetres(),
			cutsPerLevel: part.DomeCutsPerLevel}, space, material)
	case ComponentSlitDome:
		assembly.add(domeShell{radius: part.radiusMetres(), thick: part.thicknessMetres(),
			opening: part.Opening.InRadians()}, space, material)
	case ComponentRod:
		from := part.Frame.Out(position.Add(part.From))
		to := part.Frame.Out(position.Add(part.To))
		assembly.rodAlong(from, to.Sub(from),
			part.radiusMetres()*scaleOf(part.Frame), material)
	case ComponentDish:
		dish := mirrorDish{radius: part.radiusMetres(), depth: part.depthMetres(), segmented: part.Segmented}
		if part.Smooth {
			assembly.add(roundDish{mirrorDish: dish}, space, material)
		} else {
			assembly.add(dish, space, material)
		}
	}
}
