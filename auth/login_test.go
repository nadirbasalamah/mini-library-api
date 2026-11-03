package auth

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"go-digital-library-fiber/pkg/utils"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func testLoginReq() *LoginRequest {
	return &LoginRequest{
		Email:    "test@example.com",
		Password: "Password1!",
	}
}

func TestLogin_Success(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	l := login{repository: gormDB}
	req := testLoginReq()

	hashedPassword, err := utils.GeneratePassword(req.Password)
	require.NoError(t, err)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE email =`)).
		WithArgs("test@example.com", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "email", "password", "address", "province_id", "city_id", "district_id", "profile_picture", "role", "created_at", "updated_at", "deleted_at"}).
			AddRow(int64(1), "testuser", "test@example.com", string(hashedPassword), "123 Test St", int64(1), int64(2), int64(3), "", "user", nil, nil, nil))

	user, err := l.Login(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, uint(1), user.ID)
	require.Equal(t, "testuser", user.Username)
	require.Equal(t, "test@example.com", user.Email)
	require.Equal(t, "user", user.Role)
}

func TestLogin_UserNotFound(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	l := login{repository: gormDB}
	req := testLoginReq()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE email =`)).
		WithArgs("test@example.com", 1).
		WillReturnError(errors.New("record not found"))

	_, err := l.Login(context.Background(), req)
	require.Error(t, err)
	require.Contains(t, err.Error(), "record not found")
}

func TestLogin_WrongPassword(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	l := login{repository: gormDB}
	req := &LoginRequest{
		Email:    "test@example.com",
		Password: "WrongPassword1!",
	}

	hashedPassword, err := utils.GeneratePassword("CorrectPassword1!")
	require.NoError(t, err)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE email =`)).
		WithArgs("test@example.com", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "email", "password", "address", "province_id", "city_id", "district_id", "profile_picture", "role", "created_at", "updated_at", "deleted_at"}).
			AddRow(int64(1), "testuser", "test@example.com", string(hashedPassword), "123 Test St", int64(1), int64(2), int64(3), "", "user", nil, nil, nil))

	_, err = l.Login(context.Background(), req)
	require.Error(t, err)
}
