package f_repository_impl_test

import (
	"context"
	"github.com/reuben-baek/go-learning/e-domain"
	f_repository_impl "github.com/reuben-baek/go-learning/f-repository-impl"
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

func TestServerRepositoryImpl(t *testing.T) {
	logConfig := logger.New(log.New(os.Stdout, "\r\n", log.LstdFlags), logger.Config{
		SlowThreshold: 100 * time.Millisecond,
		LogLevel:      logger.Info,
		Colorful:      true,
	})
	var err error
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{Logger: logConfig})
	if err != nil {
		panic("failed to connect database")
	}

	var serverRepository e_domain.ServerRepository
	var flavorRepository e_domain.FlavorRepository
	serverRepository = f_repository_impl.NewGormServerRepository(db)
	flavorRepository = f_repository_impl.NewGormFlavorRepository(db)

	db.AutoMigrate(&f_repository_impl.Server{})
	db.AutoMigrate(&f_repository_impl.Flavor{})

	db.Create(&f_repository_impl.Flavor{
		ID:   "flavor-1",
		Name: "flavor-1_4core_16G",
	})
	db.Create(&f_repository_impl.Flavor{
		ID:   "flavor-2",
		Name: "flavor-2_8core_32G",
	})

	db.Create(&f_repository_impl.Server{
		ID:       "server-1",
		Name:     "server-1-name",
		FlavorID: "flavor-1",
	})
	db.Create(&f_repository_impl.Server{
		ID:       "server-2",
		Name:     "server-2-name",
		FlavorID: "flavor-2",
	})

	t.Run("find-one", func(t *testing.T) {
		var err error
		ctx := context.Background()
		server1, err := serverRepository.FindOne(ctx, "server-1")
		require.Nil(t, err)

		assert.Equal(t, "server-1", server1.ID())
		assert.Equal(t, "server-1-name", server1.Name())
		assert.Equal(t, "flavor-1", server1.Flavor().ID())
		assert.Equal(t, "flavor-1_4core_16G", server1.Flavor().Name())
	})

	t.Run("create", func(t *testing.T) {
		var err error
		ctx := context.Background()
		flavor, err := flavorRepository.FindOne(ctx, "flavor-1")
		assert.Nil(t, err)

		server := e_domain.ServerInstance("create-server-1", "create-server-1", flavor)
		saved, err := serverRepository.Create(ctx, server)
		assert.Nil(t, err)
		assert.Equal(t, server.ID(), saved.ID())
		assert.Equal(t, server.Name(), saved.Name())
		assert.Equal(t, "flavor-1", saved.Flavor().ID())
		assert.Equal(t, "flavor-1_4core_16G", saved.Flavor().Name())
	})
}
