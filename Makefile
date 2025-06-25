dev:
	export CONFIG_PATH=config/local.yml && air

swagger:
	swag fmt && swag init -g cmd/kushfinds/main.go