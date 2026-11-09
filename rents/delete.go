package rents

import (
	"context"
	"errors"
	"go-digital-library-fiber/db/models"

	"gorm.io/gorm"
)

type delete struct {
	repository *gorm.DB
	get
}

func (d delete) Delete(ctx context.Context, id uint) error {
	rent, err := d.get.GetByID(ctx, id)

	if err != nil {
		return err
	}

	isValidStatus := rent.Status == string(models.Cancelled) || rent.Status == string(models.Returned)

	if !isValidStatus {
		return errors.New("rent status must be returned or cancelled")
	}

	if err := d.repository.WithContext(ctx).Delete(&rent).Error; err != nil {
		return err
	}

	return nil
}
