package manufacture

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// spacecraftLooks uses authored display colours, not optical constants.
// Webb's mirrors and sunshield are then made over from its surfaces file,
// which names real metals; see webbSurfaces.
func spacecraftLooks(design string) looks {
	mirror := maths.Grey(spacecraftValue("white"))
	if design == "webb" {
		mirror = maths.RGB(spacecraftValue("gold-red"),
			spacecraftValue("gold-green"),
			spacecraftValue("gold-blue"))
	}
	white := maths.Grey(spacecraftValue("white"))
	if design == "prima" {
		white = maths.RGB(primaValue("hull-red"), primaValue("hull-green"),
			primaValue("hull-blue"))
	}
	solar := maths.RGB(spacecraftValue("solar-red"),
		spacecraftValue("solar-green"), spacecraftValue("solar-blue"))
	structure := maths.Grey(spacecraftValue("structure"))
	foil := maths.Grey(spacecraftValue("foil"))
	if design == "roman" {
		white = maths.RGB(spacecraftValue("roman-housing-red"),
			spacecraftValue("roman-housing-green"),
			spacecraftValue("roman-housing-blue"))
		solar = maths.RGB(spacecraftValue("roman-solar-red"),
			spacecraftValue("roman-solar-green"),
			spacecraftValue("roman-solar-blue"))
		structure = maths.Grey(spacecraftValue("roman-structure-reflectance"))
		foil = maths.RGB(spacecraftValue("roman-foil-red"),
			spacecraftValue("roman-foil-green"),
			spacecraftValue("roman-foil-blue"))
	}
	panelColour := structure
	goldColour := maths.Grey(spacecraftValue("white"))
	if design == "prima" {
		mirror = maths.RGB(primaValue("mirror-red"), primaValue("mirror-green"),
			primaValue("mirror-blue"))
		structure = maths.RGB(primaValue("truss-red"), primaValue("truss-green"),
			primaValue("truss-blue"))
		foil = maths.RGB(primaValue("canopy-red"), primaValue("canopy-green"),
			primaValue("canopy-blue"))
		panelColour = maths.RGB(primaValue("panel-red"),
			primaValue("panel-green"), primaValue("panel-blue"))
		goldColour = maths.RGB(primaValue("gold-red"),
			primaValue("gold-green"), primaValue("gold-blue"))
	}
	bodyLook := thing.Polished("telescope housing",
		white, spacecraftValue("gloss"), spacecraftValue("index"))
	foilLook := thing.Polished("thermal shield", foil,
		spacecraftValue("gloss"), spacecraftValue("index"))
	if design == "prima" {
		bodyLook = thing.Diffuse("telescope housing", white)
		foilLook = thing.Diffuse("thermal shield", foil)
	}
	panelLook := thing.Diffuse("radiator panel", panelColour)
	goldLook := thing.Diffuse("gold instrument hardware", goldColour)
	trussLook := thing.Diffuse("spacecraft structure", structure)
	primaryLook := thing.Polished("primary mirror", mirror,
		spacecraftValue("gloss"), spacecraftValue("index"))
	secondaryLook := thing.Polished("secondary mirror", mirror,
		spacecraftValue("gloss"), spacecraftValue("index"))
	if design == "prima" {
		// Conductor constants below are visual analogues for silver,
		// aluminium and gold. PRIMA's actual coating stack is unspecified.
		panelLook = &thing.Material{Name: "silver radiator panel",
			Colour: panelColour, Gloss: 0.88, Metal: thing.Silver()}
		goldLook = &thing.Material{Name: "gold instrument hardware",
			Colour: goldColour, Gloss: 0.88, Metal: thing.Gold()}
		trussLook = &thing.Material{Name: "silver telescope truss",
			Colour: structure, Gloss: 0.72, Metal: thing.Aluminium()}
		foilLook = thing.Polished("thermal shield", foil,
			0.13, 1.7)
		primaryLook = &thing.Material{Name: "primary mirror",
			Colour: mirror, Gloss: 0.95, Metal: thing.Silver()}
		secondaryLook = &thing.Material{Name: "secondary mirror",
			Colour: mirror, Gloss: 0.95, Metal: thing.Silver()}
	}
	made := looks{
		body: bodyLook,
		leaf: thing.Diffuse("solar array",
			solar),
		spaceMirror:    primaryLook,
		spaceSecondary: secondaryLook,
		spaceStructure: trussLook,
		spaceFoil:      foilLook,
		spacePanel:     panelLook,
		spaceGold:      goldLook,
	}
	if design == "webb" {
		made[solarCover] = thing.Polished("solar cell cover glass", solar,
			spacecraftValue("gloss"), spacecraftValue("index"))
		// the bars are polished as the mirror is, and are not the mirror
		made[solarBar] = thing.Polished("solar bus bar", mirror,
			spacecraftValue("gloss"), spacecraftValue("index"))
	}
	if design == "roman" {
		made[romanShade] = thing.Polished("black aperture shade fabric",
			maths.RGB(.024, .023, .028), .22, 1.55)
		made[romanSolarString] = thing.Diffuse("second solar cell string",
			solar.Scale(.93))
		made[romanSilverBlanket] = &thing.Material{Name: "silver thermal blanket",
			Colour: maths.Grey(.65), Gloss: .43, Metal: thing.Aluminium()}
		made[romanGoldBlanket] = &thing.Material{Name: "gold thermal blanket",
			Colour: maths.RGB(.77, .51, .12), Gloss: .58, Metal: thing.Gold()}
		// the array's metalwork is the colour of the structure, and a
		// mesh of its own
		made[solarMetal] = thing.Diffuse("solar array frame", structure)
	}
	return made
}
