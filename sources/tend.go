package sources

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
)

// unpackedDir is the folder of the shelf a member of an archive is
// unpacked to, under the member's own path.
const unpackedDir = "unpacked"

// readPath is where a set's file of a product is on the shelf: the
// unpacked member when the file is that member, the file named otherwise,
// or the product's own file when none is named.
func (shelf Shelf) readPath(product *Product, file string) string {
	if file == "" {
		return filepath.Join(shelf.dir, product.name)
	}
	return filepath.Join(shelf.dir, filepath.FromSlash(file))
}

// memberFile is the shelf-relative path a member is unpacked to.
func memberFile(member *packed) string {
	return path.Join(unpackedDir, filepath.ToSlash(member.name))
}

// isMember is whether a set's file of a product is its unpacked member.
func isMember(product *Product, file string) bool {
	return product.member != nil && file == memberFile(product.member)
}

// Reads is the path a set may read a dataset's file at: the dataset's
// own file when file is empty, or the file under the shelf it names. it
// downloads nothing, and for an unpacked member it asks nothing of the
// archive it came from.
//
// a file not there is one error that names the dataset, the path looked
// at, and the flag that fetches it (or, where flag is empty, the verb
// that does). a file there is checked against its pin: the member's own
// where the file is the unpacked member, the source's otherwise.
func (shelf Shelf) Reads(source Source, file, flag string) (string, error) {
	product := productOf(source)
	at := shelf.readPath(product, file)
	member := isMember(product, file)
	looked := []string{at}
	if !member {
		looked = append(looked, shelf.Path(source))
	}
	for _, each := range looked {
		if _, err := os.Stat(each); errors.Is(err, fs.ErrNotExist) {
			return "", absent(product, each, flag)
		}
	}
	if !member {
		_, err := shelf.Check(source)
		return at, err
	}
	sum, err := shelf.hashed(at)
	if err != nil {
		return "", fmt.Errorf("%s: %w", product.dataset, err)
	}
	if sum != product.member.sum {
		return "", fmt.Errorf("%s: sha256 %s on the shelf at %s, the list "+
			"pins %s for the unpacked member", product.dataset, sum, at,
			product.member.sum)
	}
	return at, nil
}

// absent is the refusal of a dataset not on the shelf at a path.
func absent(product *Product, looked, flag string) error {
	how := fmt.Sprintf("fetch it with `sources fetch %s`", product.dataset)
	if flag != "" {
		how = fmt.Sprintf("pass %s to fetch it from %s", flag, product.from)
	}
	return fmt.Errorf("sources: dataset %s is not on the shelf: looked "+
		"for %s: %w; nothing was downloaded; %s", product.dataset, looked,
		fs.ErrNotExist, how)
}

// A tending is one run of Tend: the shelf, where the bytes come from,
// and where to say what happened.
type tending struct {
	shelf  Shelf
	origin Origin
	say    io.Writer
}

// the places of one want: its product, the archive's path on the shelf,
// the path the set reads, and the pin of the file at that path.
type place struct {
	product       *Product
	archive, read string
	pin           string
	member        bool
}

// placed is where a want's files are.
func (tend tending) placed(want Want) (*place, error) {
	products, err := tend.origin.Products(want.key)
	if err != nil {
		return nil, err
	}
	if len(products) != single {
		return nil, fmt.Errorf("%s: %s holds %d products, want one",
			want.key, tend.origin.Name(), len(products))
	}
	var product *Product
	for _, each := range products {
		product = each
	}
	found := &place{product: product, pin: product.sum,
		archive: tend.shelf.readPath(product, ""),
		read:    tend.shelf.readPath(product, want.file),
		member:  isMember(product, want.file)}
	if found.member {
		found.pin = product.member.sum
	}
	return found, nil
}

// Tend does a mode over the datasets a set names, and says on say what
// it did: Fetching fetches those missing from the shelf, Refreshing
// fetches all of them again and replaces the files only with good ones,
// and Checking downloads nothing and compares the shelf with the source.
// the first dataset that cannot be tended stops a fetch; a check goes
// through all of them and fails if any differs or is missing.
func (shelf Shelf) Tend(ctx context.Context, origin Origin, mode Mode,
	wants []Want, say io.Writer) error {
	tend := tending{shelf: shelf, origin: origin, say: say}
	failed := nothing
	for _, want := range wants {
		found, err := tend.placed(want)
		if err == nil {
			err = tend.one(ctx, mode, found)
		}
		if err != nil && mode != Checking {
			return err
		}
		if err != nil {
			fmt.Fprintf(say, "%s: %v\n", want.key, err)
			failed++
		}
	}
	if failed > nothing {
		return fmt.Errorf("sources: %d of %d datasets differ from %s or "+
			"are missing", failed, len(wants), origin.Name())
	}
	return nil
}

// one tends one want in a mode.
func (tend tending) one(ctx context.Context, mode Mode, found *place) error {
	switch mode {
	case Refreshing:
		return tend.refresh(ctx, found)
	case Checking:
		return tend.check(ctx, found)
	}
	return tend.fetch(ctx, found)
}

// fetch brings what is missing: a file there and as pinned is left, an
// archive there and good is unpacked, and a file there that is not as
// pinned is left alone and said, for a refresh to replace.
func (tend tending) fetch(ctx context.Context, found *place) error {
	have, err := tend.holds(found.read, found.pin)
	if err != nil {
		return err
	}
	if have != "" {
		fmt.Fprintf(tend.say, "%s: %s is on the shelf, sha256 %s as %s "+
			"pins it; not fetched\n", found.product.dataset, found.read,
			have, tend.origin.Name())
		return nil
	}
	if _, err := os.Stat(found.archive); errors.Is(err, fs.ErrNotExist) {
		sum, size, err := tend.bring(ctx, found)
		if err != nil {
			return err
		}
		fmt.Fprintf(tend.say, "%s: fetched %d bytes from %s, sha256 %s\n",
			found.product.dataset, size, found.product.from, sum)
	} else if _, err := tend.holds(found.archive, found.product.sum); err != nil {
		return err
	}
	return tend.unpacked(found)
}

// holds is the sha256 of a file that is there and as pinned, or nothing
// for a file that is not there, or an error for one that is there and is
// not as pinned. a pin of nothing holds any file.
func (tend tending) holds(at, pin string) (string, error) {
	sum, err := hashOf(at)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return "", nil
	case err != nil:
		return "", err
	case pin != "" && sum != pin:
		return "", fmt.Errorf("%s has sha256 %s, %s pins %s; "+
			"--refresh-terrain replaces it", at, sum, tend.origin.Name(), pin)
	}
	return sum, nil
}

// bring fetches a product into a folder of its own beside the archive's
// place, holds it to its pin and size, and only then moves it to the
// archive's place, over any file there. a fetch that fails, or is not as
// pinned, leaves the archive's place as it was and no folder behind.
func (tend tending) bring(ctx context.Context, found *place) (string,
	int64, error) {
	product := found.product
	if err := os.MkdirAll(tend.shelf.dir, folderMode); err != nil {
		return "", nothing, err
	}
	folder, err := os.MkdirTemp(tend.shelf.dir, ".fetching-")
	if err != nil {
		return "", nothing, err
	}
	defer os.RemoveAll(folder)
	got, err := tend.origin.Fetch(ctx, product, folder)
	if err != nil {
		return "", nothing, fmt.Errorf("%s: %w", product.dataset, err)
	}
	sum, size, err := summed(got)
	if err == nil {
		err = held(product.sum, product.size, sum, size)
	}
	if err != nil {
		return "", nothing, fmt.Errorf("%s: %w", product.dataset, err)
	}
	return sum, size, os.Rename(got, found.archive)
}

// summed is a file's sha256 and size.
func summed(at string) (string, int64, error) {
	sum, err := hashOf(at)
	if err != nil {
		return "", nothing, err
	}
	size, err := fileSize(at)
	return sum, size, err
}

// fileSize is how many bytes a file has.
func fileSize(at string) (int64, error) {
	info, err := os.Stat(at)
	if err != nil {
		return nothing, err
	}
	return info.Size(), nil
}

// unpacked unpacks the member a set reads, when it reads one, from the
// archive to its place under the shelf, holds it to its pin and size,
// and says so. a member that is not as pinned is removed, and refused.
func (tend tending) unpacked(found *place) error {
	if !found.member {
		return nil
	}
	member := found.product.member
	sum, err := unpackMember(found.archive, member, found.read)
	if err != nil {
		return fmt.Errorf("%s: %w", found.product.dataset, err)
	}
	fmt.Fprintf(tend.say, "%s: unpacked %s from the archive to %s, %d "+
		"bytes, sha256 %s\n", found.product.dataset, member.name,
		found.read, member.size, sum)
	return nil
}

// unpackMember writes the one member of a zip, by exactly its name, to a
// path, through a name of its own, and moves it into place only when its
// size and sha256 are the member's. it reads no more bytes than the
// member's size and one, so an archive that lies cannot fill the disk.
// it gives the sha256.
func unpackMember(archive string, member *packed, to string) (string, error) {
	if !filepath.IsLocal(member.name) {
		return "", fmt.Errorf("member %q is not a path inside its folder",
			member.name)
	}
	opened, err := zip.OpenReader(archive)
	if err != nil {
		return "", fmt.Errorf("%s: %w", archive, err)
	}
	defer opened.Close()
	for _, each := range opened.File {
		if each.Name == member.name {
			return copyMember(each, member, to)
		}
	}
	return "", fmt.Errorf("%s holds no member %q", archive, member.name)
}

// copyMember writes a zip's file to a path as unpackMember says.
func copyMember(from *zip.File, member *packed, to string) (string, error) {
	reading, err := from.Open()
	if err != nil {
		return "", err
	}
	defer reading.Close()
	if err := os.MkdirAll(filepath.Dir(to), folderMode); err != nil {
		return "", err
	}
	out, err := os.Create(to + partEnding)
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	size, err := io.Copy(io.MultiWriter(out, hash),
		io.LimitReader(reading, member.size+single))
	err = errors.Join(err, out.Close())
	sum := hex.EncodeToString(hash.Sum(nil))
	if err == nil && (size != member.size || sum != member.sum) {
		err = fmt.Errorf("member %q has %d bytes, sha256 %s; the list "+
			"pins %d bytes, sha256 %s", member.name, size, sum, member.size,
			member.sum)
	}
	if err != nil {
		os.Remove(to + partEnding)
		return "", err
	}
	return sum, os.Rename(to+partEnding, to)
}

// unpackedFor unpacks the pinned member of a source's archive on the
// shelf to its place, unless it is there as pinned, and says which.
func (shelf Shelf) unpackedFor(source Source) (string, error) {
	product := productOf(source)
	to := shelf.readPath(product, memberFile(product.member))
	if matches(to, product.member.sum) {
		return fmt.Sprintf("member %s kept, %s", product.member.name, to), nil
	}
	if _, err := unpackMember(shelf.Path(source), product.member, to); err != nil {
		return "", fmt.Errorf("%s: %w", source.Key(), err)
	}
	return fmt.Sprintf("member %s unpacked to %s", product.member.name, to), nil
}
