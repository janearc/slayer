package preview

import (
	"reflect"
	"testing"
)

// TestRendererRoutesUseTheirOwnCLI keeps catalogue plans out of Slayer draw.
func TestRendererRoutesUseTheirOwnCLI(t *testing.T) {
	tools := Tools{Game: "/engine/game5", Slayer: "/engine/slayer", Catalogue: "/engine/catalogue"}
	for _, fixture := range []struct {
		plan Plan
		want []string
	}{
		{Plan{Kind: "job", Input: "/draft/file.yaml", Output: "/new/preview.png"}, []string{tools.Slayer, "draw", "/draft/file.yaml", "--out", "/new/preview.png", "--width", "640", "--height", "360", "--rays", "1", "--quality", "preview"}},
		{Plan{Kind: "catalogue", Input: "/draft/settings.yaml", Directory: "/new", Specimen: "subaru", Detail: "demo"}, []string{tools.Catalogue, "render", "subaru", "--plan", "/draft/settings.yaml", "--out", "/new/book"}},
	} {
		got, err := fixture.plan.arguments(tools)
		if err != nil || !reflect.DeepEqual(got, fixture.want) {
			t.Fatalf("%s route = %v, %v; want %v", fixture.plan.Kind, got, err, fixture.want)
		}
	}
}

// TestRendererRoutesRefuseMissingToolsAndUnboundedSelection prevents all-kind runs.
func TestRendererRoutesRefuseMissingToolsAndUnboundedSelection(t *testing.T) {
	for _, fixture := range []struct {
		plan  *Plan
		tools Tools
	}{
		{nil, Tools{Game: "game"}},
		{&Plan{Kind: "job"}, Tools{}},
		{&Plan{Kind: "job"}, Tools{Game: "game"}},
		{&Plan{Kind: "catalogue", Specimen: "subaru"}, Tools{Game: "game", Slayer: "slayer"}},
		{&Plan{Kind: "catalogue"}, Tools{Game: "game", Catalogue: "catalogue"}},
	} {
		if got, err := fixture.plan.arguments(fixture.tools); err == nil || len(got) != 0 {
			t.Fatalf("invalid route accepted: %v, %v", got, err)
		}
	}
}
