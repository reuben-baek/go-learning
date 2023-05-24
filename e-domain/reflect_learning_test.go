package e_domain

import (
	"fmt"
	"reflect"
	"testing"
)

func TestStructReflection(t *testing.T) {
	v := struct {
		ID   string
		Func func() bool
	}{
		ID:   "1",
		Func: func() bool { return true },
	}

	ref := reflect.ValueOf(v)
	fmt.Printf("%+v\n", ref)
	if ref.Kind() == reflect.Struct {
		idField := ref.FieldByName("ID")
		fmt.Printf("%+v\n", idField)
		funcField := ref.FieldByName("Func")
		fmt.Printf("%+v\n", funcField)
	}
}

func TestInterfaceReflection(t *testing.T) {
	type Box interface {
		X() int
		Y() int
	}

	var box Box

	ref := reflect.TypeOf(box)
	fmt.Printf("%+v\n", ref)
}
