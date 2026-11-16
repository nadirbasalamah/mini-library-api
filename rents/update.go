package rents

import (
	"context"
	"fmt"
	"go-digital-library-fiber/db/models"
	"time"

	"gorm.io/gorm"
)

type update struct {
	repository *gorm.DB
	get
}

const BY_ID_CLAUSE = "id = ?"

func updateTransaction(ctx context.Context, id uint, rent models.Rent, tx *gorm.DB) error {
	result := tx.WithContext(ctx).Where(
		"id = ? AND status NOT IN (?,?)",
		id, models.Returned, models.Cancelled,
	).Updates(rent)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("rent already returned or not found")
	}

	if rent.Status == models.Returned || rent.Status == models.Cancelled {
		rentRecord := new(Rent)
		book := new(models.Book)

		if err := tx.WithContext(ctx).Where(BY_ID_CLAUSE, id).First(rentRecord).Error; err != nil {
			return err
		}

		bookId := rentRecord.BookID

		if err := tx.WithContext(ctx).First(book, BY_ID_CLAUSE, bookId).Error; err != nil {
			return err
		}

		book.Stock = book.Stock + 1

		if err := tx.WithContext(ctx).Where(BY_ID_CLAUSE, bookId).Updates(book).Error; err != nil {
			return err
		}
	}

	return nil
}

func (u update) Update(ctx context.Context, rentReq *UpdateRentRequest, id uint) (Rent, error) {
	status := models.RentStatus(rentReq.Status)

	rent := models.Rent{
		Status: status,
	}

	if status == models.Returned {
		rent.ReturnedAt = time.Now()
	}

	txErr := u.repository.Transaction(func(tx *gorm.DB) error {
		return updateTransaction(ctx, id, rent, tx)
	})

	if txErr != nil {
		return Rent{}, txErr
	}

	rentRecord, err := u.get.GetByID(ctx, id)

	if err != nil {
		return Rent{}, err
	}

	return rentRecord, nil
}
