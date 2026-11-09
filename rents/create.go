package rents

import (
	"context"
	"go-digital-library-fiber/db/models"

	"gorm.io/gorm"
)

type create struct {
	repository *gorm.DB
}

func (c create) Create(ctx context.Context, rentReq *RentRequest) error {
	err := c.repository.Transaction(func(tx *gorm.DB) error {
		for _, bookId := range rentReq.Books {
			var book models.Book

			if err := tx.WithContext(ctx).
				Where("id = ? AND stock > 0", bookId).
				Find(&book).Error; err != nil {
				return err
			}

			rent := models.Rent{
				UserID:     rentReq.UserID,
				BookID:     book.ID,
				Fee:        rentReq.Fee,
				Courier:    rentReq.Courier,
				Status:     models.Pending,
				Duration:   rentReq.Duration,
				ReturnTime: rentReq.ReturnTime,
			}

			if err := tx.WithContext(ctx).Create(&rent).Error; err != nil {
				return err
			}

			if err := tx.WithContext(ctx).First(&book, "id = ?", bookId).Error; err != nil {
				return err
			}

			book.Stock = book.Stock - 1

			if err := tx.WithContext(ctx).Where("id = ?", bookId).Updates(book).Error; err != nil {
				return err
			}
		}
		return nil
	})

	return err
}
