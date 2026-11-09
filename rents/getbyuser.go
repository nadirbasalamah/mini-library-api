package rents

import (
	"context"

	"gorm.io/gorm"
)

type getbyuser struct {
	repository *gorm.DB
}

func (g getbyuser) GetByUser(ctx context.Context, userId uint) ([]Rent, error) {
	rents := []Rent{}

	if err := g.repository.
		WithContext(ctx).
		Preload("User").
		Preload("Book").
		Preload("Book.Category").
		Where("user_id = ?", userId).
		Find(&rents).Error; err != nil {
		return nil, err
	}

	return rents, nil
}
