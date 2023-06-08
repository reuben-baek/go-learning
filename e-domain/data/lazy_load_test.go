package data

import (
	"context"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestLazyLoad_in_Struct(t *testing.T) {
	type Company struct {
		ID   uint
		Name string
	}
	type User struct {
		CompanyID   uint
		Company     Company
		CompanyLazy *LazyLoad[Company]
	}

	kakao := Company{
		ID:   1,
		Name: "kakao",
	}

	lazyLoadFnFactory := func(ctx context.Context, entity any, id any) func() (any, error) {
		logrus.Infof("entity: %+v id: %v", entity, id)
		return func() (any, error) {
			return kakao, nil
		}
	}

	ctx := context.Background()
	lazyFn := lazyLoadFnFactory(ctx, Company{}, kakao.ID)
	reuben := User{}

	reuben.CompanyLazy = LazyLoadFn[Company](lazyFn)

	company := reuben.CompanyLazy.Get()
	assert.IsType(t, Company{}, company)
	assert.Equal(t, kakao, company)

}
