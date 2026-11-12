package books

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

func setupBooksService(t *testing.T) (Service, sqlmock.Sqlmock) {
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

func bookRows(id int64, title string) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "title", "description", "publisher", "year", "stock",
		"image_link", "image_id", "category_id", "created_at", "updated_at", "deleted_at",
	}).AddRow(id, title, "A novel", "Publisher", int64(1965), int64(3),
		"https://example.com/book.png", "book-image", int64(7), nil, nil, nil)
}

func categoryRows(id int64) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "name", "description", "created_at", "updated_at", "deleted_at",
	}).AddRow(id, "Fiction", "Fiction books", nil, nil, nil)
}

func expectCategoryPreload(mock sqlmock.Sqlmock, id int64) {
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "categories" WHERE "categories"."id" = $1`)).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(categoryRows(id))
}

func TestServiceGetByID(t *testing.T) {
	t.Run("returns book and category", func(t *testing.T) {
		service, mock := setupBooksService(t)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "books" WHERE id = $1 AND "books"."deleted_at" IS NULL ORDER BY "books"."id" LIMIT $2`)).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(bookRows(12, "Dune"))
		expectCategoryPreload(mock, 7)

		book, err := service.GetByID(context.Background(), 12)

		require.NoError(t, err)
		require.Equal(t, uint(12), book.ID)
		require.Equal(t, "Dune", book.Title)
		require.Equal(t, "Fiction", book.Category.Name)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns database error", func(t *testing.T) {
		service, mock := setupBooksService(t)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "books" WHERE id = $1 AND "books"."deleted_at" IS NULL ORDER BY "books"."id" LIMIT $2`)).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(errors.New("database unavailable"))

		_, err := service.GetByID(context.Background(), 12)

		require.EqualError(t, err, "database unavailable")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestServiceGetAll(t *testing.T) {
	t.Run("returns paginated books", func(t *testing.T) {
		service, mock := setupBooksService(t)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "books" WHERE "books"."deleted_at" IS NULL`)).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
		mock.ExpectQuery(`SELECT \* FROM "books".*title LIKE.*ORDER BY.*LIMIT.*OFFSET`).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(bookRows(12, "Dune").AddRow(13, "Foundation", "A novel", "Publisher", int64(1951), int64(2),
				"https://example.com/foundation.png", "foundation-image", int64(7), nil, nil, nil))
		expectCategoryPreload(mock, 7)

		result, err := service.GetAll(context.Background(), utils.Pagination{
			Page: 2, Limit: 1, Sort: "title asc", Search: "D", Keyword: "title",
		})

		require.NoError(t, err)
		require.EqualValues(t, 2, result.TotalRows)
		require.Equal(t, 2, result.TotalPages)
		require.Equal(t, 2, result.Page)
		require.Equal(t, 1, result.Limit)
		rows, ok := result.Rows.([]Book)
		require.True(t, ok)
		require.Len(t, rows, 2)
		require.Equal(t, "Dune", rows[0].Title)
		require.Equal(t, "Fiction", rows[0].Category.Name)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns query error", func(t *testing.T) {
		service, mock := setupBooksService(t)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "books" WHERE "books"."deleted_at" IS NULL`)).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
		mock.ExpectQuery(`SELECT \* FROM "books".*title LIKE`).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(errors.New("query failed"))

		_, err := service.GetAll(context.Background(), utils.Pagination{Keyword: "title"})

		require.EqualError(t, err, "query failed")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestServiceGetByCategory(t *testing.T) {
	t.Run("returns books filtered by category", func(t *testing.T) {
		service, mock := setupBooksService(t)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "books" WHERE category_id = $1 AND "books"."deleted_at" IS NULL`)).
			WithArgs(sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(`SELECT \* FROM "books".*title LIKE.*category_id =.*ORDER BY.*LIMIT`).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(bookRows(12, "Dune"))
		expectCategoryPreload(mock, 7)

		result, err := service.GetByCategory(context.Background(), utils.Pagination{
			Page: 1, Limit: 10, Sort: "title asc", Search: "Du", Keyword: "title",
		}, 7)

		require.NoError(t, err)
		require.EqualValues(t, 1, result.TotalRows)
		require.Equal(t, 1, result.TotalPages)
		rows, ok := result.Rows.([]Book)
		require.True(t, ok)
		require.Len(t, rows, 1)
		require.Equal(t, uint(12), rows[0].ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns query error", func(t *testing.T) {
		service, mock := setupBooksService(t)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "books" WHERE category_id = $1 AND "books"."deleted_at" IS NULL`)).
			WithArgs(sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
		mock.ExpectQuery(`SELECT \* FROM "books".*title LIKE`).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(errors.New("query failed"))

		_, err := service.GetByCategory(context.Background(), utils.Pagination{Keyword: "title"}, 7)

		require.EqualError(t, err, "query failed")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestServiceCreate(t *testing.T) {
	t.Run("creates book and loads category", func(t *testing.T) {
		service, mock := setupBooksService(t)
		mock.ExpectQuery(`INSERT INTO "books"`).
			WithArgs("Dune", "A novel", "Publisher", int64(1965), int64(3), "https://example.com/book.png",
				"book-image", int64(7), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(12))
		mock.ExpectQuery(`SELECT .* FROM "books" WHERE "books"\."deleted_at" IS NULL ORDER BY "books"\."id" DESC LIMIT \$1`).
			WithArgs(sqlmock.AnyArg()).
			WillReturnRows(bookRows(12, "Dune"))
		expectCategoryPreload(mock, 7)

		book, err := service.Create(context.Background(), &BookRequest{
			Title: "Dune", Description: "A novel", Publisher: "Publisher", Year: 1965,
			Stock: 3, ImageLink: "https://example.com/book.png", ImageID: "book-image", CategoryID: 7,
		})

		require.NoError(t, err)
		require.Equal(t, uint(12), book.ID)
		require.Equal(t, "Fiction", book.Category.Name)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns insert error", func(t *testing.T) {
		service, mock := setupBooksService(t)
		mock.ExpectQuery(`INSERT INTO "books"`).
			WithArgs("Dune", "A novel", "Publisher", int64(1965), int64(3), "", "", int64(7),
				sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(errors.New("insert failed"))

		_, err := service.Create(context.Background(), &BookRequest{
			Title: "Dune", Description: "A novel", Publisher: "Publisher", Year: 1965, Stock: 3, CategoryID: 7,
		})

		require.EqualError(t, err, "insert failed")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns reload error after insert", func(t *testing.T) {
		service, mock := setupBooksService(t)
		mock.ExpectQuery(`INSERT INTO "books"`).
			WithArgs("Dune", "A novel", "Publisher", int64(1965), int64(3), "", "", int64(7),
				sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(12))
		mock.ExpectQuery(`SELECT .* FROM "books" WHERE "books"\."deleted_at" IS NULL ORDER BY "books"\."id" DESC LIMIT \$1`).
			WithArgs(sqlmock.AnyArg()).
			WillReturnError(errors.New("reload failed"))

		_, err := service.Create(context.Background(), &BookRequest{
			Title: "Dune", Description: "A novel", Publisher: "Publisher", Year: 1965, Stock: 3, CategoryID: 7,
		})

		require.EqualError(t, err, "reload failed")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestServiceUpdate(t *testing.T) {
	t.Run("updates book and reloads category", func(t *testing.T) {
		service, mock := setupBooksService(t)
		mock.ExpectExec(`UPDATE "books" SET`).
			WithArgs("Dune Messiah", "Updated description", "Publisher", int64(1969), int64(4),
				"https://example.com/new.png", "new-image", int64(7), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "books" WHERE id = $1 AND "books"."deleted_at" IS NULL ORDER BY "books"."id" LIMIT $2`)).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(bookRows(12, "Dune Messiah"))
		expectCategoryPreload(mock, 7)

		book, err := service.Update(context.Background(), &BookRequest{
			Title: "Dune Messiah", Description: "Updated description", Publisher: "Publisher",
			Year: 1969, Stock: 4, ImageLink: "https://example.com/new.png", ImageID: "new-image", CategoryID: 7,
		}, 12)

		require.NoError(t, err)
		require.Equal(t, uint(12), book.ID)
		require.Equal(t, "Dune Messiah", book.Title)
		require.Equal(t, "Fiction", book.Category.Name)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns update error", func(t *testing.T) {
		service, mock := setupBooksService(t)
		mock.ExpectExec(`UPDATE "books" SET`).
			WithArgs("Dune Messiah", "Updated description", "Publisher", int64(1969), int64(4),
				int64(7), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(errors.New("update failed"))

		_, err := service.Update(context.Background(), &BookRequest{
			Title: "Dune Messiah", Description: "Updated description", Publisher: "Publisher",
			Year: 1969, Stock: 4, CategoryID: 7,
		}, 12)

		require.EqualError(t, err, "update failed")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns reload error after update", func(t *testing.T) {
		service, mock := setupBooksService(t)
		mock.ExpectExec(`UPDATE "books" SET`).
			WithArgs("Dune Messiah", "Updated description", "Publisher", int64(1969), int64(4),
				int64(7), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "books" WHERE id = $1 AND "books"."deleted_at" IS NULL ORDER BY "books"."id" LIMIT $2`)).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(errors.New("reload failed"))

		_, err := service.Update(context.Background(), &BookRequest{
			Title: "Dune Messiah", Description: "Updated description", Publisher: "Publisher",
			Year: 1969, Stock: 4, CategoryID: 7,
		}, 12)

		require.EqualError(t, err, "reload failed")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestServiceDelete(t *testing.T) {
	t.Run("soft deletes existing book", func(t *testing.T) {
		service, mock := setupBooksService(t)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "books" WHERE id = $1 AND "books"."deleted_at" IS NULL ORDER BY "books"."id" LIMIT $2`)).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(bookRows(12, "Dune"))
		expectCategoryPreload(mock, 7)
		mock.ExpectExec(`UPDATE "books" SET "deleted_at"=\$1 WHERE "books"."id" = \$2 AND "books"."deleted_at" IS NULL`).
			WithArgs(sqlmock.AnyArg(), int64(12)).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := service.Delete(context.Background(), 12)

		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns lookup error", func(t *testing.T) {
		service, mock := setupBooksService(t)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "books" WHERE id = $1 AND "books"."deleted_at" IS NULL ORDER BY "books"."id" LIMIT $2`)).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(errors.New("book not found"))

		err := service.Delete(context.Background(), 12)

		require.EqualError(t, err, "book not found")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns soft-delete error", func(t *testing.T) {
		service, mock := setupBooksService(t)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "books" WHERE id = $1 AND "books"."deleted_at" IS NULL ORDER BY "books"."id" LIMIT $2`)).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(bookRows(12, "Dune"))
		expectCategoryPreload(mock, 7)
		mock.ExpectExec(`UPDATE "books" SET "deleted_at"=\$1 WHERE "books"."id" = \$2 AND "books"."deleted_at" IS NULL`).
			WithArgs(sqlmock.AnyArg(), int64(12)).
			WillReturnError(errors.New("delete failed"))

		err := service.Delete(context.Background(), 12)

		require.EqualError(t, err, "delete failed")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
