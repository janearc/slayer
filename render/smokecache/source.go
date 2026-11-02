package smokecache

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Inputs is what a smoke reads from disk, in words, for its key: each of
// declared as the smoke wrote it, then each file reads finds for it, by
// its length and the sha256 of its bytes. reads is the smoke's own Reads
// for one input.
//
// A file in a $NAME folder goes by its name below that folder, which is
// all a render can tell of where it is, so the same files anywhere are the
// same key. A plain path goes by the path as declared.
func Inputs(declared []string, reads func(in string) ([]string,
	error)) (string, error) {
	if len(declared) == 0 {
		return "none", nil
	}
	var parts []string
	for _, in := range declared {
		files, err := reads(in)
		if err != nil {
			return "", fmt.Errorf("%s: %w", in, err)
		}
		if len(files) == 0 {
			parts = append(parts, in+" nothing")
			continue
		}
		var each []string
		for _, f := range files {
			name, err := below(in, f)
			if err != nil {
				return "", err
			}
			n, sum, err := measure(f)
			if err != nil {
				return "", fmt.Errorf("%s: %w", in, err)
			}
			each = append(each,
				fmt.Sprintf("%q %d %s", name, n, sum))
		}
		parts = append(parts, in+" "+strings.Join(each, ", "))
	}
	return strings.Join(parts, "; "), nil
}

// below is the name a file read for the input in goes by in the key: its
// name below the folder $NAME names, or the path as declared.
func below(in, file string) (string, error) {
	if !strings.HasPrefix(in, "$") {
		return in, nil
	}
	name, err := filepath.Rel(os.Getenv(in[1:]), file)
	if err != nil || strings.HasPrefix(name, "..") {
		return "", fmt.Errorf("%s read %s, which is not in its folder",
			in, file)
	}
	return filepath.ToSlash(name), nil
}

// measure is a file's length and the sha256 of its bytes.
func measure(path string) (int64, string, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, "", err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return 0, "", fmt.Errorf("%s: %w", path, err)
	}
	return n, hex.EncodeToString(h.Sum(nil)), nil
}

// askTimeout is how long smokecheck waits on git or go for an answer about a
// tree, for the record.
const askTimeout = 20 * time.Second

// slayerModule is slayer's module path, which its go.mod names. smokecheck
// lives in slayer now, and the trees it records are slayer's and
// libtheme's.
const slayerModule = "github.com/janearc/slayer"

// RecordText is what a run drew with and from, for people, kept beside each
// entry it stores. The commits are of the trees on disk when it drew, which
// are not always what the binary was built from; the key is the binary.
func RecordText() string {
	var b strings.Builder
	exe, _ := os.Executable()
	fmt.Fprintf(&b, "drawn %s\nby %s\n", time.Now().UTC().Format(
		time.RFC3339), exe)
	fmt.Fprintf(&b, "go %s %s/%s, -trimpath %t\n", runtime.Version(),
		runtime.GOOS, runtime.GOARCH, Trimmed())
	b.WriteString("on disk when drawn, not always what was built:\n")
	slayer, err := slayerDir()
	if err != nil {
		fmt.Fprintf(&b, "trees unknown: %v\n", err)
		return b.String()
	}
	fmt.Fprintf(&b, "slayer %s\n", commit(slayer))
	libtheme, err := replaced(slayer)
	if err != nil {
		fmt.Fprintf(&b, "libtheme unknown: %v\n", err)
		return b.String()
	}
	fmt.Fprintf(&b, "libtheme %s\n", commit(libtheme))
	return b.String()
}

// slayerDir is the slayer tree this was built from: where this file was,
// when that is known, and otherwise the nearest slayer go.mod above the
// working directory. The file is in render/smokecache, two below the tree.
func slayerDir() (string, error) {
	if _, file, _, ok := runtime.Caller(0); ok && filepath.IsAbs(file) {
		dir := filepath.Dir(filepath.Dir(filepath.Dir(file)))
		if isSlayer(dir) {
			return dir, nil
		}
	}
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if isSlayer(dir) {
			return dir, nil
		}
		up := filepath.Dir(dir)
		if up == dir {
			return "", fmt.Errorf("no slayer tree above %s", dir)
		}
		dir = up
	}
}

// isSlayer is whether dir holds slayer's go.mod.
func isSlayer(dir string) bool {
	f, err := os.Open(filepath.Join(dir, "go.mod"))
	if err != nil {
		return false
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		if fields := strings.Fields(s.Text()); len(fields) == 2 &&
			fields[0] == "module" {
			return fields[1] == slayerModule
		}
	}
	return false
}

// replaced is where slayer's build finds libtheme: the directory its
// go.mod replaces it with, as go itself resolves it.
func replaced(slayer string) (libtheme string, err error) {
	out, err := ask(slayer, "go", "list", "-m", "-f", "{{.Path}} {{.Dir}}",
		"github.com/janearc/libtheme-css")
	if err != nil {
		return "", err
	}
	path, dir, _ := strings.Cut(strings.TrimSpace(out), " ")
	if path != "github.com/janearc/libtheme-css" || dir == "" {
		return "", fmt.Errorf("go list did not say where libtheme is: %q",
			oneLine(out))
	}
	return dir, nil
}

// commit is the commit a tree is at, in words: with what is uncommitted
// in it, an untracked file included, or why it cannot be read.
func commit(dir string) string {
	head, err := ask(dir, "git", "rev-parse", "HEAD")
	if err != nil {
		return "unknown: " + err.Error()
	}
	head = strings.TrimSpace(head)
	status, err := ask(dir, "git", "status", "--porcelain",
		"--untracked-files=normal")
	switch {
	case err != nil:
		return head + ", its changes unknown: " + err.Error()
	case strings.TrimSpace(status) != "":
		return head + ", with uncommitted changes: " + oneLine(status)
	}
	return head
}

// ask runs a command in dir and is its output, or what went wrong.
func ask(dir, name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), askTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	var out, errs bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errs
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s %s: %v: %s", name,
			strings.Join(args, " "), err, oneLine(errs.String()))
	}
	return out.String(), nil
}
