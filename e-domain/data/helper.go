package data

import (
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
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

func findValue(entity any, fieldName string) any {
	valueOfEntity := reflect.ValueOf(entity)
	if valueOfEntity.Type().Kind() == reflect.Pointer {
		valueOfEntity = reflect.Indirect(valueOfEntity)
	}
	value := valueOfEntity.FieldByName(fieldName)
	if !value.IsValid() {
		panic(fmt.Sprintf("Entity '%s' has not %s field", valueOfEntity.Type(), fieldName))
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
	Name       string
	Value      any
	ID         any
	ForeignKey string
	Type       AssociationType
	FetchMode  FetchMode
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

func isSystemStructType(typeName string) bool {
	return systemStructTypeMap[typeName]
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

func findAssociations[T any](entity T) []Association {
	var associations []Association
	entityValue := reflect.ValueOf(entity)
	entityType := reflect.TypeOf(entity)

	numOfField := entityValue.NumField()
	for i := 0; i < numOfField; i++ {
		field := entityValue.Field(i)
		if field.Type().Kind() == reflect.Struct {
			fieldType := field.Type().String()
			fieldTypeName := field.Type().Name()
			if !isSystemStructType(fieldType) {
				logrus.Debugf("findAssociation: field Name[%s], Type[%s], Value[%+v]", fieldTypeName, fieldType, field.Interface())

				fetchMode := ToFetchMode(entityType.Field(i).Tag.Get("fetch"))
				switch fetchMode {
				case FetchLazyMode:
					fieldValue := reflect.New(field.Type())

					belongToForeignKey := fmt.Sprintf("%sID", fieldTypeName)
					oneToOneForeignKey := fmt.Sprintf("%sID", entityType.Name())
					if entityValue.FieldByName(belongToForeignKey).IsValid() {
						associations = append(associations, Association{
							Name:      fieldTypeName,
							Value:     fieldValue.Interface(),
							FetchMode: FetchLazyMode,
							ID:        findValue(entity, fmt.Sprintf("%sID", fieldTypeName)), // belongTo ID
							Type:      BelongTo,
						})
					} else if reflect.Indirect(fieldValue).FieldByName(oneToOneForeignKey).IsValid() {
						associations = append(associations, Association{
							Name:       fieldTypeName,
							Value:      fieldValue.Interface(),
							FetchMode:  FetchLazyMode,
							ForeignKey: toSnakeCase(oneToOneForeignKey), // belongTo ID
							Type:       HasOne,
						})
					}
				case FetchEagerMode:
					associations = append(associations, Association{
						Name:      fieldTypeName,
						Value:     field.Interface(),
						FetchMode: ToFetchMode(entityType.Field(i).Tag.Get("fetch")),
						Type:      BelongTo,
					})
				}
			}
		} else if field.Type().Kind() == reflect.Slice {
			if field.Type().Elem().Kind() == reflect.Struct {
				elementType := field.Type().Elem()
				fieldType := elementType.String()
				fieldTypeName := elementType.Name()
				logrus.Debugf("findAssocation: field Name=[%s], Type[%s], Value[%+v]", fieldTypeName, fieldType, field.Interface())

				fetchMode := ToFetchMode(entityType.Field(i).Tag.Get("fetch"))
				switch fetchMode {
				case FetchLazyMode:
					fieldValue := reflect.New(field.Type())
					foreignKey := fmt.Sprintf("%sID", entityType.Name())
					if _, ok := elementType.FieldByName(foreignKey); ok {
						associations = append(associations, Association{
							Name:       fieldTypeName + "s",
							Value:      fieldValue.Interface(),
							FetchMode:  FetchLazyMode,
							ForeignKey: toSnakeCase(foreignKey), // has-many foreign ID
							Type:       HasMany,
						})
					} else {
						associations = append(associations, Association{
							Name:       fieldTypeName + "s",
							Value:      fieldValue.Interface(),
							FetchMode:  FetchLazyMode,
							ForeignKey: toSnakeCase(foreignKey), // many-to-many foreign ID
							Type:       ManyToMany,
						})
						//panic(fmt.Sprintf("findAssociations: foreignKey %s does not exist in %s", foreignKey, fieldType))
					}
				case FetchEagerMode:
					associations = append(associations, Association{
						Name:      fieldTypeName + "s",
						Value:     field.Interface(),
						FetchMode: ToFetchMode(entityType.Field(i).Tag.Get("fetch")),
						Type:      HasMany,
					})
				}
			}
		}
	}
	return associations
}
