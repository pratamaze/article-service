include .env
export

PSQL = docker compose exec -T db psql -U $(POSTGRES_USER) -d $(POSTGRES_DB) -v ON_ERROR_STOP=1

# .PHONY: up down psql run test
.PHONY: up down psql run test migrate-up migrate-down indexes-up indexes-down sanity seed stats

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
# 	$(PSQL) < migrations/0002_create_indexes.up.sql
	$(PSQL) -c "SET maintenance_work_mem = '512MB'" -f - < migrations/0002_create_indexes.up.sql

indexes-down:
	$(PSQL) < migrations/0002_create_indexes.down.sql

sanity:
	$(PSQL) < scripts/sanity_schema.sql

seed:
	$(MAKE) indexes-down
	go run ./scripts/seed -n $(N) -reset
	$(MAKE) indexes-up
	$(PSQL) -c "VACUUM (ANALYZE) authors" -c "VACUUM (ANALYZE) articles"

stats:
	$(PSQL) -c "SELECT pg_size_pretty(pg_total_relation_size('articles')) AS tabel_plus_index" \
	        -c "SELECT relname, pg_size_pretty(pg_relation_size(oid)) AS ukuran FROM pg_class WHERE relname LIKE 'idx_%' OR relname = 'articles_pkey' ORDER BY relname"