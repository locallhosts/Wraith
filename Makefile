.PHONY: up down lint test build-go

up:
	docker compose up --build

down:
	docker compose down -v

lint:
	cd backend-go && go run . lint ../rules

test:
	cd backend-go && go test ./...

build-go:
	cd backend-go && go build -o wraith .
