package f_repository_impl

import (
	"context"
	"github.com/reuben-baek/go-learning/e-domain"
	"gorm.io/gorm"
)

type DTO[M any] interface {
	To() M
	From(m M) any
}

type DtoWrapRepository[D DTO[M], M any, ID comparable] struct {
	dtoRepository e_domain.Repository[D, ID]
}

func (d *DtoWrapRepository[D, M, ID]) FindOne(ctx context.Context, id ID) (M, error) {
	dto, err := d.dtoRepository.FindOne(ctx, id)
	return dto.To(), err
}

func (d *DtoWrapRepository[D, M, ID]) FindBy(ctx context.Context, belongTo any) ([]M, error) {
	dtos, err := d.dtoRepository.FindBy(ctx, belongTo)

	models := make([]M, 0, len(dtos))
	for _, v := range dtos {
		models = append(models, v.To())
	}
	return models, err
}

func (d *DtoWrapRepository[D, M, ID]) Create(ctx context.Context, entity M) (M, error) {
	var dto D
	dto = dto.From(entity).(D)
	created, err := d.dtoRepository.Create(ctx, dto)
	return created.To(), err
}

func (d *DtoWrapRepository[D, M, ID]) Update(ctx context.Context, entity M) (M, error) {
	var dto D
	dto = dto.From(entity).(D)
	created, err := d.dtoRepository.Update(ctx, dto)
	return created.To(), err
}

func (d *DtoWrapRepository[D, M, ID]) Delete(ctx context.Context, entity M) error {
	var dto D
	dto = dto.From(entity).(D)
	err := d.dtoRepository.Delete(ctx, dto)
	return err
}

func NewDtoWrapRepository[D DTO[M], M any, ID comparable](dtoRepository e_domain.Repository[D, ID]) *DtoWrapRepository[D, M, ID] {
	return &DtoWrapRepository[D, M, ID]{
		dtoRepository: dtoRepository,
	}
}

func NewGormDtoWrapRepository[D DTO[M], M any, ID comparable](db *gorm.DB) *DtoWrapRepository[D, M, ID] {
	return &DtoWrapRepository[D, M, ID]{
		dtoRepository: NewGormRepository[D, ID](db),
	}
}
