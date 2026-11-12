package rents

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"go-digital-library-fiber/pkg/utils"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupRentService(t *testing.T) (Service, sqlmock.Sqlmock) {
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

func rentTestRows(id, userID, bookID int64, status string) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "user_id", "book_id", "fee", "courier", "duration", "status",
		"return_time", "returned_at", "created_at", "updated_at", "deleted_at",
	}).AddRow(id, userID, bookID, float64(125), "jne", int64(7), status, nil, nil, nil, nil, nil)
}

func rentBookRows(id, categoryID int64) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "title", "description", "publisher", "year", "stock", "image_link", "image_id",
		"category_id", "created_at", "updated_at", "deleted_at",
	}).AddRow(id, "Dune", "A novel", "Publisher", int64(1965), int64(3), "", "", categoryID, nil, nil, nil)
}

func rentUserRows(id int64) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "username", "email", "password", "address", "province_id", "city_id", "district_id",
		"profile_picture", "profile_picture_id", "role", "created_at", "updated_at", "deleted_at",
	}).AddRow(id, "reader1", "reader1@example.com", "hashed", "1 Main St", int64(1), int64(2), int64(3),
		"", "", "user", nil, nil, nil)
}

func rentCategoryRows(id int64) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "name", "description", "created_at", "updated_at", "deleted_at",
	}).AddRow(id, "Fiction", "Fiction books", nil, nil, nil)
}

func expectRentAssociations(mock sqlmock.Sqlmock, userID, bookID, categoryID int64) {
	mock.ExpectQuery(`SELECT .* FROM "books" WHERE "books"\."id" = \$1`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(rentBookRows(bookID, categoryID))
	mock.ExpectQuery(`SELECT .* FROM "categories" WHERE "categories"\."id" = \$1`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(rentCategoryRows(categoryID))
	mock.ExpectQuery(`SELECT .* FROM "users" WHERE "users"\."id" = \$1`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(rentUserRows(userID))
	_ = userID
}

func TestServiceGetAll(t *testing.T) {
	t.Run("returns paginated rents with relations", func(t *testing.T) {
		service, mock := setupRentService(t)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "rents" WHERE "rents"."deleted_at" IS NULL`)).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(`SELECT .* FROM "rents".*courier LIKE.*ORDER BY.*LIMIT`).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(rentTestRows(1, 42, 12, "pending"))
		expectRentAssociations(mock, 42, 12, 7)

		results, err := service.GetAll(context.Background(), utils.Pagination{
			Page: 1, Limit: 10, Sort: "id desc", Search: "jne", Keyword: "courier",
		})

		require.NoError(t, err)
		require.Len(t, results, 1)
		require.Equal(t, uint(42), results[0].UserID)
		require.Equal(t, uint(12), results[0].BookID)
		require.Equal(t, "Dune", results[0].Book.Title)
		require.Equal(t, "Fiction", results[0].Book.Category.Name)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns query error", func(t *testing.T) {
		service, mock := setupRentService(t)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "rents" WHERE "rents"."deleted_at" IS NULL`)).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
		mock.ExpectQuery(`SELECT .* FROM "rents".*courier LIKE`).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(errors.New("query failed"))

		_, err := service.GetAll(context.Background(), utils.Pagination{Keyword: "courier"})

		require.EqualError(t, err, "query failed")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestServiceGetByUser(t *testing.T) {
	t.Run("returns user's rents", func(t *testing.T) {
		service, mock := setupRentService(t)
		mock.ExpectQuery(`SELECT .* FROM "rents" WHERE user_id = \$1 AND "rents"\."deleted_at" IS NULL`).
			WithArgs(int64(42)).
			WillReturnRows(rentTestRows(1, 42, 12, "pending"))
		expectRentAssociations(mock, 42, 12, 7)

		results, err := service.GetByUser(context.Background(), 42)

		require.NoError(t, err)
		require.Len(t, results, 1)
		require.Equal(t, uint(42), results[0].UserID)
		require.Equal(t, "Dune", results[0].Book.Title)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns query error", func(t *testing.T) {
		service, mock := setupRentService(t)
		mock.ExpectQuery(`SELECT .* FROM "rents" WHERE user_id = \$1 AND "rents"\."deleted_at" IS NULL`).
			WithArgs(int64(42)).
			WillReturnError(errors.New("query failed"))

		_, err := service.GetByUser(context.Background(), 42)

		require.EqualError(t, err, "query failed")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestServiceGetByID(t *testing.T) {
	t.Run("returns rent with relations", func(t *testing.T) {
		service, mock := setupRentService(t)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "rents" WHERE id = $1 AND "rents"."deleted_at" IS NULL ORDER BY "rents"."id" LIMIT $2`)).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(rentTestRows(1, 42, 12, "pending"))
		expectRentAssociations(mock, 42, 12, 7)

		rent, err := service.GetByID(context.Background(), 1)

		require.NoError(t, err)
		require.Equal(t, uint(1), rent.ID)
		require.Equal(t, "Dune", rent.Book.Title)
		require.Equal(t, "Fiction", rent.Book.Category.Name)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns lookup error", func(t *testing.T) {
		service, mock := setupRentService(t)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "rents" WHERE id = $1 AND "rents"."deleted_at" IS NULL ORDER BY "rents"."id" LIMIT $2`)).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(errors.New("rent not found"))

		_, err := service.GetByID(context.Background(), 1)

		require.EqualError(t, err, "rent not found")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestServiceCreate(t *testing.T) {
	t.Run("creates rents and decrements stock transactionally", func(t *testing.T) {
		service, mock := setupRentService(t)
		mock.ExpectBegin()
		for _, id := range []int64{12, 13} {
			mock.ExpectQuery(`SELECT .* FROM "books" WHERE \(id = \$1 AND stock > 0\) AND "books"\."deleted_at" IS NULL`).
				WithArgs(id).
				WillReturnRows(rentBookRows(id, 7))
			mock.ExpectQuery(`INSERT INTO "rents"`).
				WithArgs(int64(42), id, float64(125), "jne", int64(7), "pending", sqlmock.AnyArg(),
					sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
				WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(id))
			mock.ExpectQuery(`SELECT .* FROM "books" WHERE id = \$1 AND "books"\."deleted_at" IS NULL AND "books"\."id" = \$2 ORDER BY "books"\."id" LIMIT \$3`).
				WithArgs(id, id, sqlmock.AnyArg()).
				WillReturnRows(rentBookRows(id, 7))
			mock.ExpectExec(`UPDATE "books" SET`).
				WillReturnResult(sqlmock.NewResult(0, 1))
		}
		mock.ExpectCommit()

		err := service.Create(context.Background(), &RentRequest{
			Books: []uint{12, 13}, UserID: 42, Fee: 125, Courier: "jne", Duration: 7,
			ReturnTime: time.Date(2026, 11, 18, 0, 0, 0, 0, time.UTC),
		})

		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("rolls back when book lookup fails", func(t *testing.T) {
		service, mock := setupRentService(t)
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT .* FROM "books" WHERE \(id = \$1 AND stock > 0\) AND "books"\."deleted_at" IS NULL`).
			WithArgs(int64(12)).
			WillReturnError(errors.New("database down"))
		mock.ExpectRollback()

		err := service.Create(context.Background(), &RentRequest{Books: []uint{12}})

		require.EqualError(t, err, "database down")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("rolls back when rent insertion fails", func(t *testing.T) {
		service, mock := setupRentService(t)
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT .* FROM "books" WHERE \(id = \$1 AND stock > 0\) AND "books"\."deleted_at" IS NULL`).
			WithArgs(int64(12)).
			WillReturnRows(rentBookRows(12, 7))
		mock.ExpectQuery(`INSERT INTO "rents"`).
			WillReturnError(errors.New("insert failed"))
		mock.ExpectRollback()

		err := service.Create(context.Background(), &RentRequest{Books: []uint{12}, UserID: 42})

		require.EqualError(t, err, "insert failed")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("rolls back when stock update fails", func(t *testing.T) {
		service, mock := setupRentService(t)
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT .* FROM "books" WHERE \(id = \$1 AND stock > 0\) AND "books"\."deleted_at" IS NULL`).
			WithArgs(int64(12)).
			WillReturnRows(rentBookRows(12, 7))
		mock.ExpectQuery(`INSERT INTO "rents"`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
		mock.ExpectQuery(`SELECT .* FROM "books" WHERE id = \$1 AND "books"\."deleted_at" IS NULL AND "books"\."id" = \$2 ORDER BY "books"\."id" LIMIT \$3`).
			WithArgs(int64(12), int64(12), sqlmock.AnyArg()).
			WillReturnRows(rentBookRows(12, 7))
		mock.ExpectExec(`UPDATE "books" SET`).
			WillReturnError(errors.New("stock update failed"))
		mock.ExpectRollback()

		err := service.Create(context.Background(), &RentRequest{Books: []uint{12}, UserID: 42})

		require.EqualError(t, err, "stock update failed")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("rolls back when book reload fails", func(t *testing.T) {
		service, mock := setupRentService(t)
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT .* FROM "books" WHERE \(id = \$1 AND stock > 0\) AND "books"\."deleted_at" IS NULL`).
			WithArgs(int64(12)).
			WillReturnRows(rentBookRows(12, 7))
		mock.ExpectQuery(`INSERT INTO "rents"`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
		mock.ExpectQuery(`SELECT .* FROM "books" WHERE id = \$1 AND "books"\."deleted_at" IS NULL AND "books"\."id" = \$2 ORDER BY "books"\."id" LIMIT \$3`).
			WithArgs(int64(12), int64(12), sqlmock.AnyArg()).
			WillReturnError(errors.New("book reload failed"))
		mock.ExpectRollback()

		err := service.Create(context.Background(), &RentRequest{Books: []uint{12}, UserID: 42})

		require.EqualError(t, err, "book reload failed")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestServiceUpdate(t *testing.T) {
	t.Run("updates ordinary status", func(t *testing.T) {
		service, mock := setupRentService(t)
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "rents" SET`).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "rents" WHERE id = $1 AND "rents"."deleted_at" IS NULL ORDER BY "rents"."id" LIMIT $2`)).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(rentTestRows(1, 42, 12, "on_delivery"))
		expectRentAssociations(mock, 42, 12, 7)

		rent, err := service.Update(context.Background(), &UpdateRentRequest{Status: "on_delivery"}, 1)

		require.NoError(t, err)
		require.Equal(t, "on_delivery", rent.Status)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("restores book stock when returned", func(t *testing.T) {
		service, mock := setupRentService(t)
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "rents" SET`).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectQuery(`SELECT .* FROM "rents" WHERE id = \$1 AND "rents"\."deleted_at" IS NULL ORDER BY "rents"\."id" LIMIT \$2`).
			WithArgs(int64(1), sqlmock.AnyArg()).
			WillReturnRows(rentTestRows(1, 42, 12, "returned"))
		mock.ExpectQuery(`SELECT .* FROM "books" WHERE id = \$1 AND "books"\."deleted_at" IS NULL ORDER BY "books"\."id" LIMIT \$2`).
			WithArgs(int64(12), sqlmock.AnyArg()).
			WillReturnRows(rentBookRows(12, 7))
		mock.ExpectExec(`UPDATE "books" SET`).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "rents" WHERE id = $1 AND "rents"."deleted_at" IS NULL ORDER BY "rents"."id" LIMIT $2`)).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(rentTestRows(1, 42, 12, "returned"))
		expectRentAssociations(mock, 42, 12, 7)

		rent, err := service.Update(context.Background(), &UpdateRentRequest{Status: "returned"}, 1)

		require.NoError(t, err)
		require.Equal(t, "returned", rent.Status)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("restores book stock when cancelled", func(t *testing.T) {
		service, mock := setupRentService(t)
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "rents" SET`).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectQuery(`SELECT .* FROM "rents" WHERE id = \$1 AND "rents"\."deleted_at" IS NULL ORDER BY "rents"\."id" LIMIT \$2`).
			WithArgs(int64(1), sqlmock.AnyArg()).
			WillReturnRows(rentTestRows(1, 42, 12, "cancelled"))
		mock.ExpectQuery(`SELECT .* FROM "books" WHERE id = \$1 AND "books"\."deleted_at" IS NULL ORDER BY "books"\."id" LIMIT \$2`).
			WithArgs(int64(12), sqlmock.AnyArg()).
			WillReturnRows(rentBookRows(12, 7))
		mock.ExpectExec(`UPDATE "books" SET`).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "rents" WHERE id = $1 AND "rents"."deleted_at" IS NULL ORDER BY "rents"."id" LIMIT $2`)).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(rentTestRows(1, 42, 12, "cancelled"))
		expectRentAssociations(mock, 42, 12, 7)

		rent, err := service.Update(context.Background(), &UpdateRentRequest{Status: "cancelled"}, 1)

		require.NoError(t, err)
		require.Equal(t, "cancelled", rent.Status)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns update error", func(t *testing.T) {
		service, mock := setupRentService(t)
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "rents" SET`).
			WillReturnError(errors.New("update failed"))
		mock.ExpectRollback()

		_, err := service.Update(context.Background(), &UpdateRentRequest{Status: "rented"}, 1)

		require.EqualError(t, err, "update failed")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("rejects already returned or missing rent", func(t *testing.T) {
		service, mock := setupRentService(t)
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "rents" SET`).
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectRollback()

		_, err := service.Update(context.Background(), &UpdateRentRequest{Status: "returned"}, 1)

		require.EqualError(t, err, "rent already returned or not found")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns rent lookup error while restoring stock", func(t *testing.T) {
		service, mock := setupRentService(t)
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "rents" SET`).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectQuery(`SELECT .* FROM "rents" WHERE id = \$1 AND "rents"\."deleted_at" IS NULL ORDER BY "rents"\."id" LIMIT \$2`).
			WithArgs(int64(1), sqlmock.AnyArg()).
			WillReturnError(errors.New("rent lookup failed"))
		mock.ExpectRollback()

		_, err := service.Update(context.Background(), &UpdateRentRequest{Status: "returned"}, 1)

		require.EqualError(t, err, "rent lookup failed")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns book lookup error while restoring stock", func(t *testing.T) {
		service, mock := setupRentService(t)
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "rents" SET`).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectQuery(`SELECT .* FROM "rents" WHERE id = \$1 AND "rents"\."deleted_at" IS NULL ORDER BY "rents"\."id" LIMIT \$2`).
			WithArgs(int64(1), sqlmock.AnyArg()).
			WillReturnRows(rentTestRows(1, 42, 12, "returned"))
		mock.ExpectQuery(`SELECT .* FROM "books" WHERE id = \$1 AND "books"\."deleted_at" IS NULL ORDER BY "books"\."id" LIMIT \$2`).
			WithArgs(int64(12), sqlmock.AnyArg()).
			WillReturnError(errors.New("book lookup failed"))
		mock.ExpectRollback()

		_, err := service.Update(context.Background(), &UpdateRentRequest{Status: "returned"}, 1)

		require.EqualError(t, err, "book lookup failed")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns stock update error", func(t *testing.T) {
		service, mock := setupRentService(t)
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "rents" SET`).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectQuery(`SELECT .* FROM "rents" WHERE id = \$1 AND "rents"\."deleted_at" IS NULL ORDER BY "rents"\."id" LIMIT \$2`).
			WithArgs(int64(1), sqlmock.AnyArg()).
			WillReturnRows(rentTestRows(1, 42, 12, "returned"))
		mock.ExpectQuery(`SELECT .* FROM "books" WHERE id = \$1 AND "books"\."deleted_at" IS NULL ORDER BY "books"\."id" LIMIT \$2`).
			WithArgs(int64(12), sqlmock.AnyArg()).
			WillReturnRows(rentBookRows(12, 7))
		mock.ExpectExec(`UPDATE "books" SET`).
			WillReturnError(errors.New("stock update failed"))
		mock.ExpectRollback()

		_, err := service.Update(context.Background(), &UpdateRentRequest{Status: "returned"}, 1)

		require.EqualError(t, err, "stock update failed")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns final reload error", func(t *testing.T) {
		service, mock := setupRentService(t)
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "rents" SET`).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "rents" WHERE id = $1 AND "rents"."deleted_at" IS NULL ORDER BY "rents"."id" LIMIT $2`)).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(errors.New("reload failed"))

		_, err := service.Update(context.Background(), &UpdateRentRequest{Status: "rented"}, 1)

		require.EqualError(t, err, "reload failed")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestServiceDelete(t *testing.T) {
	t.Run("deletes returned rent", func(t *testing.T) {
		service, mock := setupRentService(t)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "rents" WHERE id = $1 AND "rents"."deleted_at" IS NULL ORDER BY "rents"."id" LIMIT $2`)).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(rentTestRows(1, 42, 12, "returned"))
		expectRentAssociations(mock, 42, 12, 7)
		mock.ExpectExec(`UPDATE "rents" SET "deleted_at"=\$1 WHERE "rents"\."id" = \$2 AND "rents"\."deleted_at" IS NULL`).
			WithArgs(sqlmock.AnyArg(), int64(1)).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := service.Delete(context.Background(), 1)

		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	for _, status := range []string{"pending", "on_delivery", "rented"} {
		t.Run("rejects status "+status, func(t *testing.T) {
			service, mock := setupRentService(t)
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "rents" WHERE id = $1 AND "rents"."deleted_at" IS NULL ORDER BY "rents"."id" LIMIT $2`)).
				WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
				WillReturnRows(rentTestRows(1, 42, 12, status))
			expectRentAssociations(mock, 42, 12, 7)

			err := service.Delete(context.Background(), 1)

			require.EqualError(t, err, "rent status must be returned or cancelled")
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}

	t.Run("returns lookup error", func(t *testing.T) {
		service, mock := setupRentService(t)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "rents" WHERE id = $1 AND "rents"."deleted_at" IS NULL ORDER BY "rents"."id" LIMIT $2`)).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(errors.New("rent not found"))

		err := service.Delete(context.Background(), 1)

		require.EqualError(t, err, "rent not found")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns delete error", func(t *testing.T) {
		service, mock := setupRentService(t)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "rents" WHERE id = $1 AND "rents"."deleted_at" IS NULL ORDER BY "rents"."id" LIMIT $2`)).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(rentTestRows(1, 42, 12, "cancelled"))
		expectRentAssociations(mock, 42, 12, 7)
		mock.ExpectExec(`UPDATE "rents" SET "deleted_at"=\$1 WHERE "rents"\."id" = \$2 AND "rents"\."deleted_at" IS NULL`).
			WithArgs(sqlmock.AnyArg(), int64(1)).
			WillReturnError(errors.New("delete failed"))

		err := service.Delete(context.Background(), 1)

		require.EqualError(t, err, "delete failed")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
