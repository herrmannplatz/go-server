.PHONY: generate
generate:
	sqlc generate

.PHONY: run
run:
	go run *.go

.PHONY: build
build:
	go build -o bin/main *.go

.PHONY: migration
migration:
	@if [ "$(name)" = "" ]; then \
		echo "Error: Migration name not provided. Usage: make migration name=your_migration_name"; \
		exit 1; \
	fi
	goose create $(name) -dir ./migrations sql
