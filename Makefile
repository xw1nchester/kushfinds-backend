MIGRATIONS_DIR=./migrations
DB_URL=postgres://postgres:postgres@localhost:5432/kushfinds?sslmode=disable

# make migration name=create_users_table
migrate.create:
	migrate create -ext sql -dir $(MIGRATIONS_DIR) $(name)
	
migrate.up:
	migrate -path $(MIGRATIONS_DIR) -database $(DB_URL) up

migrate.up.%:
	migrate -path $(MIGRATIONS_DIR) -database $(DB_URL) up $*

migrate.down:
	migrate -path $(MIGRATIONS_DIR) -database $(DB_URL) down

migrate.down.%:
	migrate -path $(MIGRATIONS_DIR) -database $(DB_URL) down $*
	
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