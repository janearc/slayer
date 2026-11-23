package workspace

import (
	"path/filepath"

	"github.com/janearc/slayer/ferruccio/internal/preview"
	"github.com/janearc/slayer/ferruccio/native/internal/settings"
)

// SettingsSource is how the window resolves and saves its settings: the
// settings file, and the flags and environment that override it. Path may be
// empty, which is no file.
type SettingsSource struct {
	Path              string
	Flag, Environment settings.Lookup
}

// resolved is every setting as it is now. It reads the file each time, so a
// value saved from the sheet is in force on the next call, with no restart.
func (work *Workspace) resolved() settings.Resolved {
	return settings.Resolve(work.settings.Path, work.settings.Flag, work.settings.Environment)
}

// renderTools is the tools for a render started now: the launch's tools with
// the executable, folders, renderer and processors the settings give.
func (work *Workspace) renderTools() preview.Tools {
	tools := work.options.Tools
	resolved := work.resolved()
	tools.Slayer = absolutePath(resolved.Value(settings.Slayer), tools.Slayer)
	tools.Cache = absolutePath(resolved.Value(settings.Cache), tools.Cache)
	tools.Shelf = absolutePath(resolved.Value(settings.Shelf), tools.Shelf)
	tools.Renderer = resolved.Get(settings.Renderer).Value
	tools.Processors = resolved.Get(settings.Processors).Value
	return tools
}

// examplesFolder is the folder of starting scenes now: the settings', or the
// one the window was launched with.
func (work *Workspace) examplesFolder() string {
	return absolutePath(work.resolved().Value(settings.Examples), work.options.Examples)
}

// absolutePath is value made absolute, or fallback when there is no value.
func absolutePath(value, fallback string) string {
	if value == "" {
		return fallback
	}
	if absolute, err := filepath.Abs(value); err == nil {
		return absolute
	}
	return value
}

// ownRender is how a render started from this window chooses its renderer.
// The Azazel road is another one: it keeps the CPU renderer it always had.
// The person's choice is the GPU renderer or, by default, the CPU one.
func (work *Workspace) ownRender(profile preview.Profile) preview.Profile {
	profile.CPURenderer = true
	if work.options.Tools.Transport == "slayer" && work.resolved().Get(settings.Renderer).Value == settings.RendererGPU {
		profile.CPURenderer, profile.GPU = false, true
	}
	return profile
}
