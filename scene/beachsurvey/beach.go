// Package beachsurvey stages measured land and coarse regional bathymetry.
package beachsurvey

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/janearc/slayer/arid"
	"github.com/janearc/slayer/materials"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/oceantex"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/terrain"
)

// dataRoot names the variable that holds slayer's data folder. The
// surveys are too large for the tree and live in that folder.
const dataRoot = "SLAYER_DATA"

// surveyFolder is the surveys' folder inside the data folder.
const surveyFolder = "beachsurvey"

// surveys is the folder of retained surveys, or the refusal that says
// which variable names it when none does.
func surveys() (fs.FS, error) {
	root := os.Getenv(dataRoot)
	if root == "" {
		return nil, fmt.Errorf("beach surveys are kept outside the tree: "+
			"set %s to slayer's data folder", dataRoot)
	}
	return os.DirFS(filepath.Join(root, surveyFolder)), nil
}

type survey struct {
	ID           string
	Lat, Half    float64
	From, Aim    [3]float64
	Observations struct {
		Salinity  *materials.WaterObservation `json:"salinity"`
		Sediment  *materials.WaterObservation `json:"sediment"`
		Turbidity *materials.WaterObservation `json:"turbidity"`
	} `json:"observations"`
}

// Sites returns the retained surveys in stable order, and none where the
// data folder is not named.
func Sites() []string {
	kept, err := surveys()
	if err != nil {
		return nil
	}
	files, _ := fs.Glob(kept, "*.json")
	sites := make([]string, 0, len(files))
	for _, file := range files {
		sites = append(sites, strings.TrimSuffix(path.Base(file), ".json"))
	}
	return sites
}

// New places the measured surface and calm sea in a shared cove.
func New(id string) (*arid.Cove, maths.Vec, maths.Vec, error) {
	return NewWithWater(id, maths.RGB(.14, .050, .039))
}

// NewWithWater stages the retained survey with supplied RGB water extinction
// per metre. The source and interpretation of the coefficients belong to the
// caller's recipe.
func NewWithWater(id string, extinction maths.Light) (*arid.Cove, maths.Vec, maths.Vec, error) {
	var data survey
	kept, err := surveys()
	if err != nil {
		return nil, maths.Vec{}, maths.Vec{}, err
	}
	raw, err := fs.ReadFile(kept, id+".json")
	if err != nil || json.Unmarshal(raw, &data) != nil || data.ID != id {
		return nil, maths.Vec{}, maths.Vec{},
			fmt.Errorf("beach survey %q is unavailable", id)
	}
	window, err := heights(id)
	if err != nil {
		return nil, maths.Vec{}, maths.Vec{}, err
	}
	ground, low, err := window.Ground()
	if err != nil {
		return nil, maths.Vec{}, maths.Vec{}, err
	}
	origin := maths.V(-data.Half, low, -data.Half)
	terrain := thing.Thing{
		Shape:  ground,
		Motion: thing.Still(maths.NewSubspace(origin, maths.Identity, 1)),
		Material: &thing.Material{
			Name:    "coastal relief",
			Texture: paint{window: window, low: low, latitude: data.Lat, id: id},
		},
	}
	var composition *materials.WaterComposition
	if data.Observations.Salinity != nil || data.Observations.Sediment != nil ||
		data.Observations.Turbidity != nil {
		composition, err = materials.NewWaterComposition(data.Observations.Salinity,
			data.Observations.Sediment, data.Observations.Turbidity)
		if err != nil {
			return nil, maths.Vec{}, maths.Vec{}, fmt.Errorf("beach survey %q observations: %w", id, err)
		}
	}
	location, err := arid.New([]thing.Thing{terrain}, &arid.Water{
		Name: id + " water", Extinction: extinction,
		Sea:    oceantex.Sea{Wind: 0, Fetch: 2400, Depth: 20},
		Extent: 32, Collar: 8, Samples: 65, Seed: 11,
		Composition:   composition,
		OpticalSource: "Authored comparison coefficients supplied by the scene",
	})
	return location, vector(data.From), vector(data.Aim), err
}

// heights reads the retained metric heightfield.
func heights(id string) (terrain.Window, error) {
	kept, err := surveys()
	if err != nil {
		return terrain.Window{}, err
	}
	b, err := fs.ReadFile(kept, id+".bin.gz")
	if err != nil {
		return terrain.Window{}, err
	}
	gz, err := gzip.NewReader(bytes.NewReader(b))
	if err != nil {
		return terrain.Window{}, err
	}
	defer gz.Close()
	raw, err := io.ReadAll(gz)
	if err != nil {
		return terrain.Window{}, err
	}
	return terrain.WindowFromBytes(raw)
}

// vector reads an XYZ triple in the scene's local metric frame.
func vector(xyz [3]float64) maths.Vec { return maths.V(xyz[0], xyz[1], xyz[2]) }

type paint struct {
	window        terrain.Window
	low, latitude float64
	id            string
}

// At gives the measured shape an illustrative shade by height and slope.
func (p paint) At(point maths.Vec) maths.Light {
	h := point.Y + p.low
	if h < 0 {
		f := maths.Smooth(1, 75, -h)
		return maths.RGB(.33, .50, .37).Scale(1 - f).
			Add(maths.RGB(.035, .13, .18).Scale(f))
	}
	across, down, step := p.window.Size()
	x := max(1, min(across-2, int(point.X/step+.5)))
	z := max(1, min(down-2, int(point.Z/step+.5)))
	slope := maths.V(p.window.At(x+1, z)-p.window.At(x-1, z), 0,
		p.window.At(x, z+1)-p.window.At(x, z-1)).Len() / (2 * step)
	cliff := maths.Smooth(.4, 1.8, slope)
	v := noise.Gradient(199, point.X/65, h/38, point.Z/65)
	if p.id == "antarctica" {
		// Whalers Bay is a volcanic Antarctic shoreline. This authored
		// weathering suggests dark ash, exposed rock and patches of snow.
		ash := maths.RGB(.17, .18, .18)
		rock := maths.RGB(.22, .24, .24)
		snow := maths.RGB(.76, .80, .80)
		land := ash.Scale(1 - cliff).Add(rock.Scale(cliff))
		cover := maths.Smooth(18, 120, h+14*v) * (1 - .55*cliff)
		return land.Scale(1 - cover).Add(snow.Scale(cover))
	}
	plants := maths.RGB(.21, .36, .16)
	sand := maths.RGB(.72, .59, .36)
	if p.id == "pink-sand-beach" || p.id == "raudasandur" {
		sand = maths.RGB(.78, .49, .37)
	}
	if p.id == "dune-du-pilat" || p.id == "dolfynstrand" ||
		p.id == "al-mughsail" || p.id == "turquoise-bay" {
		plants = maths.RGB(.57, .44, .29)
	}
	if p.latitude > 48 {
		plants = plants.Scale(.84)
	}
	if p.id == "bering-sea" {
		plants = maths.RGB(.22, .24, .17)
	}
	land := plants.Scale(1 - cliff).
		Add(maths.RGB(.43, .37, .32).Scale(cliff))
	beach := (1 - maths.Smooth(1.8, 11, h)) * (1 - maths.Smooth(.14, .64, slope))
	return land.Scale(1 - beach).Add(sand.Scale(beach)).
		Scale(.91 + .19*v)
}
