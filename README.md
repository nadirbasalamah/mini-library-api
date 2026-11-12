# go-digital-library-fiber

Batteries-included Go Fiber Starter Project

## Key Features

- 🔐 Basic authentication features including registration and login
- 💾 Ready-to-use middlewares including authentication, logging and request validation
- 🗂️ Extendable application architecture
- 📦 Essential custom packages including:
  - 🌐 HTTP Client
  - 🤖 OpenRouter AI Client
  - 📁 File Upload with Cloudinary
  - 🚚 RajaOngkir API Client
  - ⚙️ App configuration with Viper

## 💿 Tech Stacks

- Fiber: web framework
- GORM: ORM library
- Validator: validation library
- Zap: logging
- Viper: configuration

## How to Use

1. Clone this repository.

2. Replace the module name in `go.mod`. You can use any name you want as long its easy to remember

3. Copy the sample configuration.

```sh
cp .env.example .env
```

4. Run the seed role command. This command is required to generate user roles in the database.

```sh
go run cmd/seed/role.go
```

5. Run the application.

```sh
go run cmd/api/main.go
```

For development, consider using `air` for hot reload. Learn more [here](https://github.com/air-verse/air)

## Additional Notes

Generate the admin account using this command. Ensure the credentials are defined in `.env`

```sh
go run cmd/generate/admin.go
```

## Testing

Run the complete Go test suite with:

```sh
go test ./...
```

The HTTP handler and route tests use in-process Fiber requests and local test doubles; they do not require a running database or external Cloudinary/RajaOngkir credentials.

Test the endpoints

Register

```sh
curl -X POST "http://localhost:1323/api/v1/auth/register" \
  -H "Content-Type: application/json" \
  -d '{"username":"cooluser","email":"cooluser@mail.com","password":"$P455w0rdz","address":"Somewhere","province_id":18,"city_id":391,"district_id":3894}'
```

Login

```sh
curl -X POST "http://localhost:1323/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"email":"atmin@mail.com","password":"!5VapccCxaifxZFa$"}'
```
