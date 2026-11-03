package categories

import (
	"context"
	"go-digital-library-fiber/pkg/utils"

	"gorm.io/gorm"
)

type getall struct {
	repository *gorm.DB
}

func (g getall) GetAll(ctx context.Context, pagination utils.Pagination) (utils.Pagination, error) {
	categories := []Category{}

	if err := g.repository.
		WithContext(ctx).
		Scopes(utils.Paginate(&categories, &pagination, g.repository)).
		Find(&categories).Error; err != nil {
		return utils.Pagination{}, err
	}

	pagination.Rows = categories

	return pagination, nil
}
