package thing

// The census: a world counted as it was staged. Every placed thing is a
// row: a mesh says its faces, a mesh group its instanced faces, its
// stored faces and its instances, and anything else its kind and no
// faces at all, since an analytic surface has none to count. Volumes are
// counted by kind. The totals are Storage's, so the census and the
// scene-complexity report agree.
//
// These are the faces staged in the scene, before any test of what the
// camera sees: not the faces visible, nor the faces a ray hit.

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
)

// CensusCounts says what a census counts, in the words its file says.
const CensusCounts = "staged, not visible"

// The kinds a census gives the shapes it knows by name; any other shape
// is called by its Go type.
const (
	kindMesh        = "mesh"
	kindMeshGroup   = "mesh group"
	kindHeightField = "height field"
)

// Census is a world's geometry as staged: each placed thing, the totals,
// the analytic shapes and the volumes by kind, the height fields' grids,
// and how many fields and lamps it holds.
type Census struct {
	Total    CensusTotal   `json:"total"`
	Placed   []PlacedCount `json:"placed"`
	Analytic []KindCount   `json:"analytic"`
	Grids    []GridCount   `json:"height_fields"`
	Volumes  []KindCount   `json:"volumes"`
	Fields   int           `json:"fields"`
	Lamps    int           `json:"lamps"`
	Ocean    bool          `json:"ocean_surface"`
}

// CensusTotal is the world's faces as staged: every placement and every
// group member counted (staged), each distinct mesh once (stored), the
// group members, and the hidden things and their staged faces, which are
// staged and neither drawn nor shadowing. Storage is the whole inventory.
type CensusTotal struct {
	Things      int             `json:"things"`
	FacesStaged int             `json:"faces_staged"`
	FacesStored int             `json:"faces_stored"`
	Instances   int             `json:"instances"`
	Hidden      int             `json:"hidden_things"`
	HiddenFaces int             `json:"hidden_faces_staged"`
	Storage     GeometryStorage `json:"storage"`
}

// PlacedCount is one placed thing: its place in the world, its kind and
// material, and its faces where it has faces. A mesh group's faces are
// instanced; its stored faces count each distinct mesh once. A height
// field says its samples a side. An analytic shape has no faces field.
type PlacedCount struct {
	Index       int    `json:"index"`
	Kind        string `json:"kind"`
	Material    string `json:"material,omitempty"`
	Faces       *int   `json:"faces,omitempty"`
	StoredFaces *int   `json:"stored_faces,omitempty"`
	Instances   int    `json:"instances,omitempty"`
	Samples     int    `json:"samples_per_side,omitempty"`
	Hidden      bool   `json:"hidden,omitempty"`
}

// KindCount is how many of one kind the world holds.
type KindCount struct {
	Kind  string `json:"kind"`
	Count int    `json:"count"`
}

// GridCount is a height field's place in the world and its samples a
// side: its grid is that many squared.
type GridCount struct {
	Index   int `json:"index"`
	Samples int `json:"samples_per_side"`
}

// Count is the census of a world as staged. A missing world, thing or
// volume is refused by its place, never counted as nothing.
func Count(world *World) (Census, error) {
	if world == nil {
		return Census{}, errors.New("thing: there is no world to count")
	}
	made := Census{Fields: len(world.Fields), Lamps: len(world.Lamps),
		Ocean: world.OceanSurface != nil, Placed: []PlacedCount{},
		Grids: []GridCount{}}
	values := make([]Thing, 0, len(world.Things))
	analytic := map[string]int{}
	for index, placed := range world.Things {
		if placed == nil {
			return Census{}, fmt.Errorf("thing: thing %d is nil; it "+
				"cannot be counted", index)
		}
		values = append(values, *placed)
		made.add(placedCount(index, placed), analytic)
	}
	volumes, err := volumeKinds(world.Volumes)
	if err != nil {
		return Census{}, err
	}
	made.Volumes, made.Analytic = volumes, sortedKinds(analytic)
	made.Total.Things, made.Total.Storage = len(values), Storage(values)
	made.Total.FacesStaged = made.Total.Storage.ReferencedFaces
	made.Total.FacesStored = made.Total.Storage.StoredFaces
	made.Total.Instances = made.Total.Storage.Instances
	return made, nil
}

// add takes a row into the census: an analytic one by kind, a height
// field's grid, and a hidden one's faces.
func (made *Census) add(row PlacedCount, analytic map[string]int) {
	made.Placed = append(made.Placed, row)
	if row.Faces == nil {
		analytic[row.Kind]++
	}
	if row.Kind == kindHeightField {
		made.Grids = append(made.Grids, GridCount{Index: row.Index,
			Samples: row.Samples})
	}
	if row.Hidden {
		made.Total.Hidden++
		if row.Faces != nil {
			made.Total.HiddenFaces += *row.Faces
		}
	}
}

// placedCount is a thing's row: faces for a mesh and a group, the grid
// for a height field, and nothing more for an analytic shape.
func placedCount(index int, placed *Thing) PlacedCount {
	row := PlacedCount{Index: index, Kind: kindOf(placed.Shape),
		Hidden: placed.Hidden()}
	if placed.Material != nil {
		row.Material = placed.Material.Name
	}
	switch solid := placed.Shape.(type) {
	case *mesh.Mesh:
		row.Faces = counted(solid.FaceCount())
	case *shape.Mesh:
		row.Faces = counted(solid.Storage().Faces)
	case *shape.MeshGroup:
		row.Faces = counted(solid.FaceCount())
		row.StoredFaces = counted(solid.StoredFaces())
		row.Instances = solid.MemberCount()
	case *shape.HeightField:
		_, row.Samples, _, _, _, _ = solid.Pack()
	}
	return row
}

// counted is a count that is said, nought included.
func counted(number int) *int { return &number }

// kindOf is a shape's kind: a word for the shapes the census knows, and
// the Go type, without its pointer, for any other.
func kindOf(solid shape.Shape) string {
	switch solid.(type) {
	case *mesh.Mesh, *shape.Mesh:
		return kindMesh
	case *shape.MeshGroup:
		return kindMeshGroup
	case *shape.HeightField:
		return kindHeightField
	}
	return strings.TrimPrefix(fmt.Sprintf("%T", solid), "*")
}

// volumeKinds is the volumes by the kind of their medium. A nil volume
// is refused by its place.
func volumeKinds(volumes []*Volume) ([]KindCount, error) {
	kinds := map[string]int{}
	for index, volume := range volumes {
		if volume == nil {
			return nil, fmt.Errorf("thing: volume %d is nil; it cannot "+
				"be counted", index)
		}
		kinds[strings.TrimPrefix(fmt.Sprintf("%T", volume.Medium), "*")]++
	}
	return sortedKinds(kinds), nil
}

// sortedKinds is a count by kind as rows, in the kinds' order.
func sortedKinds(kinds map[string]int) []KindCount {
	rows := make([]KindCount, 0, len(kinds))
	for kind, count := range kinds {
		rows = append(rows, KindCount{Kind: kind, Count: count})
	}
	sort.Slice(rows, func(left, right int) bool {
		return rows[left].Kind < rows[right].Kind
	})
	return rows
}
