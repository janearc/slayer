package pages

import (
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// folderStyle is frames-index.sh's style, through the opening body tag.
//
//go:embed folder-style.html
var folderStyle string

// folderMark is azazel's mark on a folder page it wrote.
const folderMark = `<meta name="generator" content="azazel">`

// scriptMark is frames-index.sh's mark: a page it wrote is not a hand
// page, and azazel writes over it.
const scriptMark = `<meta name="generator" content="frames-index.sh">`

// scriptGrid is the grid only frames-index.sh's pages have; with the
// folder's heading it marks a page the script wrote before its mark.
const scriptGrid = "minmax(300px, 1fr)"

// More than mostInFull pictures are shown as shownOfMany, evenly spaced.
const (
	mostInFull  = 60 // defined: frames-index.sh's most
	shownOfMany = 48 // defined: frames-index.sh's shown
)

// keptNotice is what the job's log says of a hand page it cannot set
// aside.
const keptNotice = "index.html was made by hand and " + handPageName +
	" is already there; azazel leaves both and writes no folder page"

// contents is what a folder page lists: its pictures and its films, each
// by its path in the folder, in the Mac sort's order.
type contents struct {
	pictures []string
	films    []string
}

// isPicture says whether a name is a picture the folder page shows.
func isPicture(name string) bool {
	return strings.HasSuffix(name, ".png") && !strings.HasPrefix(name, "icon-")
}

// isFilm says whether a name is a film the folder page shows.
func isFilm(name string) bool {
	if strings.HasPrefix(filepath.Base(name), ".") {
		return false
	}
	return strings.HasSuffix(name, ".mp4") || strings.HasSuffix(name, ".mov") ||
		strings.HasSuffix(name, ".m4v")
}

// scanFolder finds what frames-index.sh's find does: every entry at any
// depth, a file, a link or a folder, named as a picture or a film. Links
// are not followed. An entry gone while it is read is left out.
func scanFolder(dir string) (contents, error) {
	var found contents
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry,
		err error) error {
		if err != nil {
			return gone(err, path != dir)
		}
		inside := strings.TrimPrefix(path, dir+"/")
		switch {
		case path == dir:
		case isPicture(entry.Name()):
			found.pictures = append(found.pictures, inside)
		case isFilm(entry.Name()):
			found.films = append(found.films, inside)
		}
		return nil
	})
	slices.SortFunc(found.pictures, collate)
	slices.SortFunc(found.films, collate)
	return found, err
}

// gone forgives a file that vanished while a folder was walked, other
// than the folder itself.
func gone(err error, forgiven bool) error {
	if forgiven && errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}

// folderLead is the page's count line, as frames-index.sh words it.
func folderLead(count, reels, step int) string {
	if count == 0 && reels == 0 {
		return "No pictures in this folder yet."
	}
	said := fmt.Sprintf("%d pictures.", count)
	if step > 1 {
		said = fmt.Sprintf("%d pictures; every %dth shown.", count, step)
	}
	if reels > 0 {
		said += fmt.Sprintf(" %d films.", reels)
	}
	return said
}

// folderPage is a render folder's page: frames-index.sh's page with
// azazel's mark, and a lead line for each thing in lines.
func folderPage(name string, found contents, lines []string) string {
	count := len(found.pictures)
	step := 1
	if count > mostInFull {
		step = (count + shownOfMany - 1) / shownOfMany
	}
	var page strings.Builder
	page.WriteString("<!doctype html>\n<html lang=\"en\">\n<head>\n" +
		"<meta charset=\"utf-8\">\n<meta name=\"viewport\" " +
		"content=\"width=device-width, initial-scale=1\">\n")
	page.WriteString(folderMark + "\n")
	fmt.Fprintf(&page, "<title>%s</title>\n%s", escape(name), folderStyle)
	fmt.Fprintf(&page, "<h1>%s</h1>\n", escape(name))
	lead := folderLead(count, len(found.films), step)
	for _, line := range append([]string{lead}, lines...) {
		fmt.Fprintf(&page, "<p class=\"lead\">%s</p>\n", line)
	}
	page.WriteString("<div class=\"grid\">\n")
	folderGrid(&page, found, step)
	page.WriteString("</div>\n</body>\n</html>\n")
	return page.String()
}

// folderGrid writes the films, then every step'th picture.
func folderGrid(page *strings.Builder, found contents, step int) {
	for _, film := range found.films {
		one := escape(film)
		fmt.Fprintf(page, "<figure><video controls preload=\"metadata\" "+
			"src=\"%s\"></video><figcaption><a href=\"%s\">%s</a>"+
			"</figcaption></figure>\n", one, one, one)
	}
	for index := 0; index < len(found.pictures); index += step {
		one := escape(found.pictures[index])
		fmt.Fprintf(page, "<figure><a href=\"%s\"><img loading=\"lazy\" "+
			"src=\"%s\" alt=\"%s\"></a><figcaption>%s</figcaption>"+
			"</figure>\n", one, one, one, one)
	}
}

// ours says whether a folder page was written by azazel or the script,
// and not by hand.
func ours(page, name string) bool {
	return strings.Contains(page, folderMark) ||
		strings.Contains(page, scriptMark) ||
		strings.Contains(page, scriptGrid) &&
			strings.Contains(page, "<h1>"+escape(name)+"</h1>")
}

// setAside moves a hand page out of index.html's way, once, by linking it
// as index-before-azazel.html, which a link never overwrites. It says
// whether the hand page must be kept where it is: when that name already
// holds another page. A link made before a failed write is not another.
func setAside(dir, name string) (bool, error) {
	path := filepath.Join(dir, pageName)
	current, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil || ours(string(current), name) {
		return false, err
	}
	aside := filepath.Join(dir, handPageName)
	err = os.Link(path, aside)
	if errors.Is(err, fs.ErrExist) {
		return !sameFile(path, aside), nil
	}
	return false, err
}

// sameFile says whether two names are one file.
func sameFile(one, other string) bool {
	first, err := os.Stat(one)
	second, wrong := os.Stat(other)
	return err == nil && wrong == nil && os.SameFile(first, second)
}

// writePage writes a folder page when it differs from the one there.
func writePage(dir, page string) error {
	path := filepath.Join(dir, pageName)
	current, err := os.ReadFile(path)
	if err == nil && string(current) == page {
		return nil
	}
	return writeBeside(path, []byte(page), written)
}

// handLine is the lead line linking a hand page set aside, when there is
// one.
func handLine(dir string) []string {
	if !regularFile(filepath.Join(dir, handPageName)) {
		return nil
	}
	return []string{`<a href="` + handPageName +
		`">The page made by hand before azazel kept this one</a>`}
}

// publishFolder writes a render folder's page from what it holds and
// what its film holds, and returns anything the job's log must say.
func (p publisher) publishFolder(folder Folder, found contents,
	view filmView) (string, error) {
	dir := folder.path(p.root)
	kept, err := setAside(dir, folder.Name)
	if kept || err != nil {
		return keptNoticeIf(kept), err
	}
	lines := append(handLine(dir), view.lines()...)
	return "", writePage(dir, folderPage(folder.Name, found, lines))
}

// keptNoticeIf is the kept notice when a hand page was kept, else none.
func keptNoticeIf(kept bool) string {
	if kept {
		return keptNotice
	}
	return ""
}

// escape is the shell scripts' escape: ampersands, angle brackets and
// double quotes, nothing more.
func escape(text string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;",
		`"`, "&quot;").Replace(text)
}
