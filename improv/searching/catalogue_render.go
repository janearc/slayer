package searching

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/gpu"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// catalogueSet holds one small luminous marker per measured position.
// Marker size and intensity are authored to survive a preview's sampling.
type catalogueSet struct {
	world  *thing.World
	points []*thing.Thing
	home   *thing.Thing
}

// newCatalogue keeps the shared sphere geometry and all catalogue identities.
func newCatalogue(data *catalogue) *catalogueSet {
	set := &catalogueSet{world: thing.NewWorld()}
	for _, point := range data.Points {
		level := .5 + 1.5*math.Pow(10, -.08*math.Max(0, point.Magnitude-6))
		set.points = append(set.points, set.world.Add(thing.Thing{
			Shape: shape.Sphere{Radius: 1}, Material: &thing.Material{
				Name: "Gaia " + point.ID, Glow: maths.RGB(level, level*.85, level*.65)}}))
	}
	set.home = set.world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1},
		Material: &thing.Material{Name: "Sun location marker", Glow: maths.RGB(.1, 2, 3)}})
	return set
}

// position rotates Galactic Cartesian axes into Slayer's y-up frame:
// x is toward l=0, y toward the north pole, minus z toward l=90.
// This proper rotation preserves handedness, angles and relative distances.
func (point cataloguePoint) position() maths.Vec {
	lon, lat := point.Longitude*maths.Degree, point.Latitude*maths.Degree
	return maths.V(math.Cos(lat)*math.Cos(lon), math.Sin(lat),
		-math.Cos(lat)*math.Sin(lon)).Scale(point.Parsecs)
}

// catalogueFrame rebases the entire scene uniformly for GPU precision.
// The equivalent physical camera moves in parsecs; no individual positions
// are compressed. Only unresolved marker radii grow to remain visible.
func (renderer *Renderer) catalogueFrame(ctx context.Context, state State) (*Exposure, State, error) {
	if err := ctx.Err(); err != nil {
		return nil, state, err
	}
	edit, plan := renderer.Edit, renderer.Edit.Plan.CatalogueStudy
	if renderer.stars == nil {
		renderer.stars = newCatalogue(edit.Catalogue)
	}
	state.Site, state.Chapter, state.Spectrum = "catalogue-study", "nearby stars", "authored display markers"
	drawer := renderer.drawers[state.Site]
	if drawer == nil {
		var err error
		drawer, err = gpu.NewFrames(gpu.Software)
		if err != nil {
			return nil, state, err
		}
		renderer.drawers[state.Site] = drawer
	}
	distance := plan.distance((state.At - edit.CatalogueStart) / (edit.CatalogueEnd - edit.CatalogueStart))
	state.Camera, state.Aim, state.Flash, state.SceneSeconds = distance, 0, 0, state.At
	eye := vector(plan.Eye).Scale(10 / vector(plan.Eye).Len())
	pixel := 2 * math.Tan(plan.Wide*maths.Degree/2) / float64(edit.Plan.Width)
	for index, point := range edit.Catalogue.Points {
		at := point.position().Scale(10 / distance)
		radius := at.Sub(eye).Len() * pixel * plan.Pixels
		renderer.stars.points[index].Motion = thing.Still(maths.Subspace{Origin: at, Scale: math.Max(radius, 1e-6)})
	}
	renderer.stars.home.Motion = thing.Still(maths.Subspace{Scale: 10 * pixel * plan.HomePixels})
	lens, err := film.PinholeOf(plan.Wide * maths.Degree)
	if err != nil {
		return nil, state, err
	}
	frame, _, err := drawer.Draw(renderer.stars.world, uint64(math.Floor(state.At*float64(edit.Plan.FPS)))+1,
		maths.Subspace{Origin: eye, Orientation: maths.LookAt(eye, maths.Vec{})}, lens,
		time.Duration(state.At*float64(time.Second)), edit.Plan.Width, edit.Plan.Height, edit.Plan.Rays)
	if err != nil {
		return nil, state, err
	}
	return &Exposure{linear: frame,
		tone: colour.Tone{Exposure: edit.Plan.Exposure, Knee: .5, Top: 12, Bloom: edit.Plan.Bloom},
		labels: []string{
			fmt.Sprintf("SUN | camera %.1f pc | %d sampled Gaia DR3 stars", distance, len(edit.Catalogue.Points)),
			"Catalogue positions; enlarged markers; compressed travel. No inferred voids.",
		}}, state, nil
}
