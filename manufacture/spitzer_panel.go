package manufacture

import (
	_ "embed"
	"math"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/research"
	"github.com/janearc/slayer/scene/thing"
)

//go:embed spitzer-surfaces.yaml
var spitzerSurfaceRaw []byte

// spitzerCellSurface keeps the optical material and picked finish in data.
func spitzerCellSurface(raw []byte) (*thing.Material, error) {
	var recipe struct{ Material, Finish, By, Says string }
	if err := yamlfile.Read("spitzer-surfaces.yaml", raw, &recipe); err != nil {
		return nil, err
	}
	return surfaceOf("Spitzer solar cells", recipe.Material, recipe.Finish)
}

// spitzerPanelSize separates measured hardware from its authored placement.
type spitzerPanelSize struct {
	width, height, extension, cellWidth, cellHeight, thickness float64
}

// spitzerPanels scales the source dimensions with the requested aperture.
func spitzerPanels(aperture float64) spitzerPanelSize {
	scale := aperture / research.Value("spitzer-primary-diameter-centimetres")
	return spitzerPanelSize{
		width:      scale * research.Value("spitzer-active-panel-width-centimetres"),
		height:     scale * research.Value("spitzer-active-panel-length-centimetres"),
		extension:  scale * research.Value("spitzer-shield-extension-centimetres"),
		cellWidth:  scale * research.Value("spitzer-cell-long-centimetres"),
		cellHeight: scale * research.Value("spitzer-cell-short-centimetres"),
		thickness:  aperture * spacecraftValue("panel-thickness"),
	}
}

// spitzerPanel builds both chevron leaves and the published 392 cells on
// each. The seven columns, folding angle and thickness are picked scenery
// choices, not recovered engineering drawings. See doc/spitzer-maturity.md.
func (row spacecraft) spitzerPanel(assembly *instrumentAssembly, size spacecraftSize) {
	look, err := spitzerCellSurface(spitzerSurfaceRaw)
	if err != nil {
		assembly.trouble = err
		return
	}
	look.Name = "solar array"
	assembly.object.dressed[leaf] = look
	panel := spitzerPanels(size.aperture)
	columns := int(spacecraftValue("spitzer-cell-columns"))
	rows := int(research.Value("spitzer-cells-per-panel")) / columns
	xStep, yStep := panel.width/float64(columns), panel.height/float64(rows)
	for _, side := range []float64{-spaceOne, spaceOne} {
		angle := side * spacecraftValue("spitzer-chevron-degrees") * math.Pi / 180
		seam := size.aperture * spacecraftValue("spitzer-seam-gap")
		pose := maths.NewSubspace(maths.V(side*seam*spaceHalf, spaceZero,
			-size.aperture*spacecraftValue("spitzer-panel-offset")),
			maths.Rotation(maths.YAxis, angle), spaceOne)
		left := math.Min(spaceZero, side*panel.width)
		start := len(assembly.object.parts)
		assembly.box(panel.width, panel.height, panel.thickness,
			maths.V(left, spaceZero, spaceZero), spaceStructure)
		assembly.box(panel.width, panel.extension, panel.thickness,
			maths.V(left, panel.height, spaceZero), spaceFoil)
		for column := range columns {
			x := left + (float64(column)+spaceHalf)*xStep - panel.cellWidth*spaceHalf
			for line := range rows {
				y := (float64(line)+spaceHalf)*yStep - panel.cellHeight*spaceHalf
				assembly.box(panel.cellWidth, panel.cellHeight, panel.thickness*spaceHalf,
					maths.V(x, y, -panel.thickness*spaceHalf), leaf)
			}
		}
		// Continuous reflector strips occupy the spaces between columns.
		// They are counted as strips, not as invented individual tiles.
		gap := xStep - panel.cellWidth
		for strip := range columns + spaceOne {
			x, width := left+float64(strip)*xStep-gap*spaceHalf, gap
			if strip == spaceZero {
				x, width = left, gap*spaceHalf
			} else if strip == columns {
				width = gap * spaceHalf
			}
			assembly.box(width, panel.height, panel.thickness*spaceHalf,
				maths.V(x, spaceZero, -panel.thickness*spaceHalf), spaceFoil)
		}
		for index := start; index < len(assembly.object.parts); index++ {
			part := &assembly.object.parts[index]
			part.space = part.space.Within(pose)
		}
	}
}
