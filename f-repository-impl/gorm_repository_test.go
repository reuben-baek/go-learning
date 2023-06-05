package f_repository_impl_test

import (
	"context"
	"database/sql"
	"github.com/reuben-baek/go-learning/e-domain"
	f_repository_impl "github.com/reuben-baek/go-learning/f-repository-impl"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
	"os"
	"testing"
	"time"
)

func getGormDB() *gorm.DB {
	logConfig := logger.New(log.New(os.Stdout, "\r\n", log.LstdFlags), logger.Config{
		SlowThreshold: 100 * time.Millisecond,
		LogLevel:      logger.Info,
		Colorful:      true,
	})

	var err error
	//db, err := gorm.Open(sqlite.Open("gorm_repo.db?_foreign_keys=on"), &gorm.Config{
	db, err := gorm.Open(sqlite.Open("file::memory:?_foreign_keys=on"), &gorm.Config{
		Logger: logConfig,
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	if err != nil {
		panic("failed to connect database")
	}
	return db
}

func TestGormRepositorySimpleModel(t *testing.T) {
	type User struct {
		ID           uint
		Name         string
		Email        *string
		Age          uint8
		Birthday     *time.Time
		MemberNumber sql.NullString
		ActivatedAt  sql.NullTime
		CreatedAt    time.Time
		UpdatedAt    time.Time
	}
}

func TestGormRepositoryGormModel(t *testing.T) {
	type User struct {
		gorm.Model
		Name  string
		Email *string
	}
}

func TestGormRepositoryEmbeddedModel(t *testing.T) {
	type Author struct {
		Name  string
		Email string
	}
	type Blog struct {
		ID      int
		Author  Author `gorm:"embedded"`
		Upvotes int32
	}
}

func LazyLoadableInstance(entity f_repository_impl.LazyLoadable) {
	entity.NewInstance()
}

func TestLazyLoadable(t *testing.T) {
	type User struct {
		f_repository_impl.LazyLoadableImpl `gorm:"-"`
		Name                               string
		CompanyID                          int
	}
	user := User{}
	LazyLoadableInstance(&user)
	assert.NotNil(t, user.LazyLoadableImpl)

}

func TestGormRepository_GetLazyLoadFn(t *testing.T) {
	type Company struct {
		ID   int
		Name string
	}
	type LazyUser struct {
		f_repository_impl.LazyLoadableImpl `gorm:"-"`
		gorm.Model
		Name        string
		CompanyID   int
		Company     Company                     `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
		LazyCompany *e_domain.LazyLoad[Company] `gorm:"-"`
	}
	db := getGormDB()
	db.AutoMigrate(&Company{})
	db.AutoMigrate(&LazyUser{})

	companyRepository := f_repository_impl.NewGormRepository[Company, int](db)
	userRepository := f_repository_impl.NewGormRepository[LazyUser, uint](db)

	ctx := context.Background()
	kakaoEnterprise := Company{
		Name: "kakao enterprise",
	}
	kakaoEnterpriseCreated, err := companyRepository.Create(ctx, kakaoEnterprise)
	assert.Nil(t, err)
	assert.NotEmpty(t, kakaoEnterpriseCreated.ID)

	company := &Company{}
	loadFn := userRepository.GetLazyLoadFunc(ctx, company, int(1))

	loadedCompany, _ := loadFn()
	assert.Equal(t, 1, loadedCompany.(Company).ID)
}

func TestGormRepositoryAssociations_LazyLoad(t *testing.T) {
	type Company struct {
		ID   int
		Name string
	}
	type LazyUser struct {
		f_repository_impl.LazyLoadableImpl `gorm:"-"`
		gorm.Model
		Name      string
		CompanyID int
		Company   Company `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	}

	db := getGormDB()
	db.AutoMigrate(&Company{})
	db.AutoMigrate(&LazyUser{})

	companyRepository := f_repository_impl.NewGormRepository[Company, int](db)
	userRepository := f_repository_impl.NewGormRepository[LazyUser, uint](db)

	t.Run("create", func(t *testing.T) {
		ctx := context.Background()
		kakaoEnterprise := Company{
			Name: "kakao enterprise",
		}
		kakaoEnterpriseCreated, err := companyRepository.Create(ctx, kakaoEnterprise)
		assert.Nil(t, err)
		assert.NotEmpty(t, kakaoEnterpriseCreated.ID)

		reuben := LazyUser{
			Name:    "reuben.b",
			Company: kakaoEnterpriseCreated,
		}
		created, err := userRepository.Create(ctx, reuben)
		assert.Nil(t, err)
		assert.NotEmpty(t, created.ID)
		assert.NotEmpty(t, created.CompanyID)
		assert.NotEmpty(t, created.Company)

		found, err := userRepository.FindOne(ctx, created.ID)
		assert.Nil(t, err)
		assert.Equal(t, created.CompanyID, found.CompanyID)
		assert.Empty(t, found.Company)

		company, err := f_repository_impl.LazyLoadNow[Company](&found)
		assert.Nil(t, err)
		assert.Equal(t, kakaoEnterpriseCreated, company)
	})
	db.Migrator().DropTable(&Company{})
	db.Migrator().DropTable(&User{})
}

func TestGormRepositoryAssociations_BelongsTo(t *testing.T) {
	// `User` belongs to `Company`, `CompanyID` is the foreign key
	type Company struct {
		ID   int
		Name string
	}
	type User struct {
		gorm.Model
		Name      string
		CompanyID int
		Company   Company `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" fetch:"eager"`
	}
	db := getGormDB()
	db.AutoMigrate(&Company{})
	db.AutoMigrate(&User{})

	companyRepository := f_repository_impl.NewGormRepository[Company, int](db)
	userRepository := f_repository_impl.NewGormRepository[User, uint](db)

	t.Run("create", func(t *testing.T) {
		ctx := context.Background()
		kakaoEnterprise := Company{
			Name: "kakao enterprise",
		}
		kakaoEnterpriseCreated, err := companyRepository.Create(ctx, kakaoEnterprise)
		assert.Nil(t, err)
		assert.NotEmpty(t, kakaoEnterpriseCreated.ID)

		reuben := User{
			Name:    "reuben.b",
			Company: kakaoEnterpriseCreated,
		}
		created, err := userRepository.Create(ctx, reuben)
		assert.Nil(t, err)
		assert.NotEmpty(t, created.ID)
		assert.NotEmpty(t, created.CompanyID)
		assert.NotEmpty(t, created.Company)

		found, err := userRepository.FindOne(ctx, created.ID)
		assert.Nil(t, err)
		assert.Equal(t, created, found)
	})
	t.Run("update field", func(t *testing.T) {
		ctx := context.Background()
		kakaoEnterprise := Company{
			Name: "kakao enterprise",
		}
		kakaoEnterpriseCreated, _ := companyRepository.Create(ctx, kakaoEnterprise)
		reuben := User{
			Name:    "reuben.b",
			Company: kakaoEnterpriseCreated,
		}
		created, _ := userRepository.Create(ctx, reuben)
		found, _ := userRepository.FindOne(ctx, created.ID)

		found.Name = "reuben.baek"
		updated, err := userRepository.Update(ctx, found)

		assert.Nil(t, err)
		assert.Equal(t, found.ID, updated.ID)
		assert.Equal(t, found.Name, updated.Name)
		assert.Equal(t, found.CompanyID, updated.CompanyID)
		assert.Equal(t, updated.CompanyID, updated.Company.ID)
		assert.Equal(t, found.Company, updated.Company)
	})
	t.Run("update belong to", func(t *testing.T) {
		ctx := context.Background()
		kakaoEnterprise := Company{
			Name: "kakao enterprise",
		}
		kakaoEnterpriseCreated, _ := companyRepository.Create(ctx, kakaoEnterprise)
		kakaoCloud := Company{
			Name: "kakao cloud",
		}
		kakaoCloudCreated, _ := companyRepository.Create(ctx, kakaoCloud)

		reuben := User{
			Name:    "reuben.b",
			Company: kakaoEnterpriseCreated,
		}
		created, _ := userRepository.Create(ctx, reuben)
		found, _ := userRepository.FindOne(ctx, created.ID)

		found.Company = kakaoCloudCreated
		updated, err := userRepository.Update(ctx, found)

		assert.Nil(t, err)
		assert.Equal(t, found.ID, updated.ID)
		assert.Equal(t, found.Name, updated.Name)
		assert.Equal(t, updated.CompanyID, updated.Company.ID)
		assert.Equal(t, found.Company, updated.Company)

		foundAfterUpdated, _ := userRepository.FindOne(ctx, created.ID)
		assert.Equal(t, updated, foundAfterUpdated)
	})
	t.Run("delete belong to", func(t *testing.T) {
		ctx := context.Background()
		kakaoEnterprise := Company{
			Name: "kakao enterprise",
		}
		kakaoEnterpriseCreated, _ := companyRepository.Create(ctx, kakaoEnterprise)

		reuben := User{
			Name:    "reuben.b",
			Company: kakaoEnterpriseCreated,
		}
		created, _ := userRepository.Create(ctx, reuben)

		err := companyRepository.Delete(ctx, kakaoEnterpriseCreated)
		assert.Nil(t, err)

		found, err := userRepository.FindOne(ctx, created.ID)
		assert.Nil(t, err)
		assert.Empty(t, found.Company)
		assert.Empty(t, found.CompanyID)
	})
	db.Migrator().DropTable(&User{})
	db.Migrator().DropTable(&Company{})
}

func TestGormRepositoryAssociations_HasOne(t *testing.T) {
	// User has one CreditCard, UserID is the foreign key
	type CreditCard struct {
		gorm.Model
		Number string
		UserID uint
	}
	type User struct {
		gorm.Model
		Name       string
		CreditCard CreditCard `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" fetch:"eager"`
	}
	db := getGormDB()
	db.AutoMigrate(&User{})
	db.AutoMigrate(&CreditCard{})

	userRepository := f_repository_impl.NewGormRepository[User, uint](db)

	t.Run("create", func(t *testing.T) {
		ctx := context.Background()

		reuben := User{
			Name: "reuben.b",
			CreditCard: CreditCard{
				Number: "123412341234",
			},
		}
		created, err := userRepository.Create(ctx, reuben)
		assert.Nil(t, err)
		assert.NotEmpty(t, created.ID)
		assert.Equal(t, reuben.Name, created.Name)
		assert.NotEmpty(t, created.CreditCard.ID)
		assert.Equal(t, reuben.CreditCard.Number, created.CreditCard.Number)
		assert.NotEmpty(t, created.CreditCard.UserID)

		found, err := userRepository.FindOne(ctx, created.ID)
		assert.Nil(t, err)
		assert.Equal(t, created, found)
	})

	t.Run("update field", func(t *testing.T) {
		ctx := context.Background()

		reuben := User{
			Name: "reuben.b",
			CreditCard: CreditCard{
				Number: "123412341234",
			},
		}
		created, _ := userRepository.Create(ctx, reuben)
		found, _ := userRepository.FindOne(ctx, created.ID)

		found.Name = "reuben.baek"
		updated, err := userRepository.Update(ctx, found)

		assert.Nil(t, err)
		assert.NotEmpty(t, updated.ID)
		assert.Equal(t, found.Name, updated.Name)
		assert.NotEmpty(t, updated.CreditCard.ID)
		assert.Equal(t, found.CreditCard.Number, updated.CreditCard.Number)
		assert.Equal(t, found.ID, updated.CreditCard.UserID)

		var creditCard CreditCard
		db.Model(&CreditCard{}).Where("id = ? ", created.CreditCard.ID).First(&creditCard)
		assert.Equal(t, found.CreditCard, creditCard)
	})

	t.Run("update association", func(t *testing.T) {
		ctx := context.Background()

		reuben := User{
			Name: "reuben.b",
			CreditCard: CreditCard{
				Number: "123412341234",
			},
		}
		created, _ := userRepository.Create(ctx, reuben)
		found, _ := userRepository.FindOne(ctx, created.ID)

		found.CreditCard = CreditCard{}
		updated, err := userRepository.Update(ctx, found)

		assert.Nil(t, err)
		assert.NotEmpty(t, updated.ID)
		assert.Empty(t, updated.CreditCard)

		var creditCard CreditCard
		err = db.Model(&CreditCard{}).Where("id = ? ", created.CreditCard.ID).First(&creditCard).Error
		assert.ErrorIs(t, gorm.ErrRecordNotFound, err)
		assert.Empty(t, creditCard)
	})

	t.Run("delete", func(t *testing.T) {
		ctx := context.Background()

		reuben := User{
			Name: "reuben.b",
			CreditCard: CreditCard{
				Number: "123412341234",
			},
		}
		created, _ := userRepository.Create(ctx, reuben)
		found, _ := userRepository.FindOne(ctx, created.ID)

		err := userRepository.Delete(ctx, found) // soft delete
		assert.Nil(t, err)

		_, err = userRepository.FindOne(ctx, created.ID)
		assert.ErrorIs(t, f_repository_impl.NotFoundError, err)

		var creditCard CreditCard
		result := db.Model(&CreditCard{}).Where("id = ? ", created.CreditCard.ID).First(&creditCard)
		assert.ErrorIs(t, gorm.ErrRecordNotFound, result.Error)
		assert.Empty(t, creditCard)
	})

	db.Migrator().DropTable(&User{})
	db.Migrator().DropTable(&CreditCard{})
}
func TestGormRepositoryAssociations_HasMany(t *testing.T) {
	// User has many CreditCards, UserID is the foreign key
	type CreditCard struct {
		gorm.Model
		Number string
		UserID uint
	}

	type User struct {
		gorm.Model
		Name        string
		CreditCards []CreditCard `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" fetch:"eager"`
	}
	db := getGormDB()
	db.AutoMigrate(&User{})
	db.AutoMigrate(&CreditCard{})

	userRepository := f_repository_impl.NewGormRepository[User, uint](db)

	t.Run("create", func(t *testing.T) {
		ctx := context.Background()

		creditCards := []CreditCard{
			{
				Number: "123412341234",
			},
			{
				Number: "000000000000",
			},
		}
		reuben := User{
			Name:        "reuben.b",
			CreditCards: creditCards,
		}
		created, err := userRepository.Create(ctx, reuben)
		assert.Nil(t, err)
		assert.NotEmpty(t, created.ID)
		assert.NotEmpty(t, created.CreditCards[0].ID)
		assert.NotEmpty(t, created.CreditCards[1].ID)

		found, err := userRepository.FindOne(ctx, created.ID)
		assert.Nil(t, err)
		assert.NotEmpty(t, found.ID)
		assert.Equal(t, 2, len(found.CreditCards))
		assert.NotEmpty(t, found.CreditCards[0].ID)
		assert.NotEmpty(t, found.CreditCards[0].Number)
		assert.NotEmpty(t, found.CreditCards[1].ID)
		assert.NotEmpty(t, found.CreditCards[1].Number)
	})
	t.Run("update field", func(t *testing.T) {
		ctx := context.Background()

		creditCards := []CreditCard{
			{
				Number: "123412341234",
			},
			{
				Number: "000000000000",
			},
		}
		reuben := User{
			Name:        "reuben.b",
			CreditCards: creditCards,
		}
		created, _ := userRepository.Create(ctx, reuben)
		found, _ := userRepository.FindOne(ctx, created.ID)

		found.Name = "reuben.baek"

		updated, err := userRepository.Update(ctx, found)
		assert.Nil(t, err)
		assert.Equal(t, found.Name, updated.Name)
		assert.Equal(t, found.CreditCards, updated.CreditCards)

		foundAfterUpdate, err := userRepository.FindOne(ctx, created.ID)
		assert.Nil(t, err)
		assert.Equal(t, updated, foundAfterUpdate)
	})
	t.Run("update association", func(t *testing.T) {
		ctx := context.Background()

		creditCards := []CreditCard{
			{
				Number: "123412341234",
			},
			{
				Number: "000000000000",
			},
		}
		reuben := User{
			Name:        "reuben.b",
			CreditCards: creditCards,
		}
		created, _ := userRepository.Create(ctx, reuben)
		found, _ := userRepository.FindOne(ctx, created.ID)

		// delete 000000000000, add 999999999999
		found.CreditCards = []CreditCard{
			found.CreditCards[0],
			{
				Number: "999999999999",
			},
		}
		updated, err := userRepository.Update(ctx, found)
		assert.Nil(t, err)
		assert.Equal(t, 2, len(updated.CreditCards))
		assert.NotEmpty(t, updated.CreditCards[0].ID)
		assert.NotEmpty(t, updated.CreditCards[1].ID)

		foundAfterUpdate, err := userRepository.FindOne(ctx, created.ID)
		assert.Nil(t, err)
		assert.Equal(t, updated, foundAfterUpdate)
	})
	t.Run("delete", func(t *testing.T) {
		ctx := context.Background()

		creditCards := []CreditCard{
			{
				Number: "123412341234",
			},
			{
				Number: "000000000000",
			},
		}
		reuben := User{
			Name:        "reuben.b",
			CreditCards: creditCards,
		}
		created, _ := userRepository.Create(ctx, reuben)
		found, _ := userRepository.FindOne(ctx, created.ID)

		err := userRepository.Delete(ctx, found)
		assert.Nil(t, err)

		_, err = userRepository.FindOne(ctx, created.ID)
		assert.ErrorIs(t, f_repository_impl.NotFoundError, err)

		var cards []CreditCard
		result := db.Model(&CreditCard{}).Find(&cards, []uint{created.CreditCards[0].ID, created.CreditCards[1].ID})
		assert.Equal(t, int64(0), result.RowsAffected)
		assert.Empty(t, cards)
	})

	db.Migrator().DropTable(&User{})
	db.Migrator().DropTable(&CreditCard{})
}

func TestGormRepositoryAssociations_ManyToMany(t *testing.T) {
	// User has and belongs to many languages, `user_languages` is the join table
	type Language struct {
		gorm.Model
		Name string
		//Users []User `gorm:"many2many:user_languages;" fetch:"eager"`
	}
	type User struct {
		gorm.Model
		Name      string
		Languages []Language `gorm:"many2many:user_languages;" fetch:"eager"`
	}
	db := getGormDB()
	db.AutoMigrate(&Language{})
	db.AutoMigrate(&User{})

	languageRepository := f_repository_impl.NewGormRepository[Language, uint](db)
	userRepository := f_repository_impl.NewGormRepository[User, uint](db)

	ctx := context.Background()

	languages := []Language{
		{
			Name: "kr",
		},
		{
			Name: "en",
		},
	}
	var languagesCreated []Language
	for _, language := range languages {
		created, err := languageRepository.Create(ctx, language)
		assert.Nil(t, err)
		assert.NotEmpty(t, created.ID)
		languagesCreated = append(languagesCreated, created)
	}

	t.Run("create", func(t *testing.T) {
		ctx := context.Background()
		reuben := User{
			Name: "reuben.b",
			Languages: []Language{
				languagesCreated[0],
				languagesCreated[1],
			},
		}
		created, err := userRepository.Create(ctx, reuben)
		assert.Nil(t, err)
		assert.NotEmpty(t, created.ID)
		assert.Equal(t, 2, len(created.Languages))
		assert.NotEmpty(t, created.Languages[0])
		assert.NotEmpty(t, created.Languages[1])

		found, err := userRepository.FindOne(ctx, created.ID)
		assert.Nil(t, err)
		assert.Equal(t, created, found)
	})
	t.Run("update association", func(t *testing.T) {
		ctx := context.Background()
		reuben := User{
			Name: "reuben.b",
			Languages: []Language{
				languagesCreated[0],
				languagesCreated[1],
			},
		}
		created, _ := userRepository.Create(ctx, reuben)
		found, _ := userRepository.FindOne(ctx, created.ID)

		found.Languages = []Language{languagesCreated[1]}
		updated, err := userRepository.Update(ctx, found)
		assert.Nil(t, err)
		assert.NotEmpty(t, updated.ID)
		assert.Equal(t, 1, len(updated.Languages))
		assert.Equal(t, languagesCreated[1], updated.Languages[0])

		foundAfterUpdated, err := userRepository.FindOne(ctx, created.ID)
		assert.Nil(t, err)
		assert.Equal(t, updated, foundAfterUpdated)
	})
	t.Run("delete", func(t *testing.T) {
		ctx := context.Background()
		reuben := User{
			Name: "reuben.b",
			Languages: []Language{
				languagesCreated[0],
				languagesCreated[1],
			},
		}
		created, _ := userRepository.Create(ctx, reuben)
		found, _ := userRepository.FindOne(ctx, created.ID)

		err := userRepository.Delete(ctx, found)
		assert.Nil(t, err)

		_, err = userRepository.FindOne(ctx, created.ID)
		assert.ErrorIs(t, f_repository_impl.NotFoundError, err)

		var languages []Language
		result := db.Unscoped().Table("user_languages").Where("user_id = ?", created.ID).Find(&languages)
		assert.Nil(t, result.Error)
		assert.Equal(t, int64(0), result.RowsAffected)

	})
	db.Migrator().DropTable(&User{})
	db.Migrator().DropTable(&Language{})
}
