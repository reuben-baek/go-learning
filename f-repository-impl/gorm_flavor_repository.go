package f_repository_impl

import (
	"context"
	"github.com/reuben-baek/go-learning/e-domain"
	"gorm.io/gorm"
	"time"
)

type Flavor struct {
	ID        string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (f *Flavor) to() e_domain.Flavor {
	return e_domain.FlavorInstance(f.ID, f.Name)
}

func fromFlavorInstance(flavor e_domain.Flavor) Flavor {
	return Flavor{
		ID:   flavor.ID(),
		Name: flavor.Name(),
	}
}

type GormFlavorRepository struct {
	db *gorm.DB
}

func NewGormFlavorRepository(db *gorm.DB) *GormFlavorRepository {
	return &GormFlavorRepository{db: db}
}

func (s *GormFlavorRepository) Create(ctx context.Context, flavor e_domain.Flavor) (e_domain.Flavor, error) {
	var dto Flavor
	dto = fromFlavorInstance(flavor)
	if err := s.db.Create(&dto).Error; err != nil {
		return nil, err
	}
	return dto.to(), nil
}

func (s *GormFlavorRepository) Update(ctx context.Context, flavor e_domain.Flavor) (e_domain.Flavor, error) {
	if flavor.ID() == "" {
		panic("object.id is missing")
	}
	var dto Flavor
	dto = fromFlavorInstance(flavor)
	if err := s.db.Save(&dto).Error; err != nil {
		return nil, err
	}
	return dto.to(), nil
}

func (s *GormFlavorRepository) Delete(ctx context.Context, flavor e_domain.Flavor) error {
	if flavor.ID() == "" {
		panic("object.id is missing")
	}
	var dto Flavor
	dto = fromFlavorInstance(flavor)
	if err := s.db.Delete(&dto).Error; err != nil {
		return err
	}
	return nil
}

func (s *GormFlavorRepository) FindOne(ctx context.Context, id string) (e_domain.Flavor, error) {
	var dto Flavor
	if err := s.db.First(&dto, "id = ?", id).Error; err != nil {
		return nil, err
	}

	return dto.to(), nil
}
