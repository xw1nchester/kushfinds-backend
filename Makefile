migrate.up:
	migrate -path ./migrations -database "postgres://postgres:postgres@localhost:5432/kushfinds?sslmode=disable" up

migrate.up.%:
	migrate -path ./migrations -database "postgres://postgres:postgres@localhost:5432/kushfinds?sslmode=disable" up $*

migrate.down:
	migrate -path ./migrations -database "postgres://postgres:postgres@localhost:5432/kushfinds?sslmode=disable" down

migrate.down.%:
	migrate -path ./migrations -database "postgres://postgres:postgres@localhost:5432/kushfinds?sslmode=disable" down $*
	
dev:
	export CONFIG_PATH=config/local.yml && air

docker.up:
	docker compose up --build

swagger:
	swag fmt && swag init -g cmd/kushfinds/main.go

gen:
	go generate ./...

test.unit:
	go test ./internal/... -count=1 -v

test.integration:
	go test ./tests/... -count=1 -v

cover:
	go test -count=1 -race -coverprofile=coverage.out ./internal/...
	go tool cover -html=coverage.out
	rm coverage.out

minio:
	docker compose up -d minio