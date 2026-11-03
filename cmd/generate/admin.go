package main

import (
	"errors"
	"go-digital-library-fiber/db/drivers"
	"go-digital-library-fiber/db/models"
	"go-digital-library-fiber/pkg/constant"
	"go-digital-library-fiber/pkg/utils"
	"log"
	"strconv"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	dbConfig := drivers.DBConfig{
		Username: utils.GetConfig(constant.DB_USERNAME),
		Password: utils.GetConfig(constant.DB_PASSWORD),
		Database: utils.GetConfig(constant.DB_NAME),
		Host:     utils.GetConfig(constant.DB_HOST),
		Port:     utils.GetConfig(constant.DB_PORT),
	}

	repository := dbConfig.InitDB()

	provinceId, errProv := strconv.Atoi(utils.GetConfig(constant.ADMIN_PROVINCE_ID))
	cityId, errCity := strconv.Atoi(utils.GetConfig(constant.ADMIN_CITY_ID))
	districtId, errDis := strconv.Atoi(utils.GetConfig(constant.ADMIN_DISTRICT_ID))

	if err := errors.Join(errProv, errCity, errDis); err != nil {
		log.Fatalf("failed to read admin address configurations: %v\n", err)
	}

	password, err := bcrypt.GenerateFromPassword([]byte(utils.GetConfig(constant.ADMIN_PASSWORD)), bcrypt.DefaultCost)

	if err != nil {
		log.Fatalf("failed to create admin password: %v\n", err)
	}

	record := models.User{
		Username:   utils.GetConfig(constant.ADMIN_USERNAME),
		Email:      utils.GetConfig(constant.ADMIN_EMAIL),
		Password:   string(password),
		Address:    utils.GetConfig(constant.ADMIN_ADDRESS),
		ProvinceID: uint(provinceId),
		CityID:     uint(cityId),
		DistrictID: uint(districtId),
		Role:       models.Admin,
	}

	if err := repository.Create(&record).Error; err != nil {
		log.Fatalf("failed to create admin: %v\n", err)
	}

	log.Println("admin created successfully")
}
