package categories

import (
	"context"

	"gorm.io/gorm"
)

type delete struct {
	repository *gorm.DB
	get
}

func (d delete) Delete(ctx context.Context, id uint) error {
	category, err := d.get.GetByID(ctx, id)

	if err != nil {
		return err
	}

	if err := d.repository.Delete(&category).Error; err != nil {
		return err
	}

	return nil
}
