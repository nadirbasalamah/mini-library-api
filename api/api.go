package api

import (
	"go-digital-library-fiber/api/handlers"
	"go-digital-library-fiber/api/middlewares"
	"go-digital-library-fiber/auth"
	"go-digital-library-fiber/books"
	"go-digital-library-fiber/categories"
	"go-digital-library-fiber/pkg/fileupload"
	"go-digital-library-fiber/pkg/rajaongkir"
	"go-digital-library-fiber/rents"
	"go-digital-library-fiber/users"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/helmet"

	zapMiddleware "github.com/gofiber/contrib/v3/zap"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func NewApp(repository *gorm.DB, cld *cloudinary.Cloudinary, jwtConfig middlewares.JWTConfig) *fiber.App {
	var (
		app       = fiber.New()
		uploader  = fileupload.InitUploader(cld)
		roService = rajaongkir.InitService()

		authService     = auth.New(repository)
		bookService     = books.New(repository)
		rentService     = rents.New(repository)
		userService     = users.New(repository)
		categoryService = categories.New(repository)

		authHandler     = handlers.NewAuth(authService, jwtConfig)
		bookHandler     = handlers.NewBooks(bookService, uploader)
		rentHandler     = handlers.NewRents(rentService, userService, roService)
		usersHandler    = handlers.NewUsers(userService, uploader)
		categoryHandler = handlers.NewCategories(categoryService)
	)

	logger, _ := zap.NewProduction()
	defer logger.Sync()

	logMiddleware := middlewares.LoggerConfig{Config: zapMiddleware.Config{
		Logger: logger,
	}}

	jwtMiddleware := jwtConfig.Init()

	app.Use(logMiddleware.Init())
	app.Use(helmet.New())

	authRoutes := app.Group("/api/v1/auth")

	authRoutes.Post("/register", middlewares.ValidateBody(&auth.RegisterRequest{}), authHandler.Register)
	authRoutes.Post("/login", middlewares.ValidateBody(&auth.LoginRequest{}), authHandler.Login)

	userRoutes := app.Group("/api/v1", jwtMiddleware, middlewares.VerifyToken())

	userRoutes.Get("/profile", usersHandler.GetProfile)
	userRoutes.Patch("/profile/edit", middlewares.ValidateBody(&users.EditProfileRequest{}), usersHandler.EditProfile)

	categoryRoutes := app.Group("/api/v1", jwtMiddleware, middlewares.VerifyToken())
	categoryRoutes.Get("/categories", categoryHandler.GetAll)
	categoryRoutes.Get("/categories/:id", categoryHandler.GetByID)
	categoryRoutes.Post("/categories", middlewares.VerifyAdmin(), middlewares.ValidateBody(&categories.CategoryRequest{}), categoryHandler.Create)
	categoryRoutes.Patch("/categories/:id", middlewares.VerifyAdmin(), middlewares.ValidateBody(&categories.CategoryRequest{}), categoryHandler.Update)
	categoryRoutes.Delete("/categories/:id", middlewares.VerifyAdmin(), categoryHandler.Delete)

	bookRoutes := app.Group("/api/v1", jwtMiddleware, middlewares.VerifyToken())
	bookRoutes.Get("/books", bookHandler.GetAll)
	bookRoutes.Get("/books/category/:id", bookHandler.GetByCategory)
	bookRoutes.Get("/books/:id", bookHandler.GetByID)
	bookRoutes.Post("/books", middlewares.VerifyAdmin(), middlewares.ValidateBody(&books.BookRequest{}), bookHandler.Create)
	bookRoutes.Patch("/books/:id", middlewares.VerifyAdmin(), middlewares.ValidateBody(&books.BookRequest{}), bookHandler.Update)
	bookRoutes.Delete("/books/:id", middlewares.VerifyAdmin(), bookHandler.Delete)

	rentRoutes := app.Group("/api/v1", jwtMiddleware, middlewares.VerifyToken())
	rentRoutes.Get("/rents", middlewares.VerifyAdmin(), rentHandler.GetAll)
	rentRoutes.Get("/rents/me", rentHandler.GetByUser)
	rentRoutes.Get("/rents/:id", rentHandler.GetByID)
	rentRoutes.Post("/rents", middlewares.ValidateBody(&rents.RentRequest{}), rentHandler.Create)
	rentRoutes.Patch("/rents/:id", middlewares.VerifyAdmin(), middlewares.ValidateBody(&rents.UpdateRentRequest{}), rentHandler.Update)
	rentRoutes.Delete("/rents/:id", middlewares.VerifyAdmin(), rentHandler.Delete)

	return app
}
