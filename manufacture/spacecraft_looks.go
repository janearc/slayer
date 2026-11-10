package manufacture

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// spacecraftLooks uses authored display colours, not optical constants.
func spacecraftLooks(design string) looks {
	mirror := maths.Grey(spacecraftValue("white"))
	if design == "webb" {
		mirror = maths.RGB(spacecraftValue("gold-red"),
			spacecraftValue("gold-green"),
			spacecraftValue("gold-blue"))
	}
	white := maths.Grey(spacecraftValue("white"))
	solar := maths.RGB(spacecraftValue("solar-red"),
		spacecraftValue("solar-green"), spacecraftValue("solar-blue"))
	structure := maths.Grey(spacecraftValue("structure"))
	foil := maths.Grey(spacecraftValue("foil"))
	return looks{
		body: thing.Polished("telescope housing",
			white, spacecraftValue("gloss"),
			spacecraftValue("index")),
		leaf: thing.Diffuse("solar array",
			solar),
		spaceMirror: thing.Polished("primary mirror", mirror,
			spacecraftValue("gloss"), spacecraftValue("index")),
		spaceSecondary: thing.Polished("secondary mirror", mirror,
			spacecraftValue("gloss"), spacecraftValue("index")),
		spaceStructure: thing.Diffuse("spacecraft structure",
			structure),
		spaceFoil: thing.Polished("thermal shield",
			foil, spacecraftValue("gloss"),
			spacecraftValue("index")),
	}
}
