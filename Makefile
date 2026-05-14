# load .env so make commands can use your env vars
include .env
export

run:
	go run main.go

build:
	go build -o bin/server main.go

migrate-up:
	goose -dir sql/schema postgres "$(DB_URL)" up

migrate-down:
	goose -dir sql/schema postgres "$(DB_URL)" down

migrate-status:
	goose -dir sql/schema postgres "$(DB_URL)" status

generate:
	sqlc generate

test:
	go test ./...

dbcon:
	psql $(DB_URL)

run-log:
	go run main.go 2>&1 | tee logs/server.log