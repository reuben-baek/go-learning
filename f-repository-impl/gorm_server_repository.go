package f_repository_impl

import (
	"context"
	"github.com/reuben-baek/go-learning/e-domain"
	"github.com/reuben-baek/go-learning/e-domain/data"
	"gorm.io/gorm"
	"time"
)

type Server struct {
	ID        string `gorm:"primaryKey"`
	Name      string
	FlavorID  string
	Flavor    Flavor `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (s *Server) to(flavorLoadFn func() (any, error)) *lazyServer {
	return &lazyServer{
		Server: e_domain.ServerInstance(s.ID, s.Name, nil),
		flavor: data.LazyLoadFn[e_domain.Flavor](flavorLoadFn),
	}
}

type lazyServer struct {
	e_domain.Server
	flavor *data.LazyLoad[e_domain.Flavor]
}

func (s *lazyServer) Flavor() e_domain.Flavor {
	return s.flavor.Get()
}

func fromServerInstance(s e_domain.Server) Server {
	return Server{
		ID:       s.ID(),
		Name:     s.Name(),
		FlavorID: s.Flavor().ID(),
	}
}

type GormServerRepository struct {
	db *gorm.DB
}

func NewGormServerRepository(db *gorm.DB) *GormServerRepository {
	return &GormServerRepository{db: db}
}

func (s *GormServerRepository) FindBy(ctx context.Context, belongTo any) ([]e_domain.Server, error) {
	//TODO implement me
	panic("implement me")
}

func (s *GormServerRepository) Create(ctx context.Context, server e_domain.Server) (e_domain.Server, error) {
	var dto Server
	dto = fromServerInstance(server)
	if err := s.db.Create(&dto).Error; err != nil {
		return nil, err
	}
	return dto.to(s.loadFlavor(ctx, dto.FlavorID)), nil
}

func (s *GormServerRepository) Update(ctx context.Context, server e_domain.Server) (e_domain.Server, error) {
	if server.ID() == "" {
		panic("object.id is missing")
	}
	var dto Server
	dto = fromServerInstance(server)
	if err := s.db.Save(&dto).Error; err != nil {
		return nil, err
	}
	return dto.to(s.loadFlavor(ctx, dto.FlavorID)), nil
}

func (s *GormServerRepository) Delete(ctx context.Context, server e_domain.Server) error {
	if server.ID() == "" {
		panic("object.id is missing")
	}
	var dto Server
	dto = fromServerInstance(server)
	if err := s.db.Delete(&dto).Error; err != nil {
		return err
	}
	return nil
}

func (s *GormServerRepository) FindOne(ctx context.Context, id string) (e_domain.Server, error) {
	var dto Server
	if err := s.db.First(&dto, "id = ?", id).Error; err != nil {
		return nil, err
	}

	return dto.to(s.loadFlavor(ctx, dto.FlavorID)), nil
}

func (s *GormServerRepository) loadFlavor(ctx context.Context, id string) func() (any, error) {
	return func() (any, error) {
		var flavorDto Flavor
		if err := s.db.First(&flavorDto, "id = ?", id).Error; err != nil {
			return nil, err
		}

		return flavorDto.to(), nil
	}
}
