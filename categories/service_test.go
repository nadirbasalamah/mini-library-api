package categories

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"go-digital-library-fiber/pkg/utils"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupCategoryService(t *testing.T) (Service, sqlmock.Sqlmock) {
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

func categoryTestRows(id int64, name string) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "name", "description", "created_at", "updated_at", "deleted_at",
	}).AddRow(id, name, "Category description", nil, nil, nil)
}

func TestServiceGetAll(t *testing.T) {
	t.Run("returns paginated categories", func(t *testing.T) {
		service, mock := setupCategoryService(t)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "categories" WHERE "categories"."deleted_at" IS NULL`)).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
		mock.ExpectQuery(`SELECT \* FROM "categories".*name LIKE.*ORDER BY.*LIMIT`).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(categoryTestRows(1, "Fiction").AddRow(2, "History", "Category description", nil, nil, nil))

		result, err := service.GetAll(context.Background(), utils.Pagination{
			Page: 2, Limit: 1, Sort: "name asc", Search: "F", Keyword: "name",
		})

		require.NoError(t, err)
		require.EqualValues(t, 2, result.TotalRows)
		require.Equal(t, 2, result.TotalPages)
		require.Equal(t, 2, result.Page)
		require.Equal(t, 1, result.Limit)
		rows, ok := result.Rows.([]Category)
		require.True(t, ok)
		require.Len(t, rows, 2)
		require.Equal(t, "Fiction", rows[0].Name)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns query error", func(t *testing.T) {
		service, mock := setupCategoryService(t)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "categories" WHERE "categories"."deleted_at" IS NULL`)).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
		mock.ExpectQuery(`SELECT \* FROM "categories".*name LIKE`).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(errors.New("query failed"))

		_, err := service.GetAll(context.Background(), utils.Pagination{Keyword: "name"})

		require.EqualError(t, err, "query failed")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestServiceGetByID(t *testing.T) {
	t.Run("returns category", func(t *testing.T) {
		service, mock := setupCategoryService(t)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "categories" WHERE id = $1 AND "categories"."deleted_at" IS NULL ORDER BY "categories"."id" LIMIT $2`)).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(categoryTestRows(7, "Fiction"))

		category, err := service.GetByID(context.Background(), 7)

		require.NoError(t, err)
		require.Equal(t, uint(7), category.ID)
		require.Equal(t, "Fiction", category.Name)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns lookup error", func(t *testing.T) {
		service, mock := setupCategoryService(t)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "categories" WHERE id = $1 AND "categories"."deleted_at" IS NULL ORDER BY "categories"."id" LIMIT $2`)).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(errors.New("category not found"))

		_, err := service.GetByID(context.Background(), 7)

		require.EqualError(t, err, "category not found")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestServiceCreate(t *testing.T) {
	t.Run("creates and reloads category", func(t *testing.T) {
		service, mock := setupCategoryService(t)
		mock.ExpectQuery(`INSERT INTO "categories"`).
			WithArgs("Fiction", "Category description", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(7))
		mock.ExpectQuery(`SELECT .* FROM "categories" WHERE "categories"\."deleted_at" IS NULL ORDER BY "categories"\."id" DESC LIMIT \$1`).
			WithArgs(sqlmock.AnyArg()).
			WillReturnRows(categoryTestRows(7, "Fiction"))

		category, err := service.Create(context.Background(), &CategoryRequest{
			Name: "Fiction", Description: "Category description",
		})

		require.NoError(t, err)
		require.Equal(t, uint(7), category.ID)
		require.Equal(t, "Fiction", category.Name)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns insert error", func(t *testing.T) {
		service, mock := setupCategoryService(t)
		mock.ExpectQuery(`INSERT INTO "categories"`).
			WithArgs("Fiction", "Category description", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(errors.New("insert failed"))

		_, err := service.Create(context.Background(), &CategoryRequest{
			Name: "Fiction", Description: "Category description",
		})

		require.EqualError(t, err, "insert failed")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns reload error", func(t *testing.T) {
		service, mock := setupCategoryService(t)
		mock.ExpectQuery(`INSERT INTO "categories"`).
			WithArgs("Fiction", "Category description", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(7))
		mock.ExpectQuery(`SELECT .* FROM "categories" WHERE "categories"\."deleted_at" IS NULL ORDER BY "categories"\."id" DESC LIMIT \$1`).
			WithArgs(sqlmock.AnyArg()).
			WillReturnError(errors.New("reload failed"))

		_, err := service.Create(context.Background(), &CategoryRequest{
			Name: "Fiction", Description: "Category description",
		})

		require.EqualError(t, err, "reload failed")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestServiceUpdate(t *testing.T) {
	t.Run("updates and returns category", func(t *testing.T) {
		service, mock := setupCategoryService(t)
		mock.ExpectExec(`UPDATE "categories" SET`).
			WithArgs("Science Fiction", "Updated description", sqlmock.AnyArg(), int64(7)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "categories" WHERE id = $1 AND "categories"."deleted_at" IS NULL ORDER BY "categories"."id" LIMIT $2`)).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(categoryTestRows(7, "Science Fiction"))

		category, err := service.Update(context.Background(), &CategoryRequest{
			Name: "Science Fiction", Description: "Updated description",
		}, 7)

		require.NoError(t, err)
		require.Equal(t, uint(7), category.ID)
		require.Equal(t, "Science Fiction", category.Name)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns update error", func(t *testing.T) {
		service, mock := setupCategoryService(t)
		mock.ExpectExec(`UPDATE "categories" SET`).
			WithArgs("Science Fiction", "Updated description", sqlmock.AnyArg(), int64(7)).
			WillReturnError(errors.New("update failed"))

		_, err := service.Update(context.Background(), &CategoryRequest{
			Name: "Science Fiction", Description: "Updated description",
		}, 7)

		require.EqualError(t, err, "update failed")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns lookup error after update", func(t *testing.T) {
		service, mock := setupCategoryService(t)
		mock.ExpectExec(`UPDATE "categories" SET`).
			WithArgs("Science Fiction", "Updated description", sqlmock.AnyArg(), int64(7)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "categories" WHERE id = $1 AND "categories"."deleted_at" IS NULL ORDER BY "categories"."id" LIMIT $2`)).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(errors.New("category not found"))

		_, err := service.Update(context.Background(), &CategoryRequest{
			Name: "Science Fiction", Description: "Updated description",
		}, 7)

		require.EqualError(t, err, "category not found")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestServiceDelete(t *testing.T) {
	t.Run("soft deletes existing category", func(t *testing.T) {
		service, mock := setupCategoryService(t)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "categories" WHERE id = $1 AND "categories"."deleted_at" IS NULL ORDER BY "categories"."id" LIMIT $2`)).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(categoryTestRows(7, "Fiction"))
		mock.ExpectExec(`UPDATE "categories" SET "deleted_at"=\$1 WHERE "categories"\."id" = \$2 AND "categories"\."deleted_at" IS NULL`).
			WithArgs(sqlmock.AnyArg(), int64(7)).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := service.Delete(context.Background(), 7)

		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns lookup error", func(t *testing.T) {
		service, mock := setupCategoryService(t)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "categories" WHERE id = $1 AND "categories"."deleted_at" IS NULL ORDER BY "categories"."id" LIMIT $2`)).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(errors.New("category not found"))

		err := service.Delete(context.Background(), 7)

		require.EqualError(t, err, "category not found")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns soft-delete error", func(t *testing.T) {
		service, mock := setupCategoryService(t)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "categories" WHERE id = $1 AND "categories"."deleted_at" IS NULL ORDER BY "categories"."id" LIMIT $2`)).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(categoryTestRows(7, "Fiction"))
		mock.ExpectExec(`UPDATE "categories" SET "deleted_at"=\$1 WHERE "categories"\."id" = \$2 AND "categories"\."deleted_at" IS NULL`).
			WithArgs(sqlmock.AnyArg(), int64(7)).
			WillReturnError(errors.New("delete failed"))

		err := service.Delete(context.Background(), 7)

		require.EqualError(t, err, "delete failed")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
