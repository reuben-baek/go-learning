package data

import (
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
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
				ID   int
				Name string
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
					ID:   1,
					Name: "order-1",
				},
			}

			expected := []Association{
				{
					Name:       "Order",
					Value:      user.Order,
					ID:         nil,
					ForeignKey: "",
					Type:       BelongTo,
					FetchMode:  FetchEagerMode,
				},
			}
			associations := findAssociations[User](user)
			assert.Equal(t, expected, associations)
		})
		t.Run("1:n cardinality", func(t *testing.T) {
			type Order struct {
				ID   int
				Name string
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
						ID:   1,
						Name: "order-1",
					},
				},
			}

			expected := []Association{
				{
					Name:       "Orders",
					Value:      user.Orders,
					ID:         nil,
					ForeignKey: "",
					Type:       HasMany,
					FetchMode:  FetchEagerMode,
				},
			}
			associations := findAssociations[User](user)
			assert.Equal(t, expected, associations)
		})
	})
	t.Run("lazy", func(t *testing.T) {
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
					Name:       "Company",
					Value:      &Company{},
					ID:         1,
					ForeignKey: "",
					Type:       BelongTo,
					FetchMode:  FetchLazyMode,
				},
			}
			associations := findAssociations[User](reuben)

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
					Name:       "CreditCard",
					Value:      &CreditCard{},
					ForeignKey: "UserID",
					Type:       HasOne,
					FetchMode:  FetchLazyMode,
				},
			}
			associations := findAssociations[User](reuben)

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
					Name:       "CreditCards",
					Value:      []CreditCard(nil),
					ForeignKey: "UserID",
					Type:       HasMany,
					FetchMode:  FetchLazyMode,
				},
			}
			associations := findAssociations[User](reuben)

			assert.Equal(t, expected, associations)
		})
	})
}

func TestFindLazyEntity(t *testing.T) {
	type Company struct {
		ID int
	}
	type Order struct {
		ID int
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
	expected := []Association{
		{
			Name:      "Company",
			Value:     &Company{},
			FetchMode: FetchLazyMode,
			ID:        "1",
		},
		{
			Name:      "Role",
			Value:     &Role{},
			FetchMode: FetchEagerMode,
		},
		{
			Name:      "Orders",
			Value:     []Order(nil),
			FetchMode: FetchEagerMode,
		},
	}
	associations := findAssociations[User](reuben)

	assert.Equal(t, expected, associations)
}
