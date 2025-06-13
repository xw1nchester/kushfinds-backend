Применение миграций (способ 1):  
go run cmd/migrator/main.go  
(опциональные флаги: -migrations-path=/path/to/migrations/folder -dns=postgres://postgres:postgres@localhost:5432/kushfinds?sslmode=disable)

Применение миграций (способ 2):  
Требуется установить утилиту migrate (https://github.com/golang-migrate/migrate/releases)  
migrate -path ./migrations -database postgres://postgres:postgres@localhost:5432/kushfinds?sslmode=disable up

---

Запуск приложения (способ 1):  
go run cmd/kushfinds/main.go  
(опциональный флаг: -config=config/local.yml)

Запуск приложения (способ 2):  
export CONFIG_PATH=config/local.yml  
air