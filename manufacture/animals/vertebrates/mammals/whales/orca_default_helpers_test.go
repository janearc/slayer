package whales

import (
	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/manufacture/animals/vertebrates/mammals"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// Existing anatomical witnesses use the embedded orca, while profile tests
// construct other models directly and never change this default.
func orcaSkeleton() (*skeleton.Skeleton, error)    { return defaultOrcaModel().orcaSkeleton() }
func orcaRest() *orcaRestSkin                      { return defaultOrcaModel().orcaRest() }
func orcaTrunkProfile() *orcaTrunkTable            { return defaultOrcaModel().orcaTrunkProfile() }
func orcaBinding() ([][]skeleton.Influence, error) { return defaultOrcaModel().orcaBinding() }
func orcaField(p maths.Vec) float64                { return defaultOrcaModel().orcaField(p) }
func orcaMouthLine(x float64) float64              { return defaultOrcaModel().orcaMouthLine(x) }
func orcaTeeth(p maths.Vec) float64                { return defaultOrcaModel().orcaTeeth(p) }
func orcaEyeCentre() maths.Vec                     { return defaultOrcaModel().orcaEyeCentre() }
func orcaBind(bones *skeleton.Skeleton, spans []mammals.Span, p maths.Vec) []skeleton.Influence {
	return defaultOrcaModel().orcaBind(bones, spans, p)
}
func orcaPosed(bones *skeleton.Skeleton, pose skeleton.Pose, at maths.Subspace) ([]thing.Thing, error) {
	profile, _ := Default().Get("orca")
	return defaultOrcaModel().orcaPosed(bones, pose, at, profile.Palette, profile.Satin)
}
func orcaBones(at maths.Subspace) ([]thing.Thing, error) {
	return defaultOrcaModel().orcaBones(at)
}
func orcaForms(bones *skeleton.Skeleton) map[string]skeleton.BoneForm {
	return defaultOrcaModel().orcaForms(bones)
}

func orcaSwimming(phase float64, at maths.Subspace) ([]thing.Thing, error) {
	profile, _ := Default().Get("orca")
	return defaultOrcaModel().swimming(phase, at, profile.Palette, profile.Satin)
}
