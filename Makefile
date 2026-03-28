APP_NAME := meshchat-server

.PHONY: fmt build run compose-up compose-down

fmt:
	gofmt -w $$(find . -name '*.go' | sort)

build:
	go build -o $(APP_NAME) ./cmd/server

run:
	go run ./cmd/server

compose-up:
	cd docker-compose && docker compose up --build -d

compose-down:
	cd docker-compose && docker compose down

