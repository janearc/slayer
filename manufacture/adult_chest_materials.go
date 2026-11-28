package manufacture

import (
	"github.com/janearc/slayer/manufacture/animals/vertebrates/mammals/primates"
	"github.com/janearc/slayer/maths"
)

type chestMaterialStudy struct {
	skin   primates.SkinTone
	region primates.AreolarTone
}

func chestMaterialRows() map[string]maker {
	low := primates.AreolarTone{
		Left: primates.AreolarSideTone{
			Areola:  primates.PigmentShift{Melanin: -.62},
			Papilla: primates.PigmentShift{Melanin: -.35},
		},
		Right: primates.AreolarSideTone{
			Areola:  primates.PigmentShift{Melanin: -.54},
			Papilla: primates.PigmentShift{Melanin: -.27},
		},
	}
	high := primates.AreolarTone{
		Left: primates.AreolarSideTone{
			Areola:  primates.PigmentShift{Melanin: .38},
			Papilla: primates.PigmentShift{Melanin: .20},
		},
		Right: primates.AreolarSideTone{
			Areola:  primates.PigmentShift{Melanin: .48},
			Papilla: primates.PigmentShift{Melanin: .28},
		},
	}
	chroma := high
	chroma.Left.Areola.Chroma = primates.Chroma{Red: .58, Gold: -.32}
	chroma.Right.Areola.Chroma = primates.Chroma{Red: .51, Gold: -.25}
	chroma.Left.Papilla.Chroma = primates.Chroma{Red: .30, Gold: -.18}
	chroma.Right.Papilla.Chroma = primates.Chroma{Red: .36, Gold: -.18}

	complexions := [...]struct {
		name    string
		melanin float64
	}{{"light", .15}, {"medium", .47}, {"dark", .85}}
	variants := [...]struct {
		name string
		tone primates.AreolarTone
	}{{"low-contrast", low}, {"high-contrast", high}, {"high-chroma", chroma}}

	rows := make(map[string]maker, len(complexions)*len(variants))
	for _, complexion := range complexions {
		for _, variant := range variants {
			name := "adult-chest-" + complexion.name + "-" + variant.name
			rows[name] = chestMaterialStudy{
				skin:   primates.SkinTone{Melanin: complexion.melanin, Vascular: .50},
				region: variant.tone,
			}
		}
	}
	return rows
}

func (row chestMaterialStudy) extent(from seed) (maths.Box, error) {
	// The neutral study uses the same anatomy. Its allowance for the
	// scalp groom is conservative for this bare mesh.
	return (adultHumanStudy{bareScalp: true}).extent(from)
}

func (row chestMaterialStudy) made(from seed) (good, error) {
	spec := primates.AdultHuman().WithSkinTone(row.skin).WithAreolarTone(row.region)
	return (nativeChestStudy{spec}).made(from)
}
