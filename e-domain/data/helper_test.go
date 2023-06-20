package data

import (
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	"reflect"
	"testing"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
}

func TestFindID(t *testing.T) {
	t.Run("success - notempty id", func(t *testing.T) {
		type User struct {
			ID   string
			Name string
		}

		reuben := User{
			ID: "reuben.b",
		}

		id, zero := findID[User, string](reuben)
		assert.Equal(t, reuben.ID, id)
		assert.False(t, zero)
	})
	t.Run("success - empty id", func(t *testing.T) {
		type User struct {
			ID   string
			Name string
		}

		reuben := User{}

		id, zero := findID[User, string](reuben)
		assert.Equal(t, reuben.ID, id)
		assert.True(t, zero)
	})
	t.Run("fail - Entity has not ID field", func(t *testing.T) {
		assert.PanicsWithValue(t, "Entity 'data.TT' has not ID field", func() {
			type TT struct {
				Name string
			}
			reuben := TT{
				Name: "reuben",
			}
			findID[TT, string](reuben)
		})
	})
	t.Run("fail - Entity ID type is not comparable", func(t *testing.T) {
		assert.PanicsWithValue(t, "ID field type 'map[string]string' of 'data.TT' is not comparable", func() {
			type TT struct {
				ID map[string]string
			}
			reuben := TT{
				ID: nil,
			}
			findID[TT, string](reuben)
		})
	})
	t.Run("fail - Entity's ID field type is different from ID type constraint", func(t *testing.T) {
		assert.PanicsWithValue(t, "Entity's ID field type is different from ID type constraint", func() {
			type TT struct {
				ID string
			}
			reuben := TT{
				ID: "reuben",
			}
			findID[TT, int](reuben)
		})
	})
}

func TestFindAssociations(t *testing.T) {
	t.Run("eager", func(t *testing.T) {
		t.Run("1:1 cardinality", func(t *testing.T) {
			type Order struct {
				ID     int
				Name   string
				UserID uint
			}
			type User struct {
				gorm.Model
				Name  string
				Order Order `fetch:"eager"`
			}

			user := User{
				Model: gorm.Model{
					ID: 1,
				},
				Name: "reuben",
				Order: Order{
					ID:     1,
					Name:   "order-1",
					UserID: 1,
				},
			}

			expected := []Association{
				{
					Name:        "Order",
					PtrToEntity: &Order{},
					ID:          nil,
					ForeignKey:  "user_id",
					Type:        HasOne,
					FetchMode:   FetchEagerMode,
				},
			}
			associations := findAssociations(user)
			assert.Equal(t, expected, associations)
		})
		t.Run("1:n cardinality", func(t *testing.T) {
			type Order struct {
				ID     int
				Name   string
				UserID uint
			}
			type User struct {
				gorm.Model
				Name   string
				Orders []Order `fetch:"eager"`
			}

			user := User{
				Model: gorm.Model{
					ID: 1,
				},
				Name: "reuben",
				Orders: []Order{
					{
						ID:     1,
						Name:   "order-1",
						UserID: 1,
					},
				},
			}

			orders := []Order(nil)
			expected := []Association{
				{
					Name:        "Orders",
					PtrToEntity: &orders,
					ID:          nil,
					ForeignKey:  "user_id",
					Type:        HasMany,
					FetchMode:   FetchEagerMode,
				},
			}
			associations := findAssociations(user)
			assert.Equal(t, expected, associations)
		})
	})
	t.Run("lazy", func(t *testing.T) {
		t.Run("self-referential", func(t *testing.T) {
			t.Run("empty", func(t *testing.T) {
				type Category struct {
					ID       int
					Name     string
					ParentID *int
					Parent   *Category
				}

				computer := Category{
					ID:       1,
					Name:     "computer",
					ParentID: nil,
					Parent:   nil,
				}
				expected := []Association{
					{
						Name:        "Parent",
						PtrToEntity: &Category{},
						ID:          nil,
						ForeignKey:  "",
						Type:        BelongTo,
						FetchMode:   FetchLazyMode,
					},
				}
				associations := findAssociations(computer)

				assert.Equal(t, expected, associations)
			})
			t.Run("not empty", func(t *testing.T) {
				type Category struct {
					ID       int
					Name     string
					ParentID *int
					Parent   *Category
				}
				deviceID := 1
				computer := Category{
					ID:       2,
					Name:     "computer",
					ParentID: &deviceID,
					Parent:   nil,
				}
				expected := []Association{
					{
						Name:        "Parent",
						PtrToEntity: &Category{},
						ID:          &deviceID,
						ForeignKey:  "",
						Type:        BelongTo,
						FetchMode:   FetchLazyMode,
					},
				}
				associations := findAssociations(computer)

				assert.Equal(t, expected, associations)
			})
		})
		t.Run("belong-to", func(t *testing.T) {
			type Company struct {
				ID int
			}
			type User struct {
				LazyLoader `gorm:"-"`
				gorm.Model
				Name      string
				CompanyID int
				Company   Company `fetch:"lazy"`
			}

			reuben := User{
				Model: gorm.Model{
					ID: 1,
				},
				Name:      "reuben",
				CompanyID: 1,
			}
			expected := []Association{
				{
					Name:        "Company",
					PtrToEntity: &Company{},
					ID:          1,
					ForeignKey:  "",
					Type:        BelongTo,
					FetchMode:   FetchLazyMode,
				},
			}
			associations := findAssociations(reuben)

			assert.Equal(t, expected, associations)
		})
		t.Run("one-to-one", func(t *testing.T) {
			type CreditCard struct {
				ID     int
				UserID uint
			}
			type User struct {
				LazyLoader `gorm:"-"`
				gorm.Model
				Name       string
				CreditCard CreditCard `fetch:"lazy"`
			}

			reuben := User{
				Model: gorm.Model{
					ID: 1,
				},
				Name: "reuben",
			}
			expected := []Association{
				{
					Name:        "CreditCard",
					PtrToEntity: &CreditCard{},
					ForeignKey:  "user_id",
					Type:        HasOne,
					FetchMode:   FetchLazyMode,
				},
			}
			associations := findAssociations(reuben)

			assert.Equal(t, expected, associations)
		})
		t.Run("one-to-many", func(t *testing.T) {
			type CreditCard struct {
				ID     int
				UserID uint
			}
			type User struct {
				LazyLoader `gorm:"-"`
				gorm.Model
				Name        string
				CreditCards []CreditCard `fetch:"lazy"`
			}

			reuben := User{
				Model: gorm.Model{
					ID: 1,
				},
				Name: "reuben",
			}
			expected := []Association{
				{
					Name:        "CreditCards",
					PtrToEntity: &reuben.CreditCards,
					ForeignKey:  "user_id",
					Type:        HasMany,
					FetchMode:   FetchLazyMode,
				},
			}
			associations := findAssociations(reuben)

			assert.Equal(t, expected, associations)
		})
	})
}

func TestFindLazyEntity(t *testing.T) {
	type Company struct {
		ID int
	}
	type Order struct {
		ID     int
		UserID int
	}
	type Role struct {
		ID   int
		Name string
	}
	type User struct {
		LazyLoader `gorm:"-"`
		gorm.Model
		ID        int
		Name      string
		CompanyID string
		Company   Company `fetch:"lazy"`
		RoleID    int
		Role      Role    `fetch:"eager"`
		Orders    []Order `fetch:"eager"`
	}

	reuben := User{
		Model: gorm.Model{
			ID: 1,
		},
		Name:      "reuben",
		CompanyID: "1",
	}
	orders := []Order(nil)
	expected := []Association{
		{
			Name:        "Company",
			PtrToEntity: &Company{},
			ID:          "1",
			ForeignKey:  "",
			Type:        BelongTo,
			FetchMode:   FetchLazyMode,
		},
		{
			Name:        "Role",
			PtrToEntity: &Role{},
			ID:          nil,
			ForeignKey:  "",
			FetchMode:   FetchEagerMode,
			Type:        BelongTo,
		},
		{
			Name:        "Orders",
			PtrToEntity: &orders,
			ForeignKey:  "user_id",
			FetchMode:   FetchEagerMode,
			Type:        HasMany,
		},
	}
	associations := findAssociations(reuben)

	assert.Equal(t, expected, associations)
}

func appendByReflection(slice any, item any) any {
	sliceValue := reflect.ValueOf(slice)
	itemValue := reflect.ValueOf(item)
	if sliceValue.IsNil() {
		sliceValue = reflect.MakeSlice(reflect.SliceOf(itemValue.Type()), 0, 1)
	}
	sliceValue = reflect.Append(sliceValue, itemValue)
	return sliceValue.Interface()
}

func TestReflectionHelper(t *testing.T) {
	t.Run("slice", func(t *testing.T) {
		type Company struct {
			ID int
		}
		var expected, actual []Company

		expected = append(expected, Company{ID: 1})

		actual = appendByReflection(actual, Company{ID: 1}).([]Company)
		assert.Equal(t, expected, actual)
	})

	t.Run("pointer", func(t *testing.T) {
		type Company struct {
			ID     int
			Parent *Company
		}

		var actual Company

		kakao := Company{
			ID: 1,
		}
		kep := Company{
			ID:     2,
			Parent: &kakao,
		}

		actual.ID = 2
		actualValue := reflect.Indirect(reflect.ValueOf(&actual)).FieldByName("Parent")
		parentValue := reflect.ValueOf(kakao)
		actualValue.Set(reflect.New(reflect.TypeOf(Company{})))
		actualValue.Elem().Set(parentValue)

		assert.Equal(t, kep, actual)
	})

	t.Run("ptrToEmptyElementOfPtrToSlice", func(t *testing.T) {
		type Company struct {
			ID int
		}
		var companies []Company

		ptrToElement := ptrToEmptyElementOfPtrToSlice(&companies)
		assert.Equal(t, &Company{}, ptrToElement)
	})
}
