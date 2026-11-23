package pages

import (
	"cmp"
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// latestHead is latest.sh's page head, through the end of its style.
//
//go:embed latest-head.html
var latestHead string

// latestFilter is latest.sh's script that shows a team its own folders,
// on a page reached through its compositing folder.
//
//go:embed latest-filter.html
var latestFilter string

// How many cards each gallery page holds.
const (
	latestCards = 120  // defined: latest.sh's default
	indexCards  = 2000 // defined: what indexes.sh asks latest.sh for
)

// The two time formats on a gallery page: when it was made, and when a
// card's newest file was written.
const (
	madeFormat  = "2006-01-02 15:04" // defined: latest.sh's date '+%Y-%m-%d %H:%M'
	stampFormat = "01-02 15:04"      // defined: latest.sh's date '+%m-%d %H:%M'
)

// seen is one picture or film a gallery page can show: where it is, and
// when it was last written, in the whole seconds stat gives.
type seen struct {
	path string
	when int64
}

// note is one line of a team's notes file: the folder it is about, under
// the renders root, and what it says.
type note struct {
	folder string
	says   string
}

// card is one folder on a gallery page: its newest picture or film, when
// that was written, how many it holds, and the folder.
type card struct {
	newest string
	when   int64
	count  int
	folder string
}

// galleryPage is one page latest.sh makes: its name under the renders
// root, its title, how many cards, and the folders it is limited to.
type galleryPage struct {
	name  string
	title string
	most  int
	keep  map[string]bool
}

// gallery is what one walk of the renders tree finds, for every page.
type gallery struct {
	root  string
	files []seen
	notes []note
	when  time.Time
	zone  *time.Location
}

// shown says whether latest.sh shows a file: a picture or a film, but
// not a catalogue icon.
func shown(name string) bool {
	if strings.HasPrefix(name, "icon-") && strings.HasSuffix(name, ".png") {
		return false
	}
	return strings.HasSuffix(name, ".png") || strings.HasSuffix(name, ".jpg") ||
		strings.HasSuffix(name, ".mp4")
}

// pruned says whether the walk leaves a folder out, as latest.sh does:
// the cache, the applications, the skeleton and any sources.
func pruned(root, path string) bool {
	switch path {
	case filepath.Join(root, "cache"), filepath.Join(root, "applications"),
		filepath.Join(root, "skel"):
		return true
	}
	return filepath.Base(path) == "sources"
}

// newestFirst orders files as latest.sh's sort does: newest first, and
// files of one second by their paths in the Mac sort's order.
func newestFirst(a, b seen) int {
	if order := cmp.Compare(b.when, a.when); order != 0 {
		return order
	}
	return collate(a.path, b.path)
}

// walkGallery finds every file a gallery page can show. Links are not
// followed, and a file gone while it is read is left out.
func walkGallery(root string) ([]seen, error) {
	var found []seen
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry,
		err error) error {
		if err != nil {
			return gone(err, path != root)
		}
		if entry.IsDir() && pruned(root, path) {
			return filepath.SkipDir
		}
		if !entry.Type().IsRegular() || !shown(entry.Name()) {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return gone(err, true)
		}
		found = append(found, seen{path, info.ModTime().Unix()})
		return nil
	})
	slices.SortFunc(found, newestFirst)
	return found, err
}

// readNotes reads every team's notes, RENDERS/TEAM/latest-notes.txt, in
// the Mac sort's order of their paths, as latest.sh does.
func readNotes(root string) ([]note, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, entry := range entries {
		path := filepath.Join(root, entry.Name(), notesName)
		info, err := os.Lstat(path)
		if entry.IsDir() && err == nil && info.Mode().IsRegular() {
			files = append(files, path)
		}
	}
	slices.SortFunc(files, collate)
	var all []note
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		dataText := string(data)
		all = append(all, notesIn(dataText)...)
	}
	return all, nil
}

// notesIn is the lines of one notes file that hold a tab, cut as cut and
// awk cut them: the folder before the first tab, the note before the
// next.
func notesIn(text string) []note {
	var found []note
	for line := range strings.Lines(text) {
		fields := strings.Split(strings.TrimSuffix(line, "\n"), "\t")
		if len(fields) > 1 {
			found = append(found, note{folder: fields[0], says: fields[1]})
		}
	}
	return found
}

// up is a path with its last part taken off, as latest.sh's awk does:
// nothing at all when there is no slash left.
func up(path string) string {
	at := strings.LastIndex(path, "/")
	if at < 0 {
		return ""
	}
	return path[:at]
}

// renderFolder is the render folder a picture's folder is in: at most two
// below the renders root.
func renderFolder(root, dir string) string {
	parts := strings.Split(strings.TrimPrefix(dir, root+"/"), "/")
	if dir == root || len(parts) <= 2 {
		return dir
	}
	return root + "/" + parts[0] + "/" + parts[1]
}

// noteFolder is the card of a picture's folder on the latest page: the
// nearest folder at or above it with a note of its own, or, failing that,
// the render folder it is in.
func noteFolder(root string, noted map[string]bool, dir string) string {
	for at := dir; at != root && at != ""; at = up(at) {
		if noted[at] {
			return at
		}
	}
	return renderFolder(root, dir)
}

// keptFolder is the card of a picture's folder on a page of one set of
// folders: the one of the set it is in or under, else its own.
func keptFolder(keep map[string]bool, dir string) string {
	for at := dir; at != ""; at = up(at) {
		if keep[at] {
			return at
		}
	}
	return dir
}

// cards groups the files into folders, newest first, at most page.most.
func (g gallery) cards(page galleryPage) []card {
	noted := map[string]bool{}
	for _, each := range g.notes {
		noted[g.root+"/"+each.folder] = true
	}
	place := func(dir string) string { return noteFolder(g.root, noted, dir) }
	if page.keep != nil {
		place = func(dir string) string { return keptFolder(page.keep, dir) }
	}
	index := map[string]int{}
	var all []card
	for _, file := range g.files {
		folder := place(filepath.Dir(file.path))
		if at, found := index[folder]; found {
			all[at].count++
			continue
		}
		index[folder] = len(all)
		all = append(all, card{file.path, file.when, 1, folder})
	}
	return kept(all, page)
}

// kept is the first page.most cards of the page's own folders.
func kept(all []card, page galleryPage) []card {
	var chosen []card
	for _, each := range all {
		if len(chosen) == page.most {
			break
		}
		if page.keep == nil || page.keep[each.folder] {
			chosen = append(chosen, each)
		}
	}
	return chosen
}

// noteOf is the last note on a folder, as awk keeps the last one.
func (g gallery) noteOf(dir string) string {
	said := ""
	for _, each := range g.notes {
		if each.folder == dir {
			said = each.says
		}
	}
	return said
}

// relative is a path under the renders root, as the shell strips it.
func (g gallery) relative(path string) string {
	return strings.TrimPrefix(path, g.root+"/")
}

// lastPart is what follows the last slash, as the shell's ${dir##*/}.
func lastPart(path string) string {
	return path[strings.LastIndex(path, "/")+1:]
}

// firstYaml is the first set file directly in a folder, in the Mac sort's
// order, or nothing.
func firstYaml(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	var found []string
	for _, entry := range entries {
		name := entry.Name()
		yaml := strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml")
		if yaml && entry.Type().IsRegular() {
			found = append(found, filepath.Join(dir, name))
		}
	}
	if len(found) == 0 {
		return "", nil
	}
	return slices.MinFunc(found, collate), nil
}

// copyNewer writes a set file's copy, NAME-yaml.txt, when the copy is
// missing or older by the second, as latest.sh's cp does.
func copyNewer(from, to string) error {
	source, err := os.Stat(from)
	if err != nil {
		return err
	}
	copied, err := os.Stat(to)
	fresh := err == nil && copied.Mode().IsRegular() &&
		copied.ModTime().Unix() >= source.ModTime().Unix()
	if fresh {
		return nil
	}
	data, err := os.ReadFile(from)
	if err != nil {
		return err
	}
	return writeBeside(to, data, source.Mode().Perm())
}

// recipe finds the set file a card's picture was drawn from, beside the
// picture or in the card's folder, makes sure of its copy and returns the
// copy's path under the renders root, or nothing.
func (g gallery) recipe(each card) (string, error) {
	for _, where := range []string{filepath.Dir(each.newest), each.folder} {
		found, err := firstYaml(where)
		if err != nil {
			return "", err
		}
		if found == "" {
			continue
		}
		copied := strings.TrimSuffix(found, filepath.Ext(found)) + "-yaml.txt"
		return g.relative(copied), copyNewer(found, copied)
	}
	return "", nil
}

// figure is one card, as latest.sh prints it.
func (g gallery) figure(each card) (string, error) {
	path, dir := g.relative(each.newest), g.relative(each.folder)
	link, book, opens := path, "", ""
	if each.count > 1 && regularFile(filepath.Join(each.folder, pageName)) {
		link = dir + "/" + pageName
		book = fmt.Sprintf("%d pictures here: click the picture to open "+
			"them all", each.count)
	}
	film := strings.HasSuffix(path, ".mp4")
	if film && book != "" {
		book, opens = fmt.Sprintf("%d files here", each.count), link
	}
	recipe, err := g.recipe(each)
	var out strings.Builder
	partName := lastPart(dir)
	fmt.Fprintf(&out, `<figure data-folder="%s">`, escape(partName))
	g.picture(&out, path, link, film)
	out.WriteString("<figcaption>")
	g.caption(&out, each, dir, book)
	links(&out, opens, recipe)
	out.WriteString("</figcaption></figure>\n")
	return out.String(), err
}

// picture is a card's player for a film, or its picture as a link.
func (g gallery) picture(out *strings.Builder, path, link string, film bool) {
	source := escape(path)
	if film {
		fmt.Fprintf(out, `<video src="%s" controls preload="metadata">`+
			`</video>`, source)
		return
	}
	fmt.Fprintf(out, `<a href="%s"><img loading="lazy" src="%s" alt="%s">`+
		`</a>`, escape(link), source, source)
}

// caption is a card's note, its count of pictures, and where and when.
func (g gallery) caption(out *strings.Builder, each card, dir, book string) {
	if said := g.noteOf(dir); said != "" {
		fmt.Fprintf(out, `<span class="note">%s</span>`, escape(said))
	}
	if book != "" {
		fmt.Fprintf(out, `<span class="note">%s</span>`, escape(book))
	}
	stamp := time.Unix(each.when, 0).In(g.zone).Format(stampFormat)
	fmt.Fprintf(out, `<span class="where">%s &middot; %d file(s) &middot; `+
		`%s</span>`, stamp, each.count, escape(dir))
}

// links is a card's buttons: to the folder's page, and to its set file.
func links(out *strings.Builder, opens, recipe string) {
	if opens != "" {
		fmt.Fprintf(out, `<a class="recipe" href="%s">open them all</a> `,
			escape(opens))
	}
	if recipe != "" {
		fmt.Fprintf(out, `<a class="recipe" href="%s">the yaml it was `+
			`drawn from</a>`, escape(recipe))
	}
}

// madeLine is the line that says when a gallery page was made.
const madeLine = `<p class="lead">Newest first, one card per folder. ` +
	`Made %s. Press cmd-r after a new render.</p>` + "\n"

// html is one gallery page, as latest.sh writes it.
func (g gallery) html(page galleryPage) (string, error) {
	var out strings.Builder
	out.WriteString(latestHead)
	base := ""
	if strings.Contains(page.name, "/") {
		base = `<base href="../">`
	}
	fmt.Fprintf(&out, "<title>%s</title>\n%s\n</head>\n<body>\n<h1>%s</h1>\n",
		escape(page.title), base, escape(page.title))
	fmt.Fprintf(&out, madeLine, g.when.In(g.zone).Format(madeFormat))
	err := g.grid(&out, page)
	if base != "" {
		out.WriteString(latestFilter)
	}
	out.WriteString("</body>\n</html>\n")
	return out.String(), err
}

// grid is a gallery page's cards.
func (g gallery) grid(out *strings.Builder, page galleryPage) error {
	out.WriteString("<div class=\"grid\">\n")
	var failures []error
	for _, each := range g.cards(page) {
		figure, err := g.figure(each)
		out.WriteString(figure)
		failures = append(failures, err)
	}
	out.WriteString("</div>\n")
	return errors.Join(failures...)
}

// shelfFolders is every folder directly on a shelf, not following links,
// as indexes.sh's find gives them.
func shelfFolders(root, shelf string) (map[string]bool, error) {
	entries, err := os.ReadDir(filepath.Join(root, shelf))
	keep := map[string]bool{}
	for _, entry := range entries {
		if entry.IsDir() {
			keep[filepath.Join(root, shelf, entry.Name())] = true
		}
	}
	return keep, err
}

// galleryPages is latest.html and the two shelves' indexes.
func galleryPages(root string) ([]galleryPage, error) {
	frames, err := shelfFolders(root, framesShelf)
	animations, wrong := shelfFolders(root, animationsShelf)
	return []galleryPage{
		{name: latestName, title: "Latest renders", most: latestCards},
		{name: framesShelf + "/" + pageName, title: "Frames",
			most: indexCards, keep: frames},
		{name: animationsShelf + "/" + pageName, title: "Animations",
			most: indexCards, keep: animations},
	}, errors.Join(err, wrong)
}

// publishGallery writes latest.html and the two indexes from one walk of
// the renders tree. A page that cannot be made whole is not written.
func (p publisher) publishGallery() error {
	files, err := walkGallery(p.root)
	notes, wrong := readNotes(p.root)
	if err = errors.Join(err, wrong); err != nil {
		return err
	}
	pages, missing := galleryPages(p.root)
	found := gallery{root: p.root, files: files, notes: notes, when: p.now(),
		zone: p.location}
	failures := []error{missing}
	for _, page := range pages {
		html, err := found.html(page)
		if err == nil {
			err = writeBeside(filepath.Join(p.root, page.name), []byte(html),
				written)
		}
		failures = append(failures, err)
	}
	return errors.Join(failures...)
}
