package auth

import (
	"context"
	"go-digital-library-fiber/pkg/utils"

	"gorm.io/gorm"
)

type login struct {
	repository *gorm.DB
}

func (l login) Login(ctx context.Context, req *LoginRequest) (User, error) {
	user := new(User)

	err := l.repository.WithContext(ctx).First(user, "email = ?", req.Email).Error

	if err != nil {
		return User{}, err
	}

	err = utils.ComparePassword(user.Password, req.Password)

	if err != nil {
		return User{}, err
	}

	return *user, nil
}
