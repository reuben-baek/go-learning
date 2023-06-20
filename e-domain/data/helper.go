package data

import (
	"errors"
	"fmt"
	"gorm.io/gorm"
	"reflect"
	"strings"
)

var NotFoundError = errors.New("not found")

func findID[T any, ID comparable](entity T) (ID, bool) {
	valueOfEntity := reflect.ValueOf(entity)
	if valueOfEntity.Type().Kind() == reflect.Pointer {
		valueOfEntity = reflect.Indirect(valueOfEntity)
	}
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

func findIDValue(ptrToEntity any, fieldName string) any {
	valueOfEntity := reflect.ValueOf(ptrToEntity)
	if valueOfEntity.Type().Kind() == reflect.Pointer {
		valueOfEntity = reflect.Indirect(valueOfEntity)
	}
	if !valueOfEntity.IsValid() {
		return nil
	}
	value := valueOfEntity.FieldByName(fieldName)
	if !value.IsValid() {
		panic(fmt.Sprintf("Entity '%s' has not %s field", valueOfEntity.Type(), fieldName))
	}
	if value.Type().Kind() == reflect.Pointer && value.IsNil() {
		return nil
	}
	if value.IsZero() {
		return nil
	}
	return value.Interface()
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

type AssociationType int

const (
	BelongTo AssociationType = iota + 1
	HasOne
	HasMany
	ManyToMany
)

type Association struct {
	Name        string
	PtrToEntity any
	ID          any
	ForeignKey  string
	Type        AssociationType
	FetchMode   FetchMode
}

var systemStructTypes = []any{
	gorm.Model{},
	LazyLoader{},
}

var systemStructTypeMap map[string]bool

func init() {
	systemStructTypeMap = make(map[string]bool)
	for _, s := range systemStructTypes {
		typeName := reflect.TypeOf(s).String()
		systemStructTypeMap[typeName] = true
	}
}

func isSystemStructType(typeName reflect.Type) bool {
	return systemStructTypeMap[typeName.String()]
}

func toSnakeCase(camel string) string {
	var b strings.Builder
	diff := 'a' - 'A'
	l := len(camel)
	for i, v := range camel {
		// A is 65, a is 97
		if v >= 'a' {
			b.WriteRune(v)
			continue
		}
		if (i != 0 || i == l-1) && (          // head and tail
		(i > 0 && rune(camel[i-1]) >= 'a') || // pre
			(i < l-1 && rune(camel[i+1]) >= 'a')) { //next
			b.WriteRune('_')
		}
		b.WriteRune(v + diff)
	}
	return b.String()
}

func findAssociations(ptrToEntity any) []Association {
	var associations []Association
	entityType := reflect.TypeOf(ptrToEntity)

	if entityType.Kind() == reflect.Pointer || entityType.Kind() == reflect.Slice {
		entityType = entityType.Elem()
		if entityType.Kind() == reflect.Slice {
			entityType = entityType.Elem()
		}
	}
	if entityType.Kind() != reflect.Struct {
		panic(fmt.Sprintf("findAssociation: entity[%s] is not struct type", entityType.String()))
	}
	numOfField := entityType.NumField()
	for i := 0; i < numOfField; i++ {
		field := entityType.Field(i)
		if isSystemStructType(field.Type) {
			continue
		}

		var association Association
		association.Name = field.Name
		association.FetchMode = ToFetchMode(field.Tag.Get("fetch"))

		belongToForeignKey := fmt.Sprintf("%sID", field.Name)
		hasForeignKey := fmt.Sprintf("%sID", entityType.Name())
		if field.Type.Kind() == reflect.Struct {
			association.PtrToEntity = reflect.New(field.Type).Interface()
			if _, ok := entityType.FieldByName(belongToForeignKey); ok {
				association.Type = BelongTo
				association.ID = findIDValue(ptrToEntity, belongToForeignKey)
			} else if _, ok := field.Type.FieldByName(hasForeignKey); ok {
				association.Type = HasOne
				association.ForeignKey = toSnakeCase(hasForeignKey)
			}
		} else if field.Type.Kind() == reflect.Pointer && field.Type.Elem().Kind() == reflect.Struct {
			association.PtrToEntity = reflect.New(field.Type.Elem()).Interface()
			if _, ok := entityType.FieldByName(belongToForeignKey); ok {
				association.Type = BelongTo
				association.ID = findIDValue(ptrToEntity, belongToForeignKey)
			} else if _, ok := field.Type.Elem().FieldByName(hasForeignKey); ok {
				association.Type = HasOne
				association.ForeignKey = toSnakeCase(hasForeignKey)
			}
		} else if field.Type.Kind() == reflect.Slice && field.Type.Elem().Kind() == reflect.Struct {
			association.PtrToEntity = reflect.New(field.Type).Interface()
			if _, ok := field.Type.Elem().FieldByName(hasForeignKey); ok {
				association.Type = HasMany
				association.ForeignKey = toSnakeCase(hasForeignKey)
			} else {
				association.Type = ManyToMany
				association.ForeignKey = toSnakeCase(hasForeignKey)
			}
		} else {
			continue
		}

		associations = append(associations, association)
	}
	return associations
}

func ptrToEmptyElementOfPtrToSlice(ptrToSlice any) any {
	ptrToSliceType := reflect.TypeOf(ptrToSlice)
	if ptrToSliceType.Kind() == reflect.Pointer {
		if ptrToSliceType.Elem().Kind() == reflect.Slice {
			ptrToElementValue := reflect.New(ptrToSliceType.Elem().Elem())
			return ptrToElementValue.Interface()
		} else {
			panic(fmt.Sprintf("%v is not pointer to slice", ptrToSliceType))
		}
	} else {
		panic(fmt.Sprintf("%v is not pointer to slice", ptrToSliceType))
	}
}
