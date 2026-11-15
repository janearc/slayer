package sources_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"

	"github.com/janearc/slayer/sources"
)

// listedAt is the origin of a one-source list whose source is at a url,
// and the folder the list is in.
func listedAt(from string) (*sources.Listed, string) {
	dir, _ := os.MkdirTemp("", "origin")
	text := "sources:\n  - key: moon\n    from: " + from +
		"/moon.tif\n    licence: a test's own\n"
	listed := filepath.Join(dir, "sources.yaml")
	os.WriteFile(listed, []byte(text), fileMode)
	list, _ := sources.Read(listed)
	return sources.NewListed(list, http.DefaultClient), dir
}

// tinySize is how many bytes the tiny moon has; defined: the length of
// "tiny moon".
const tinySize = 9

// theProduct is the one product the origin holds for the moon.
func theProduct(origin *sources.Listed) *sources.Product {
	var only *sources.Product
	held, _ := origin.Products("moon")
	for _, each := range held {
		only = each
	}
	return only
}

// an origin is anything that names itself, lists the products it holds
// for a dataset, fetches one into a folder and says what it is now; the
// list in sources.yaml is the one there is today.
func ExampleOrigin() {
	var origin sources.Origin
	origin, dir := listedAt("https://x.test")
	defer os.RemoveAll(dir)
	fmt.Println(origin.Name())
	// Output: sources.yaml
}

// a product is made with its name on the shelf, its url, and the pin
// the origin gives it.
func ExampleNewProduct() {
	product := sources.NewProduct("moon", "moon.tif", "https://x.test/moon.tif",
		"", none)
	fmt.Printf("%T\n", product)
	// Output: *sources.Product
}

// a declaration is made with the size the origin gives, if it gives one.
func ExampleNewDeclaration() {
	declared := sources.NewDeclaration(tinySize, true, "", "HEAD https://x.test/moon.tif")
	fmt.Printf("%T\n", declared)
	// Output: *sources.Declaration
}

// the list's origin is named for the file.
func ExampleListed_Name() {
	origin, dir := listedAt("https://x.test")
	defer os.RemoveAll(dir)
	fmt.Println(origin.Name())
	// Output: sources.yaml
}

// the list holds one product for a dataset it names, and none for one it
// does not.
func ExampleListed_Products() {
	origin, dir := listedAt("https://x.test")
	defer os.RemoveAll(dir)
	held, _ := origin.Products("moon")
	_, err := origin.Products("sun")
	fmt.Println(len(held), err != nil)
	// Output: 1 true
}

// a fetch brings a product into a folder under its name on the shelf.
func ExampleListed_Fetch() {
	server := httptest.NewServer(http.HandlerFunc(tinyHandler))
	defer server.Close()
	origin, dir := listedAt(server.URL)
	defer os.RemoveAll(dir)
	got, _ := origin.Fetch(context.Background(), theProduct(origin), dir)
	fmt.Println(filepath.Base(got))
	// Output: moon.tif
}

// a declaration is asked of the server with a HEAD request.
func ExampleListed_Declared() {
	server := httptest.NewServer(http.HandlerFunc(tinyHandler))
	defer server.Close()
	origin, dir := listedAt(server.URL)
	defer os.RemoveAll(dir)
	_, err := origin.Declared(context.Background(), theProduct(origin))
	fmt.Println(err == nil)
	// Output: true
}
