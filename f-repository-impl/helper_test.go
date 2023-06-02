package f_repository_impl

import (
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	"testing"
)

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
		assert.PanicsWithValue(t, "Entity 'f_repository_impl.TT' has not ID field", func() {
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
		assert.PanicsWithValue(t, "ID field type 'map[string]string' of 'f_repository_impl.TT' is not comparable", func() {
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

func TestFindPreloadModels(t *testing.T) {
	type Company struct {
		ID   int
		Name string
	}
	type Role struct {
		ID   int
		Name string
	}
	type Order struct {
		ID   int
		Name string
	}
	type User struct {
		gorm.Model
		Name      string
		CompanyID uint
		Company   Company `fetch:"eager"`
		Role      Role
		Orders    []Order `fetch:"eager"`
	}

	models := findPreloadModels[User](User{})
	expected := []string{"Company", "Orders"}
	assert.Equal(t, expected, models)
}

func TestFindAssociations(t *testing.T) {
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
	associations := findAssociations[User](user)
	assert.Equal(t, 1, len(associations))
}
