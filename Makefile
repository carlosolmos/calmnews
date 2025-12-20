.PHONY: build clean

build:
	@mkdir -p bin
	@go build -o bin/calmnews ./cmd/calmnews

clean:
	@rm -f bin/calmnews

