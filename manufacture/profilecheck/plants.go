package profilecheck

import (
	"fmt"
	"reflect"

	"github.com/janearc/puzzlemouth/yamlfile"
	"gopkg.in/yaml.v3"
)

// PlantRows reads one bounded plant book and returns independently owned rows.
func PlantRows[T any](file string, raw []byte, optional func(*yaml.Node, string) bool) ([]T, error) {
	return Rows[T](file, raw, "plant profile book", "plant profile", optional)
}

// Rows reads one bounded sequence book, retaining its family's diagnostics.
func Rows[T any](file string, raw []byte, bookName, profileName string, optional func(*yaml.Node, string) bool) ([]T, error) {
	if len(raw) == 0 || len(raw) > 1<<20 {
		return nil, fmt.Errorf("%s: empty or oversized %s", file, bookName)
	}
	if err := NumbersPresent(file, raw, reflect.TypeOf((*T)(nil)).Elem(), optional); err != nil {
		return nil, err
	}
	var book struct{ Profiles []T }
	if err := yamlfile.Read(file, raw, &book); err != nil {
		return nil, err
	}
	if len(book.Profiles) == 0 || len(book.Profiles) > 256 {
		return nil, fmt.Errorf("%s: invalid %s count", file, profileName)
	}
	return book.Profiles, nil
}
