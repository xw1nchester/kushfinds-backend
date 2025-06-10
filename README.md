Применение миграций:  
migrate -path ./migrations -database postgres://postgres:postgres@localhost:5432/kushfinds?sslmode=disable up

Запуск приложения (способ 1):  
go run cmd/kushfinds/main.go -config=config/local.yml

Запуск приложения (способ 2):  
air