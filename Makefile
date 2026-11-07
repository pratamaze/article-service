include .env
export

PSQL = docker compose exec -T db psql -U $(POSTGRES_USER) -d $(POSTGRES_DB) -v ON_ERROR_STOP=1

.PHONY: up down psql run test

up:
	docker compose up -d

down:
	docker compose down

psql:
	docker compose exec db psql -U $(POSTGRES_USER) -d $(POSTGRES_DB)

run:
	go run ./cmd/api

test:
	go test ./...

migrate-up:
	@for f in $$(ls migrations/*.up.sql | sort); do echo "-> $$f"; $(PSQL) < $$f || exit 1; done

migrate-down:
	@for f in $$(ls migrations/*.down.sql | sort -r); do echo "-> $$f"; $(PSQL) < $$f || exit 1; done

indexes-up:
	$(PSQL) < migrations/0002_create_indexes.up.sql

indexes-down:
	$(PSQL) < migrations/0002_create_indexes.down.sql

sanity:
	$(PSQL) < scripts/sanity_schema.sql