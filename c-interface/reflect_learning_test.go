package c_interface_test

import (
	"encoding/json"
	"fmt"
	c_interface "github.com/reuben-baek/go-learning/c-interface"
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
	user := c_interface.UserInstance(1, "reuben")

	s := reflect.TypeOf(user)
	for i := 0; i < s.NumMethod(); i++ {
		f := s.Method(i)
		fmt.Printf("%d: %+v\n", i, f)
	}
}

func TestJsonMarshalReflection(t *testing.T) {
	user := c_interface.UserInstance(1, "reuben")

	bytes, _ := json.Marshal(user)
	fmt.Printf("%s\n", bytes)
}
