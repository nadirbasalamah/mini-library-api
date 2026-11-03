package main

import (
	"go-digital-library-fiber/db/drivers"
	"go-digital-library-fiber/pkg/constant"
	"go-digital-library-fiber/pkg/utils"
	"log"

	"gorm.io/gorm"
)

func createTypeIfNotExists(db *gorm.DB, typeName, createSQL string) error {
	var exists int
	if err := db.Raw("SELECT 1 FROM pg_type WHERE typname = ?", typeName).Scan(&exists).Error; err != nil {
		return err
	}

	if exists == 1 {
		log.Printf("type %q already exists, skipping\n", typeName)
		return nil
	}

	if err := db.Exec(createSQL).Error; err != nil {
		return err
	}

	log.Printf("type %q created successfully\n", typeName)
	return nil
}

func main() {
	dbConfig := drivers.DBConfig{
		Username: utils.GetConfig(constant.DB_USERNAME),
		Password: utils.GetConfig(constant.DB_PASSWORD),
		Database: utils.GetConfig(constant.DB_NAME),
		Host:     utils.GetConfig(constant.DB_HOST),
		Port:     utils.GetConfig(constant.DB_PORT),
	}

	repository := dbConfig.InitDB()

	roleTypeSQL := "CREATE TYPE role AS ENUM ('user', 'admin')"
	if err := createTypeIfNotExists(repository, "role", roleTypeSQL); err != nil {
		log.Fatalf("failed to create role type: %v\n", err)
	}

	rentStatusTypeSQL := "CREATE TYPE rent_status AS ENUM ('pending','on_delivery','rented','returned','cancelled')"
	if err := createTypeIfNotExists(repository, "rent_status", rentStatusTypeSQL); err != nil {
		log.Fatalf("failed to create rent_status type: %v\n", err)
	}
}
