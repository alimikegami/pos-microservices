.PHONY: test migrate-up migrate-down

migrate-up:
	migrate -path database/migration -database "postgresql://postgres:testpass@localhost:5433/testdb?sslmode=disable" up

migrate-down:
	migrate -path database/migration -database "postgresql://postgres:testpass@localhost:5433/testdb?sslmode=disable" down

test:
	docker-compose -f docker-compose.yml up -d
	@while ! docker-compose exec -T postgres pg_isready -U postgres -d testdb; do \
		sleep 1; \
	done
	migrate -path database/migration -database "postgresql://postgres:testpass@localhost:5433/testdb?sslmode=disable" up
	go test ./... -tags=integration || true
	docker-compose -f docker-compose.yml down