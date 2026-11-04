package books

import (
	"context"
	"go-digital-library-fiber/pkg/utils"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type getall struct {
	repository *gorm.DB
}

func (g getall) GetAll(ctx context.Context, pagination utils.Pagination) (utils.Pagination, error) {
	books := []Book{}

	if err := g.repository.
		WithContext(ctx).
		Scopes(utils.Paginate(&books, &pagination, g.repository)).
		Preload(clause.Associations).
		Find(&books).Error; err != nil {
		return utils.Pagination{}, err
	}

	pagination.Rows = books

	return pagination, nil
}
