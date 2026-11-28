package draft

import (
	"fmt"
	"strconv"

	"gopkg.in/yaml.v3"
)

// These bounds match the physical lens in the set reader and ecosystem schema.
const (
	minFocalMM = 1
	maxFocalMM = 100000
	minFStop   = 0.5
	maxFStop   = 1000
)

// PhysicalLens holds the lens's focal length in millimetres and its f-number.
type PhysicalLens struct {
	FocalMM float64
	FStop   float64
}

// PhysicalLens reads the optics of an authored physical set rig.
func (document *Document) PhysicalLens() (PhysicalLens, bool) {
	_, film := document.physicalFilm()
	return physicalLens(film)
}

// SetPhysicalLens changes the focal length and f-stop together.
func (document *Document) SetPhysicalLens(lens PhysicalLens) error {
	if err := lens.check(); err != nil {
		return err
	}
	tree, film := document.physicalFilm()
	current, ok := physicalLens(film)
	if !ok {
		return fmt.Errorf("the set does not have a complete physical film rig")
	}
	if current == lens {
		return nil
	}
	nodes := member(film, "lens")
	setNumber(member(nodes, "focal"), lens.FocalMM)
	setNumber(member(nodes, "stop"), lens.FStop)
	return document.replacePhysicalFilm(tree, "lens")
}

// physicalLens reads the numeric optics of a complete physical rig.
func physicalLens(film *yaml.Node) (PhysicalLens, bool) {
	if _, ok := physicalFrame(film); !ok {
		return PhysicalLens{}, false
	}
	nodes := member(film, "lens")
	focal, focalErr := strconv.ParseFloat(member(nodes, "focal").Value, 64)
	stop, stopErr := strconv.ParseFloat(member(nodes, "stop").Value, 64)
	lens := PhysicalLens{FocalMM: focal, FStop: stop}
	return lens, focalErr == nil && stopErr == nil && lens.check() == nil
}

// check also rejects NaN and infinities, including on files without a schema.
func (lens PhysicalLens) check() error {
	if !(lens.FocalMM >= minFocalMM && lens.FocalMM <= maxFocalMM) {
		return fmt.Errorf("focal length must be between %d and %d mm", minFocalMM, maxFocalMM)
	}
	if !(lens.FStop >= minFStop && lens.FStop <= maxFStop) {
		return fmt.Errorf("f-stop must be between %g and %d", minFStop, maxFStop)
	}
	return nil
}
