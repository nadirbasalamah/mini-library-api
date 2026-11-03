package categories

import (
	"context"

	"gorm.io/gorm"
)

type get struct {
	repository *gorm.DB
}

func (g get) GetByID(ctx context.Context, id uint) (Category, error) {
	category := new(Category)

	if err := g.repository.WithContext(ctx).First(category, "id = ?", id).Error; err != nil {
		return Category{}, err
	}

	return *category, nil
}
