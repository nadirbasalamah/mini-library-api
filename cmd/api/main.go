package main

import (
	"context"
	"errors"
	"fmt"
	"go-digital-library-fiber/api"
	"go-digital-library-fiber/api/middlewares"
	"go-digital-library-fiber/db/drivers"
	"go-digital-library-fiber/pkg/constant"
	"go-digital-library-fiber/pkg/fileupload"
	"go-digital-library-fiber/pkg/utils"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

var app *fiber.App
var repository *gorm.DB

func main() {
	dbConfig := drivers.DBConfig{
		Username: utils.GetConfig(constant.DB_USERNAME),
		Password: utils.GetConfig(constant.DB_PASSWORD),
		Database: utils.GetConfig(constant.DB_NAME),
		Host:     utils.GetConfig(constant.DB_HOST),
		Port:     utils.GetConfig(constant.DB_PORT),
	}

	cloudinaryConfig := fileupload.CloudinaryConfig{
		CloudinaryURL: utils.GetConfig(constant.CLOUDINARY_URL),
	}

	expireDuration, err := strconv.Atoi(utils.GetConfig(constant.JWT_EXPIRE_DURATION))

	if err != nil {
		log.Fatalf("error when parsing expire duration: %v\n", err)
	}

	jwtConfig := middlewares.JWTConfig{
		SecretKey:       utils.GetConfig(constant.JWT_SECRET_KEY),
		ExpiresDuration: expireDuration,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {

		var cloudinary = cloudinaryConfig.InitCloudinary()

		repository = dbConfig.InitDB()
		app = api.NewApp(repository, cloudinary, jwtConfig)

		drivers.MigrateDB(repository)

		appPort := fmt.Sprintf(":%s", utils.GetConfig(constant.PORT))

		log.Printf("Server starting on port %s...\n", appPort)

		if err := app.Listen(appPort); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("failed to start server %v\n", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutdown signal received, starting graceful cleanup...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	if err := drivers.CloseDB(repository); err != nil {
		log.Fatalf("error occurred when database closed: %v", err)
	}
	log.Println("Server gracefully stopped")

}
