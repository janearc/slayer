package sources

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
)

// A getter is how a request reaches a server. An *http.Client is one, so
// a test may hand in a client of a server on localhost.
type getter interface {
	Do(request *http.Request) (*http.Response, error)
}

// A packed is the one file of an archive a set reads: its path inside the
// archive, and its sha256 and size once unpacked.
type packed struct {
	name, sum string
	size      int64
}

// A Product is one file a dataset comes as: the dataset it is of, the
// name it has on the shelf, where it is fetched from, the sha256 and size
// the list pins for it (empty and zero where it pins none), and the
// member a set reads from it when it is an archive (nil when it is not).
type Product struct {
	dataset, name, from, sum string
	size                     int64
	member                   *packed
}

// A Declaration is what a source says a product is, now: its size where
// the server gives one, and its sha256 where the source gives one.
type Declaration struct {
	size  int64
	sized bool
	sum   string
	asked string
}

// NewProduct is a product of a dataset: the name it has on the shelf,
// where it is fetched from, and the sha256 and size the origin pins for it
// (empty and zero where it pins none).
func NewProduct(dataset, name, from, sum string, size int64) *Product {
	return &Product{dataset: dataset, name: name, from: from, sum: sum,
		size: size}
}

// NewDeclaration is what an origin says a product is: its size where
// sized is true, its sha256 where it gives one (empty where it does not),
// and the request or record it said it by, for saying what was compared.
func NewDeclaration(size int64, sized bool, sum, asked string) *Declaration {
	return &Declaration{size: size, sized: sized, sum: sum, asked: asked}
}

// An Origin is where a named dataset's bytes come from. celestial-data's
// Archive and Fetch satisfy it through a thin wrapper.
type Origin interface {
	// Name is what the origin is called in what is said of it.
	Name() string
	// Products is the files the origin holds for a dataset.
	Products(dataset string) ([]*Product, error)
	// Fetch brings a product into a folder and gives the path of the
	// file; it never replaces one.
	Fetch(ctx context.Context, product *Product, into string) (string, error)
	// Declared is what the origin says the product is now.
	Declared(ctx context.Context, product *Product) (*Declaration, error)
}

// Listed is the origin of sources.yaml: the url and sha256 a list gives.
type Listed struct {
	list List
	get  getter
}

// NewListed is the origin of a list, fetching through a getter.
func NewListed(list List, get getter) *Listed {
	return &Listed{list: list, get: get}
}

// Name is the list's file name.
func (listed *Listed) Name() string { return listFile }

// listFile is what the list is called in what is said of it.
const listFile = "sources.yaml"

// Products is the one product the list holds for a dataset.
func (listed *Listed) Products(dataset string) ([]*Product, error) {
	source, found := listed.list.Find(dataset)
	if !found {
		return nil, fmt.Errorf("%s is not in %s", dataset, listFile)
	}
	if source.Never() != "" {
		return nil, fmt.Errorf("%s: never fetched: %s", dataset, source.Never())
	}
	return []*Product{productOf(source)}, nil
}

// productOf is the product a source is, with its member where it has one.
func productOf(source Source) *Product {
	product := &Product{dataset: source.Key(), name: fileName(source),
		from: source.From(), sum: source.row.Sha256, size: source.row.Bytes}
	if inside := source.row.Unpack; inside != nil {
		product.member = &packed{name: inside.member, sum: inside.sum,
			size: inside.size}
	}
	return product
}

// Fetch brings the product's bytes into a folder under its name, and
// refuses to replace a file already there. bytes cut short are an error.
func (listed *Listed) Fetch(ctx context.Context, product *Product,
	into string) (string, error) {
	if err := os.MkdirAll(into, folderMode); err != nil {
		return "", err
	}
	at := filepath.Join(into, product.name)
	if _, err := os.Lstat(at); err == nil {
		return "", fmt.Errorf("%w: %s is already there; a fetch never "+
			"replaces a file", errFetch, at)
	}
	if _, _, err := download(ctx, listed.get, product.from, at+partEnding); err != nil {
		os.Remove(at + partEnding)
		return "", fmt.Errorf("%s: %w", product.dataset, err)
	}
	return at, os.Rename(at+partEnding, at)
}

// Declared is what a HEAD request to the product's url says of its size,
// and the sha256 the list pins for it. a server that cannot be reached,
// or does not answer 200, is an error and not a size.
func (listed *Listed) Declared(ctx context.Context,
	product *Product) (*Declaration, error) {
	asking, err := http.NewRequestWithContext(ctx, http.MethodHead,
		product.from, nil)
	if err != nil {
		return nil, err
	}
	answer, err := listed.get.Do(asking)
	if err != nil {
		return nil, fmt.Errorf("could not reach %s: %w", product.from, err)
	}
	answer.Body.Close()
	if answer.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("could not reach %s: it answered %s",
			product.from, answer.Status)
	}
	return &Declaration{size: answer.ContentLength,
		sized: answer.ContentLength >= nothing, sum: product.sum,
		asked: "HEAD " + product.from}, nil
}

const (
	// folderMode is the mode of a folder a fetch makes; defined: the
	// owner writes it, all others read it.
	folderMode = 0o755
	// partEnding is what a file is called while it is still coming in;
	// defined: the ending the sources fetch already gives.
	partEnding = ".part"
	// nothing is zero; defined: no bytes, and a length the server did not
	// give is below it.
	nothing = 0
	// single is the count of products the list holds for a dataset;
	// defined: one.
	single = 1
)

// A Mode is what is done with the datasets a set names.
type Mode int

// the three things done with a set's datasets.
const (
	Fetching   Mode = iota // fetch those missing from the shelf
	Refreshing             // fetch all again, replace when good
	Checking               // download nothing; compare with the source
)

// A Want is a dataset a set names and the file of it the set reads.
type Want struct{ key, file string }

// NewWant is the dataset of a key, and the file the set reads of it under
// the shelf, or nothing for the dataset's own file.
func NewWant(key, file string) Want { return Want{key: key, file: file} }
