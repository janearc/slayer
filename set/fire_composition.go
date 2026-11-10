package set

import (
	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/flame"
	yaml "gopkg.in/yaml.v3"
)

// fireComposition keeps wavelength provenance alongside authored line power.
func (reading reader) fireComposition(target **flame.CompositionSpec) step {
	return func(_, node *yaml.Node) error {
		spec := flame.CompositionSpec{}
		err := reading.keys(node, map[string]step{
			"continuum": reading.number(&spec.Continuum, 0, 1),
			"lines": func(_, lines *yaml.Node) error {
				if lines.Kind != yaml.SequenceNode || len(lines.Content) > 32 {
					return reading.refuse(lines, "lines takes at most 32 entries")
				}
				for _, entry := range lines.Content {
					line := flame.LineSpec{}
					var wavelength float64
					err := reading.keys(entry, map[string]step{
						"species": reading.text(&line.Species), "source": reading.text(&line.Source),
						"convention": reading.word(&line.Convention, "air", "vacuum"),
						"wavelength": reading.quantity(&wavelength, "nm", 1, 1e6),
						"radiance":   reading.quantity(&line.Radiance, "W/m^2/sr", 0, 1e12),
					}, nil)
					if err != nil {
						return err
					}
					if err = reading.all(entry, "emission line", "species", "source", "convention", "wavelength", "radiance"); err != nil {
						return err
					}
					line.Wavelength = measure.Metres(wavelength * 1e-9)
					spec.Lines = append(spec.Lines, line)
				}
				return nil
			},
		}, nil)
		if err != nil {
			return err
		}
		if err = reading.all(node, "composition", "continuum", "lines"); err != nil {
			return err
		}
		*target = &spec
		return nil
	}
}
