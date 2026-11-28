package preview

import (
	"reflect"
	"testing"
	"time"
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

func TestNamedCameraAngleReachesPreviewAndFullDraw(t *testing.T) {
	tools := Tools{Slayer: "/engine/slayer"}
	for _, full := range []bool{false, true} {
		plan := Plan{
			Kind: "set", Input: "/draft/job.yaml", Output: "/new/image.png",
			Profile: Profile{Full: full, Camera: "bay", CameraWide: 31.25},
		}
		args, err := plan.arguments(tools)
		want := []string{"--camera", "bay", "--camera-wide", "31.25"}
		if err != nil || len(args) < len(want) || !reflect.DeepEqual(args[len(args)-len(want):], want) {
			t.Fatalf("full=%t: draw args = %v, %v; want trailing %v", full, args, err, want)
		}
	}
}

func TestStillCameraPathReachesPreviewAndFullDraw(t *testing.T) {
	const recipe = `{"camera":{"unit":"m","moves":[{}]},"at":{"value":20.125,"unit":"s"}}`
	for _, full := range []bool{false, true} {
		plan := Plan{
			Kind: "set", Input: "/draft/set.yaml", Output: "/new/image.png",
			Profile: Profile{Full: full, CameraPathRecipe: recipe},
		}
		args, err := plan.arguments(Tools{Slayer: "/engine/slayer"})
		want := []string{"--still-camera", recipe}
		if err != nil || len(args) < len(want) || !reflect.DeepEqual(args[len(args)-len(want):], want) {
			t.Fatalf("full=%t: draw args = %v, %v; want trailing %v", full, args, err, want)
		}
	}
	if width, height, err := fitCameraAspect(960, 540, 640, 400); err != nil || width != 640 || height != 360 {
		t.Fatalf("Preview fit = %dx%d, %v", width, height, err)
	}
}

func TestExplicitSceneCameraKeepsTheStillTime(t *testing.T) {
	plan := Plan{
		Kind: "set", Input: "/draft/set.yaml", Output: "/new/image.png",
		Profile: Profile{Full: true, Camera: "bay", StillAt: 20*time.Second + 125*time.Millisecond},
	}
	args, err := plan.arguments(Tools{Slayer: "/engine/slayer"})
	want := []string{"--camera", "bay", "--at", "20.125s"}
	if err != nil || len(args) < len(want) || !reflect.DeepEqual(args[len(args)-len(want):], want) {
		t.Fatalf("draw args = %v, %v; want trailing %v", args, err, want)
	}
}
