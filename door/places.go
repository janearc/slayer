package door

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/janearc/slayer/diskcache"
	"github.com/janearc/slayer/job"
)

// places is where a job's outputs go, each an absolute path or nothing
// for a role the job does not write, and what goes with them: the folder
// under the renders home, the copy of the job file and the note.
type places struct {
	picture, raw  string
	frames, movie string
	// folder is the output's folder under the renders home, relative
	// to it, or nothing where the output is not under one.
	folder string
	// copied is where the file given is copied, or nothing.
	copied string
	// notes is the notes file the note goes in, or nothing; why says
	// why a note the job asks for is not written.
	notes, note, why string
	// resumed is the folder of an earlier day this film resumes, under
	// the renders home, or nothing.
	resumed string
	// work is whether the frames are working files: drawn for the movie
	// alone, and removed once it is written.
	work bool
}

// placing is what places are worked out from.
type placing struct {
	wanted job.Job
	given  string // the file given, absolute
	env    Env
	key    string // the frames' key, as the ledger records it
}

// placesOf is where the job's outputs go: as the job says them, under
// out/ where it says none, and under the renders home where one is named
// and an output is said with no directory or not said at all. A file
// given from inside the renders home is in its folder already: its
// outputs stay where it says them. A film that keeps working frames
// is refused where the cache home is not named.
func placesOf(from placing) (places, error) {
	home, hasHome := from.env.lookup(rendersVariable)
	_, inside := relativeTo(home, filepath.Dir(from.given))
	moves := hasHome && !inside
	made, err := from.laid(home, moves)
	if err != nil {
		return places{}, err
	}
	if moves && made.folder != "" {
		made.copied = filepath.Join(home, made.folder,
			from.name()+jobExtension)
	}
	return from.noted(made, home, hasHome), nil
}

// laid is the places of the picture or the film, before its copy and its
// note: a film that moves resumes the folder of an earlier day where its
// ledger is there.
func (from placing) laid(home string, moves bool) (places, error) {
	folder := from.datedFolder(home)
	if from.wanted.Clock().Still() {
		return from.still(folder, moves), nil
	}
	earlier := ""
	if moves {
		earlier = from.earlierFolder(home)
	}
	if earlier != "" {
		folder = earlier
	}
	made, err := from.film(folder, moves)
	made.resumed = earlier
	return made, err
}

// name is the base name of the file given, without its extension.
func (from placing) name() string {
	base := filepath.Base(from.given)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// datedFolder is the folder, under a renders home's frames or
// animations, that the job's outputs go in when moved there: the date
// and the name.
func (from placing) datedFolder(home string) string {
	date := from.env.now().Format(dateLayout)
	kind := filmsFolder
	if from.wanted.Clock().Still() {
		kind = stillsFolder
	}
	return filepath.Join(kind, date+"-"+from.name())
}

// earlierFolder is the newest folder of this film's name from another
// day, under the renders home's animations, whose ledger holds frames of
// this film's key, so a film resumed after midnight finishes where it
// began; or nothing where there is none.
func (from placing) earlierFolder(home string) string {
	if from.key == "" {
		return ""
	}
	today := from.datedFolder(home)
	pattern := filepath.Join(home, filmsFolder, datePattern+"-"+
		globbed(from.name()))
	found, _ := filepath.Glob(pattern)
	slices.Sort(found)
	slices.Reverse(found)
	planned := from.plannedWork()
	for _, folder := range found {
		relative, _ := filepath.Rel(home, folder)
		if relative == today {
			continue
		}
		if keyOfFolder(folder) == from.key || planned == folder {
			return relative
		}
	}
	return ""
}

// plannedWork is the folder of the movie this film's working frames are
// for; or nothing where there is no working folder to look in.
func (from placing) plannedWork() string {
	work, err := from.workFolder()
	if err != nil {
		return ""
	}
	return plannedFolder(work, from.key)
}

// plannedFolder is the folder of the movie a working folder's frames are
// for, where its ledger holds frames of key; or nothing.
func plannedFolder(work, key string) string {
	found, read := ledgerOf(work)
	if !read || found.Key != key || found.Planned == "" {
		return ""
	}
	return filepath.Dir(found.Planned)
}

// globbed is a name with the characters a glob reads escaped.
func globbed(name string) string {
	var escaped strings.Builder
	for _, each := range name {
		if strings.ContainsRune(`*?[\`, each) {
			escaped.WriteRune('\\')
		}
		escaped.WriteRune(each)
	}
	return escaped.String()
}

// keyOfFolder is the key a frames folder's ledger records, or nothing.
func keyOfFolder(folder string) string {
	found, _ := ledgerOf(folder)
	return found.Key
}

// ledgerOf is a frames folder's ledger as it is written, and whether it
// could be read.
func ledgerOf(folder string) (manifest, bool) {
	raw, err := os.ReadFile(filepath.Join(folder, manifestName))
	if err != nil {
		return manifest{}, false
	}
	var found manifest
	return found, json.Unmarshal(raw, &found) == nil
}

// bare is whether a path was said with no directory: the job resolves a
// bare name into the given file's own directory.
func (from placing) bare(path string) bool {
	return filepath.Dir(path) == filepath.Dir(from.given)
}

// out is a path under the working directory's out/.
func (from placing) out(name string) string {
	return filepath.Join(from.env.directory, outFolder, name)
}

// still is a still's places: its picture, and its raw light where said.
func (from placing) still(folder string, hasHome bool) places {
	home, _ := from.env.lookup(rendersVariable)
	picture, said := from.wanted.Outputs().Picture()
	raw, hasRaw := from.wanted.Outputs().Raw()
	forms := from.wanted.Outputs()
	if !said {
		picture = from.out(from.name() + pictureExtension)
	}
	if !hasHome || (said && (!forms.PictureBare() || !from.bare(picture))) {
		return places{picture: picture, raw: raw}
	}
	inside := filepath.Join(home, folder)
	made := places{folder: folder,
		picture: filepath.Join(inside, filepath.Base(picture)), raw: raw}
	if hasRaw && forms.RawBare() && from.bare(raw) {
		made.raw = filepath.Join(inside, filepath.Base(raw))
	}
	return made
}

// film is a film's places: its frames and its movie, both under out/
// where the job says neither.
func (from placing) film(folder string, hasHome bool) (places, error) {
	made := from.saidFilm(folder, hasHome)
	if made.frames != "" || made.movie == "" {
		return made, nil
	}
	work, err := from.workFolder()
	made.frames, made.work = work, true
	return made, err
}

// saidFilm is a film's frames and movie as the job says them: under the
// dated folder where moved there, and a movie alone, under out/ or the
// renders home, where it says neither.
func (from placing) saidFilm(folder string, hasHome bool) places {
	home, _ := from.env.lookup(rendersVariable)
	frames, hasFrames := from.wanted.Outputs().Frames()
	movie, hasMovie := from.wanted.Outputs().Movie()
	forms := from.wanted.Outputs()
	inside := filepath.Join(home, folder)
	if !hasFrames && !hasMovie && hasHome {
		return places{folder: folder,
			movie: filepath.Join(inside, from.name()+movieExtension)}
	}
	if !hasFrames && !hasMovie {
		return places{movie: from.out(from.name() + movieExtension)}
	}
	made := places{frames: frames, movie: movie}
	if !hasHome {
		return made
	}
	if hasFrames && forms.FramesBare() && from.bare(frames) {
		made.frames, made.folder = inside, folder
	}
	if hasMovie && forms.MovieBare() && from.bare(movie) {
		made.movie = filepath.Join(inside, filepath.Base(movie))
		made.folder = folder
	}
	return made
}

// workFolder is where a film that writes a movie and no frames keeps
// its frames while it draws: the frames folder of the cache home, by
// team where SLAYER_TEAM is set and with no team folder where it is not,
// since a person outside this machine has no team.
//
// It refuses by name where SLAYER_CACHE is not set; there is no other
// place. Its name is the job's and a digest of the job file's path, so it
// is the same folder on every day and for every version of the job.
func (from placing) workFolder() (string, error) {
	sum := sha256.Sum256([]byte(from.given))
	name := from.name() + "-" + hex.EncodeToString(sum[:])[:workDigits]
	home, err := diskcache.Home(from.env.lookup)
	if err != nil {
		return "", err
	}
	team, _ := from.env.lookup(teamVariable)
	return filepath.Join(home, workFolder, team, name), nil
}

// noted is the places with the note the job asks for, where it can be
// written, and why not where it cannot.
func (from placing) noted(made places, home string, hasHome bool) places {
	note, said := from.wanted.Outputs().Note()
	if !said {
		return made
	}
	team, hasTeam := from.env.lookup(teamVariable)
	folder, under := made.underHome(home)
	switch {
	case !hasHome:
		made.why = "no renders home is named (" + rendersVariable + ")"
	case !hasTeam:
		made.why = "no team is named (" + teamVariable + ")"
	case !under:
		made.why = "the output is not under the renders home " + home
	default:
		made.folder = folder
		made.notes = filepath.Join(home, team, notesName)
		made.note = note
	}
	return made
}

// underHome is the output's folder relative to the renders home, and
// whether it is under it: the folder moved there, or else the movie's
// folder, the frames folder, or the picture's folder.
func (made places) underHome(home string) (string, bool) {
	if made.folder != "" {
		return made.folder, true
	}
	folder := filepath.Dir(made.picture)
	switch {
	case made.movie != "":
		folder = filepath.Dir(made.movie)
	case made.frames != "":
		folder = made.frames
	}
	return relativeTo(home, folder)
}

// relativeTo is a folder's path under home, and whether it is under it;
// home itself is not under it.
func relativeTo(home, folder string) (string, bool) {
	relative, err := filepath.Rel(home, folder)
	if home == "" || err != nil || relative == "." || relative == ".." ||
		strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", false
	}
	return relative, true
}

// written is each file the places write, in words, one a line.
func (made places) written() []string {
	var lines []string
	add := func(path, says string) {
		if path != "" {
			lines = append(lines, fmt.Sprintf(says, path))
		}
	}
	add(made.picture, "picture %s")
	add(made.raw, "raw light %s")
	if made.work {
		add(made.frames, "working frames in %s, removed once the movie "+
			"is written")
	} else {
		add(made.frames, "frames in %s, with its "+manifestName)
	}
	add(made.movie, "movie %s")
	add(made.copied, "the job, copied to %s")
	add(made.notes, "a note line in %s")
	return lines
}
