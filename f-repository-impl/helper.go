package f_repository_impl

import (
	"errors"
	"fmt"
	"reflect"
)

var NotFoundError = errors.New("not found")

func findID[T any, ID comparable](entity T) (ID, bool) {
	valueOfEntity := reflect.ValueOf(entity)
	value := valueOfEntity.FieldByName("ID")
	if !value.IsValid() {
		panic(fmt.Sprintf("Entity '%s' has not ID field", valueOfEntity.Type()))
	}
	if !value.Comparable() {
		panic(fmt.Sprintf("ID field type '%s' of '%s' is not comparable", value.Type(), valueOfEntity.Type()))
	}
	v := value.Interface()
	switch v.(type) {
	case ID:
		return v.(ID), value.IsZero()
	default:
		panic("Entity's ID field type is different from ID type constraint")
	}
}

func isZero[ID comparable](id ID) bool {
	value := reflect.ValueOf(id)
	return value.IsZero()
}
