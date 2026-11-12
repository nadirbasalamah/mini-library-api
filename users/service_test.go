package users

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupUserService(t *testing.T) (Service, sqlmock.Sqlmock) {
	t.Helper()

	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{
		SkipDefaultTransaction: true,
		Logger:                 logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	return New(db), mock
}

func userTestRows(id int64, username, email, address, picture, pictureID string) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "username", "email", "password", "address", "province_id", "city_id", "district_id",
		"profile_picture", "profile_picture_id", "role", "created_at", "updated_at", "deleted_at",
	}).AddRow(id, username, email, "hashed-password", address, int64(1), int64(2), int64(3),
		picture, pictureID, "user", nil, nil, nil)
}

func TestServiceGetProfile(t *testing.T) {
	t.Run("returns profile by user ID", func(t *testing.T) {
		service, mock := setupUserService(t)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1 AND "users"."deleted_at" IS NULL ORDER BY "users"."id" LIMIT $2`)).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(userTestRows(42, "reader1", "reader1@example.com", "1 Main St", "https://example.com/avatar.png", "avatar"))

		profile, err := service.GetProfile(context.Background(), 42)

		require.NoError(t, err)
		require.Equal(t, uint(42), profile.ID)
		require.Equal(t, "reader1", profile.Username)
		require.Equal(t, "reader1@example.com", profile.Email)
		require.Equal(t, "https://example.com/avatar.png", profile.ProfilePicture)
		require.Equal(t, "avatar", profile.ProfilePictureID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns lookup error", func(t *testing.T) {
		service, mock := setupUserService(t)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1 AND "users"."deleted_at" IS NULL ORDER BY "users"."id" LIMIT $2`)).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(errors.New("user not found"))

		_, err := service.GetProfile(context.Background(), 42)

		require.EqualError(t, err, "user not found")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestServiceUpdate(t *testing.T) {
	request := &EditProfileRequest{
		Username:         "reader2",
		Email:            "reader2@example.com",
		Address:          "2 Main St",
		ProfilePicture:   "https://example.com/new-avatar.png",
		ProfilePictureID: "new-avatar",
	}

	t.Run("updates profile and returns refreshed user", func(t *testing.T) {
		service, mock := setupUserService(t)
		mock.ExpectExec(`UPDATE "users" SET`).
			WithArgs("reader2", "reader2@example.com", "2 Main St", "https://example.com/new-avatar.png",
				"new-avatar", sqlmock.AnyArg(), int64(42)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1 AND "users"."deleted_at" IS NULL ORDER BY "users"."id" LIMIT $2`)).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(userTestRows(42, "reader2", "reader2@example.com", "2 Main St", "https://example.com/new-avatar.png", "new-avatar"))

		profile, err := service.Update(context.Background(), request, 42)

		require.NoError(t, err)
		require.Equal(t, uint(42), profile.ID)
		require.Equal(t, "reader2", profile.Username)
		require.Equal(t, "reader2@example.com", profile.Email)
		require.Equal(t, "2 Main St", profile.Address)
		require.Equal(t, "new-avatar", profile.ProfilePictureID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns database update error", func(t *testing.T) {
		service, mock := setupUserService(t)
		mock.ExpectExec(`UPDATE "users" SET`).
			WithArgs("reader2", "reader2@example.com", "2 Main St", "https://example.com/new-avatar.png",
				"new-avatar", sqlmock.AnyArg(), int64(42)).
			WillReturnError(errors.New("update failed"))

		_, err := service.Update(context.Background(), request, 42)

		require.EqualError(t, err, "update failed")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns refreshed-profile lookup error", func(t *testing.T) {
		service, mock := setupUserService(t)
		mock.ExpectExec(`UPDATE "users" SET`).
			WithArgs("reader2", "reader2@example.com", "2 Main St", "https://example.com/new-avatar.png",
				"new-avatar", sqlmock.AnyArg(), int64(42)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1 AND "users"."deleted_at" IS NULL ORDER BY "users"."id" LIMIT $2`)).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(errors.New("reload failed"))

		_, err := service.Update(context.Background(), request, 42)

		require.EqualError(t, err, "reload failed")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
