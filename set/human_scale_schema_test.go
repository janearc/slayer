package set_test

import (
	"os"
	"strings"
	"testing"

	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/set"
)

// Native catalogue figures must pass both public set schemas.
func TestScaleHumanPassesBothSetSchemas(t *testing.T) {
	raw, err := os.ReadFile("../sets/ferruccio-playground.yaml")
	if err != nil {
		t.Fatal(err)
	}
	for _, version := range []string{"v1", "v2"} {
		source := strings.ReplaceAll(string(raw), "ecosystem-v2.json",
			"ecosystem-"+version+".json")
		if version == "v1" {
			source = strings.ReplaceAll(source, "  setting: space\n", "")
		}
		if _, err := set.Read("native-human.yaml", []byte(source)); err != nil {
			t.Fatal(version, err)
		}
		broken := strings.ReplaceAll(source, "human-scale-2m", "human-scale-20m")
		if _, err := set.Read("unknown-human.yaml", []byte(broken)); err == nil {
			t.Fatal(version, "accepted unknown human kind")
		}
	}
}

func TestAlfaHeroPassesBothSetSchemasAndResolvesToAMaker(t *testing.T) {
	raw, err := os.ReadFile("../sets/ferruccio-playground.yaml")
	if err != nil {
		t.Fatal(err)
	}
	for _, version := range []string{"v1", "v2"} {
		source := strings.ReplaceAll(string(raw), "ecosystem-v2.json",
			"ecosystem-"+version+".json")
		if version == "v1" {
			source = strings.ReplaceAll(source, "  setting: space\n", "")
		}
		source = strings.Replace(source, "kind: human-scale-2m",
			"kind: alfa-hero", 1)
		made, err := set.Read("alfa-hero.yaml", []byte(source))
		if err != nil {
			t.Fatalf("%s schema refused alfa-hero: %v", version, err)
		}
		if len(made.Things()) < 2 || made.Things()[1].Kind() != "alfa-hero" {
			t.Fatalf("%s schema parsed the wrong thing: %+v", version, made.Things())
		}
		kind, err := manufacture.KindNamed(made.Things()[1].Kind())
		if err != nil {
			t.Fatalf("%s schema kind has no maker: %v", version, err)
		}
		if _, err := manufacture.MakeWithComplexity(kind,
			manufacture.Sow(5), "quick"); err != nil {
			t.Fatalf("%s schema kind cannot be constructed: %v", version, err)
		}
		broken := strings.Replace(source, "kind: alfa-hero",
			"kind: alfa-heroo", 1)
		if _, err := set.Read("unknown-alfa-hero.yaml", []byte(broken)); err == nil {
			t.Fatalf("%s schema accepted a near-miss hero kind", version)
		}
	}
}

func TestSOHOPassesBothSetSchemasAndResolvesToAMaker(t *testing.T) {
	raw, err := os.ReadFile("../sets/ferruccio-playground.yaml")
	if err != nil {
		t.Fatal(err)
	}
	for _, version := range []string{"v1", "v2"} {
		source := strings.ReplaceAll(string(raw), "ecosystem-v2.json",
			"ecosystem-"+version+".json")
		if version == "v1" {
			source = strings.ReplaceAll(source, "  setting: space\n", "")
		}
		source = strings.Replace(source, "kind: human-scale-2m",
			"kind: soho", 1)
		made, err := set.Read("soho.yaml", []byte(source))
		if err != nil {
			t.Fatalf("%s schema refused soho: %v", version, err)
		}
		if len(made.Things()) < 2 || made.Things()[1].Kind() != "soho" {
			t.Fatalf("%s schema parsed the wrong thing: %+v", version, made.Things())
		}
		kind, err := manufacture.KindNamed(made.Things()[1].Kind())
		if err != nil {
			t.Fatalf("%s schema kind has no maker: %v", version, err)
		}
		if _, err := manufacture.MakeWithComplexity(kind,
			manufacture.Sow(5), "quick"); err != nil {
			t.Fatalf("%s schema kind cannot be constructed: %v", version, err)
		}
		broken := strings.Replace(source, "kind: soho",
			"kind: sohoo", 1)
		if _, err := set.Read("unknown-soho.yaml", []byte(broken)); err == nil {
			t.Fatalf("%s schema accepted a near-miss hero kind", version)
		}
	}
}

func TestCassiniKindPassesBothSetSchemasAndResolvesToAMaker(t *testing.T) {
	raw, err := os.ReadFile("../sets/ferruccio-playground.yaml")
	if err != nil {
		t.Fatal(err)
	}
	for _, version := range []string{"v1", "v2"} {
		source := strings.ReplaceAll(string(raw), "ecosystem-v2.json",
			"ecosystem-"+version+".json")
		if version == "v1" {
			source = strings.ReplaceAll(source, "  setting: space\n", "")
		}
		source = strings.Replace(source, "kind: human-scale-2m",
			"kind: cassini", 1)
		made, err := set.Read("cassini.yaml", []byte(source))
		if err != nil {
			t.Fatalf("%s schema refused cassini: %v", version, err)
		}
		if len(made.Things()) < 2 || made.Things()[1].Kind() != "cassini" {
			t.Fatalf("%s schema parsed the wrong thing: %+v", version, made.Things())
		}
		kind, err := manufacture.KindNamed(made.Things()[1].Kind())
		if err != nil {
			t.Fatalf("%s schema kind has no maker: %v", version, err)
		}
		if _, err := manufacture.MakeWithComplexity(kind,
			manufacture.Sow(5), "quick"); err != nil {
			t.Fatalf("%s schema kind cannot be constructed: %v", version, err)
		}
		broken := strings.Replace(source, "kind: cassini",
			"kind: cassinio", 1)
		if _, err := set.Read("unknown-cassini.yaml", []byte(broken)); err == nil {
			t.Fatalf("%s schema accepted a near-miss hero kind", version)
		}
	}
}

func TestHuygensKindPassesBothSetSchemasAndResolvesToAMaker(t *testing.T) {
	raw, err := os.ReadFile("../sets/ferruccio-playground.yaml")
	if err != nil {
		t.Fatal(err)
	}
	for _, version := range []string{"v1", "v2"} {
		source := strings.ReplaceAll(string(raw), "ecosystem-v2.json",
			"ecosystem-"+version+".json")
		if version == "v1" {
			source = strings.ReplaceAll(source, "  setting: space\n", "")
		}
		source = strings.Replace(source, "kind: human-scale-2m",
			"kind: huygens", 1)
		made, err := set.Read("huygens.yaml", []byte(source))
		if err != nil {
			t.Fatalf("%s schema refused huygens: %v", version, err)
		}
		if len(made.Things()) < 2 || made.Things()[1].Kind() != "huygens" {
			t.Fatalf("%s schema parsed the wrong thing: %+v", version, made.Things())
		}
		kind, err := manufacture.KindNamed(made.Things()[1].Kind())
		if err != nil {
			t.Fatalf("%s schema kind has no maker: %v", version, err)
		}
		if _, err := manufacture.MakeWithComplexity(kind,
			manufacture.Sow(5), "quick"); err != nil {
			t.Fatalf("%s schema kind cannot be constructed: %v", version, err)
		}
		broken := strings.Replace(source, "kind: huygens",
			"kind: huygenso", 1)
		if _, err := set.Read("unknown-huygens.yaml", []byte(broken)); err == nil {
			t.Fatalf("%s schema accepted a near-miss hero kind", version)
		}
	}
}

func TestCassiniHuygensKindPassesBothSetSchemasAndResolvesToAMaker(t *testing.T) {
	raw, err := os.ReadFile("../sets/ferruccio-playground.yaml")
	if err != nil {
		t.Fatal(err)
	}
	for _, version := range []string{"v1", "v2"} {
		source := strings.ReplaceAll(string(raw), "ecosystem-v2.json",
			"ecosystem-"+version+".json")
		if version == "v1" {
			source = strings.ReplaceAll(source, "  setting: space\n", "")
		}
		source = strings.Replace(source, "kind: human-scale-2m",
			"kind: cassini-huygens", 1)
		made, err := set.Read("cassini-huygens.yaml", []byte(source))
		if err != nil {
			t.Fatalf("%s schema refused cassini-huygens: %v", version, err)
		}
		if len(made.Things()) < 2 || made.Things()[1].Kind() != "cassini-huygens" {
			t.Fatalf("%s schema parsed the wrong thing: %+v", version, made.Things())
		}
		kind, err := manufacture.KindNamed(made.Things()[1].Kind())
		if err != nil {
			t.Fatalf("%s schema kind has no maker: %v", version, err)
		}
		if _, err := manufacture.MakeWithComplexity(kind,
			manufacture.Sow(5), "quick"); err != nil {
			t.Fatalf("%s schema kind cannot be constructed: %v", version, err)
		}
		broken := strings.Replace(source, "kind: cassini-huygens",
			"kind: cassini-huygenso", 1)
		if _, err := set.Read("unknown-cassini-huygens.yaml", []byte(broken)); err == nil {
			t.Fatalf("%s schema accepted a near-miss hero kind", version)
		}
	}
}

func TestNewHorizonsPassesBothSetSchemasAndResolvesToAMaker(t *testing.T) {
	raw, err := os.ReadFile("../sets/ferruccio-playground.yaml")
	if err != nil {
		t.Fatal(err)
	}
	for _, version := range []string{"v1", "v2"} {
		source := strings.ReplaceAll(string(raw), "ecosystem-v2.json",
			"ecosystem-"+version+".json")
		if version == "v1" {
			source = strings.ReplaceAll(source, "  setting: space\n", "")
		}
		source = strings.Replace(source, "kind: human-scale-2m",
			"kind: new-horizons", 1)
		made, err := set.Read("new-horizons.yaml", []byte(source))
		if err != nil {
			t.Fatalf("%s schema refused new-horizons: %v", version, err)
		}
		if len(made.Things()) < 2 || made.Things()[1].Kind() != "new-horizons" {
			t.Fatalf("%s schema parsed the wrong thing: %+v", version, made.Things())
		}
		kind, err := manufacture.KindNamed(made.Things()[1].Kind())
		if err != nil {
			t.Fatalf("%s schema kind has no maker: %v", version, err)
		}
		if _, err := manufacture.MakeWithComplexity(kind,
			manufacture.Sow(5), "quick"); err != nil {
			t.Fatalf("%s schema kind cannot be constructed: %v", version, err)
		}
		broken := strings.Replace(source, "kind: new-horizons",
			"kind: new-horizonso", 1)
		if _, err := set.Read("unknown-new-horizons.yaml", []byte(broken)); err == nil {
			t.Fatalf("%s schema accepted a near-miss hero kind", version)
		}
	}
}

func TestNewHorizonsVTADPassesBothSetSchemasAndResolvesToAMaker(t *testing.T) {
	raw, err := os.ReadFile("../sets/ferruccio-playground.yaml")
	if err != nil {
		t.Fatal(err)
	}
	for _, version := range []string{"v1", "v2"} {
		source := strings.ReplaceAll(string(raw), "ecosystem-v2.json",
			"ecosystem-"+version+".json")
		if version == "v1" {
			source = strings.ReplaceAll(source, "  setting: space\n", "")
		}
		source = strings.Replace(source, "kind: human-scale-2m",
			"kind: new-horizons-vtad", 1)
		made, err := set.Read("new-horizons-vtad.yaml", []byte(source))
		if err != nil {
			t.Fatalf("%s schema refused new-horizons-vtad: %v", version, err)
		}
		if len(made.Things()) < 2 || made.Things()[1].Kind() != "new-horizons-vtad" {
			t.Fatalf("%s schema parsed the wrong thing: %+v", version, made.Things())
		}
		kind, err := manufacture.KindNamed(made.Things()[1].Kind())
		if err != nil {
			t.Fatalf("%s schema kind has no maker: %v", version, err)
		}
		if _, err := manufacture.MakeWithComplexity(kind,
			manufacture.Sow(5), "quick"); err != nil {
			t.Fatalf("%s schema kind cannot be constructed: %v", version, err)
		}
		broken := strings.Replace(source, "kind: new-horizons-vtad",
			"kind: new-horizons-vtado", 1)
		if _, err := set.Read("unknown-new-horizons-vtad.yaml", []byte(broken)); err == nil {
			t.Fatalf("%s schema accepted a near-miss hero kind", version)
		}
	}
}
