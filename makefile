.PHONY: generate
generate:
	sqlc generate

.PHONY: run
run:
	go run *.go

.PHONY: build
build:
	go build -o main *.go