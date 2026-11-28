package preview

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"

	"github.com/janearc/slayer/ferruccio/internal/draft"
	"gopkg.in/yaml.v3"
)

// FullFilm holds checked authored settings shared by preparation and setup.
type FullFilm struct {
	film, setFilm *yaml.Node
	preset        outputPreset
	stock         *developmentStock
	rig           *physicalRig
	reference     filmReference
	region        *renderRegion
	exactError    string
}

// physicalRig records the authored optics and exposure in producer units.
type physicalRig struct {
	WidthMM        float64 `json:"width_mm"`
	HeightMM       float64 `json:"height_mm"`
	FocalMM        float64 `json:"focal_mm"`
	FStop          float64 `json:"f_stop"`
	Focus          string  `json:"focus"`
	ShutterSeconds float64 `json:"shutter_seconds"`
	Glimpses       int     `json:"glimpses"`
}

// developmentStock keeps omissions distinct from an explicitly authored zero.
type developmentStock struct {
	Name  string   `json:"name" yaml:"name"`
	Seed  *int64   `json:"seed,omitempty" yaml:"seed"`
	Rated *float64 `json:"rated,omitempty" yaml:"rated"`
}

type filmReference struct {
	path, digest string
}

// renderRegion uses image coordinates from the upper left of the frame.
type renderRegion struct {
	Left, Top, Width, Height int
}

// HasPhysicalRig says whether the still uses a physical exposure.
func (full FullFilm) HasPhysicalRig() bool { return full.rig != nil }

// ReadFullFilm reads the authored settings without creating a snapshot.
func ReadFullFilm(path, source string) (FullFilm, error) {
	document, err := draft.Open(filepath.Base(path), []byte(source))
	if err != nil {
		return FullFilm{}, err
	}
	tree, err := document.Tree()
	if err != nil {
		return FullFilm{}, err
	}
	return readFullFilm(path, document, tree)
}

// readFullFilm extracts still settings from the already checked draft.
func readFullFilm(path string, document *draft.Document, tree *yaml.Node) (FullFilm, error) {
	var full FullFilm
	root := tree.Content[0]
	switch document.Kind() {
	case "catalogue":
		return full, fmt.Errorf("full render is not supported for catalogue plans; use the catalogue producer")
	case "set", "legacy-set":
		full.setFilm = field(field(root, "scene"), "film")
	case "job":
		if movieJob(root) {
			return full, fmt.Errorf("full still rendering does not support movie jobs")
		}
		full.film = field(root, "film")
		preset, err := readOutputPreset(field(full.film, "preset"))
		if err != nil {
			return full, err
		}
		full.preset = preset
		full.exactError = jobExactRefusal(root, full.film)
		if region := field(full.film, "region"); region != nil {
			if err := checkAuthoredRegion(root, full.film); err != nil {
				return FullFilm{}, err
			}
			full.region = new(renderRegion)
			if err := region.Decode(full.region); err != nil {
				return FullFilm{}, err
			}
		}
		if stock := field(root, "stock"); stock != nil {
			full.stock = new(developmentStock)
			if err := stock.Decode(full.stock); err != nil {
				return FullFilm{}, err
			}
		}
		if reference := field(field(root, "scene"), "set"); reference != nil {
			var name string
			if err := reference.Decode(&name); err != nil {
				return FullFilm{}, err
			}
			if err := full.readReference(path, name); err != nil {
				return FullFilm{}, err
			}
		}
	}
	if field(full.setFilm, "frame") != nil {
		rig, err := readPhysicalRig(full.setFilm)
		if err != nil {
			return FullFilm{}, err
		}
		full.rig = &rig
	}
	return full, nil
}

// checkAuthoredRegion checks job fields that always conflict with a region.
// The producer still checks its resolved traversal, scene and development.
func checkAuthoredRegion(job, film *yaml.Node) error {
	switch {
	case field(field(job, "scene"), "set") == nil:
		return fmt.Errorf("a film region requires a referenced set")
	case field(film, "mode") != nil && field(film, "mode").Value == "wire":
		return fmt.Errorf("a film region does not support wire mode")
	case field(film, "aperture") != nil:
		return fmt.Errorf("a film region does not support a job aperture")
	case field(job, "stock") != nil:
		return fmt.Errorf("a film region does not support stock development")
	}
	return nil
}

// readReference keeps the identity of the set used to read the film.
func (full *FullFilm) readReference(jobPath, path string) error {
	if !filepath.IsAbs(path) {
		path = filepath.Join(filepath.Dir(jobPath), path)
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	source, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	document, err := draft.Open(path, source)
	if err != nil {
		return err
	}
	tree, err := document.Tree()
	if err != nil {
		return err
	}
	if document.Kind() != "set" && document.Kind() != "legacy-set" {
		return fmt.Errorf("%s is not a scene set", path)
	}
	full.setFilm = field(field(tree.Content[0], "scene"), "film")
	full.reference = filmReference{path: path, digest: fmt.Sprintf("%x", sha256.Sum256(source))}
	return nil
}

// readPhysicalRig copies the checked set fields without converting their units.
func readPhysicalRig(film *yaml.Node) (physicalRig, error) {
	var source struct {
		Frame struct{ Width, Height float64 }
		Lens  struct {
			Focal, Stop float64
			Focus       string
		}
		Shutter  float64
		Glimpses int
	}
	if err := film.Decode(&source); err != nil {
		return physicalRig{}, err
	}
	return physicalRig{WidthMM: source.Frame.Width, HeightMM: source.Frame.Height,
		FocalMM: source.Lens.Focal, FStop: source.Lens.Stop, Focus: source.Lens.Focus,
		ShutterSeconds: source.Shutter, Glimpses: source.Glimpses}, nil
}

// jobExactRefusal mirrors the unsupported modes of the exact still producer.
func jobExactRefusal(job, film *yaml.Node) string {
	switch {
	case field(field(job, "scene"), "frames") != nil:
		return "exact samples require a scene to trace"
	case field(film, "region") != nil:
		return "exact samples do not yet support regions"
	case field(film, "mode") != nil && field(film, "mode").Value == "wire":
		return "exact samples do not yet support wire mode"
	case field(film, "aperture") != nil:
		return "exact samples do not yet support a job aperture"
	}
	return ""
}

// movieJob follows the producer's clock fields. A preset's rate alone
// does not make a movie.
func movieJob(root *yaml.Node) bool {
	for _, name := range []string{"fps", "duration", "start", "scene-duration"} {
		if field(root, name) != nil {
			return true
		}
	}
	return false
}
