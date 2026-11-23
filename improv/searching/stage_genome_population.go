package searching

import "github.com/janearc/slayer/maths"

// makeAuthoredMultitude builds one being for each authored population member.
func (s *heardStage) makeAuthoredMultitude(plan *stagePlan) error {
	if err := checkAuthoredPopulation(plan.Population); err != nil {
		return err
	}
	crowd := &multitude{beings: make([]crowdBeing, 0, len(plan.Population))}
	cache := anatomyCacheFor(plan)
	for index, authored := range plan.Population {
		identity := identityOf(authored.Identity)
		being := crowdBeing{
			at:    maths.V(authored.At[0], authored.At[1], authored.At[2]),
			scale: authored.Scale, phase: identity.Phase, identity: identity,
			family: identity.Purpose, system: index, members: 1,
		}
		if err := s.buildAuthoredIndividual(&being, authored.Anatomy, cache); err != nil {
			return err
		}
		crowd.beings = append(crowd.beings, being)
	}
	s.crowd = crowd
	return nil
}
