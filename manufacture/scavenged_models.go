package manufacture

import (
	"fmt"
	avian "github.com/janearc/slayer/manufacture/animals/vertebrates/dinosaurs/theropods/birds"
	"sync"
)

// Retired catalogue labels remain private to the default cluster's seeded
// draw. They are never offered as factory kinds or accepted as asked parts.
// Books own the anatomy; these closures select only historical variants.
var scavengedModels = sync.OnceValues(func() (map[string]func() (Kind, error), error) {
	models := map[string]func() (Kind, error){}
	var collision error
	add := func(name string, makeModel func() (Kind, error)) {
		if collision == nil {
			collision = addScavengedModel(models, name, makeModel)
		}
	}
	if kelpError != nil {
		return nil, kelpError
	}
	for _, forest := range []bool{false, true} {
		prefix := "giant-kelp"
		if forest {
			prefix = "kelp-forest"
		}
		add(prefix, func() (Kind, error) { return defaultKelpProfiles.NewGiantKelp(defaultKelpWater, forest) })
		for _, water := range []string{"slack", "moderate", "surge"} {
			add(prefix+"-"+water, func() (Kind, error) { return defaultKelpProfiles.NewGiantKelp(water, forest) })
		}
	}
	plants := defaultPlantProfiles
	for _, name := range []string{"valley-oak", "alder", "poplar", "manzanita", "chaparral-brush", "joshua-tree", "kandelia-obovata", "aegiceras-corniculatum", "avicennia-marina", "bruguiera-gymnorhiza", "sonneratia-apetala", "pandanus-tectorius", "scaevola-taccada", "metrosideros-polymorpha", "barringtonia-asiatica", "quercus-ilex", "erica-arborea", "salix-alba"} {
		add(name, func() (Kind, error) { return plants.NewPlant(name) })
	}
	for _, name := range []string{"spruce", "norway", "coast-redwood", "giant-sequoia"} {
		add(name, func() (Kind, error) { return defaultConiferProfiles.NewConifer(name) })
	}
	for _, name := range []string{"coast-redwood", "giant-sequoia"} {
		add(name+"-shoot", func() (Kind, error) { return defaultConiferProfiles.NewConiferShoot(name) })
	}
	fern, err := ReadFernProfiles("fern-profiles.yaml", fernProfilesRaw)
	if err != nil {
		return nil, err
	}
	add("fern", fern.NewGenericFern)
	cactus, err := ReadCactusProfiles("cactus-profiles.yaml", cactusProfilesRaw)
	if err != nil {
		return nil, err
	}
	add("saguaro", cactus.NewSaguaro)
	birds, err := ReadBirdProfiles("birds.yaml", birdRaw)
	if err != nil {
		return nil, err
	}
	for _, species := range []string{"black-faced-spoonbill", "little-egret", "grey-heron", "pacific-reef-heron", "yellow-legged-gull", "white-tern", "great-frigatebird", "white-tailed-tropicbird"} {
		for _, stance := range []string{avian.Standing, avian.Gliding, avian.Bones, avian.Outlined} {
			add("bird-"+species+"-"+stance, func() (Kind, error) { return birds.NewBird(species, stance) })
		}
	}
	fish, err := ReadFishProfiles("fish.yaml", fishRaw)
	if err != nil {
		return nil, err
	}
	for _, species := range []string{"pacific-bluefin-tuna", "california-yellowtail", "california-barracuda", "pacific-sardine", "convict-tang", "dusky-grouper", "mediterranean-chromis", "european-perch", "flathead-grey-mullet"} {
		for _, variant := range []struct {
			suffix  string
			options FishOptions
		}{
			{"", FishOptions{}}, {"-glide", FishOptions{Glide: true}}, {"-bones", FishOptions{Bones: true}},
		} {
			add(species+variant.suffix, func() (Kind, error) { return fish.NewFish(species, variant.options) })
		}
	}
	for _, school := range []struct {
		species string
		count   int
	}{
		{"pacific-sardine", 500}, {"pacific-sardine", 5000},
		{"mediterranean-chromis", 60}, {"flathead-grey-mullet", 120},
	} {
		add(fmt.Sprintf("%s-school-%d", school.species, school.count), func() (Kind, error) {
			return fish.NewFish(school.species, FishOptions{School: school.count})
		})
	}
	for _, variant := range []struct{ label, species, pose, representation, study string }{
		{"bay-seal", "harbor-seal", "", "", "bay"},
		{"harbor-seal", "harbor-seal", "", "", "detailed-wet"},
		{"harbor-seal-dry", "harbor-seal", "", "", "detailed-dry"},
		{"harbor-seal-hauled", "harbor-seal", "hauled", "", ""},
		{"harbor-seal-swimming", "harbor-seal", "swimming", "", ""},
		{"harbor-seal-bones", "harbor-seal", "", "bones", ""},
		{"hawaiian-monk-seal-hauled", "hawaiian-monk-seal", "hauled", "", ""},
		{"walrus-fresh-5", "walrus", "lying", "", ""},
		{"walrus-fresh-5-hauled-out", "walrus", "hauled-out", "", ""},
		{"walrus-fresh-5-swimming", "walrus", "swimming", "", ""},
		{"walrus-fresh-5-skeleton", "walrus", "", "bones", ""},
		{"walrus-fresh-5-volumes", "walrus", "", "volumes", ""},
	} {
		add(variant.label, func() (Kind, error) {
			if variant.study != "" {
				return defaultPinnipedProfiles.NewPinnipedStudy(variant.species, variant.study)
			}
			return defaultPinnipedProfiles.NewPinnipedRepresentation(variant.species, variant.pose, variant.representation)
		})
	}
	for _, variant := range []struct{ label, pose, representation string }{
		{"orca-clean-05", "swim", ""},
		{"orca-clean-05-spy-hop", "spy-hop", ""},
		{"orca-clean-05-bones", "", "bones"},
	} {
		add(variant.label, func() (Kind, error) {
			return defaultWhaleProfiles.NewOrca(variant.pose, variant.representation)
		})
	}
	return models, collision
})

// scavengedModel makes a candidate only after it is drawn. Its historical
// label remains in refusal messages and the cluster's report.
func scavengedModel(name string) (Kind, error) {
	models, err := scavengedModels()
	if err != nil {
		return Kind{}, err
	}
	if makeModel, ok := models[name]; ok {
		row, err := makeModel()
		row.name = name
		return row, err
	}
	return kindNamed(name)
}

// addScavengedModel rejects ambiguity between historical model families.
func addScavengedModel(models map[string]func() (Kind, error), name string, makeModel func() (Kind, error)) error {
	if _, exists := models[name]; exists {
		return fmt.Errorf("manufacture: duplicate private cluster candidate %q", name)
	}
	models[name] = makeModel
	return nil
}
