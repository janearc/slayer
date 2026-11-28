package workspace

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gogpu/ui/core/button"
	"github.com/gogpu/ui/core/scrollview"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
	"github.com/janearc/slayer/ferruccio/internal/preview"
	"github.com/janearc/slayer/ferruccio/native/internal/settings"
)

func TestMissingRenderQueueOffersSettings(t *testing.T) {
	work := playWorkspace(t)
	work.options.Tools = preview.Tools{Transport: "azazel", Azazel: "/tools/azazel", Slayer: "/tools/slayer"}
	work.options.Evidence = t.TempDir()
	work.settings.Path = filepath.Join(t.TempDir(), "config")
	work.rebuild()
	control, ok := libraryWidget(work.root, "Set render queue…").(*button.Widget)
	if !ok {
		t.Fatal("the default workspace offers no way to configure its missing queue")
	}

	before := work.state.document.Text()
	control.SetBounds(geometry.NewRect(0, 0, 160, 44))
	point := geometry.Pt(80, 22)
	ctx := widget.NewContext()
	control.Event(ctx, event.NewMouseEvent(event.MousePress, event.ButtonLeft, event.ButtonStateLeft, point, point, event.ModNone))
	control.Event(ctx, event.NewMouseEvent(event.MouseRelease, event.ButtonLeft, 0, point, point, event.ModNone))
	if !work.wantSettings.Load() || work.cancel != nil || work.state.document.Text() != before {
		t.Fatal("choosing queue settings failed to request Settings or altered scene work")
	}

	work.saveSetting(settings.Queue, t.TempDir())
	work.rebuild()
	previewControl, ok := libraryWidget(work.root, "Preview").(*button.Widget)
	if !ok || !previewControl.IsFocusable() || libraryWidget(work.root, "Set render queue…") != nil {
		t.Fatal("saving a queue did not restore the Preview control")
	}
}

func TestPreviewPreparesAnUncreatedDefaultQueueBeforeSubmitting(t *testing.T) {
	work := playWorkspace(t)
	queue := filepath.Join(t.TempDir(), "Queue")
	tools := preview.Tools{Transport: "azazel", Azazel: "/tools/azazel", Queue: queue, Slayer: "/tools/slayer"}
	work.options.Tools, work.options.Evidence = tools, t.TempDir()
	stopped := errors.New("stop after reaching the queue initializer")
	work.options.PrepareQueue = func(_ context.Context, executable, selected string) error {
		if executable != tools.Azazel || selected != queue {
			return fmt.Errorf("wrong initializer arguments: %q, %q", executable, selected)
		}
		if _, err := os.Stat(selected); !os.IsNotExist(err) {
			return fmt.Errorf("queue already exists before initialization")
		}
		return stopped
	}
	work.rebuild()
	control, ok := libraryWidget(work.root, "Preview").(*button.Widget)
	if !ok || !control.IsFocusable() || libraryWidget(work.root, "Set render queue…") != nil {
		t.Fatal("an uninitialized default queue did not enable Preview")
	}

	control.SetBounds(geometry.NewRect(0, 0, 160, 44))
	point := geometry.Pt(80, 22)
	ctx := widget.NewContext()
	control.Event(ctx, event.NewMouseEvent(event.MousePress, event.ButtonLeft, event.ButtonStateLeft, point, point, event.ModNone))
	control.Event(ctx, event.NewMouseEvent(event.MouseRelease, event.ButtonLeft, 0, point, point, event.ModNone))
	select {
	case result := <-work.results:
		if !errors.Is(result.err, stopped) {
			t.Fatalf("render returned %v before reaching the initializer", result.err)
		}
		if _, err := os.Stat(filepath.Join(result.plan.Directory, "azazel-request.json")); !os.IsNotExist(err) {
			t.Fatal("render created an Azazel request after queue initialization failed")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Preview did not reach the background initializer")
	}
}

func TestRenderQueueControlLeavesConfiguredAndDirectRenderingAlone(t *testing.T) {
	for _, tools := range []preview.Tools{
		{Transport: "azazel", Queue: "/queue"},
		{Transport: "slayer"},
	} {
		work := playWorkspace(t)
		work.options.Tools = tools
		control := work.drawControl()
		if id := control.(interface{ ID() string }).ID(); id != "Preview" {
			t.Fatalf("%s shows %q instead of Preview", tools.Transport, id)
		}
	}
}

func TestQueueSetupKeepsPlayToolbarReachable(t *testing.T) {
	for _, size := range []float32{16, 18, 20, 22, 24} {
		t.Run(fmt.Sprintf("%g pt", size), func(t *testing.T) {
			work := libraryEventWorkspace(t)
			work.options.Tools = preview.Tools{Transport: "azazel", Azazel: "/tools/azazel", Slayer: "/tools/slayer"}
			work.options.Evidence = t.TempDir()
			work.size = size
			work.ui.Window().HandleResize(int(smallestWindow.Width), int(smallestWindow.Height))
			work.rebuild()
			drawLibraryWindow(t, work)

			ids := []string{"Scene library", "Open…", "Save", "Set render queue…", "Render fully", "Source"}
			last := float32(0)
			for _, id := range ids {
				control := libraryWidget(work.root, id)
				if control == nil {
					t.Fatalf("missing %q", id)
				}
				bounds := control.(interface{ ScreenBounds() geometry.Rect }).ScreenBounds()
				if bounds.IsEmpty() || bounds.Min.X < last || bounds.Max.X > smallestWindow.Width ||
					bounds.Min.Y < 0 || bounds.Max.Y > smallestWindow.Height {
					t.Errorf("%q lies outside the toolbar or overlaps a neighbor: %v", id, bounds)
				}
				last = bounds.Max.X
			}
			control := libraryWidget(work.root, "Set render queue…")
			bounds := control.(interface{ ScreenBounds() geometry.Rect }).ScreenBounds()
			clickLibraryPoint(work, geometry.Pt((bounds.Min.X+bounds.Max.X)/2, (bounds.Min.Y+bounds.Max.Y)/2))
			if !work.wantSettings.Load() {
				t.Fatal("clicking queue setup at its displayed location did not request Settings")
			}
		})
	}
}

func TestSettingsScrollReachesRenderControlsAtLargestType(t *testing.T) {
	work := libraryEventWorkspace(t)
	work.options.Tools = preview.Tools{Transport: "azazel"}
	work.settings.Path = filepath.Join(t.TempDir(), "config")
	work.settingsOpen, work.size = true, 24
	work.ui.Window().HandleResize(int(smallestWindow.Width), int(smallestWindow.Height))
	work.rebuild()
	drawLibraryWindow(t, work)

	var scroll *scrollview.Widget
	if !holds(work.root, func(item widget.Widget) bool {
		var ok bool
		scroll, ok = item.(*scrollview.Widget)
		return ok
	}) {
		t.Fatal("Settings has no scroll view")
	}
	viewport := scroll.ScreenBounds()
	queue := libraryWidget(work.root, "Choose render queue").(interface{ ScreenBounds() geometry.Rect })
	processor := libraryWidget(work.root, "Setting processors").(interface{ ScreenBounds() geometry.Rect })
	if !viewport.ContainsRect(queue.ScreenBounds()) || viewport.ContainsRect(processor.ScreenBounds()) {
		t.Fatal("the fixture should initially show the queue and require scrolling to the processors")
	}

	point := geometry.Pt((viewport.Min.X+viewport.Max.X)/2, (viewport.Min.Y+viewport.Max.Y)/2)
	work.ui.HandleEvent(event.NewWheelEvent(geometry.Pt(0, 100), point, point, event.ModNone))
	drawLibraryWindow(t, work)
	for _, id := range []string{"Setting theme", "Setting renderer", "Setting processors"} {
		control := libraryWidget(work.root, id).(interface{ ScreenBounds() geometry.Rect })
		if bounds := control.ScreenBounds(); !viewport.ContainsRect(bounds) {
			t.Errorf("scrolling did not reveal %q in the Settings viewport: %v in %v", id, bounds, viewport)
		}
	}
	bounds := processor.ScreenBounds()
	clickLibraryPoint(work, geometry.Pt((bounds.Min.X+bounds.Max.X)/2, (bounds.Min.Y+bounds.Max.Y)/2))
	if len(work.ui.Window().OverlayContentWidgets()) == 0 {
		t.Fatal("the processors control could not be opened after scrolling")
	}
}

func TestClearingSavedQueueAndShelfRestoresLaunchDefaults(t *testing.T) {
	work := playWorkspace(t)
	home := t.TempDir()
	work.settings = SettingsSource{Path: settings.Path(home)}
	privateQueue := filepath.Join(home, "Library", "Application Support", "Ferruccio", "Queue")
	customQueue := filepath.Join(home, "other queue")
	customShelf := filepath.Join(home, "other shelf")
	work.options.Tools = preview.Tools{Transport: "azazel", Queue: privateQueue, Azazel: "/tools/azazel"}
	for name, value := range map[settings.Name]string{settings.Queue: customQueue, settings.Shelf: customShelf} {
		if err := settings.Save(work.settings.Path, name, value); err != nil {
			t.Fatal(err)
		}
	}
	if tools := work.renderTools(); tools.Queue != customQueue || tools.Shelf != customShelf {
		t.Fatalf("saved choices were ignored: %+v", tools)
	}

	work.saveSetting(settings.Queue, "")
	work.saveSetting(settings.Shelf, "")
	if tools := work.renderTools(); tools.Queue != privateQueue || tools.Shelf != "" {
		t.Fatalf("cleared choices still affect runtime paths: %+v", tools)
	}
	if _, err := os.Stat(privateQueue); !os.IsNotExist(err) {
		t.Fatalf("clearing settings should leave private initialization to Preview: %v", err)
	}
}
