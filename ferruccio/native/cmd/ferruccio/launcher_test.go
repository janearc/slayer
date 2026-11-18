package main

// These tests run the two launchers, the bundle launcher that
// bundle-gogpu.sh writes and ferruccio/scripts/launch-native.sh, through
// POSIX sh. The native CLI at the end of each is this test binary, started
// again with cliModeVariable set, so the arguments the CLI really receives
// are the ones compared. Hostile defaults carry quotes, command
// substitution, backticks and newlines. The Azazel flags are not checked:
// 489a84be removed them while production Azazel is unavailable, and their
// checks return with the flags.

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const (
	// executableMode lets anyone run a script the test writes.
	executableMode = 0o755
	// plainMode is the mode of a data file the test writes.
	plainMode = 0o644
	// cliModeVariable makes TestMain run the CLI instead of the tests.
	cliModeVariable = "FERRUCCIO_TEST_AS_CLI"
	// argumentsLogVariable names the file the CLI stand-in writes its
	// arguments to, each one ended by a NUL byte.
	argumentsLogVariable = "FERRUCCIO_ARGUMENTS_LOG"
	// testBinaryVariable names the executable the CLI stand-in starts.
	testBinaryVariable = "FERRUCCIO_TEST_BINARY"
	// scriptsDirectory is the launcher scripts, from this package.
	scriptsDirectory = "../../../scripts"
	// dryRunText is what the CLI prints for open --dry-run.
	dryRunText = "dry-run opens no window and starts no preview"
	// quotingFault is the safe-byte set in shell_word with a space added,
	// so a space is no longer escaped.
	quotingFault = "[^A-Za-z0-9_.,:/@ -]"
	// quotingSafe is the safe-byte set in shell_word as it is written.
	quotingSafe = "[^A-Za-z0-9_.,:/@-]"
	// overrideSlayer, overrideAgent, overrideCache and overrideEvidence
	// are hostile values set when a launcher runs.
	overrideSlayer   = "/override/' client $literal `value`"
	overrideAgent    = "operator ' $literal `value`"
	overrideCache    = "/override/queue $literal\nsecond line"
	overrideEvidence = "../shared ' override $literal `value`\nline"
)

// stubProgram is the content of an executable that does nothing.
const stubProgram = "#!/bin/sh\n"

// cliStandIn records its arguments, then runs this test binary as the CLI.
const cliStandIn = `#!/bin/sh
printf '%s\0' "$@" > "$FERRUCCIO_ARGUMENTS_LOG"
exec env FERRUCCIO_TEST_AS_CLI=1 "$FERRUCCIO_TEST_BINARY" "$@"
`

// launchVariables are cleared from every launcher run, so the machine's own
// settings never decide a result.
var launchVariables = []string{
	"FERRUCCIO_SLAYER", "FERRUCCIO_AGENT", "FERRUCCIO_EVIDENCE",
	"FERRUCCIO_CACHE", "SLAYER_CACHE", "FERRUCCIO_STARTER",
	"FERRUCCIO_BUNDLE", "FERRUCCIO_SHELF", "FERRUCCIO_DISPLAY",
	"GAME_BIN", "GAME_HOME",
}

// TestMain runs the CLI when the launchers start this binary as one.
func TestMain(main *testing.M) {
	if os.Getenv(cliModeVariable) != "" {
		os.Exit(command().RunWith(os.Args[1:], os.Stdin, os.Stdout,
			os.Stderr))
	}
	os.Exit(main.Run())
}

// hostileDefaults are the values written into a bundle launcher.
type hostileDefaults struct{ slayer, agent, cache, evidence string }

// launcherCheck holds what one run of every launcher check needs.
type launcherCheck struct {
	scripts, directory, sentinel, captured, home string
	hostile                                      hostileDefaults
}

// newLauncherCheck lays out a check inside directory, which is made
// absolute so paths the launchers print compare equal.
func newLauncherCheck(scripts, directory string) (*launcherCheck, error) {
	directory, err := filepath.EvalSymlinks(directory)
	if err != nil {
		return nil, err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	sentinel := filepath.Join(directory, "must-not-exist")
	touch := "$(touch " + sentinel + ")"
	return &launcherCheck{
		scripts: scripts, directory: directory, sentinel: sentinel,
		captured: filepath.Join(directory, "argv"), home: home,
		hostile: hostileDefaults{
			slayer:   "slayer ' " + touch + " `literal`\nname",
			agent:    "preview-owner ' " + touch + " `literal`\nline",
			cache:    "cache with 'quote' $dollar\nand newline",
			evidence: "shared ' evidence $literal `value`\nline",
		},
	}, nil
}

// writeFile writes content at path, making its directory, with mode.
func writeFile(path, content string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), executableMode); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), mode)
}

// copyScript copies the launcher script named from the scripts directory to
// destination.
func copyScript(scripts, name, destination string) error {
	content, err := os.ReadFile(filepath.Join(scripts, name))
	if err != nil {
		return err
	}
	return writeFile(destination, string(content), executableMode)
}

// environmentWith is the process environment without the launch variables
// and without the names in settings, followed by settings.
func environmentWith(settings map[string]string) []string {
	skip := map[string]bool{}
	for _, name := range launchVariables {
		skip[name] = true
	}
	for name := range settings {
		skip[name] = true
	}
	var result []string
	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		if !skip[name] {
			result = append(result, entry)
		}
	}
	for name, value := range settings {
		result = append(result, name+"="+value)
	}
	return result
}

// shellOutput starts sh on arguments with environment and returns its
// output.
func shellOutput(environment []string, arguments ...string) (string, error) {
	command := exec.Command("sh", arguments...)
	command.Env = environment
	output, err := command.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("sh %v: %w: %s", arguments, err, output)
	}
	return string(output), nil
}

// buildBundle runs the real bundle-gogpu.sh in a new root named name, with
// settings as its environment, and returns the root and the launcher path.
func (check *launcherCheck) buildBundle(name string,
	settings map[string]string) (string, string, error) {
	root := filepath.Join(check.directory, name)
	slayer := filepath.Join(check.directory, "outside", check.hostile.slayer)
	files := []string{"starter/source.yaml", "starter/preview.png",
		"starter/receipt.json", "ferruccio/examples/one.yaml",
		"sets/one.yaml", "sources.yaml"}
	for _, file := range files {
		path := filepath.Join(root, file)
		if err := writeFile(path, "x\n", plainMode); err != nil {
			return "", "", err
		}
	}
	for path, content := range map[string]string{
		filepath.Join(root, "bin", "ferruccio"): cliStandIn,
		filepath.Join(root, "bin", "game5"):     stubProgram,
		slayer:                                  stubProgram,
	} {
		if err := writeFile(path, content, executableMode); err != nil {
			return "", "", err
		}
	}
	script := filepath.Join(root, "ferruccio", "scripts", "bundle-gogpu.sh")
	if err := copyScript(check.scripts, "bundle-gogpu.sh", script); err != nil {
		return "", "", err
	}
	settings["FERRUCCIO_STARTER"] = filepath.Join(root, "starter")
	settings["GAME_BIN"] = filepath.Join(root, "bin", "game5")
	settings["FERRUCCIO_SLAYER"] = slayer
	settings[argumentsLogVariable] = check.captured
	settings[testBinaryVariable], _ = os.Executable()
	_, err := shellOutput(environmentWith(settings), script)
	launch := filepath.Join(root, "bin", "Ferruccio-Live.app", "Contents",
		"MacOS", "launch")
	return root, launch, err
}

// dryRun runs launch with --dry-run under sh and returns the value that
// followed each flag the CLI received.
func (check *launcherCheck) dryRun(launch string,
	settings map[string]string) (map[string]string, error) {
	syntax := environmentWith(nil)
	if _, err := shellOutput(syntax, "-n", launch); err != nil {
		return nil, err
	}
	executable, err := os.Executable()
	if err != nil {
		return nil, err
	}
	settings[argumentsLogVariable] = check.captured
	settings[testBinaryVariable] = executable
	output, err := shellOutput(environmentWith(settings), launch,
		"--dry-run")
	if err != nil {
		return nil, err
	}
	if !strings.Contains(output, dryRunText) {
		return nil, fmt.Errorf("dry-run text missing: %s", output)
	}
	return check.receivedFlags()
}

// receivedFlags reads the argument log the CLI stand-in wrote.
func (check *launcherCheck) receivedFlags() (map[string]string, error) {
	content, err := os.ReadFile(check.captured)
	if err != nil {
		return nil, err
	}
	arguments := strings.Split(string(content), "\x00")
	flags := map[string]string{}
	for index, argument := range arguments {
		if strings.HasPrefix(argument, "--") && index+1 < len(arguments) {
			flags[argument] = arguments[index+1]
		}
	}
	return flags, nil
}

// expectFlags reports the first flag in want that is missing from got or
// has another value.
func expectFlags(got, want map[string]string) error {
	for flag, value := range want {
		if actual, found := got[flag]; !found || actual != value {
			return fmt.Errorf("missing or changed %s: got %q, want %q",
				flag, actual, value)
		}
	}
	return nil
}

// expectDryRun runs launch under settings and compares the flags it passed.
func (check *launcherCheck) expectDryRun(launch string,
	settings, want map[string]string) error {
	got, err := check.dryRun(launch, settings)
	if err != nil {
		return err
	}
	return expectFlags(got, want)
}

// checkBundle builds the bundle launcher twice and checks its defaults,
// its overrides and the shared evidence default.
func (check *launcherCheck) checkBundle() error {
	hostile := check.hostile
	root, launch, err := check.buildBundle("quote ' app $literal one",
		map[string]string{"FERRUCCIO_AGENT": hostile.agent,
			"FERRUCCIO_CACHE":    hostile.cache,
			"FERRUCCIO_EVIDENCE": hostile.evidence})
	if err != nil {
		return err
	}
	defaults := map[string]string{"--slayer": filepath.Join(check.directory,
		"outside", hostile.slayer), "--agent": hostile.agent,
		"--cache": hostile.cache, "--evidence": root + "/" + hostile.evidence}
	none := map[string]string{}
	if err := check.expectDryRun(launch, none, defaults); err != nil {
		return err
	}
	return check.expectOverrides(launch)
}

// expectOverrides checks that launch-time settings win over the defaults.
func (check *launcherCheck) expectOverrides(launch string) error {
	settings := map[string]string{"FERRUCCIO_SLAYER": overrideSlayer,
		"FERRUCCIO_AGENT": overrideAgent, "FERRUCCIO_CACHE": overrideCache,
		"FERRUCCIO_EVIDENCE": overrideEvidence}
	return check.expectDryRun(launch, settings, map[string]string{
		"--slayer": overrideSlayer, "--agent": overrideAgent,
		"--cache": overrideCache, "--evidence": overrideEvidence})
}

// checkBundleFallbacks builds a bundle launcher with no agent, no evidence
// and only SLAYER_CACHE set, and checks the shared defaults.
func (check *launcherCheck) checkBundleFallbacks() error {
	_, launch, err := check.buildBundle("fallbacks", map[string]string{
		"SLAYER_CACHE": check.hostile.cache})
	if err != nil {
		return err
	}
	return check.expectDryRun(launch, map[string]string{},
		map[string]string{"--agent": "ferruccio",
			"--cache":    check.hostile.cache,
			"--evidence": check.sharedEvidence()})
}

// sharedEvidence is the evidence directory used when none is given.
func (check *launcherCheck) sharedEvidence() string {
	return filepath.Join(check.home, "mesh", "the-vorrh", "renders",
		"frames")
}

// checkDeveloper runs a copy of launch-native.sh with no settings, with a
// hostile agent alone, and with every override.
func (check *launcherCheck) checkDeveloper() error {
	root := filepath.Join(check.directory, "developer")
	launch := filepath.Join(root, "ferruccio", "scripts", "launch-native.sh")
	err := copyScript(check.scripts, "launch-native.sh", launch)
	if err != nil {
		return err
	}
	game := filepath.Join(root, "bin", "game5")
	stand := filepath.Join(root, "bin", "ferruccio")
	for path, content := range map[string]string{
		stand: cliStandIn, game: stubProgram} {
		if err := writeFile(path, content, executableMode); err != nil {
			return err
		}
	}
	slayer := filepath.Join(check.home, "mesh", "the-vorrh", "renders",
		"applications", "slayer", "bin", "slayer")
	defaults := map[string]string{"--evidence": check.sharedEvidence(),
		"--agent": "ferruccio", "--slayer": slayer}
	cases := []struct{ settings, want map[string]string }{
		{map[string]string{}, defaults},
		{map[string]string{"FERRUCCIO_AGENT": check.hostile.agent},
			map[string]string{"--agent": check.hostile.agent,
				"--evidence": check.sharedEvidence(), "--slayer": slayer}},
		{map[string]string{"FERRUCCIO_SLAYER": overrideSlayer,
			"FERRUCCIO_AGENT":    overrideAgent,
			"FERRUCCIO_EVIDENCE": overrideEvidence},
			map[string]string{"--slayer": overrideSlayer,
				"--agent": overrideAgent, "--evidence": overrideEvidence}},
	}
	for _, each := range cases {
		each.settings["GAME_BIN"] = game
		err := check.expectDryRun(launch, each.settings, each.want)
		if err != nil {
			return err
		}
	}
	return nil
}

// runAll makes every check: no python3 in the bundle script, both launchers
// and no command substitution having run.
func (check *launcherCheck) runAll() error {
	source, err := os.ReadFile(filepath.Join(check.scripts, "bundle-gogpu.sh"))
	if err != nil {
		return err
	}
	if strings.Contains(string(source), "python3") {
		return fmt.Errorf("Python packaging dependency remains")
	}
	for _, step := range []func() error{check.checkBundle,
		check.checkBundleFallbacks, check.checkDeveloper} {
		if err := step(); err != nil {
			return err
		}
	}
	if _, err := os.Stat(check.sentinel); err == nil {
		return fmt.Errorf("unexpected command substitution")
	}
	return nil
}

// TestLaunchersHonourDefaultsOverridesAndQuoting runs every launcher check
// against the scripts in this checkout.
func TestLaunchersHonourDefaultsOverridesAndQuoting(t *testing.T) {
	check, err := newLauncherCheck(scriptsDirectory, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := check.runAll(); err != nil {
		t.Fatal(err)
	}
}

// TestLaunchCheckFailsOnQuotingFault plants a quoting fault in a copy of the
// scripts and wants the checks to fail on it.
func TestLaunchCheckFailsOnQuotingFault(t *testing.T) {
	scratch := t.TempDir()
	source, err := os.ReadFile(filepath.Join(scriptsDirectory,
		"bundle-gogpu.sh"))
	if err != nil || strings.Count(string(source), quotingSafe) != 1 {
		t.Fatalf("quoting set not found once in bundle-gogpu.sh: %v", err)
	}
	faulted := strings.Replace(string(source), quotingSafe, quotingFault, 1)
	scripts := filepath.Join(scratch, "scripts")
	if err := writeFile(filepath.Join(scripts, "bundle-gogpu.sh"), faulted,
		executableMode); err != nil {
		t.Fatal(err)
	}
	if err := copyScript(scriptsDirectory, "launch-native.sh",
		filepath.Join(scripts, "launch-native.sh")); err != nil {
		t.Fatal(err)
	}
	work := filepath.Join(scratch, "work")
	if err := os.MkdirAll(work, executableMode); err != nil {
		t.Fatal(err)
	}
	check, err := newLauncherCheck(scripts, work)
	if err != nil {
		t.Fatal(err)
	}
	if err = check.runAll(); err == nil {
		t.Fatal("a launcher with a quoting fault passed every check")
	}
	t.Logf("the planted fault was caught: %v", err)
}

// TestLaunchCheckPartsCanFail shows each part of the check can fail on its
// own: flag comparison, the launcher status, the dry-run text and sh -n.
func TestLaunchCheckPartsCanFail(t *testing.T) {
	if expectFlags(map[string]string{"--agent": "a"},
		map[string]string{"--agent": "b"}) == nil {
		t.Fatal("a changed flag value passed")
	}
	if expectFlags(nil, map[string]string{"--agent": "a"}) == nil {
		t.Fatal("a missing flag passed")
	}
	check, err := newLauncherCheck(scriptsDirectory, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{
		"fails.sh": "exit 3\n", "silent.sh": "exit 0\n",
		"broken.sh": "if then\n"} {
		path := filepath.Join(check.directory, name)
		if err := writeFile(path, content, executableMode); err != nil {
			t.Fatal(err)
		}
		if _, err := check.dryRun(path, map[string]string{}); err == nil {
			t.Fatalf("%s passed the dry-run check", name)
		}
	}
}
