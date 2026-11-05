.DEFAULT_GOAL := dev

-include .env
export

MIGRATE = migrate
MIGRATIONS_PATH = ./migrations

.PHONY: migrate-up migrate-down migrate-down-all migrate-create migrate-version swagger build dev

migrate-up:
	$(MIGRATE) -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" up

# Roll back a specified number of migrations (default: 1)
migrate-down:
	$(MIGRATE) -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" down $(or $(STEP),1)

migrate-down-all:
	$(MIGRATE) -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" down -all

migrate-create:
	$(MIGRATE) create -ext sql -dir $(MIGRATIONS_PATH) -seq $(NAME)


migrate-version:
	$(MIGRATE) -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" version

migrate-force:
	$(MIGRATE) -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" force $(VERSION)

swagger:
	swag init -g cmd/api/main.go -o docs --parseInternal

build:
	go build -o bin/app ./cmd/api/main.go

dev:
	air
