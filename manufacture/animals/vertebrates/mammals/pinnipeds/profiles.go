package pinnipeds

import (
	_ "embed"
	"fmt"
	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/manufacture/profilecheck"
	"gopkg.in/yaml.v3"
	"maps"
	"math"
	"slices"
	"strings"
)

//go:embed profiles.yaml
var profileRaw []byte

// Profile describes the shared skeleton's variable anatomy and,
// for lofted seals, the rest-space outline and length. The bones' rest
// positions, paddles, and walrus tusks remain in the scaffold.
type Profile struct {
	Kind, Form, Status, By, Date, Source       string
	Cervical, Thoracic, Lumbar, Sacral, Caudal int
	SternalRibs                                int `yaml:"sternal-ribs"`
	Sternebrae                                 int
	AnkleForward                               float64 `yaml:"ankle-forward"`
	Length                                     *float64
	Outline                                    [][4]float64
	Loft                                       *loftProfile
	WalrusOutline                              [][5]float64 `yaml:"walrus-outline"`
	Evidence                                   map[string]pinnipedEvidence
	Studies                                    map[string]pinnipedStudy
}

// loftProfile records the picked coat and its measured outline reference.
// The source measurements are shared by the two lofted forms, while the
// selected radius and colours belong to each profile.
type loftProfile struct {
	Axillary       float64
	AxillaryBasis  string `yaml:"axillary-basis"`
	AxillarySource string `yaml:"axillary-source"`
	PeltBasis      string `yaml:"pelt-basis"`
	PeltSource     string `yaml:"pelt-source"`
	Pelt           loftPelt
	Reference      sealSource
}

type loftPelt struct {
	Back, Belly, Mark, Flipper, Pad [3]float64
	BackLevel                       float64 `yaml:"back-level"`
	BellyLevel                      float64 `yaml:"belly-level"`
	Depth                           float64
	RingLight                       float64 `yaml:"ring-light"`
}

// The detailed and bay studies use their own drawn-profile rigs. Their
// material settings are profile data; their geometry does not use the loft
// skeleton counts stored alongside them.
type pinnipedStudy struct {
	Scaffold, Status, By, Date, Source string
	Wet, Nostrils                      *float64
}

type pinnipedEvidence struct {
	Basis, Source string
	Value         *float64
}

// Book is an immutable checked book of pinniped anatomy.
type Book struct{ profiles map[string]Profile }

// Read reads a bounded authored anatomy book.
func Read(file string, raw []byte) (*Book, error) {
	rows, err := profilecheck.Rows[Profile](file, raw, "pinniped book", "pinniped profile", func(_ *yaml.Node, path string) bool { return path == "outline" || path == "walrus-outline" })
	if err != nil {
		return nil, err
	}
	book := &Book{profiles: make(map[string]Profile, len(rows))}
	for _, p := range rows {
		if !profilecheck.Authored(p.Kind, p.Status, p.By, p.Date, p.Source) {
			return nil, fmt.Errorf("%s: invalid pinniped provenance for %q", file, p.Kind)
		}
		if _, exists := book.profiles[p.Kind]; exists {
			return nil, fmt.Errorf("%s: duplicate pinniped %q", file, p.Kind)
		}
		figures := map[string]float64{"cervical": float64(p.Cervical), "thoracic": float64(p.Thoracic),
			"lumbar": float64(p.Lumbar), "sacral": float64(p.Sacral), "caudal": float64(p.Caudal),
			"sternal-ribs": float64(p.SternalRibs), "sternebrae": float64(p.Sternebrae), "ankle-forward": p.AnkleForward}
		if len(p.Evidence) != len(figures) {
			return nil, fmt.Errorf("%s: pinniped %q needs evidence for each figure", file, p.Kind)
		}
		for name, actual := range figures {
			evidence, ok := p.Evidence[name]
			if !ok || (evidence.Basis != "picked" && evidence.Basis != "read" && evidence.Basis != "derived") {
				return nil, fmt.Errorf("%s: pinniped %q has invalid evidence for %s", file, p.Kind, name)
			}
			if evidence.Basis != "picked" && (evidence.Source == "" || evidence.Value == nil || *evidence.Value != actual) {
				return nil, fmt.Errorf("%s: pinniped %q needs a source and matching value for %s", file, p.Kind, name)
			}
			if evidence.Basis == "picked" && evidence.Value != nil && *evidence.Value != actual {
				return nil, fmt.Errorf("%s: pinniped %q has stale evidence for %s", file, p.Kind, name)
			}
		}
		if p.Form != "harbor-loft" && p.Form != "monk-loft" && p.Form != "walrus" {
			return nil, fmt.Errorf("%s: unsupported pinniped form %q", file, p.Form)
		}
		if p.Form == "walrus" {
			if p.Length != nil || p.Outline != nil || p.Loft != nil || !validWalrusStations(p.WalrusOutline) {
				return nil, fmt.Errorf("%s: invalid walrus outline for %q", file, p.Kind)
			}
		} else if p.WalrusOutline != nil || p.Length == nil || !finite(*p.Length) || *p.Length < 1 || *p.Length > 4 || !validSealOutline(p.Outline) {
			return nil, fmt.Errorf("%s: invalid loft outline or length for pinniped %q", file, p.Kind)
		} else if err := validateLoftProfile(p.Loft); err != nil {
			return nil, fmt.Errorf("%s: pinniped %q: %w", file, p.Kind, err)
		}
		if len(p.Studies) > 32 {
			return nil, fmt.Errorf("%s: too many pinniped studies for %q", file, p.Kind)
		}
		for name, study := range p.Studies {
			if name == "" || p.Form != "harbor-loft" || study.By == "" || study.Date == "" ||
				(study.Status != "picked" && study.Status != "read" && study.Status != "derived") ||
				((study.Status == "read" || study.Status == "derived") && study.Source == "") {
				return nil, fmt.Errorf("%s: invalid pinniped study %q", file, name)
			}
			switch study.Scaffold {
			case "detailed-rig":
				if study.Wet == nil || study.Nostrils == nil || !finite(*study.Wet) || *study.Wet < 0 || *study.Wet > 1 ||
					!finite(*study.Nostrils) || *study.Nostrils < 0 || *study.Nostrils > 1 {
					return nil, fmt.Errorf("%s: invalid pinniped study look %q", file, name)
				}
			case "bay-rig":
				if study.Wet != nil || study.Nostrils != nil {
					return nil, fmt.Errorf("%s: bay study %q has fixed materials", file, name)
				}
			default:
				return nil, fmt.Errorf("%s: unknown pinniped study scaffold %q", file, study.Scaffold)
			}
		}
		if p.Cervical < 7 || p.Cervical > 32 || p.Thoracic < 10 || p.Thoracic > 32 || p.Lumbar < 6 && p.Form == "walrus" || p.Lumbar < 5 || p.Lumbar > 32 || p.Sacral < 1 || p.Sacral > 8 || p.Caudal < 2 || p.Caudal > 32 || p.SternalRibs < 5 || p.SternalRibs > p.Thoracic || p.Sternebrae < 1 || p.Sternebrae > 32 || math.IsNaN(p.AnkleForward) || math.IsInf(p.AnkleForward, 0) || p.AnkleForward < 0 || p.AnkleForward > math.Pi {
			return nil, fmt.Errorf("%s: invalid pinniped anatomy for %q", file, p.Kind)
		}
		book.profiles[p.Kind] = p
	}
	return book, nil
}

func Embedded() []byte { return append([]byte(nil), profileRaw...) }
func (book *Book) Get(name string) (Profile, bool) {
	if book == nil {
		return Profile{}, false
	}
	p, found := book.profiles[name]
	return p.snapshot(), found
}

func (p Profile) snapshot() Profile {
	p.Outline = slices.Clone(p.Outline)
	p.WalrusOutline = slices.Clone(p.WalrusOutline)
	p.Length = copyNumber(p.Length)
	if p.Loft != nil {
		loft := *p.Loft
		loft.Reference.Sites = slices.Clone(loft.Reference.Sites)
		p.Loft = &loft
	}
	p.Evidence = maps.Clone(p.Evidence)
	for name, evidence := range p.Evidence {
		evidence.Value = copyNumber(evidence.Value)
		p.Evidence[name] = evidence
	}
	p.Studies = maps.Clone(p.Studies)
	for name, study := range p.Studies {
		study.Wet, study.Nostrils = copyNumber(study.Wet), copyNumber(study.Nostrils)
		p.Studies[name] = study
	}
	return p
}

// validateLoftProfile limits authored colours and the measured reference to
// finite values without treating the fixed scaffold as a general rig.
func validateLoftProfile(loft *loftProfile) error {
	if loft == nil {
		return fmt.Errorf("missing loft data")
	}
	switch {
	case !finite(loft.Axillary) || loft.Axillary <= 0 || loft.Axillary > 1:
		return fmt.Errorf("invalid loft axillary")
	case !slices.Contains([]string{"picked", "read", "derived"}, loft.AxillaryBasis):
		return fmt.Errorf("invalid loft axillary-basis")
	case strings.TrimSpace(loft.AxillarySource) == "":
		return fmt.Errorf("missing loft axillary-source")
	case !slices.Contains([]string{"picked", "read", "derived"}, loft.PeltBasis):
		return fmt.Errorf("invalid loft pelt-basis")
	case strings.TrimSpace(loft.PeltSource) == "":
		return fmt.Errorf("missing loft pelt-source")
	}
	for _, field := range []struct {
		name   string
		values []float64
	}{
		{"back", loft.Pelt.Back[:]}, {"belly", loft.Pelt.Belly[:]}, {"mark", loft.Pelt.Mark[:]},
		{"flipper", loft.Pelt.Flipper[:]}, {"pad", loft.Pelt.Pad[:]},
		{"back-level", []float64{loft.Pelt.BackLevel}}, {"belly-level", []float64{loft.Pelt.BellyLevel}},
		{"depth", []float64{loft.Pelt.Depth}}, {"ring-light", []float64{loft.Pelt.RingLight}},
	} {
		for _, share := range field.values {
			if !finite(share) || share < 0 || share > 1 {
				return fmt.Errorf("invalid loft pelt %s", field.name)
			}
		}
	}
	source := loft.Reference
	switch {
	case strings.TrimSpace(source.Citation) == "":
		return fmt.Errorf("missing loft reference citation")
	case !finite(source.Length) || source.Length <= 0 || source.Length > 1000:
		return fmt.Errorf("invalid loft reference length")
	case !finite(source.Axillary) || source.Axillary <= 0 || source.Axillary > source.Length:
		return fmt.Errorf("invalid loft reference axillary")
	case len(source.Sites) < 2 || len(source.Sites) > 64:
		return fmt.Errorf("invalid loft reference sites count")
	}
	seen := make(map[string]bool, len(source.Sites))
	last := -1.0
	for i, site := range source.Sites {
		if strings.TrimSpace(site.Name) == "" || seen[site.Name] {
			return fmt.Errorf("invalid loft reference site %d name", i)
		}
		if !finite(site.Behind) || site.Behind < 0 || site.Behind > source.Length || site.Behind <= last {
			return fmt.Errorf("invalid loft reference site %s behind", site.Name)
		}
		if !finite(site.Round) || site.Round <= 0 || site.Round > source.Length {
			return fmt.Errorf("invalid loft reference site %s round", site.Name)
		}
		seen[site.Name], last = true, site.Behind
	}
	return nil
}

func copyNumber(p *float64) *float64 {
	if p == nil {
		return nil
	}
	value := *p
	return &value
}

var defaultBook = func() *Book {
	book, err := Read("profiles.yaml", profileRaw)
	if err != nil {
		panic(err)
	}
	return book
}()

func Default() *Book { return defaultBook }
func defaultPinniped(name string) Profile {
	p, found := defaultBook.Get(name)
	if !found {
		panic("missing pinniped " + name)
	}
	return p
}

type pinnipedProfile = Profile

// sealOutline returns the profile's stations as a private copy. The lofted
// scaffold uses its shared rest space; only the animal's length scales it.
func (p Profile) sealOutline() sealOutline {
	outline := make(sealOutline, len(p.Outline))
	for i, station := range p.Outline {
		outline[i] = sealStation{x: station[0], high: station[1], wide: station[2], centre: station[3]}
	}
	return outline
}

// validSealOutline bounds the authored cross sections and their interpolated
// radii. Endpoints stay fixed because the shared skeleton uses that rest space.
func validSealOutline(stations [][4]float64) bool {
	if len(stations) < 2 || len(stations) > 64 || stations[0][0] != sealTailTipX || stations[len(stations)-1][0] != sealNoseX {
		return false
	}
	for i, row := range stations {
		for _, v := range row {
			if !finite(v) {
				return false
			}
		}
		if row[1] <= 0 || row[1] > .75 || row[2] <= 0 || row[2] > .75 || math.Abs(row[3]) > .5 || (i > 0 && row[0] <= stations[i-1][0]) {
			return false
		}
	}
	outline := (Profile{Outline: stations}).sealOutline()
	for i := 1; i < len(outline); i++ {
		for j := 1; j < 8; j++ {
			x := outline[i-1].x + (outline[i].x-outline[i-1].x)*float64(j)/8
			section := outline.at(x)
			if !finite(section.height) || !finite(section.width) || section.height <= 0 || section.width <= 0 {
				return false
			}
		}
	}
	return true
}

func validWalrusStations(rows [][5]float64) bool {
	if len(rows) < 2 || len(rows) > 64 {
		return false
	}
	for i, row := range rows {
		for _, value := range row {
			if !finite(value) {
				return false
			}
		}
		if row[0] < 0 || row[0] > 4 || row[1] <= 0 || row[1] > 1 || row[2] <= row[3] || row[2] > 2 || row[3] < -1 || row[4] < 1 || row[4] > 5 || (i > 0 && row[0] <= rows[i-1][0]) {
			return false
		}
	}
	return true
}

// Ribs visits each thoracic rib in skeleton insertion order.
func Ribs(count int, add func(int, float64) error) error {
	for number := 1; number <= count; number++ {
		for _, side := range [...]float64{-1, 1} {
			if err := add(number, side); err != nil {
				return err
			}
		}
	}
	return nil
}

// SpinePose distributes a body turn over its chosen vertebrae,
// weighted by each joint's reach. Each species supplies its joint axes and
// limits, as well as which part of the spine moves.
func SpinePose(pose skeleton.Pose, names []string, flex, bend float64,
	weight func(string) (float64, float64),
	turn func(skeleton.Pose, string, float64, float64) (skeleton.Pose, error),
) (skeleton.Pose, error) {
	var flexes, bends float64
	for _, name := range names {
		f, b := weight(name)
		flexes, bends = flexes+f, bends+b
	}
	if (flex != 0 && flexes == 0) || (bend != 0 && bends == 0) {
		return pose, fmt.Errorf("pinniped: spine cannot take its requested turn")
	}
	for _, name := range names {
		f, b := weight(name)
		var a, c float64
		if flexes != 0 {
			a = flex * f / flexes
		}
		if bends != 0 {
			c = bend * b / bends
		}
		var err error
		pose, err = turn(pose, name, a, c)
		if err != nil {
			return pose, err
		}
	}
	return pose, nil
}

func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }
