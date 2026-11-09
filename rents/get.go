package rents

import (
	"context"

	"gorm.io/gorm"
)

type get struct {
	repository *gorm.DB
}

func (g get) GetByID(ctx context.Context, id uint) (Rent, error) {
	rent := new(Rent)

	if err := g.repository.
		WithContext(ctx).
		Preload("User").
		Preload("Book").
		Preload("Book.Category").
		First(rent, "id = ?", id).Error; err != nil {
		return Rent{}, err
	}

	return *rent, nil
}
