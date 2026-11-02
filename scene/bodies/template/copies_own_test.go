package template

import (
	"reflect"
	"testing"

	"github.com/janearc/slayer/scene/bodies"
)

// makers is every template by its name, as named.go names them, which
// TestEveryTemplateHasAName holds is every one there is.
var makers = named

// scribble changes everything reachable from v that can be changed: each
// number is raised, each string lengthened, each bool turned, through
// pointers, interfaces, structs, slices, arrays and maps. a field that
// is not exported is left, as reflect leaves it.
func scribble(v reflect.Value) {
	switch v.Kind() {
	case reflect.Pointer:
		if !v.IsNil() {
			scribble(v.Elem())
		}
	case reflect.Interface:
		if v.IsNil() || !v.CanSet() {
			return
		}
		inside := reflect.New(v.Elem().Type()).Elem()
		inside.Set(v.Elem())
		scribble(inside)
		v.Set(inside)
	case reflect.Struct:
		for i := range v.NumField() {
			if v.Field(i).CanSet() {
				scribble(v.Field(i))
			}
		}
	case reflect.Slice, reflect.Array:
		for i := range v.Len() {
			scribble(v.Index(i))
		}
	case reflect.Map:
		for _, k := range v.MapKeys() {
			inside := reflect.New(v.Type().Elem()).Elem()
			inside.Set(v.MapIndex(k))
			scribble(inside)
			v.SetMapIndex(k, inside)
		}
	case reflect.Float32, reflect.Float64:
		v.SetFloat(v.Float() + 1)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v.SetInt(v.Int() + 1)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v.SetUint(v.Uint() + 1)
	case reflect.String:
		v.SetString(v.String() + "!")
	case reflect.Bool:
		v.SetBool(!v.Bool())
	}
}

// each body hands back a copy of its own: with every number, string and
// part of one call's result changed, a fresh call of any template, this
// one or another, is still as it was.
//
// scribble reaches every part a body has, colours included, and the
// first check holds that it changed the copy, so the second is not
// passed by a scribble that did nothing.
func TestEachBodyHandsBackItsOwnCopy(t *testing.T) {
	pristine := map[string]bodies.Body{}
	for name, make := range makers {
		pristine[name] = make()
	}
	for name, make := range makers {
		mine := make()
		scribble(reflect.ValueOf(&mine).Elem())
		if len(apartWithColours(mine, pristine[name])) == 0 {
			t.Errorf("%s: scribbling changed nothing, so the check below proves nothing", name)
		}
		for other, again := range makers {
			for _, a := range apartWithColours(again(), pristine[other]) {
				t.Errorf("after changing a %s, a fresh %s differs: %s", name, other, a)
			}
		}
	}
}

// a call twice is the same body twice, before anything is changed.
func TestTwoCallsOfATemplateAreTheSameBody(t *testing.T) {
	for name, make := range makers {
		for _, a := range apartWithColours(make(), make()) {
			t.Errorf("%s: two calls differ: %s", name, a)
		}
	}
}
