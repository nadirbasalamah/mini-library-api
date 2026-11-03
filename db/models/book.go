package models

import (
	"time"

	"gorm.io/gorm"
)

type Book struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Publisher   string         `json:"publisher"`
	Year        uint           `json:"year"`
	Stock       uint           `json:"stock"`
	ImageLink   string         `json:"image_link"`
	ImageID     string         `json:"image_id"`
	CategoryID  uint           `json:"category_id"`
	Category    Category       `json:"category"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}
