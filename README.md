Применение миграций (способ 1):  
go run cmd/migrator/main.go  
(опциональные флаги: -migrations-path=/path/to/migrations/folder -dns=postgres://postgres:postgres@localhost:5432/kushfinds?sslmode=disable)

Применение миграций (способ 2):  
Требуется установить утилиту migrate (https://github.com/golang-migrate/migrate/releases)  
migrate -path ./migrations -database postgres://postgres:postgres@localhost:5432/kushfinds?sslmode=disable up

---

Создайте файл с конфигурацией (config/local.yml) взяв за основу config/example.yml

---

Запуск приложения (способ 1):  
CONFIG_PATH=config/local.yml docker compose up  

Запуск приложения (способ 2):  
go run cmd/kushfinds/main.go -config=config/local.yml  

Запуск приложения (способ 3):  
export CONFIG_PATH=config/local.yml  
air  

---

Swagger:  
http://localhost:8080/swagger/index.html