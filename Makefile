.PHONY: test-domain test-tools generate generate-client tidy test up down lint lint-web test-web test-web-ci \
        build-web security security-go security-npm loadtest ci ci-integration tools-check

# Dependency-free domain + app tests only — no DB, no generated code needed.
# This is the T0 resume gate (HANDOFF.md): if this isn't green, nothing else matters.
test-domain:
	go test ./internal/.../domain/... ./internal/.../app/... -race -count=1

# Build/CI tooling that carries real logic and therefore real tests
# (currently the vulngate security gate). Not part of test-domain, which is
# deliberately scoped to the product's pure core.
test-tools:
	go test ./tools/... -race -count=1

# buf (proto -> gRPC/gateway/OpenAPI) + sqlc (SQL -> typed Go) into internal/gen
# (gitignored, regenerate locally — see CLAUDE.md).
generate:
	buf generate
	sqlc generate

# The typed TypeScript client for the Vue app, from the same OpenAPI output.
# Runs `make generate` itself first (see web/scripts/generate-client.mjs), so
# this is the one command that brings ALL generated code up to date.
generate-client:
	npm --prefix web run generate:client

tidy:
	go mod tidy

# Full suite: everything, including packages that depend on generated code
# and the testcontainers-based concurrency integration test (T4), which
# needs Docker. Requires `make generate` to have been run first.
test:
	mkdir -p build
	gotestsum --junitfile build/junit.xml -- -race -tags=integration -coverprofile=build/coverage.out ./...

# `make up` now builds the web image too, which needs the generated TS
# client present on the host (see web/Dockerfile's guard).
up: generate-client
	docker compose up --build -d

down:
	docker compose down -v

lint:
	golangci-lint run ./...

# The web equivalent of `make lint`. web/package.json has no eslint setup,
# so there is no `lint` script to call; `type-check` (vue-tsc) is the real
# static-analysis gate the web app has, and it catches genuine defects —
# it is what surfaced the T9.2 fixture drift SCRUM-6 fixed. Adding eslint
# is a reasonable follow-up but was out of scope for the CI ticket.
lint-web:
	npm --prefix web run type-check

test-web:
	npm --prefix web run test

# Same tests, plus a JUnit XML report for Jenkins to publish.
test-web-ci:
	mkdir -p build
	npm --prefix web run test:ci

build-web:
	npm --prefix web run build

# --- Security scanning ----------------------------------------------------
#
# Two scanners feed one gate (cmd/vulngate): new HIGH/CRITICAL findings fail
# the build, explicitly baselined ones do not. See security/vuln-baseline.json.

# govulncheck needs to reach the Go vulnerability database at vuln.go.dev.
# SKIP_GOVULNCHECK=1 is an escape hatch for an offline or egress-restricted
# machine. It is deliberately loud and deliberately NOT set in CI: a
# security scan that silently no-ops is worse than none, because it reports
# green while checking nothing.
security-go:
	mkdir -p build
ifeq ($(SKIP_GOVULNCHECK),1)
	@echo "*** WARNING: SKIPPING the Go vulnerability scan (SKIP_GOVULNCHECK=1). ***"
	@echo "*** No Go dependency was checked. Never set this in CI.              ***"
	@rm -f build/govulncheck.json
else
	go run golang.org/x/vuln/cmd/govulncheck@latest -format json ./... > build/govulncheck.json
endif

security-npm:
	mkdir -p build
	# npm audit exits non-zero when it FINDS vulnerabilities, which is not a
	# tool failure — vulngate decides what fails the build, so the report is
	# captured either way. `|| true` guards the exit code, not the output.
	cd web && npm audit --json > ../build/npm-audit.json || true

# The -go report is passed only if the Go scan actually produced one. This
# is a shell test, not make's $(wildcard), on purpose: make can evaluate (and
# cache) $(wildcard) before this rule's prerequisites have created the file.
security: security-go security-npm
	@if [ -f build/govulncheck.json ]; then \
	  go run ./cmd/vulngate -baseline security/vuln-baseline.json \
	    -go build/govulncheck.json -npm build/npm-audit.json; \
	else \
	  echo "NOTE: no Go scan report present — gating on npm findings ONLY."; \
	  go run ./cmd/vulngate -baseline security/vuln-baseline.json \
	    -npm build/npm-audit.json; \
	fi

# --- Load testing ---------------------------------------------------------
#
# Opt-in, never a per-PR gate — see loadtest/README.md. Needs a running
# stack (`make up`) and the k6 binary.
loadtest:
	@command -v k6 >/dev/null 2>&1 || { \
	  echo "k6 not found. Install it (https://k6.io/docs/get-started/installation/) then re-run."; \
	  exit 1; }
	k6 run loadtest/booking-quote.js

# --- CI parity ------------------------------------------------------------
#
# `make ci` runs exactly the checks the Jenkins pipeline's gating stages
# run, in the same order, so "works on my machine" and "green in CI" mean
# the same thing. It deliberately EXCLUDES the Docker-dependent integration
# stage — see `make ci-integration` below, and the Jenkinsfile, which skips
# that stage on the same condition.
ci: generate tidy lint test-domain test-tools generate-client lint-web test-web build-web
	go build ./...
	$(MAKE) security
	@echo
	@echo "make ci: OK — lint, unit tests, codegen, build, and the security gate all passed."
	@echo "NOT covered here: the integration/concurrency tests (need Docker)."
	@echo "Run 'make ci-integration' on a machine with a Docker daemon."

# The Docker-dependent half: the testcontainers-backed integration and
# concurrency tests that actually prove the no-double-booking invariant.
ci-integration:
	@docker info >/dev/null 2>&1 || { \
	  echo "No Docker daemon reachable — the integration tests (testcontainers) cannot run here."; \
	  echo "This is the documented gap in CLAUDE.md's gotchas, not a new problem."; \
	  exit 1; }
	$(MAKE) test

# Reports which of the required tools are present, so a new machine finds
# out up front rather than three stages into a build.
tools-check:
	@for t in go buf sqlc gotestsum golangci-lint node npm; do \
	  if command -v $$t >/dev/null 2>&1; then echo "  ok      $$t"; \
	  else echo "  MISSING $$t"; fi; \
	done
	@if docker info >/dev/null 2>&1; then echo "  ok      docker (daemon reachable)"; \
	 else echo "  MISSING docker daemon (integration tests + make up unavailable)"; fi
	@if command -v k6 >/dev/null 2>&1; then echo "  ok      k6 (optional, load tests)"; \
	 else echo "  absent  k6 (optional — only needed for 'make loadtest')"; fi
