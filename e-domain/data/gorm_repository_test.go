package data_test

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/reuben-baek/go-learning/e-domain/data"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
	"os"
	"testing"
	"time"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
}

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

func TestGormRepository_Simple(t *testing.T) {
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

func TestGormRepository_GormSimple(t *testing.T) {
	type User struct {
		gorm.Model
		Name  string
		Email *string
	}
}

func TestGormRepository_Embedded(t *testing.T) {
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

func TestGormRepository_GetLazyLoadFn(t *testing.T) {
	type Company struct {
		ID   int
		Name string
	}
	type LazyUser struct {
		data.LazyLoader `gorm:"-"`
		gorm.Model
		Name        string
		CompanyID   int
		Company     Company                 `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
		LazyCompany *data.LazyLoad[Company] `gorm:"-"`
	}
	db := getGormDB()
	db.AutoMigrate(&Company{})
	db.AutoMigrate(&LazyUser{})

	transactionManager := data.NewGormTransactionManager(db)
	companyRepository := data.NewGormRepository[Company, int](transactionManager)
	userRepository := data.NewGormRepository[LazyUser, uint](transactionManager)

	ctx := context.Background()
	kakaoEnterprise := Company{
		Name: "kakao enterprise",
	}
	kakaoEnterpriseCreated, err := companyRepository.Create(ctx, kakaoEnterprise)
	assert.Nil(t, err)
	assert.NotEmpty(t, kakaoEnterpriseCreated.ID)

	company := &Company{}
	loadFn := userRepository.GetLazyLoadFuncOfBelongTo(ctx, company, int(1))

	loadedCompany, err := loadFn()
	assert.Nil(t, err)
	assert.Equal(t, 1, loadedCompany.(Company).ID)
}

func TestGormRepository_SelfRef_Lazy(t *testing.T) {
	type Category struct {
		data.LazyLoader `gorm:"-"`
		ID              uint
		Name            string
		ParentID        *uint
		Parent          *Category `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	}
	db := getGormDB()
	db.AutoMigrate(&Category{})

	transactionManager := data.NewGormTransactionManager(db)
	categoryRepository := data.NewGormRepository[Category, uint](transactionManager)

	t.Run("create", func(t *testing.T) {
		ctx := context.Background()
		computer := Category{
			Name: "computer",
		}
		computer, err := categoryRepository.Create(ctx, computer)
		assert.Nil(t, err)
		assert.NotEmpty(t, computer.ID)
		assert.Empty(t, computer.ParentID)
		assert.Empty(t, computer.Parent)

		parentID := computer.ID
		mac := Category{
			Name:     "mac",
			ParentID: &parentID,
		}
		mac, err = categoryRepository.Create(ctx, mac)
		assert.Nil(t, err)
		assert.NotEmpty(t, mac.ID)
		assert.Equal(t, parentID, *mac.ParentID)

		found, err := categoryRepository.FindOne(ctx, mac.ID)
		assert.Nil(t, err)
		assert.Equal(t, mac.ID, found.ID)
		assert.Equal(t, *mac.ParentID, *found.ParentID)
		assert.Empty(t, found.Parent)

		parent, err := data.LazyLoadNow[*Category]("Parent", &found)
		assert.Nil(t, err)
		assert.Equal(t, computer.ID, parent.ID)
		assert.Equal(t, computer.Name, parent.Name)
		assert.Empty(t, parent.Parent)
		assert.Equal(t, found.Parent, parent)
	})
	t.Run("find by", func(t *testing.T) {
		ctx := context.Background()
		computer := Category{
			Name: "computer",
		}
		computer, _ = categoryRepository.Create(ctx, computer)

		parentID := computer.ID
		mac := Category{
			Name:     "mac",
			ParentID: &parentID,
		}
		mac, _ = categoryRepository.Create(ctx, mac)

		found, err := categoryRepository.FindBy(ctx, "Parent", computer)
		assert.Nil(t, err)
		assert.Equal(t, 1, len(found))
		assert.Equal(t, mac.ID, found[0].ID)
		assert.Equal(t, *mac.ParentID, *found[0].ParentID)
		assert.Empty(t, found[0].Parent)

		parent, err := data.LazyLoadNow[*Category]("Parent", &found[0])
		assert.Nil(t, err)
		assert.Equal(t, computer.ID, parent.ID)
		assert.Equal(t, computer.Name, parent.Name)
		assert.Empty(t, parent.Parent)
		assert.Equal(t, found[0].Parent, parent)
	})
	t.Run("update", func(t *testing.T) {
		ctx := context.Background()
		computer := Category{
			Name: "computer",
		}
		computer, _ = categoryRepository.Create(ctx, computer)

		apple := Category{
			Name: "apple",
		}
		apple, _ = categoryRepository.Create(ctx, apple)

		parentID := computer.ID
		mac := Category{
			Name:     "mac",
			ParentID: &parentID,
		}
		mac, _ = categoryRepository.Create(ctx, mac)

		t.Run("name before load", func(t *testing.T) {
			found, _ := categoryRepository.FindOne(ctx, mac.ID)

			found.Name = "mac-m2"
			updated, err := categoryRepository.Update(ctx, found)
			assert.Nil(t, err)
			assert.Equal(t, found.ID, updated.ID)
			assert.Equal(t, found.Name, updated.Name)
			assert.Equal(t, found.ParentID, updated.ParentID)

			parent, err := data.LazyLoadNow[*Category]("Parent", &updated)
			assert.Nil(t, err)
			assert.Equal(t, computer.ID, updated.Parent.ID)
			assert.Equal(t, computer.Name, updated.Parent.Name)
			assert.Equal(t, updated.Parent, parent)

			// rollback for next tests
			categoryRepository.Update(ctx, mac)
		})
		t.Run("name after load", func(t *testing.T) {
			found, _ := categoryRepository.FindOne(ctx, mac.ID)

			parent, _ := data.LazyLoadNow[*Category]("Parent", &found)
			assert.Equal(t, computer.ID, found.Parent.ID)

			found.Name = "mac-m2"
			updated, err := categoryRepository.Update(ctx, found)
			assert.Nil(t, err)
			assert.Equal(t, found.ID, updated.ID)
			assert.Equal(t, found.Name, updated.Name)
			assert.Equal(t, found.ParentID, updated.ParentID)
			assert.Empty(t, updated.Parent)

			parent, err = data.LazyLoadNow[*Category]("Parent", &updated)
			assert.Nil(t, err)
			assert.Equal(t, computer.ID, parent.ID)
			assert.Equal(t, computer.Name, parent.Name)
			assert.Empty(t, parent.Parent)

			// rollback for next tests
			categoryRepository.Update(ctx, mac)
		})
		t.Run("parent", func(t *testing.T) {
			found, _ := categoryRepository.FindOne(ctx, mac.ID)
			appleID := apple.ID
			found.ParentID = &appleID
			updated, err := categoryRepository.Update(ctx, found)
			assert.Nil(t, err)
			assert.Equal(t, found.ID, updated.ID)
			assert.Equal(t, found.Name, updated.Name)
			assert.Equal(t, found.ParentID, updated.ParentID)
			assert.Empty(t, updated.Parent)

			parent, err := data.LazyLoadNow[*Category]("Parent", &updated)
			assert.Nil(t, err)
			assert.Equal(t, updated.Parent, parent)
			assert.Equal(t, apple.ID, updated.Parent.ID)
			assert.Equal(t, apple.Name, updated.Parent.Name)
			assert.Empty(t, updated.Parent.Parent)

			// rollback for next tests
			categoryRepository.Update(ctx, mac)
		})
	})
	t.Run("delete", func(t *testing.T) {
		ctx := context.Background()

		computer := Category{
			Name: "computer",
		}
		computer, _ = categoryRepository.Create(ctx, computer)

		parentID := computer.ID
		mac := Category{
			Name:     "mac",
			ParentID: &parentID,
		}
		mac, _ = categoryRepository.Create(ctx, mac)

		found, _ := categoryRepository.FindOne(ctx, mac.ID)

		err := categoryRepository.Delete(ctx, found)
		assert.Nil(t, err)

		_, err = categoryRepository.FindOne(ctx, mac.ID)
		assert.Equal(t, data.NotFoundError, err)
	})
	db.Migrator().DropTable(&Category{})
}
func TestGormRepository_BelongTo_Lazy(t *testing.T) {
	type Company struct {
		ID   int
		Name string
	}
	type User struct {
		data.LazyLoader `gorm:"-"`
		gorm.Model
		Name      string
		CompanyID int
		Company   Company `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	}

	db := getGormDB()
	db.AutoMigrate(&Company{})
	db.AutoMigrate(&User{})

	transactionManager := data.NewGormTransactionManager(db)
	companyRepository := data.NewGormRepository[Company, int](transactionManager)
	userRepository := data.NewGormRepository[User, uint](transactionManager)

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
		assert.Equal(t, created.CompanyID, found.CompanyID)
		assert.Empty(t, found.Company)

		company, err := data.LazyLoadNow[Company]("Company", &found)
		assert.Nil(t, err)
		assert.Equal(t, kakaoEnterpriseCreated, company)
	})
	t.Run("find by", func(t *testing.T) {
		ctx := context.Background()
		kakaoEnterprise := Company{
			Name: "kakao enterprise",
		}
		kakaoEnterprise, _ = companyRepository.Create(ctx, kakaoEnterprise)

		reuben := User{
			Name:    "reuben.b",
			Company: kakaoEnterprise,
		}
		created, _ := userRepository.Create(ctx, reuben)

		found, err := userRepository.FindBy(ctx, "Company", kakaoEnterprise)
		assert.Nil(t, err)
		assert.Equal(t, 1, len(found))
		assert.Equal(t, created.ID, found[0].ID)
		assert.Equal(t, created.Name, found[0].Name)
		assert.Equal(t, created.CompanyID, found[0].CompanyID)
		assert.Empty(t, found[0].Company)

		company, err := data.LazyLoadNow[Company]("Company", &found[0])
		assert.Nil(t, err)
		assert.Equal(t, kakaoEnterprise, company)
	})

	t.Run("update", func(t *testing.T) {
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
		reuben, _ = userRepository.Create(ctx, reuben)

		t.Run("name field without lazy load", func(t *testing.T) {
			found, _ := userRepository.FindOne(ctx, reuben.ID)
			assert.Empty(t, found.Company)
			found.Name = "reuben.baek"
			updated, err := userRepository.Update(ctx, found)
			assert.Nil(t, err)
			assert.Equal(t, found.Name, updated.Name)
			assert.Equal(t, found.CompanyID, updated.CompanyID)
			assert.Empty(t, updated.Company)
			company, err := data.LazyLoadNow[Company]("Company", &updated)
			assert.Nil(t, err)
			assert.Equal(t, kakaoEnterpriseCreated, updated.Company)
			assert.Equal(t, kakaoEnterpriseCreated, company)

			// rollback for next tests
			userRepository.Update(ctx, reuben)
		})
		t.Run("name field after lazy load", func(t *testing.T) {
			found, _ := userRepository.FindOne(ctx, reuben.ID)
			company, err := data.LazyLoadNow[Company]("Company", &found)
			assert.Nil(t, err)
			assert.Equal(t, kakaoEnterpriseCreated, found.Company)
			assert.Equal(t, kakaoEnterpriseCreated, company)
			found.Name = "reuben.baek"
			updated, err := userRepository.Update(ctx, found)
			assert.Nil(t, err)
			assert.Equal(t, found.Name, updated.Name)
			assert.Equal(t, found.CompanyID, updated.CompanyID)
			assert.Equal(t, updated.CompanyID, updated.CompanyID)
			assert.Empty(t, updated.Company)
			// rollback for next tests
			userRepository.Update(ctx, reuben)
		})
		t.Run("belongTo CompanyID field without lazy load", func(t *testing.T) {
			found, _ := userRepository.FindOne(ctx, reuben.ID)
			assert.Empty(t, found.Company)
			found.CompanyID = kakaoCloudCreated.ID
			updated, err := userRepository.Update(ctx, found)
			assert.Nil(t, err)
			assert.Equal(t, found.Name, updated.Name)
			assert.Equal(t, found.CompanyID, updated.CompanyID)
			assert.Empty(t, updated.Company)
			company, err := data.LazyLoadNow[Company]("Company", &updated)
			assert.Nil(t, err)
			assert.Equal(t, kakaoCloudCreated, updated.Company)
			assert.Equal(t, updated.Company, company)
			// rollback for next tests
			userRepository.Update(ctx, reuben)
		})
		t.Run("belongTo CompanyID field after lazy load - only foreignKey", func(t *testing.T) {
			found, _ := userRepository.FindOne(ctx, reuben.ID)
			require.Equal(t, kakaoEnterpriseCreated.ID, found.CompanyID)

			company, err := data.LazyLoadNow[Company]("Company", &found)
			assert.Nil(t, err)
			assert.Equal(t, kakaoEnterpriseCreated, found.Company)
			assert.Equal(t, kakaoEnterpriseCreated, company)

			found.CompanyID = kakaoCloudCreated.ID
			updated, err := userRepository.Update(ctx, found)
			assert.Nil(t, err)
			assert.Equal(t, found.CompanyID, updated.CompanyID)
			assert.Empty(t, updated.Company)

			// rollback for next tests
			userRepository.Update(ctx, reuben)
		})
		t.Run("belongTo CompanyID field after lazy load - foreignKey and value", func(t *testing.T) {
			found, _ := userRepository.FindOne(ctx, reuben.ID)
			require.Equal(t, kakaoEnterpriseCreated.ID, found.CompanyID)

			company, err := data.LazyLoadNow[Company]("Company", &found)
			assert.Nil(t, err)
			assert.Equal(t, kakaoEnterpriseCreated, found.Company)
			assert.Equal(t, kakaoEnterpriseCreated, company)

			found.CompanyID = kakaoCloudCreated.ID
			found.Company = Company{}
			updated, err := userRepository.Update(ctx, found)
			assert.Nil(t, err)
			assert.Equal(t, found.CompanyID, updated.CompanyID)
			assert.Empty(t, updated.Company)

			company, err = data.LazyLoadNow[Company]("Company", &updated)
			assert.Nil(t, err)
			assert.Equal(t, kakaoCloudCreated, updated.Company)
			assert.Equal(t, updated.Company, company)
			// rollback for next tests
			userRepository.Update(ctx, reuben)
		})
		t.Run("company field before lazy load", func(t *testing.T) {
			found, _ := userRepository.FindOne(ctx, reuben.ID)
			require.Equal(t, kakaoEnterpriseCreated.ID, found.CompanyID)

			found.CompanyID = kakaoCloudCreated.ID
			found.Company = kakaoCloudCreated
			updated, err := userRepository.Update(ctx, found)
			assert.Nil(t, err)
			assert.Equal(t, kakaoCloudCreated.ID, updated.CompanyID)
			assert.Empty(t, updated.Company)
			company, err := data.LazyLoadNow[Company]("Company", &updated)
			assert.Nil(t, err)
			assert.Equal(t, kakaoCloudCreated, updated.Company)
			assert.Equal(t, updated.Company, company)
			// rollback for next tests
			userRepository.Update(ctx, reuben)
		})
	})
	db.Migrator().DropTable(&Company{})
	db.Migrator().DropTable(&User{})
}

func TestGormRepository_BelongTo_Eager(t *testing.T) {
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

	transactionManager := data.NewGormTransactionManager(db)
	companyRepository := data.NewGormRepository[Company, int](transactionManager)
	userRepository := data.NewGormRepository[User, uint](transactionManager)

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
	t.Run("findBy belongTo", func(t *testing.T) {
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

		users, err := userRepository.FindBy(ctx, "Company", kakaoEnterpriseCreated)
		assert.Nil(t, err)
		assert.Equal(t, 1, len(users))
		assert.Equal(t, created, users[0])
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

		found.CompanyID = kakaoCloudCreated.ID
		updated, err := userRepository.Update(ctx, found)

		assert.Nil(t, err)
		assert.Equal(t, found.ID, updated.ID)
		assert.Equal(t, found.Name, updated.Name)
		assert.Equal(t, updated.CompanyID, updated.Company.ID)
		assert.Equal(t, kakaoCloudCreated, updated.Company)

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

func TestGormRepository_HasOne_Lazy(t *testing.T) {
	// User has one CreditCard, UserID is the foreign key
	type CreditCard struct {
		gorm.Model
		Number string
		UserID uint
	}
	type User struct {
		data.LazyLoader `gorm:"-"`
		gorm.Model
		Name       string
		CreditCard CreditCard `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	}
	db := getGormDB()
	db.AutoMigrate(&User{})
	db.AutoMigrate(&CreditCard{})

	transactionManager := data.NewGormTransactionManager(db)
	userRepository := data.NewGormRepository[User, uint](transactionManager)

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
		assert.Equal(t, created.ID, found.ID)
		assert.Empty(t, found.CreditCard)
		creditCard, err := data.LazyLoadNow[CreditCard]("CreditCard", &found)
		assert.Nil(t, err)
		assert.Equal(t, created.CreditCard, creditCard)
	})
	t.Run("find by", func(t *testing.T) {
		ctx := context.Background()

		reuben := User{
			Name: "reuben.b",
			CreditCard: CreditCard{
				Number: "123412341234",
			},
		}
		created, _ := userRepository.Create(ctx, reuben)

		found, err := userRepository.FindBy(ctx, "CreditCard", created.CreditCard)
		assert.Nil(t, err)
		assert.Equal(t, created.ID, found[0].ID)
		assert.Empty(t, found[0].CreditCard)
		creditCard, err := data.LazyLoadNow[CreditCard]("CreditCard", &found[0])
		assert.Nil(t, err)
		assert.Equal(t, created.CreditCard, creditCard)
	})
	t.Run("update field", func(t *testing.T) {
		ctx := context.Background()

		reuben := User{
			Name: "reuben.b",
			CreditCard: CreditCard{
				Number: "123412341234",
			},
		}
		reuben, _ = userRepository.Create(ctx, reuben)

		t.Run("name field without lazy load", func(t *testing.T) {
			found, _ := userRepository.FindOne(ctx, reuben.ID)
			found.Name = "reuben.baek"
			updated, err := userRepository.Update(ctx, found)

			assert.Nil(t, err)
			assert.NotEmpty(t, updated.ID)
			assert.Equal(t, found.Name, updated.Name)
			assert.Empty(t, updated.CreditCard)

			creditCard, err := data.LazyLoadNow[CreditCard]("CreditCard", &updated)
			assert.Nil(t, err)
			assert.Equal(t, reuben.CreditCard, creditCard)

			// rollback for next tests
			rollback, _ := userRepository.Update(ctx, reuben)
			fmt.Printf("%+v\n", rollback)
		})
		t.Run("name field after lazy load", func(t *testing.T) {
			found, _ := userRepository.FindOne(ctx, reuben.ID)
			creditCard, err := data.LazyLoadNow[CreditCard]("CreditCard", &found)
			assert.Nil(t, err)
			assert.NotEmpty(t, found.CreditCard)
			assert.Equal(t, found.CreditCard, creditCard)

			found.Name = "reuben.baek"
			updated, err := userRepository.Update(ctx, found)
			assert.Nil(t, err)
			assert.NotEmpty(t, updated.ID)
			assert.Equal(t, found.Name, updated.Name)
			assert.Empty(t, updated.CreditCard)
			// rollback for next tests
			userRepository.Update(ctx, reuben)
		})
		t.Run("hasOne field before lazy load - success", func(t *testing.T) {
			found, _ := userRepository.FindOne(ctx, reuben.ID)

			found.CreditCard = CreditCard{Number: "999999999999"}

			updated, err := userRepository.Update(ctx, found)
			assert.Nil(t, err)
			assert.Empty(t, updated.CreditCard)

			creditCard, err := data.LazyLoadNow[CreditCard]("CreditCard", &updated)
			assert.Nil(t, err)
			assert.Equal(t, updated.CreditCard, creditCard)
			assert.NotEmpty(t, updated.CreditCard.ID)
			assert.Equal(t, reuben.ID, updated.CreditCard.UserID)

			// rollback for next tests
			userRepository.Update(ctx, reuben)
		})
		t.Run("hasOne field after lazy load - success", func(t *testing.T) {
			found, _ := userRepository.FindOne(ctx, reuben.ID)

			creditCard, err := data.LazyLoadNow[CreditCard]("CreditCard", &found)
			assert.Nil(t, err)
			assert.NotEmpty(t, found.CreditCard)
			assert.Equal(t, found.CreditCard, creditCard)

			found.CreditCard = CreditCard{Number: "999999999999"}

			updated, err := userRepository.Update(ctx, found)
			assert.Nil(t, err)
			assert.Empty(t, updated.CreditCard)

			creditCard, err = data.LazyLoadNow[CreditCard]("CreditCard", &updated)
			assert.Nil(t, err)
			assert.Equal(t, found.CreditCard.Number, updated.CreditCard.Number)
			assert.NotEmpty(t, updated.CreditCard.ID)
			assert.Equal(t, reuben.ID, updated.CreditCard.UserID)

			// rollback for next tests
			userRepository.Update(ctx, reuben)
		})
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
		assert.ErrorIs(t, data.NotFoundError, err)

		var creditCard CreditCard
		result := db.Model(&CreditCard{}).Where("id = ? ", created.CreditCard.ID).First(&creditCard)
		assert.ErrorIs(t, gorm.ErrRecordNotFound, result.Error)
		assert.Empty(t, creditCard)
	})

	db.Migrator().DropTable(&User{})
	db.Migrator().DropTable(&CreditCard{})
}
func TestGormRepository_HasOne_Eager(t *testing.T) {
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

	transactionManager := data.NewGormTransactionManager(db)
	userRepository := data.NewGormRepository[User, uint](transactionManager)

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
		assert.Equal(t, updated.CreditCard, creditCard)
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
		assert.ErrorIs(t, data.NotFoundError, err)

		var creditCard CreditCard
		result := db.Model(&CreditCard{}).Where("id = ? ", created.CreditCard.ID).First(&creditCard)
		assert.ErrorIs(t, gorm.ErrRecordNotFound, result.Error)
		assert.Empty(t, creditCard)
	})

	db.Migrator().DropTable(&User{})
	db.Migrator().DropTable(&CreditCard{})
}

func TestGormRepository_HasMany_Lazy(t *testing.T) {
	// User has many CreditCards, UserID is the foreign key
	type CreditCard struct {
		gorm.Model
		Number string
		UserID uint
	}

	type User struct {
		data.LazyLoader `gorm:"-"`
		gorm.Model
		Name        string
		CreditCards []CreditCard `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" fetch:"lazy"`
	}
	db := getGormDB()
	db.AutoMigrate(&User{})
	db.AutoMigrate(&CreditCard{})

	transactionManager := data.NewGormTransactionManager(db)
	userRepository := data.NewGormRepository[User, uint](transactionManager)

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
		assert.Equal(t, 0, len(found.CreditCards))

		foundCreditCards, err := data.LazyLoadNow[[]CreditCard]("CreditCards", &found)
		assert.Nil(t, err)
		assert.Equal(t, created.CreditCards, foundCreditCards)
	})
	t.Run("find by", func(t *testing.T) {
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

		found, err := userRepository.FindBy(ctx, "CreditCard", created.CreditCards[1])
		assert.Nil(t, err)
		assert.Equal(t, 1, len(found))
		assert.NotEmpty(t, found[0].ID)
		assert.Empty(t, found[0].CreditCards)

		foundCreditCards, err := data.LazyLoadNow[[]CreditCard]("CreditCards", &found[0])
		assert.Nil(t, err)
		assert.Equal(t, created.CreditCards, foundCreditCards)
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
		reuben, _ = userRepository.Create(ctx, reuben)

		t.Run("name field without lazy load", func(t *testing.T) {
			found, _ := userRepository.FindOne(ctx, reuben.ID)
			found.Name = "reuben.baek"
			updated, err := userRepository.Update(ctx, found)

			assert.Nil(t, err)
			assert.NotEmpty(t, updated.ID)
			assert.Equal(t, found.Name, updated.Name)
			assert.Empty(t, updated.CreditCards)

			creditCards, err := data.LazyLoadNow[[]CreditCard]("CreditCards", &updated)
			assert.Nil(t, err)
			assert.Equal(t, reuben.CreditCards, creditCards)

			// rollback for next tests
			userRepository.Update(ctx, reuben)
		})

		t.Run("name field after lazy load", func(t *testing.T) {
			found, _ := userRepository.FindOne(ctx, reuben.ID)
			creditCards, err := data.LazyLoadNow[[]CreditCard]("CreditCards", &found)
			assert.Nil(t, err)
			assert.NotEmpty(t, found.CreditCards)
			assert.Equal(t, found.CreditCards, creditCards)

			found.Name = "reuben.baek"
			updated, err := userRepository.Update(ctx, found)
			assert.Nil(t, err)
			assert.NotEmpty(t, updated.ID)
			assert.Equal(t, found.Name, updated.Name)
			assert.Empty(t, updated.CreditCards)
			// rollback for next tests
			userRepository.Update(ctx, reuben)
		})
		t.Run("hasMany field before lazy load", func(t *testing.T) {
			found, _ := userRepository.FindOne(ctx, reuben.ID)

			found.CreditCards = []CreditCard{
				{
					Number: "999999999999",
				},
			}

			updated, err := userRepository.Update(ctx, found)
			assert.Nil(t, err)
			assert.Empty(t, updated.CreditCards)

			creditCards, err := data.LazyLoadNow[[]CreditCard]("CreditCards", &updated)
			assert.Nil(t, err)
			assert.Equal(t, updated.CreditCards, creditCards)
			assert.Equal(t, 1, len(updated.CreditCards))

			// rollback for next tests
			userRepository.Update(ctx, reuben)
		})
		t.Run("empty hasMany field before lazy load", func(t *testing.T) {
			t.Run("nil hasMany field", func(t *testing.T) {
				found, _ := userRepository.FindOne(ctx, reuben.ID)

				found.CreditCards = nil

				updated, err := userRepository.Update(ctx, found)
				assert.Nil(t, err)
				assert.Empty(t, updated.CreditCards)

				creditCards, err := data.LazyLoadNow[[]CreditCard]("CreditCards", &updated)
				assert.Nil(t, err)
				assert.Equal(t, updated.CreditCards, creditCards)
				assert.Equal(t, 2, len(updated.CreditCards))

				// rollback for next tests
				userRepository.Update(ctx, reuben)
			})
			t.Run("empty slice hasMany field", func(t *testing.T) {
				found, _ := userRepository.FindOne(ctx, reuben.ID)

				found.CreditCards = []CreditCard{}

				updated, err := userRepository.Update(ctx, found)
				assert.Nil(t, err)
				assert.Empty(t, updated.CreditCards)

				creditCards, err := data.LazyLoadNow[[]CreditCard]("CreditCards", &updated)
				assert.Nil(t, err)
				assert.Equal(t, updated.CreditCards, creditCards)
				assert.Equal(t, 0, len(updated.CreditCards))

				// rollback for next tests
				userRepository.Update(ctx, reuben)
			})
		})
		t.Run("hasMany field after lazy load - success", func(t *testing.T) {
			found, _ := userRepository.FindOne(ctx, reuben.ID)
			data.LazyLoadNow[[]CreditCard]("CreditCards", &found)

			found.CreditCards = []CreditCard{
				{
					Number: "999999999999",
				},
			}

			updated, err := userRepository.Update(ctx, found)
			assert.Nil(t, err)
			assert.Empty(t, updated.CreditCards)

			creditCards, err := data.LazyLoadNow[[]CreditCard]("CreditCards", &updated)
			assert.Nil(t, err)
			assert.Equal(t, updated.CreditCards, creditCards)
			assert.Equal(t, 1, len(updated.CreditCards))

			// rollback for next tests
			userRepository.Update(ctx, reuben)
		})
		t.Run("empty hasMany field after lazy load", func(t *testing.T) {
			t.Run("nil hasMany field", func(t *testing.T) {
				found, _ := userRepository.FindOne(ctx, reuben.ID)
				data.LazyLoadNow[[]CreditCard]("CreditCards", &found)

				found.CreditCards = nil

				updated, err := userRepository.Update(ctx, found)
				assert.Nil(t, err)
				assert.Empty(t, updated.CreditCards)

				creditCards, err := data.LazyLoadNow[[]CreditCard]("CreditCards", &updated)
				assert.Nil(t, err)
				assert.Equal(t, updated.CreditCards, creditCards)
				assert.Equal(t, 0, len(updated.CreditCards))

				// rollback for next tests
				userRepository.Update(ctx, reuben)
			})
			t.Run("empty slice hasMany field", func(t *testing.T) {
				found, _ := userRepository.FindOne(ctx, reuben.ID)
				data.LazyLoadNow[[]CreditCard]("CreditCards", &found)

				found.CreditCards = []CreditCard{}

				updated, err := userRepository.Update(ctx, found)
				assert.Nil(t, err)
				assert.Empty(t, updated.CreditCards)

				creditCards, err := data.LazyLoadNow[[]CreditCard]("CreditCards", &updated)
				assert.Nil(t, err)
				assert.Equal(t, updated.CreditCards, creditCards)
				assert.Equal(t, 0, len(updated.CreditCards))

				// rollback for next tests
				userRepository.Update(ctx, reuben)
			})
		})
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
		reuben, _ = userRepository.Create(ctx, reuben)

		found, _ := userRepository.FindOne(ctx, reuben.ID)

		err := userRepository.Delete(ctx, found) // soft delete
		assert.Nil(t, err)

		_, err = userRepository.FindOne(ctx, reuben.ID)
		assert.ErrorIs(t, data.NotFoundError, err)

		var cards []CreditCard
		result := db.Model(&[]CreditCard{}).Where("user_id = ? ", reuben.ID).Find(&cards)
		assert.Nil(t, result.Error)
		assert.Empty(t, cards)
	})

	db.Migrator().DropTable(&User{})
	db.Migrator().DropTable(&CreditCard{})
}

func TestGormRepository_HasMany_Eager(t *testing.T) {
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

	transactionManager := data.NewGormTransactionManager(db)
	userRepository := data.NewGormRepository[User, uint](transactionManager)

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
		assert.ErrorIs(t, data.NotFoundError, err)

		var cards []CreditCard
		result := db.Model(&CreditCard{}).Find(&cards, []uint{created.CreditCards[0].ID, created.CreditCards[1].ID})
		assert.Equal(t, int64(0), result.RowsAffected)
		assert.Empty(t, cards)
	})

	db.Migrator().DropTable(&User{})
	db.Migrator().DropTable(&CreditCard{})
}

func TestGormRepository_ManyToMany_Lazy(t *testing.T) {
	// User has and belongs to many languages, `user_languages` is the join table
	type Language struct {
		ID        uint
		Name      string
		CreatedAt time.Time
		DeletedAt gorm.DeletedAt `gorm:"index"`
	}
	type User struct {
		data.LazyLoader `gorm:"-"`
		gorm.Model
		Name      string
		Languages []Language `gorm:"many2many:user_languages;" fetch:"lazy"`
	}
	db := getGormDB()
	db.AutoMigrate(&Language{})
	db.AutoMigrate(&User{})

	transactionManager := data.NewGormTransactionManager(db)
	languageRepository := data.NewGormRepository[Language, uint](transactionManager)
	userRepository := data.NewGormRepository[User, uint](transactionManager)

	ctx := context.Background()

	languages := []Language{
		{
			Name: "kr",
		},
		{
			Name: "en",
		},
		{
			Name: "ch",
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

		languages, err := data.LazyLoadNow[[]Language]("Languages", &found)
		assert.Nil(t, err)
		assert.Equal(t, created.Languages, languages)

		t.Run("find by", func(t *testing.T) {
			found, err := userRepository.FindBy(ctx, "Language", languagesCreated[1])
			assert.Nil(t, err)

			assert.Equal(t, 1, len(found))
			assert.Equal(t, created.ID, found[0].ID)
			assert.Equal(t, created.Name, found[0].Name)

			languages, err := data.LazyLoadNow[[]Language]("Languages", &found[0])
			assert.Nil(t, err)
			assert.Equal(t, created.Languages, languages)
		})
	})

	t.Run("update field", func(t *testing.T) {
		ctx := context.Background()
		reuben := User{
			Name: "reuben.b",
			Languages: []Language{
				languagesCreated[0],
				languagesCreated[1],
			},
		}
		reuben, _ = userRepository.Create(ctx, reuben)
		t.Run("name field without lazy load", func(t *testing.T) {
			found, _ := userRepository.FindOne(ctx, reuben.ID)
			found.Name = "reuben.baek"
			updated, err := userRepository.Update(ctx, found)

			assert.Nil(t, err)
			assert.NotEmpty(t, updated.ID)
			assert.Equal(t, found.Name, updated.Name)
			assert.Empty(t, updated.Languages)

			languages, err := data.LazyLoadNow[[]Language]("Languages", &updated)
			assert.Nil(t, err)
			assert.Equal(t, reuben.Languages, languages)

			// rollback for next tests
			userRepository.Update(ctx, reuben)
		})

		t.Run("name field after lazy load", func(t *testing.T) {
			found, _ := userRepository.FindOne(ctx, reuben.ID)
			languages, err := data.LazyLoadNow[[]Language]("Languages", &found)
			assert.Nil(t, err)
			assert.NotEmpty(t, found.Languages)
			assert.Equal(t, found.Languages, languages)

			found.Name = "reuben.baek"
			updated, err := userRepository.Update(ctx, found)
			assert.Nil(t, err)
			assert.NotEmpty(t, updated.ID)
			assert.Equal(t, found.Name, updated.Name)
			assert.Empty(t, updated.Languages)
			// rollback for next tests
			userRepository.Update(ctx, reuben)
		})
		t.Run("many-to-many field before lazy load - success", func(t *testing.T) {
			found, _ := userRepository.FindOne(ctx, reuben.ID)

			found.Languages = []Language{
				languagesCreated[2],
			}

			updated, err := userRepository.Update(ctx, found)
			assert.Nil(t, err)
			assert.Empty(t, updated.Languages)

			languages, err := data.LazyLoadNow[[]Language]("Languages", &updated)
			assert.Nil(t, err)
			assert.Equal(t, updated.Languages, languages)
			assert.Equal(t, 1, len(updated.Languages))

			// rollback for next tests
			userRepository.Update(ctx, reuben)
		})
		t.Run("many-to-many field after lazy load - success", func(t *testing.T) {
			found, _ := userRepository.FindOne(ctx, reuben.ID)
			data.LazyLoadNow[[]Language]("Languages", &found)

			found.Languages = []Language{
				languagesCreated[2],
			}

			updated, err := userRepository.Update(ctx, found)
			assert.Nil(t, err)
			assert.Empty(t, updated.Languages)

			languages, err := data.LazyLoadNow[[]Language]("Languages", &updated)
			assert.Nil(t, err)
			assert.Equal(t, updated.Languages, languages)
			assert.Equal(t, 1, len(updated.Languages))

			// rollback for next tests
			userRepository.Update(ctx, reuben)
		})
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
		assert.ErrorIs(t, data.NotFoundError, err)

		var languages []Language
		result := db.Unscoped().Table("user_languages").Where("user_id = ?", created.ID).Find(&languages)
		assert.Nil(t, result.Error)
		assert.Equal(t, int64(0), result.RowsAffected)

	})
	db.Migrator().DropTable(&User{})
	db.Migrator().DropTable(&Language{})
}

func TestGormRepository_ManyToMany_Eager(t *testing.T) {
	// User has and belongs to many languages, `user_languages` is the join table
	type Language struct {
		ID   uint
		Name string
		//Users []User `gorm:"many2many:user_languages;" fetch:"eager"`
		CreatedAt time.Time
		DeletedAt gorm.DeletedAt `gorm:"index"`
	}
	type User struct {
		gorm.Model
		Name      string
		Languages []Language `gorm:"many2many:user_languages;" fetch:"eager"`
	}
	db := getGormDB()
	db.AutoMigrate(&Language{})
	db.AutoMigrate(&User{})

	transactionManager := data.NewGormTransactionManager(db)
	languageRepository := data.NewGormRepository[Language, uint](transactionManager)
	userRepository := data.NewGormRepository[User, uint](transactionManager)

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
		assert.ErrorIs(t, data.NotFoundError, err)

		var languages []Language
		result := db.Unscoped().Table("user_languages").Where("user_id = ?", created.ID).Find(&languages)
		assert.Nil(t, result.Error)
		assert.Equal(t, int64(0), result.RowsAffected)

	})
	db.Migrator().DropTable(&User{})
	db.Migrator().DropTable(&Language{})
}
