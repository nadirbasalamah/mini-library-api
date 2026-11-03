package auth

import (
	"context"
	"go-digital-library-fiber/db/models"
	"go-digital-library-fiber/pkg/utils"

	"gorm.io/gorm"
)

type register struct {
	repository *gorm.DB
}

func (r register) Register(ctx context.Context, req *RegisterRequest) (User, error) {
	password, err := utils.GeneratePassword(req.Password)

	if err != nil {
		return User{}, err
	}

	user := models.User{
		Username:   req.Username,
		Email:      req.Email,
		Address:    req.Address,
		ProvinceID: req.ProvinceID,
		CityID:     req.CityID,
		DistrictID: req.DistrictID,
		Password:   string(password),
		Role:       models.Enduser,
	}

	result := r.repository.WithContext(ctx).Create(&user)

	if err := result.Error; err != nil {
		return User{}, err
	}

	record := new(User)

	if err := result.WithContext(ctx).Last(record).Error; err != nil {
		return User{}, err
	}

	return *record, nil
}
