package sources

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"
)

// changed says what a refresh did to a file, from its sha256 before (empty
// for a file that was not there) and after.
func changed(before, after string) string {
	switch before {
	case "":
		return "it was not on the shelf"
	case after:
		return "same bytes as before"
	}
	return fmt.Sprintf("changed, sha256 was %s, now %s", before, after)
}

// refresh fetches the dataset again though it is there, and puts the new
// file over the old only when it is as pinned; a fetch that fails leaves
// the old file. a member a set reads is unpacked again from the new
// archive the same way. it says what changed.
func (tend tending) refresh(ctx context.Context, found *place) error {
	name := found.product.dataset
	oldArchive, err := tend.holds(found.archive, "")
	if err != nil {
		return err
	}
	oldRead, err := tend.holds(found.read, "")
	if err != nil {
		return err
	}
	sum, size, err := tend.bring(ctx, found)
	if err != nil {
		return err
	}
	fmt.Fprintf(tend.say, "%s: refreshed %d bytes from %s, sha256 %s; %s\n",
		name, size, found.product.from, sum, changed(oldArchive, sum))
	if !found.member {
		return nil
	}
	unpacked, err := unpackMember(found.archive, found.product.member,
		found.read)
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	fmt.Fprintf(tend.say, "%s: unpacked %s to %s, sha256 %s; %s\n", name,
		found.product.member.name, found.read, unpacked,
		changed(oldRead, unpacked))
	return nil
}

// check compares the file a set reads with the source, and downloads
// nothing: its sha256 with the one the list pins, and its size with the
// size the server reports to a HEAD request. it says each comparison it
// made and each it could not, and is an error for a file that is
// missing, a server that cannot be reached, or a difference.
func (tend tending) check(ctx context.Context, found *place) error {
	name := found.product.dataset
	sum, err := tend.holds(found.read, "")
	if err != nil {
		return err
	}
	if sum == "" {
		return fmt.Errorf("missing: no file at %s", found.read)
	}
	declared, err := tend.origin.Declared(ctx, found.product)
	if err != nil {
		return err
	}
	var compared, differs []string
	for _, each := range []comparison{tend.comparedSum(found, sum),
		tend.comparedSize(found, declared)} {
		fmt.Fprintf(tend.say, "%s: %s\n", name, each.said)
		if each.compared {
			compared = append(compared, each.what)
		}
		if each.differs {
			differs = append(differs, each.what)
		}
	}
	if len(differs) > nothing {
		return fmt.Errorf("differs from %s in %s", tend.origin.Name(),
			strings.Join(differs, " and "))
	}
	fmt.Fprintf(tend.say, "%s: same as %s in what was compared: %s\n", name,
		tend.origin.Name(), strings.Join(compared, " and "))
	return nil
}

// a comparison is what one check of a file said, what it compared, and
// whether it could and found a difference.
type comparison struct {
	what, said        string
	compared, differs bool
}

// comparedSum is the file's sha256 against the pin of the list.
func (tend tending) comparedSum(found *place, sum string) comparison {
	switch {
	case found.pin == "":
		return comparison{what: "sha256", said: fmt.Sprintf("sha256 not "+
			"compared: %s pins none; the file's is %s", tend.origin.Name(),
			sum)}
	case sum == found.pin:
		return comparison{what: "sha256", compared: true,
			said: fmt.Sprintf("sha256 matches the pin of %s (%s)",
				tend.origin.Name(), sum)}
	}
	return comparison{what: "sha256", compared: true, differs: true,
		said: fmt.Sprintf("sha256 differs: the file at %s has %s, %s pins %s",
			found.read, sum, tend.origin.Name(), found.pin)}
}

// comparedSize is the size of the file the server's declaration is of
// against the size it declares: the file itself, or for a member the
// archive it came from, when that is on the shelf.
func (tend tending) comparedSize(found *place,
	declared *Declaration) comparison {
	what := comparison{what: "size"}
	at := found.read
	if found.member {
		at = found.archive
	}
	size, err := fileSize(at)
	want, given := declared.size, declared.sized
	switch {
	case !given:
		what.said = fmt.Sprintf("size not compared: the server gave no "+
			"size (%s)", declared.asked)
	case errors.Is(err, fs.ErrNotExist):
		what.said = fmt.Sprintf("size not compared: %s is not on the "+
			"shelf; the server reports %d bytes", at, want)
	case err != nil:
		what.said = fmt.Sprintf("size not compared: %v", err)
	case size == want:
		what.compared = true
		what.said = fmt.Sprintf("size matches the server (%d bytes, %s)",
			size, declared.asked)
	default:
		what.compared, what.differs = true, true
		what.said = fmt.Sprintf("size differs: %s has %d bytes, the server "+
			"reports %d (%s)", at, size, want, declared.asked)
	}
	return what
}
