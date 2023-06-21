package data

import (
	"context"
)

type DTO[M any] interface {
	To() M
	From(m M) any
}

type DtoWrapRepository[D DTO[M], M any, ID comparable] struct {
	dtoRepository Repository[D, ID]
}

func (d *DtoWrapRepository[D, M, ID]) FindOne(ctx context.Context, id ID) (M, error) {
	dto, err := d.dtoRepository.FindOne(ctx, id)
	return dto.To(), err
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

func NewDtoWrapRepository[D DTO[M], M any, ID comparable](dtoRepository Repository[D, ID]) *DtoWrapRepository[D, M, ID] {
	return &DtoWrapRepository[D, M, ID]{
		dtoRepository: dtoRepository,
	}
}

type DtoWrapFindByRepository[D DTO[M], M any, E DTO[S], S any] struct {
	dtoRepository FindByRepository[D, E]
}

func NewDtoWrapFindByRepository[D DTO[M], M any, E DTO[S], S any](dtoRepository FindByRepository[D, E]) *DtoWrapFindByRepository[D, M, E, S] {
	return &DtoWrapFindByRepository[D, M, E, S]{dtoRepository: dtoRepository}
}

func (d *DtoWrapFindByRepository[D, M, E, S]) FindBy(ctx context.Context, name string, byEntity S) ([]M, error) {
	var dto E
	dto = dto.From(byEntity).(E)
	dtos, err := d.dtoRepository.FindBy(ctx, name, dto)

	models := make([]M, 0, len(dtos))
	for _, v := range dtos {
		models = append(models, v.To())
	}
	return models, err
}
