package constant

const PORT = "PORT"

const (
	DB_HOST     = "DB_HOST"
	DB_PORT     = "DB_PORT"
	DB_USERNAME = "DB_USERNAME"
	DB_PASSWORD = "DB_PASSWORD"
	DB_NAME     = "DB_NAME"
)

const (
	JWT_EXPIRE_DURATION = "JWT_EXPIRE_DURATION"
	JWT_SECRET_KEY      = "JWT_SECRET_KEY"
)

const (
	ADMIN_USERNAME    = "ADMIN_USERNAME"
	ADMIN_EMAIL       = "ADMIN_EMAIL"
	ADMIN_PASSWORD    = "ADMIN_PASSWORD"
	ADMIN_ADDRESS     = "ADMIN_ADDRESS"
	ADMIN_PROVINCE_ID = "ADMIN_PROVINCE_ID"
	ADMIN_CITY_ID     = "ADMIN_CITY_ID"
	ADMIN_DISTRICT_ID = "ADMIN_DISTRICT_ID"
)

const (
	DISTRICT_ID = "DISTRICT_ID"
)

const (
	CLOUDINARY_URL     = "CLOUDINARY_URL"
	RAJAONGKIR_API_KEY = "RAJAONGKIR_API_KEY"
	AI_API_KEY         = "AI_API_KEY"
	AI_MODEL           = "AI_MODEL"
)

var ALLOWED_EXTENSIONS map[string]bool = map[string]bool{
	".png":  true,
	".jpg":  true,
	".jpeg": true,
}

const (
	RESOURCE_IMAGE = "image"
)

const (
	ERR_INVALID_TOKEN = "invalid token"
	ERR_INVALID_ID    = "invalid id"
)
