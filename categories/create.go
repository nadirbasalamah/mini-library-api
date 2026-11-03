package categories

import (
	"context"
	"go-digital-library-fiber/db/models"

	"gorm.io/gorm"
)

type create struct {
	repository *gorm.DB
}

func (c create) Create(ctx context.Context, categoryReq *CategoryRequest) (Category, error) {
	category := models.Category{
		Name:        categoryReq.Name,
		Description: categoryReq.Description,
	}

	result := c.repository.WithContext(ctx).Create(&category)
	record := new(Category)

	if err := result.Error; err != nil {
		return Category{}, err
	}

	if err := result.WithContext(ctx).Last(record).Error; err != nil {
		return Category{}, err
	}

	return *record, nil
}
