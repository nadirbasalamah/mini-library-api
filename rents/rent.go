package rents

import (
	"go-digital-library-fiber/books"
	"go-digital-library-fiber/users"
	"time"

	"gorm.io/gorm"
)

type Rent struct {
	ID         uint           `json:"id" gorm:"primaryKey"`
	UserID     uint           `json:"user_id"`
	BookID     uint           `json:"book_id"`
	User       users.User     `json:"user"`
	Book       books.Book     `json:"book"`
	Fee        float64        `json:"fee"`
	Courier    string         `json:"courier"`
	Duration   uint           `json:"duration"`
	Status     string         `json:"rent_status" gorm:"type:rent_status"`
	ReturnTime time.Time      `json:"return_time"`
	ReturnedAt time.Time      `json:"returned_at"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}

type RentRequest struct {
	Books      []uint `json:"books" validate:"required"`
	Duration   uint   `json:"duration" validate:"required,gte=1"`
	Courier    string `json:"courier" validate:"required,validCourier"`
	UserID     uint
	Fee        float64
	ReturnTime time.Time
}

type UpdateRentRequest struct {
	Status string `json:"status" validate:"required"`
}
