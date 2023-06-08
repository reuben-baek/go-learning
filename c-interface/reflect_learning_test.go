package c_interface_test

import (
	"encoding/json"
	"fmt"
	c_interface "github.com/reuben-baek/go-learning/c-interface"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

func TestStructReflection_Get(t *testing.T) {
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

func TestStructReflection_Set(t *testing.T) {
	type Meta struct {
		ID int
	}
	type Object struct {
		ID   int
		Meta Meta
	}

	object := Object{}
	valueOfObject := reflect.ValueOf(&object)
	fmt.Printf("%+v\n", valueOfObject)

	assert.Equal(t, reflect.Ptr, valueOfObject.Kind())

	indirectValueOfObject := reflect.Indirect(valueOfObject)
	idField := indirectValueOfObject.FieldByName("ID")
	assert.Equal(t, reflect.Int, idField.Kind())
	idField.SetInt(10)
	assert.Equal(t, 10, object.ID)

	metaValueOf := reflect.Indirect(reflect.ValueOf(&object.Meta))
	metaIdField := metaValueOf.FieldByName("ID")

	assert.Equal(t, reflect.Int, metaIdField.Kind())
	metaIdField.SetInt(10)
	assert.Equal(t, 10, object.Meta.ID)

	metaField := indirectValueOfObject.FieldByName("Meta")
	meta := Meta{ID: 20}
	metaField.Set(reflect.ValueOf(meta))
	assert.Equal(t, meta, object.Meta)
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
