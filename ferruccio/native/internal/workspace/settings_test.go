package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/ferruccio/internal/draft"
	"github.com/janearc/slayer/ferruccio/internal/preview"
	"github.com/janearc/slayer/ferruccio/native/internal/settings"
)

// nothing is a Lookup that knows nothing.
func nothing(string) string { return "" }

// settingsWorkspace is a workspace on the Slayer road whose settings file is
// in a folder of its own, and has the text in file when it is not empty. It
// has no window.
func settingsWorkspace(t *testing.T, file string) *Workspace {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config")
	if file != "" {
		if err := os.WriteFile(path, []byte(file), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	document, err := draft.Open("set.yaml", []byte(checkingSource))
	if err != nil {
		t.Fatal(err)
	}
	work := &Workspace{
		options:  Options{Evidence: "/evidence", Tools: preview.Tools{Transport: "slayer"}},
		settings: SettingsSource{Path: path, Flag: nothing, Environment: nothing},
	}
	work.state = documentState{document: document, path: "/sets/set.yaml"}
	return work
}

func TestRenderToolsTakeTheSettingsFileAndNeedNoRestart(t *testing.T) {
	work := settingsWorkspace(t, "")
	if tools := work.renderTools(); tools.Cache != "" || tools.Renderer != "" || tools.Processors != "" {
		t.Fatalf("an empty file gave %+v", tools)
	}
	if err := settings.Save(work.settings.Path, settings.Cache, "/the/cache"); err != nil {
		t.Fatal(err)
	}
	if err := settings.Save(work.settings.Path, settings.Processors, "8"); err != nil {
		t.Fatal(err)
	}
	tools := work.renderTools()
	if tools.Cache != "/the/cache" || tools.Processors != "8" {
		t.Fatalf("a saved value was not in force at the next render: %+v", tools)
	}
}

func TestAFlagOrTheEnvironmentBeatsTheFileInTheRenderToolsToo(t *testing.T) {
	work := settingsWorkspace(t, "cache :: /from/the/file\n")
	work.settings.Flag = func(name string) string {
		if name == "cache" {
			return "/from/the/flag"
		}
		return ""
	}
	if got := work.renderTools().Cache; got != "/from/the/flag" {
		t.Fatalf("the cache was %q", got)
	}
}

func TestRenderToolsKeepTheLaunchsToolsTheSettingsDoNotSay(t *testing.T) {
	work := settingsWorkspace(t, "")
	work.options.Tools.Slayer = "/launched/slayer"
	work.options.Tools.Agent = "someone"
	tools := work.renderTools()
	if tools.Slayer != "/launched/slayer" || tools.Agent != "someone" || tools.Transport != "slayer" {
		t.Fatalf("the launch's tools were lost: %+v", tools)
	}
}

func TestOwnRenderIsTheCPUByDefaultAndTheGPUOnlyWhenChosen(t *testing.T) {
	work := settingsWorkspace(t, "")
	if profile := work.ownRender(preview.Profile{Full: true}); !profile.CPURenderer || profile.GPU || !profile.Full {
		t.Fatalf("the default is %+v", profile)
	}
	if err := settings.Save(work.settings.Path, settings.Renderer, "gpu"); err != nil {
		t.Fatal(err)
	}
	if profile := work.ownRender(preview.Profile{Full: true}); profile.CPURenderer || !profile.GPU || !profile.Full {
		t.Fatalf("the chosen GPU renderer is %+v", profile)
	}
}

func TestTheAzazelRoadKeepsTheCPURendererWhateverIsChosen(t *testing.T) {
	work := settingsWorkspace(t, "renderer :: gpu\n")
	work.options.Tools.Transport = "azazel"
	if profile := work.ownRender(preview.Profile{}); !profile.CPURenderer || profile.GPU {
		t.Fatalf("the Azazel road was given %+v", profile)
	}
}

func TestSlayerNeedsNoGameNowThatNoLaneIsUsed(t *testing.T) {
	work := settingsWorkspace(t, "slayer :: /a/slayer\n")
	work.options.Tools.Game = ""
	if !work.rendererConfigured() {
		t.Fatal("a render was refused for want of a Game it no longer uses")
	}
	work.settings.Path = filepath.Join(t.TempDir(), "none")
	if work.rendererConfigured() {
		t.Fatal("a render with no Slayer was taken as configured")
	}
}

func TestARenderWithoutACacheIsRefusedWithTheLineAndAsksForSettings(t *testing.T) {
	t.Setenv(preview.CacheVariable, "")
	work := settingsWorkspace(t, "")
	err := work.refuseWithoutCache(work.renderTools())
	if err != preview.ErrNoCache {
		t.Fatalf("got %v", err)
	}
	if !work.wantSettings.Load() {
		t.Fatal("Settings was not asked for")
	}
}

func TestARenderWithACacheFromTheFileIsNotRefused(t *testing.T) {
	t.Setenv(preview.CacheVariable, "")
	work := settingsWorkspace(t, "cache :: /the/cache\n")
	if err := work.refuseWithoutCache(work.renderTools()); err != nil {
		t.Fatalf("a render with a cache was refused: %v", err)
	}
	if work.wantSettings.Load() {
		t.Fatal("Settings was asked for although nothing was missing")
	}
}

func TestTheAzazelRoadIsNotAskedForACache(t *testing.T) {
	t.Setenv(preview.CacheVariable, "")
	work := settingsWorkspace(t, "")
	work.options.Tools.Transport = "azazel"
	if err := work.refuseWithoutCache(work.renderTools()); err != nil || work.wantSettings.Load() {
		t.Fatalf("the Azazel road was refused for a cache: %v", err)
	}
}

func TestLaunchWithoutACacheSaysSoInOneLineAndAsksForSettingsButStillOpens(t *testing.T) {
	t.Setenv(preview.CacheVariable, "")
	work := settingsWorkspace(t, "")
	work.autoPreview = true
	work.noteMissingCache()
	if work.status != preview.ErrNoCache.Error() || strings.Contains(work.status, "\n") {
		t.Fatalf("the status was %q", work.status)
	}
	if !work.wantSettings.Load() || work.autoPreview {
		t.Fatal("Settings was not asked for, or the automatic preview was left to fail")
	}
}

func TestLaunchWithACacheOrAProofSaysNothing(t *testing.T) {
	t.Setenv(preview.CacheVariable, "")
	work := settingsWorkspace(t, "cache :: /c\n")
	work.noteMissingCache()
	if work.status != "" || work.wantSettings.Load() {
		t.Fatal("a launch with a cache was told it had none")
	}
	proof := settingsWorkspace(t, "")
	proof.options.SuppressAutomaticPreview = true
	proof.noteMissingCache()
	if proof.status != "" || proof.wantSettings.Load() {
		t.Fatal("a proof was given a sheet to answer")
	}
}

func TestSettingNoteSaysWhyAValueFromAFlagOrTheEnvironmentCannotBeChanged(t *testing.T) {
	key, _ := settings.KeyOf(settings.Cache)
	flag := settingNote(settings.Entry{Key: key, Value: "/f", Source: settings.FromFlag, Origin: "--cache"})
	environment := settingNote(settings.Entry{Key: key, Value: "/e", Source: settings.FromEnvironment, Origin: "SLAYER_CACHE"})
	for _, note := range []string{flag, environment} {
		if !strings.Contains(note, "cannot be changed here") {
			t.Fatalf("the note does not say why: %s", note)
		}
	}
	if !strings.Contains(flag, "--cache flag") || !strings.Contains(environment, "SLAYER_CACHE variable") {
		t.Fatalf("the notes do not name where the value came from: %s | %s", flag, environment)
	}
	if file := settingNote(settings.Entry{Key: key, Value: "/c", Source: settings.FromFile}); strings.Contains(file, "cannot") {
		t.Fatalf("a value from the file was said to be fixed: %s", file)
	}
}

func TestTheRendererNoteSaysWhatCPUAndGPUAreInPlainWords(t *testing.T) {
	key, _ := settings.KeyOf(settings.Renderer)
	note := settingNote(settings.Entry{Key: key, Source: settings.Unset})
	for _, want := range []string{"cpu draws every scene", "reference", "gpu", "refuses scenes it cannot draw"} {
		if !strings.Contains(note, want) {
			t.Fatalf("the renderer note lacks %q: %s", want, note)
		}
	}
}

func TestSettingValueShowsDefaultsAndNothingElseAsADefault(t *testing.T) {
	work := settingsWorkspace(t, "")
	resolved := work.resolved()
	if got := work.settingValue(resolved.Get(settings.Processors)); got != "2 (default)" {
		t.Fatalf("processors show %q", got)
	}
	if got := work.settingValue(resolved.Get(settings.Cache)); got != "not set" {
		t.Fatalf("an unset cache shows %q", got)
	}
}

func TestExamplesFolderIsTheSettingsOrTheLaunchs(t *testing.T) {
	work := settingsWorkspace(t, "")
	work.options.Examples = "/launched/examples"
	if got := work.examplesFolder(); got != "/launched/examples" {
		t.Fatalf("got %s", got)
	}
	if err := settings.Save(work.settings.Path, settings.Examples, "/chosen/examples"); err != nil {
		t.Fatal(err)
	}
	if got := work.examplesFolder(); got != "/chosen/examples" {
		t.Fatalf("got %s", got)
	}
}
