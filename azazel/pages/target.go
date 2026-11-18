package pages

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/janearc/slayer/azazel/internal/queue"
)

// outFlags are the arguments that name where a job's tool writes.
var outFlags = []string{"--out", "--output"}

// outOf is the path a job's arguments write to, as --out PATH or
// --out=PATH, or --output in either form.
func outOf(args []string) (string, bool) {
	for index, argument := range args {
		for _, flag := range outFlags {
			if value, given := strings.CutPrefix(argument, flag+"="); given {
				return value, true
			}
			if argument == flag && index+1 < len(args) {
				return args[index+1], true
			}
		}
	}
	return "", false
}

// isShelf says whether a name is one of the two shelves of render folders.
func isShelf(name string) bool {
	return name == framesShelf || name == animationsShelf
}

// folderUnder is the render folder a clean absolute path is in or names:
// ROOT/frames/X or ROOT/animations/X. A picture or film directly on a
// shelf is in no render folder.
func folderUnder(root, path string) (Folder, bool) {
	rest, under := strings.CutPrefix(path, root+"/")
	parts := strings.Split(rest, "/")
	if !under || len(parts) < 2 || !isShelf(parts[0]) {
		return Folder{}, false
	}
	if len(parts) == 2 && (shown(parts[1]) || isFilm(parts[1])) {
		return Folder{}, false
	}
	return Folder{Shelf: parts[0], Name: parts[1]}, true
}

// outputsSaid is what a job or set file says it writes: its outputs, each
// by name, as heard's chunk job files and slayer's sets have them.
type outputsSaid struct {
	Outputs map[string]any `json:"outputs" yaml:"outputs"`
}

// namedFiles is every job or set file the job's arguments name, read from
// the job's directory when it is not absolute.
func namedFiles(request queue.Request) []string {
	var files []string
	for _, argument := range request.Args {
		record := strings.HasSuffix(argument, ".yaml") ||
			strings.HasSuffix(argument, ".yml") ||
			strings.HasSuffix(argument, ".json")
		if record {
			files = append(files, besideRecord(request.Directory+"/",
				argument))
		}
	}
	return files
}

// foldersNamed is the render folders a job or set file's outputs land in.
func foldersNamed(root, file string) []Folder {
	var said outputsSaid
	if !decodeRecord(file, &said) {
		return nil
	}
	var found []Folder
	for _, value := range said.Outputs {
		path, isPath := value.(string)
		if !isPath || path == "" {
			continue
		}
		folder, under := folderUnder(root,
			filepath.Clean(besideRecord(file, path)))
		if under {
			found = append(found, folder)
		}
	}
	return found
}

// folderOf is the render folder a job writes into: where its --out or
// --output lands, else where the outputs of the job or set files it names
// land. With none, or two, it says why it publishes nothing.
func folderOf(root string, request queue.Request) (Folder, string) {
	if out, given := outOf(request.Args); given {
		folder, under := folderUnder(root, filepath.Clean(besideRecord(
			request.Directory+"/", out)))
		if !under {
			return Folder{}, "its --out is not in a render folder under " + root
		}
		return folder, ""
	}
	var found []Folder
	for _, file := range namedFiles(request) {
		found = append(found, foldersNamed(root, file)...)
	}
	if len(found) == 0 {
		return Folder{}, "it has no --out, and no job or set file it names " +
			"has outputs in a render folder under " + root
	}
	for _, folder := range found {
		if folder != found[0] {
			return Folder{}, "its job or set files' outputs land in two " +
				"render folders"
		}
	}
	return found[0], ""
}

// ParseFolder reads the publish verb's --folder: frames/X or
// animations/X, under the renders root or as an absolute path in it.
func ParseFolder(root, given string) (Folder, error) {
	path := given
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	path = filepath.Clean(path)
	folder, found := folderUnder(root, path)
	if !found || path != folder.path(root) {
		return Folder{}, fmt.Errorf("--folder must name a render folder, "+
			"frames/X or animations/X, under %s: %q", root, given)
	}
	return folder, nil
}
