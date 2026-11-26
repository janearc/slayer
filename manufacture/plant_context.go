package manufacture

// seasonalTree resolves a species in exactly one checked book. Embedded
// defaults are used only when no external book was supplied.
func seasonalTree(book *PlantProfiles, species string) (Kind, error) {
	if book == nil {
		book = defaultPlantProfiles
	}
	return book.NewPlant(species)
}
