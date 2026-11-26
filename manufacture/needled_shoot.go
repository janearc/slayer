package manufacture

import (
	_ "embed"
	"fmt"
	"math"
	"sync"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/puzzlemouth/numbers"
	"github.com/janearc/slayer/manufacture/plants/conifers"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
)

//go:embed needled-shoot.yml
var shootChoices []byte

var shootNumbers, shootNumbersError = numbers.Read("needled-shoot.yml", shootChoices)

// shootValue reads an authored model choice, including its provenance.
func shootValue(name string) float64 {
	if shootNumbersError != nil {
		panic(shootNumbersError)
	}
	value, err := shootNumbers.Get(name)
	if err != nil {
		panic(err)
	}
	noted(value)
	return value.Value
}

type shootPlan = conifers.Plan

// shootPrototype is shared geometry, not a new copy for every branchlet.
type shootPrototype struct {
	wood, leaves *mesh.Mesh
	physical     maths.Box
	plan         shootPlan
}

type shootKey struct {
	style, bin, variant int
	detail              string
}

type shootProfile = conifers.ShootProfile

// A book retains a bounded number of reusable prototypes. Once full,
// construction still works, but additional prototypes are not retained.
type shootStore struct {
	mu    sync.Mutex
	cache sync.Map
	count int
	limit int
}

const maxCachedShootPrototypes = 128

type shootCacheKey struct {
	shootKey
	foliage shootProfile
}

func (store *shootStore) prototype(style, bin, variant int, detail string, foliage shootProfile) (*shootPrototype, error) {
	key := shootCacheKey{shootKey{style, bin, variant, detail}, foliage}
	if known, found := store.cache.Load(key); found {
		return known.(*shootPrototype), nil
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if known, found := store.cache.Load(key); found {
		return known.(*shootPrototype), nil
	}
	limit := store.limit
	if limit == 0 {
		limit = maxCachedShootPrototypes
	}
	if store.count >= limit {
		var transient sync.Map
		return prototypeWith(style, bin, variant, detail, foliage, &transient)
	}
	made, err := prototypeWith(style, bin, variant, detail, foliage, &store.cache)
	if err == nil {
		store.count++
	}
	return made, err
}

type shootConfig struct {
	profile          shootProfile
	store            *shootStore
	measured, picked []numbers.Number
}

// NewConiferShoot constructs a loose shoot from the checked species profile.
// Flat sprays and scale foliage have defined specimen representations.
func (book *ConiferProfiles) NewConiferShoot(name string) (Kind, error) {
	row, _, err := book.selectTree(name)
	if err != nil {
		return Kind{}, err
	}
	if row.FoliageStyle != 1 && row.FoliageStyle != 2 {
		return Kind{}, fmt.Errorf("manufacture: conifer %q has no loose shoot representation", name)
	}
	return Kind{name: name + "-shoot", how: shootKind{style: row.FoliageStyle,
		config: row.shootConfig, dressed: row.dressed}}, nil
}

// NewCoastRedwoodShoot and NewGiantSequoiaShoot are named conveniences.
func (book *ConiferProfiles) NewCoastRedwoodShoot() (Kind, error) {
	return book.NewConiferShoot("coast-redwood")
}
func (book *ConiferProfiles) NewGiantSequoiaShoot() (Kind, error) {
	return book.NewConiferShoot("giant-sequoia")
}

// shootKind exposes the same physical foliage used on trees as a specimen.
// It is useful for close views and for checking a recipe before scattering it.
type shootKind struct {
	style   int
	config  *shootConfig
	dressed looks
}

// made is the loose shoot at the default surface detail.
func (row shootKind) made(from seed) (good, error) { return row.madeAt(from, "quick") }

// madeAt changes only the shoot's surface detail, retaining physical leaves.
func (row shootKind) madeAt(from seed, detail string) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	minimum, cell := shootValue("shoot-minimum-cell"), shootValue("shoot-cell-length")
	bin := int(math.Round(math.Log2(cell / minimum)))
	level := detail
	if level == "hull" {
		level = "quick"
	}
	if row.config == nil || row.config.store == nil {
		return good{}, fmt.Errorf("manufacture: shoot needs a checked profile")
	}
	noteConiferFoliage(row.config)
	shoot, err := row.config.store.prototype(row.style, bin, int(from.number%2), level, row.config.profile)
	if err != nil {
		return good{}, err
	}
	box := shoot.physical
	// A loose shoot lies horizontally, with its lowest point on the ground.
	space := maths.Subspace{Orientation: maths.About(maths.ZAxis, -math.Pi/2)}
	turned := transformed(box, space)
	space.Origin.Y = -turned.Min.Y
	space.Origin.X = -(turned.Min.X + turned.Max.X) / 2
	space.Origin.Z = -(turned.Min.Z + turned.Max.Z) / 2
	box = transformed(box, space)
	made := good{dressed: row.dressed, extent: &box}
	if detail == "hull" {
		made.parts = []part{{shape: crownShape{box}, of: leaf, space: maths.Subspace{}}}
		return made, nil
	}
	return made.withMeshGroups(map[slot][]shape.MeshMember{
		body: {{Mesh: shoot.wood, Space: space}}, leaf: {{Mesh: shoot.leaves, Space: space}},
	})
}

func prototypeWith(style, bin, variant int, detail string, foliage shootProfile, cache *sync.Map) (*shootPrototype, error) {
	maximumBin := int(math.Ceil(math.Log2(
		shootValue("shoot-cell-length") / shootValue("shoot-minimum-cell"))))
	if style < 1 || style > spruceShootStyle || bin < 0 || bin > maximumBin ||
		variant < 0 || variant > 1 ||
		(detail != "clumps" && detail != "quick" && detail != "demo") {
		return nil, fmt.Errorf("manufacture: unsupported shoot prototype key")
	}

	key := shootCacheKey{shootKey{style, bin, variant, detail}, foliage}
	if known, ok := cache.Load(key); ok {
		return known.(*shootPrototype), nil
	}
	length := shootValue("shoot-minimum-cell") * math.Pow(2, float64(bin))
	plan := conifers.PlanShoot(style, length, Sow(uint64(variant)), foliage, shootValue, shootNormal)
	var wood, leaves, physical surface
	// Both redwood ranks and all three sequoia rows land on tube vertices.
	// Finer cuts retain those vertices; leaf bases stay attached. Six is
	// the smallest common ring for two ranks and three rows. Spruce uses
	// eight vertices for four ranks at alternating half-rank phases.
	coarse := foliage.WoodSides
	fine := cut{around: coarse}
	if detail == "demo" {
		fine.around = 2 * coarse
	}
	for _, axis := range plan.Axes {
		tube, err := newTube(measure.Metres(axis.Length),
			measure.Metres(axis.Radius),
			measure.Metres(axis.Radius*shootValue("shoot-tip-radius")))
		if err != nil {
			return nil, err
		}
		space := conifers.AxisSpace(axis)
		wood = append(wood, tube.skin(fine).placed(space)...)
		physical = append(physical,
			tube.skin(cut{around: 2 * coarse}).placed(space)...)
	}
	for _, needle := range plan.Needles {
		leaves = append(leaves, conifers.NeedleSkin[face](needle, detail)...)
		// Bound all detail levels, including the quick tetrahedra.
		physical = append(physical, conifers.NeedleSkin[face](needle, "quick")...)
		physical = append(physical, conifers.NeedleSkin[face](needle, "demo")...)
	}
	if style == spruceShootStyle && detail == "clumps" {
		bounds := partBounds([]part{{shape: fixedSurface(physical), space: maths.Local()}})
		leaves = crownShape{box: bounds}.skin(cut{})
	}
	var meshes [2]*mesh.Mesh
	for index, skin := range []surface{wood, leaves} {
		made, err := skin.meshed()
		if err != nil {
			return nil, err
		}
		meshes[index] = made
	}
	bounds := partBounds([]part{{shape: fixedSurface(physical), space: maths.Subspace{}}})
	made := &shootPrototype{wood: meshes[0], leaves: meshes[1], physical: bounds, plan: plan}
	actual, _ := cache.LoadOrStore(key, made)
	return actual.(*shootPrototype), nil
}

// fixedSurface lets the existing bounds helper read already made vertices.
type fixedSurface surface

// skin supplies vertices already resolved by the shoot construction.
func (skin fixedSurface) skin(cut) surface { return surface(skin) }

// maxConiferCells bounds actual placed shoot prototypes; excess is refused.
const maxConiferCells = 100000

func recursiveFoliageWith(made good, from seed, style int, detail string, config *shootConfig) (good, error) {
	if config == nil || config.store == nil {
		return good{}, fmt.Errorf("manufacture: conifer foliage needs a checked profile")
	}
	noteConiferFoliage(config)
	foliage := config.profile
	var woodParts []part
	var wood, leaves []shape.MeshMember
	count := 0
	var bounds *maths.Box
	for index, piece := range made.parts {
		if piece.of != leaf {
			woodParts = append(woodParts, piece)
			continue
		}
		// Terminal foliage has a role, independent of part order.
		if !piece.terminal {
			continue
		}
		tube, ok := piece.shape.(tube)
		if !ok {
			return good{}, fmt.Errorf("manufacture: foliage needs a shoot axis")
		}
		length := tube.length.InMetres()
		cell := shootValue("shoot-cell-length")
		if !positive(length) || !positive(cell) || length/cell > float64(maxConiferCells) {
			return good{}, fmt.Errorf("manufacture: conifer foliage exceeds its cell budget")
		}
		for node := 0; float64(node)*cell < length; node++ {
			if count == maxConiferCells {
				return good{}, fmt.Errorf("manufacture: conifer foliage exceeds its cell budget")
			}
			count++
			remaining := math.Min(cell, length-float64(node)*cell)
			minimum := shootValue("shoot-minimum-cell")
			bin := max(0, int(math.Ceil(math.Log2(remaining/minimum))))
			variant := int(from.draw(index, node, 0) * 2)
			level := detail
			if detail == "hull" {
				level = "quick"
			}
			shoot, err := config.store.prototype(style, bin, variant, level, foliage)
			if err != nil {
				return good{}, err
			}
			// Adjacent sprays may roll into different planes.
			roll := from.swing(index, node, 1) *
				shootValue("shoot-roll") * maths.Degree
			space := maths.Subspace{
				Origin:      maths.V(0, float64(node)*cell, 0),
				Orientation: maths.About(maths.YAxis, roll),
			}.Within(piece.space)
			wood = append(wood, shape.MeshMember{Mesh: shoot.wood, Space: space})
			leaves = append(leaves, shape.MeshMember{Mesh: shoot.leaves, Space: space})
			physical := transformed(shoot.physical, space)
			if bounds == nil {
				bounds = &physical
			} else {
				joined := enclosing(*bounds, physical)
				bounds = &joined
			}
		}
	}
	if len(leaves) == 0 {
		return good{}, fmt.Errorf("manufacture: tree has no terminal shoots")
	}
	base := partBounds(woodParts)
	box := enclosing(base, *bounds)
	made.parts, made.extent = woodParts, &box
	if detail == "hull" {
		made.parts = []part{woodParts[0], {shape: crownShape{*bounds}, of: leaf, space: maths.Subspace{}}}
		return made, nil
	}
	var bark surface
	for _, part := range woodParts {
		bark = append(bark, part.skin()...)
	}
	barkMesh, err := bark.meshed()
	if err != nil {
		return good{}, err
	}
	wood = append(wood, shape.MeshMember{Mesh: barkMesh})
	return made.withMeshGroups(map[slot][]shape.MeshMember{body: wood, leaf: leaves})
}
