.PHONY: build build-linux docker-build docker-up docker-down clean

build:
	@mkdir -p bin
	@go build -o bin/calmnews ./cmd/calmnews

build-linux:
	@mkdir -p bin
	@GOOS=linux GOARCH=amd64 go build -o bin/calmnews ./cmd/calmnews

docker-build:
	@docker build -t calmnews:latest .

docker-up:
	@docker compose up -d

docker-down:
	@docker compose down

clean:
	@rm -f bin/calmnews

