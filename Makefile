.PHONY: test-domain generate tidy test up down lint

# Dependency-free domain + app tests only — no DB, no generated code needed.
# This is the T0 resume gate (HANDOFF.md): if this isn't green, nothing else matters.
test-domain:
	go test ./internal/.../domain/... ./internal/.../app/... -race -count=1

# buf (proto -> gRPC/gateway/OpenAPI) + sqlc (SQL -> typed Go) into internal/gen
# (gitignored, regenerate locally — see CLAUDE.md).
generate:
	buf generate
	sqlc generate

tidy:
	go mod tidy

# Full suite: everything, including packages that depend on generated code
# and the testcontainers-based concurrency integration test (T4), which
# needs Docker. Requires `make generate` to have been run first.
test:
	gotestsum --junitfile build/junit.xml -- -race -tags=integration -coverprofile=build/coverage.out ./...

up:
	docker compose up --build -d

down:
	docker compose down -v

lint:
	golangci-lint run ./...
