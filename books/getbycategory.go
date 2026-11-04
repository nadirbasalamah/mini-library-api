package books

import (
	"context"
	"go-digital-library-fiber/pkg/utils"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type getbycategory struct {
	repository *gorm.DB
}

func (g getbycategory) GetByCategory(ctx context.Context, pagination utils.Pagination, categoryId uint) (utils.Pagination, error) {
	books := []Book{}

	if err := g.repository.
		WithContext(ctx).
		Scopes(utils.PaginateByBookCategory(&books, &pagination, categoryId, g.repository)).
		Preload(clause.Associations).
		Find(&books).Error; err != nil {
		return utils.Pagination{}, err
	}

	pagination.Rows = books

	return pagination, nil
}
