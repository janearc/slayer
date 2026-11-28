package manufacture

import (
	"github.com/janearc/slayer/manufacture/animals/vertebrates/mammals/primates"
	"github.com/janearc/slayer/materials/fibre"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/scene/thing"
)

// hairStyle gives the study populations independent materials and draws.
func (row adultHumanStudy) hairStyle(from seed) (primates.HumanHair, error) {
	var style primates.HumanHair
	if !row.bareScalp {
		material, err := row.scalpMaterial()
		if err != nil {
			return primates.HumanHair{}, err
		}
		style.Scalp = &primates.ScalpHair{
			Seed: from.number, Length: longestStudyStrand, Material: material,
			Line: primates.Hairline{FrontRise: .003, TempleRecession: .005, Peak: .0015, Asymmetry: .002},
			Part: -.018, Curl: .22,
		}
		if row.mediumScalp {
			style.Scalp.Style = primates.PartedMedium
			style.Scalp.Length = 0
			style.Scalp.Curl = .39
			style.Scalp.MediumSweep = .94
			style.Scalp.CrownLift = row.crownLift
			style.Scalp.ScalpField = row.scalpField
			if row.fieldWisps {
				style.Scalp.FieldWisps = &primates.ScalpWispShape{BlendPart: true}
			}
		}
	}
	if row.bodyHair {
		model, err := fibre.New(fibre.Optics{
			Index: 1.55, Absorption: maths.RGB(.50, .87, 1.33),
			Longitudinal: .36, Azimuthal: .43, Cuticle: .04,
		})
		if err != nil {
			return primates.HumanHair{}, err
		}
		style.Body = &primates.BodyHair{
			Seed:    noise.Mix(from.number ^ 0x626f64792d686169),
			Density: 65_000, Length: .014, Radius: .000071,
			Regions:  primates.BodyHairRegions{Arms: .80, Legs: .70},
			Material: &thing.Material{Name: "forearm and shin fibres", Fibre: model},
		}
	}
	if row.facialHair {
		style.Eyebrows = &primates.Eyebrows{
			Seed:  noise.Mix(from.number ^ 0x65796562726f7773),
			Left:  primates.BrowShape{Rise: .00048, Fullness: 1.04},
			Right: primates.BrowShape{Arch: -.00055, Fullness: .96},
			Material: &thing.Material{
				Name: "brown eyebrow hair", Colour: maths.RGB(.044, .025, .016),
				Gloss: .10, Index: 1.54, Satin: .52,
			},
		}
		style.Eyelashes = &primates.Eyelashes{
			Seed:  noise.Mix(from.number ^ 0x6579656c61736865),
			Upper: &primates.LashLine{Curl: .43},
			Lower: &primates.LashLine{Curl: .25},
			Material: &thing.Material{
				Name: "dark eyelash hair", Colour: maths.RGB(.023, .013, .009),
				Gloss: .16, Index: 1.54, Satin: .47,
			},
		}
	}
	if row.vellusHair {
		style.Vellus = &primates.VellusHair{
			Seed:    noise.Mix(from.number ^ 0x76656c6c75736872),
			Density: 150_000, Length: .0026, Radius: .000016,
			Regions: primates.VellusRegions{Cheeks: .94, Fingers: .81, Toes: .74},
			Material: &thing.Material{
				Name: "light fine skin hair", Colour: maths.RGB(.32, .218, .147),
				Gloss: .15, Index: 1.54, Satin: .47,
			},
		}
	}
	return style, nil
}
