package f_repository_impl

import (
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
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

func findValue(entity any, fieldName string) any {
	valueOfEntity := reflect.ValueOf(entity)
	value := valueOfEntity.FieldByName(fieldName)
	if !value.IsValid() {
		panic(fmt.Sprintf("Entity '%s' has not %s field", valueOfEntity.Type(), fieldName))
	}
	return value.Interface()
}

func isZero[ID comparable](id ID) bool {
	value := reflect.ValueOf(id)
	return value.IsZero()
}

func findPreloadModels[T any](entity T) []string {
	// todo cache, recursive
	var models []string
	typeOf := reflect.TypeOf(entity)
	numOfField := typeOf.NumField()
	for i := 0; i < numOfField; i++ {
		field := typeOf.Field(i)
		if field.Type.Kind() == reflect.Struct {
			fieldType := field.Type.String()
			fieldTypeName := field.Type.Name()
			logrus.Debugf("field: Name=%s, Type=%s", fieldTypeName, fieldType)
			if fieldType != "gorm.Model" {
				if field.Tag.Get("fetch") == FetchEagerMode {
					models = append(models, fieldTypeName)
				}
			}
		} else if field.Type.Kind() == reflect.Slice {
			if field.Type.Elem().Kind() == reflect.Struct {
				elementType := field.Type.Elem()
				fieldType := elementType.String()
				fieldTypeName := elementType.Name()
				logrus.Debugf("field: Name=%s, Type=%s", fieldTypeName, fieldType)
				if field.Tag.Get("fetch") == FetchEagerMode {
					models = append(models, fieldTypeName+"s")
				}
			}
		}
	}
	return models
}

type FetchMode string

const (
	FetchEagerMode = "eager"
	FetchLazyMode  = "lazy"
)

func ToFetchMode(m string) FetchMode {
	switch m {
	case "", FetchLazyMode:
		return FetchLazyMode
	case FetchEagerMode:
		return FetchEagerMode
	default:
		panic(fmt.Sprintf("wrong fetch-mode - %s", m))
	}
}

type Association struct {
	Name      string
	Value     any
	ID        any
	FetchMode FetchMode
}

func findAssociations[T any](entity T) []Association {
	var associations []Association
	valueOf := reflect.ValueOf(entity)
	typeOf := reflect.TypeOf(entity)

	numOfField := valueOf.NumField()
	for i := 0; i < numOfField; i++ {
		field := valueOf.Field(i)
		if field.Type().Kind() == reflect.Struct {
			fieldType := field.Type().String()
			if fieldType != "gorm.Model" && fieldType != "f_repository_impl.LazyLoadableImpl" && fieldType != "f_repository_impl.LazyLoad" {
				fieldTypeName := field.Type().Name()
				logrus.Debugf("field: Name=%s, Type=%s, Value=%+v", fieldTypeName, fieldType, field.Interface())

				fetchMode := ToFetchMode(typeOf.Field(i).Tag.Get("fetch"))
				switch fetchMode {
				case FetchLazyMode:
					v := reflect.New(field.Type())
					associations = append(associations, Association{
						Name:      fieldTypeName,
						Value:     v.Interface(),
						FetchMode: ToFetchMode(typeOf.Field(i).Tag.Get("fetch")),
						ID:        findValue(entity, fmt.Sprintf("%sID", fieldTypeName)), // belongTo ID
					})
				case FetchEagerMode:
					v := reflect.New(field.Type())
					associations = append(associations, Association{
						Name:      fieldTypeName,
						Value:     v.Interface(),
						FetchMode: ToFetchMode(typeOf.Field(i).Tag.Get("fetch")),
					})
				}
			}
		} else if field.Type().Kind() == reflect.Slice {
			if field.Type().Elem().Kind() == reflect.Struct {
				elementType := field.Type().Elem()
				fieldType := elementType.String()
				fieldTypeName := elementType.Name()
				logrus.Debugf("field: Name=%s, Type=%s, Value=%+v", fieldTypeName, fieldType, field.Interface())

				associations = append(associations, Association{
					Name:      fieldTypeName + "s",
					Value:     field.Interface(),
					FetchMode: ToFetchMode(typeOf.Field(i).Tag.Get("fetch")),
				})
			}
		}
	}
	return associations
}
