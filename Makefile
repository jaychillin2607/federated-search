.PHONY: build run-server run-indexer seed test docker-up docker-down

build:
	go build -o bin/server  ./cmd/server
	go build -o bin/indexer ./cmd/indexer
	go build -o bin/seeder  ./cmd/seeder

run-server:  ; go run ./cmd/server
run-indexer: ; go run ./cmd/indexer
seed:        ; go run ./cmd/seeder
test:        ; go test ./... -race -count=1
docker-up:   ; docker compose up --build
docker-down: ; docker compose down -v
