package set_test

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

func TestExplicitHeadingAndVisibleLauncherSurviveBothReaders(test *testing.T) {
	for _, source := range laserSources() {
		source = strings.Replace(source, "recipe: ridge telescope",
			"recipe: ridge telescope\n        heading: {value: 0, unit: deg}",
			singleLaser)
		source = strings.Replace(source, "    guide:\n",
			"    guide:\n      launcher: true\n", singleLaser)
		made, err := set.Read("heading.yaml", []byte(source))
		if err != nil {
			test.Fatal(err)
		}
		if !made.Lasers()[firstLaser].Launcher() {
			test.Fatal("lost launcher")
		}
		angle, said := made.Things()[firstLaser].Heading()
		if !said || angle != firstLaser {
			test.Fatal("explicit zero heading lost")
		}
		bad := strings.ReplaceAll(source, "unit: deg}", "unit: rad}")
		if _, err := set.Read("heading.yaml", []byte(bad)); err == nil {
			test.Fatal("heading accepted wrong units")
		}
	}
}
