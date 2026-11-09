package rents

import (
	"context"
	"go-digital-library-fiber/pkg/utils"

	"gorm.io/gorm"
)

type getall struct {
	repository *gorm.DB
}

func (g getall) GetAll(ctx context.Context, pagination utils.Pagination) ([]Rent, error) {
	rents := []Rent{}

	if err := g.repository.
		WithContext(ctx).
		Scopes(utils.Paginate(&rents, &pagination, g.repository)).
		Preload("User").
		Preload("Book").
		Preload("Book.Category").
		Find(&rents).Error; err != nil {
		return nil, err
	}

	return rents, nil
}
