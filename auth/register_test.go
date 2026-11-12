package auth

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = mockDB.Close() })

	gormDB, err := gorm.Open(
		postgres.New(postgres.Config{Conn: mockDB}),
		&gorm.Config{SkipDefaultTransaction: true},
	)
	require.NoError(t, err)

	return gormDB, mock
}

func testRegisterReq() *RegisterRequest {
	return &RegisterRequest{
		Username:   "testuser",
		Email:      "test@example.com",
		Password:   "Password1!",
		Address:    "123 Test St",
		ProvinceID: 1,
		CityID:     2,
		DistrictID: 3,
	}
}

func TestRegister_Success(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	r := register{repository: gormDB}
	req := testRegisterReq()

	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "users"`)).
		WithArgs("testuser", "test@example.com", sqlmock.AnyArg(), "123 Test St", int64(1), int64(2), int64(3), "", "", "user", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT "users"."id","users"."username"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "email", "password", "address", "province_id", "city_id", "district_id", "profile_picture", "role", "created_at", "updated_at", "deleted_at"}).
			AddRow(int64(1), "testuser", "test@example.com", "hashed_pass", "123 Test St", int64(1), int64(2), int64(3), "", "user", nil, nil, nil))

	user, err := r.Register(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, uint(1), user.ID)
	require.Equal(t, "testuser", user.Username)
	require.Equal(t, "test@example.com", user.Email)
	require.Equal(t, "123 Test St", user.Address)
	require.Equal(t, uint(1), user.ProvinceID)
	require.Equal(t, uint(2), user.CityID)
	require.Equal(t, uint(3), user.DistrictID)
	require.Equal(t, "user", user.Role)
}

func TestRegister_ErrorOnInsert(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	r := register{repository: gormDB}
	req := testRegisterReq()

	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "users"`)).
		WithArgs("testuser", "test@example.com", sqlmock.AnyArg(), "123 Test St", int64(1), int64(2), int64(3), "", "", "user", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnError(errors.New("insert error"))

	_, err := r.Register(context.Background(), req)
	require.Error(t, err)
	require.Contains(t, err.Error(), "insert error")
}

func TestRegister_ErrorOnLastQuery(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	r := register{repository: gormDB}
	req := testRegisterReq()

	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "users"`)).
		WithArgs("testuser", "test@example.com", sqlmock.AnyArg(), "123 Test St", int64(1), int64(2), int64(3), "", "", "user", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT "users"."id","users"."username"`)).
		WillReturnError(errors.New("last query error"))

	_, err := r.Register(context.Background(), req)
	require.Error(t, err)
	require.Contains(t, err.Error(), "last query error")
}
