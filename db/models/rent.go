package models

import (
	"database/sql/driver"
	"time"

	"gorm.io/gorm"
)

type Rent struct {
	ID         uint           `json:"id" gorm:"primaryKey"`
	UserID     uint           `json:"user_id"`
	BookID     uint           `json:"book_id"`
	User       User           `json:"user"`
	Book       Book           `json:"book"`
	Fee        float64        `json:"fee"`
	Courier    string         `json:"courier"`
	Duration   uint           `json:"duration"`
	Status     RentStatus     `json:"rent_status" gorm:"type:rent_status"`
	ReturnTime time.Time      `json:"return_time"`
	ReturnedAt time.Time      `json:"returned_at"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}

type RentStatus string

const (
	Pending    RentStatus = "pending"
	OnDelivery RentStatus = "on_delivery"
	Rented     RentStatus = "rented"
	Returned   RentStatus = "returned"
	Cancelled  RentStatus = "cancelled"
)

func (p *RentStatus) Scan(value any) error {
	*p = RentStatus(value.([]byte))
	return nil
}

func (p RentStatus) Value() (driver.Value, error) {
	return string(p), nil
}
