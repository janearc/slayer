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
// choices, not recovered engineering drawings. See
// doc/model/spacecraft/spitzer-maturity.md.
func (row spacecraft) spitzerPanel(assembly *instrumentAssembly, size spacecraftSize) {
	look, err := spitzerCellSurface(spitzerSurfaceRaw)
	if err != nil {
		assembly.trouble = err
		return
	}
	look.Name = "solar array"
	assembly.object.dressed[leaf] = look
	panel := spitzerPanels(size.aperture)
	for _, side := range []float64{-spaceOne, spaceOne} {
		// the rails across a leaf's ends stand a rail's width past its
		// sides, so the leaf stands that far up from the floor, and the
		// lowest face of its frame is on the floor and not under it
		rise := spitzerLeaf(panel, side, maths.Local()).rail()
		pose := spitzerChevron(side, size.aperture, rise)
		spitzerLeaf(panel, side, pose).lay(assembly)
		// the upper extension is a bare panel in line with the leaf,
		// past the end of its cells; it is shorter by the rise, so that
		// the leaf and it end at the published length
		left := math.Min(spaceZero, side*panel.width)
		assembly.boxAt(panel.width, panel.extension-rise, panel.thickness,
			maths.V(left, panel.height, spaceZero), pose, spaceFoil)
	}
}

// spitzerChevron is where a leaf stands: turned by the chevron angle
// about the seam, a seam gap apart from the other leaf, behind the
// optical axis, and a rise up from the floor.
func spitzerChevron(side, aperture, rise float64) maths.Subspace {
	angle := side * spacecraftValue("spitzer-chevron-degrees") * math.Pi / 180
	seam := aperture * spacecraftValue("spitzer-seam-gap")
	return maths.NewSubspace(maths.V(side*seam*spaceHalf, rise,
		-aperture*spacecraftValue("spitzer-panel-offset")),
		maths.Rotation(maths.YAxis, angle), spaceOne)
}

// spitzerLeaf is one leaf as an array: a single panel out from the seam,
// its cells on the face towards -z at the published size, packed into the
// chosen columns, with a bus bar filling each gap between columns. The
// leaf has no yoke, as it hangs from the seam.
func spitzerLeaf(panel spitzerPanelSize, side float64,
	pose maths.Subspace,
) solarArray {
	columns := int(spacecraftValue("spitzer-cell-columns"))
	rows := int(research.Value("spitzer-cells-per-panel")) / columns
	middle := maths.V(spaceZero, panel.height*spaceHalf,
		panel.thickness*spaceHalf)
	return solarArray{
		frame: solarFrame(middle, maths.V(side, spaceZero, spaceZero),
			maths.V(spaceZero, spaceZero, -spaceOne)).Within(pose),
		panels: solarPanels{along: spaceOne, across: spaceOne,
			width: panel.width, depth: panel.height,
			thickness: panel.thickness},
		cells: solarCells{columns: columns, rows: rows,
			width: panel.cellWidth, depth: panel.cellHeight,
			height: panel.thickness * spaceHalf,
			pitchX: panel.width / float64(columns),
			pitchZ: panel.height / float64(rows), barShare: spaceOne},
		looks: solarLooks{cell: leaf, bar: spaceFoil,
			substrate: spaceStructure, frame: spacePanel,
			hardware: spaceStructure},
	}
}
