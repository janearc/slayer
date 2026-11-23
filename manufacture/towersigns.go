package manufacture

import "github.com/janearc/slayer/manufacture/cities"

// signSalt is mixed into the seed of the signs a tower carries, so that
// what they say is another draw than the tower's own.
const signSalt = 0x5349474e53544f57

// signed is the good with the signs of a tower on it, if its recipe pins
// signs on: package cities draws a sign on each place the tower offers by
// the chance the sign culture gives it, at the detail of the tower, and
// they are made a good and gathered with it. A tower whose recipe does not
// pin signs on, or that offers no place a sign was drawn for, is returned
// as it is.
func signed(made good, spec TowerSpec, tower *cities.Tower, from seed,
	detail string) (good, error) {
	if spec.Signs == nil || !*spec.Signs {
		return made, nil
	}
	parts, err := cities.SignsIn(tower, cities.Seed(from.number^signSalt),
		cities.ConstantLevel(cities.Detail(detail)), spec.Lettering)
	if err != nil || len(parts) == 0 {
		return made, err
	}
	drawn, err := cities.Design{Name: "tower signs",
		Elements: parts}.Rendered(cities.Detail(detail))
	if err != nil {
		return good{}, err
	}
	hung, err := goodOfRendered(drawn)
	if err != nil {
		return good{}, err
	}
	return gather(made, hung)
}
