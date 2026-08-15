.PHONY: test-domain test-platform test-tools test-adapters test-cmd gate-coverage dev-token generate generate-client tidy test vet-integration up down lint fmt-check lint-web test-web test-web-ci \
        build-web security security-go security-npm loadtest ci ci-checks ci-integration tools-check

# Dependency-free domain + app tests only — no DB, no generated code needed.
# This is the T0 resume gate (HANDOFF.md): if this isn't green, nothing else matters.
test-domain:
	go test ./internal/.../domain/... ./internal/.../app/... -race -count=1

# The cross-cutting platform packages — the auth spine above all. Until T13.4
# these ran in NO gate at all (#138): test-domain's pattern is
# `./internal/.../domain/... ./internal/.../app/...`, which matches nothing
# under internal/platform/, and no other `ci` step reached them either. The
# single most security-critical package in the repo was the least verified.
#
# Deliberately a SEPARATE target rather than widening test-domain's glob:
# test-domain is defined (above, and in HANDOFF.md) as the product's pure
# domain+app resume gate, and internal/platform is neither domain nor app.
# test-tools already set the precedent for "real tests that aren't the pure
# core get their own target".
#
# One flat pattern covers all five packages with NO exclusions, which was
# checked rather than assumed — `go list -deps ./internal/platform/...`
# resolves nothing under internal/gen, so this needs no codegen, and
# internal/platform/pg only builds a pgxpool config (it opens no connection
# at test time, and ships no test files), so it needs no database. That makes
# this the same Docker-free, codegen-free gate class as test-domain, which is
# why it can sit next to it in `ci` rather than behind `generate`.
#   auth, auth/rs256, grpcrecovery  - real tests (stdlib + grpc + jwt only)
#   idgen, pg                       - no test files; compiled, not run
test-platform:
	go test ./internal/platform/... -race -count=1

# Build/CI tooling that carries real logic and therefore real tests
# (currently the vulngate security gate). Not part of test-domain, which is
# deliberately scoped to the product's pure core.
test-tools:
	go test ./tools/... -race -count=1

# The adapter ring: every internal/<context>/adapter/<x> package. Until T14.1
# these 22 packages ran in NO gate (#157) — including the five behavioural
# tests T13.1 wrote precisely because #146 had shipped without them, and the
# 216-line regression test PR #151 added to prove #146 stays fixed. Tests that
# no gate runs cannot fail a build, which is the failure mode one level out
# from "no tests at all".
#
# Depends on `generate` because SOME of these import the gitignored
# internal/gen (every adapter/postgres and adapter/grpcapi). The other 11 are
# codegen-free and could in principle sit in a codegen-free gate — but
# splitting them would take a hand-maintained list of which is which, and a
# hand-maintained list going stale inside one sprint is the exact defect
# T14.1 exists to remove (#157's own list missed
# internal/facilities/adapter/identity, created by its sprint sibling T13.3).
# One glob, no exclusions, no list.
#
# Docker-free: the testcontainers tests in these same packages sit behind
# //go:build integration and are not selected here. `make vet-integration`
# compiles those; `make ci-integration` runs them.
test-adapters: generate
	go test ./internal/.../adapter/... -race -count=1

# The composition root. cmd/server/main_test.go holds T13.5's three tests
# proving the server REFUSES to start without an auth verifier configured
# (#136) — a fail-closed property that is a property of this package's wiring
# and cannot be tested from internal/platform/auth. It ran in no gate either,
# and unlike the 22 it was missed by #157 and by the T14 plan's own
# enumeration too, both of which scanned `internal tools` and not `cmd`.
# Found by the check in this ticket rather than by another human re-reading
# the tree, which is the point of the check.
test-cmd: generate
	go test ./cmd/... -race -count=1

# The standing answer to "which packages hold tests that no gate executes?"
#
# Three consecutive sprints each closed that question for the packages named
# in one issue and left the next set open (T11 build tags -> T12
# internal/platform -> T13's 22 adapter packages; docs/process/t13-retro.md
# finding 2). Every fix was a glob written from a list, and every list was
# stale before its sprint ended. This target asks the GENERAL question
# instead, and both sides of it are computed at run time:
#
#   side A  every package holding a `func Test`, from a scan of the tree
#   side B  every package a `go test` reachable from `ci-checks` executes,
#           by parsing THIS Makefile and expanding its patterns with `go list`
#
# Side B being derived from the Makefile rather than copied into the tool is
# what stops the check from desynchronising when a target's pattern changes.
# There is no package list anywhere in the tool — that is the whole design
# constraint, and `go run` re-derives both sides on every invocation, so a
# package created by a future ticket is covered with no edit here.
#
# It exits non-zero AND names the offending packages: whoever trips this
# should not have to re-derive the analysis to act on it. Fix a failure by
# widening a `go test` pattern in a target reachable from `ci-checks` — never
# by adding an exclusion to the tool.
gate-coverage:
	go run ./cmd/gatecoverage

# Mints a local-development bearer token against the committed dev key
# fixture in dev/auth/ (T14.9, issue #160), so an authenticated endpoint can
# actually be called locally:
#
#   TOKEN=$$(make -s dev-token)
#   curl -H "Authorization: Bearer $$TOKEN" localhost:8080/v1/recurring-hire-templates
#
# `@` and the banner-on-stderr split inside cmd/devtoken are what make that
# capture work: stdout carries the token and nothing else.
#
# This is NOT an auth bypass and does not weaken the deployed path. It signs
# with a keypair committed to this repository — public, therefore worthless —
# and cmd/server only ever trusts the JWKS that AUTH_JWKS_FILE names, which
# in a deployment is a real provider's. `make up` mounts the fixture; nothing
# else does. See dev/auth/README.md.
dev-token:
	@go run ./cmd/devtoken

# buf (proto -> gRPC/gateway/OpenAPI) + sqlc (SQL -> typed Go) into internal/gen
# (gitignored, regenerate locally — see CLAUDE.md).
generate:
	buf generate
	sqlc generate

# The typed TypeScript client for the Vue app, from the same OpenAPI output.
# Runs `make generate` itself first (see web/scripts/generate-client.mjs), so
# this is the one command that brings ALL generated code up to date.
generate-client:
	@test -d web/node_modules || (echo "web/node_modules missing — running npm ci first"; npm --prefix web ci)
	npm --prefix web run generate:client

tidy:
	go mod tidy

# Full suite: everything, including packages that depend on generated code
# and the testcontainers-based concurrency integration test (T4), which
# needs Docker. Requires `make generate` to have been run first.
test:
	mkdir -p build
	gotestsum --junitfile build/junit.xml -- -race -tags=integration -coverprofile=build/coverage.out ./...

# Type-checks the //go:build integration files WITHOUT running them, so they
# stop being invisible to every gate a machine without Docker can run.
# `make test` and `make ci-integration` are the only other targets that compile
# these files, and both are hard-gated on a Docker daemon — so on a machine
# without one (which is most of them, and every agent session so far) a broken
# integration-tagged file reported green everywhere until a human happened to
# notice. 11 files across 4 contexts (socialplay 5, payments 3, competitions 2,
# booking 1) sit behind that tag; booking's concurrency test broke twice in T11
# for exactly this reason. See docs/process/t11-retro.md finding 2.
#
# Depends on `generate` deliberately: internal/gen/** is gitignored (see
# CLAUDE.md gotchas), so on a clean checkout the bare `go vet -tags=integration
# ./...` fails with "no required module provides package .../internal/gen/..."
# — a missing-codegen error that has nothing to do with build tags. Do NOT
# "fix" that by dropping -tags or narrowing the package pattern; the
# prerequisite is the fix. No Docker needed: vet compiles, it does not run.
vet-integration: generate
	go vet -tags=integration ./...

# `make up` now builds the web image too, which needs the generated TS
# client present on the host (see web/Dockerfile's guard).
up: generate-client
	docker compose up --build -d

down:
	docker compose down -v

lint:
	golangci-lint run ./...

# The formatting gate. Added T14.2; before it `make ci` had NO formatting
# check at all (`grep -n gofmt Makefile Jenkinsfile` returned nothing), so
# gofmt was enforced by review convention only. That is how two lines of
# struct-field alignment in internal/socialplay/domain/game.go survived nine
# PRs and three reviews which each independently re-observed them and
# correctly declined to fix an unrelated file inside their own scoped diff
# (#165). Every refusal was right; the outcome was still that nothing
# happened, because no gate owned the repository.
#
# This is ADDITIVE, not a duplicate of `lint`: .golangci.yml runs
# `default: none` plus errcheck/govet/ineffassign/staticcheck/unused and
# declares no `formatters` section, so `golangci-lint run ./...` reported
# "0 issues" on that exact unformatted file. Verified in T14.2's PR.
#
# `gofmt -l` prints the paths of files whose formatting differs from gofmt's
# and exits 0 either way, so the non-empty OUTPUT is the failure signal, not
# the exit code. A bare `gofmt -l ./...` as a recipe line would therefore
# pass silently on a mis-formatted tree — hence the capture-and-test below.
# Deliberately `-l` (list) and never `-w` (write): a gate that rewrites the
# agent's checked-out source reports green precisely by destroying the
# evidence it exists to surface, and leaves the workspace dirty.
#
# `set -e` matters: without it, a gofmt invocation that dies (e.g. a file
# with a syntax error) yields empty output and the gate would PASS on a tree
# it never actually checked.
#
# Scope is the Go tree: ./internal ./cmd ./tools. That INCLUDES the
# gitignored internal/gen, deliberately — which is also why this target sits
# AFTER `generate` in ci-checks rather than before it, so the set of files
# checked is identical on a clean checkout and a warm one instead of
# silently depending on whether codegen had been run. All 47 generated files
# are gofmt-clean today (buf's and sqlc's Go plugins format their output). If
# a future plugin ever emits unformatted code, CLAUDE.md rule 6 forbids
# hand-editing it and the fix is to narrow the path list here — not to drop
# the gate.
fmt-check:
	@set -e; \
	unformatted="$$(gofmt -l ./internal ./cmd ./tools)"; \
	if [ -n "$$unformatted" ]; then \
	  echo "fmt-check: FAIL — these files are not gofmt-formatted:"; \
	  for f in $$unformatted; do echo "    $$f"; done; \
	  echo; \
	  echo "Inspect with 'gofmt -d <file>', fix with 'gofmt -w <file>', then commit."; \
	  exit 1; \
	fi; \
	echo "fmt-check: OK — gofmt clean across ./internal ./cmd ./tools."

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
# the same thing. As of T13.4 that is true BY CONSTRUCTION rather than by
# discipline: the pipeline calls `make ci-checks` as a single step, so a
# check added here reaches CI with no Jenkinsfile edit. It had NOT been true
# by discipline — see ci-checks below for the step CI was silently missing.
#
# Both targets deliberately EXCLUDE *running* the Docker-dependent
# integration tests — but `vet-integration` still type-checks them here, so a
# broken integration-tagged file fails this gate. See `make ci-integration`
# below, and the Jenkinsfile, which skips that stage on the same condition.

# Every gating check EXCEPT the vulnerability scan. Split out of `ci` in
# T13.4 so the Jenkinsfile can invoke this gate as ONE command instead of
# re-listing its steps stage by stage — that re-listing is what #129 records,
# and it had already drifted: `vet-integration` has been in `make ci` since
# T12.1 and appeared nowhere in the pipeline, so the check that closes the
# integration-tag hole never actually reached CI.
#
# Why the scan is NOT in here: the Jenkinsfile's Security stage is strictly
# stronger than the `security` target below. It records govulncheck's exit
# code to build/govulncheck.exit and gates on THAT, whereas this Makefile
# gates on the report file merely existing — and a scan that dies partway
# (unreachable vuln.go.dev on an egress-restricted agent) still leaves a
# well-formed, non-empty, zero-finding JSON file behind. PR #95's review
# caught exactly that. Folding the scan into the command Jenkins calls would
# have downgraded CI's gate to the weaker check and hard-failed the whole
# build on any agent that cannot reach vuln.go.dev, which ADR-0011 section 3
# forbids ("findings UNKNOWN, never no findings"). So: Jenkins runs
# ci-checks, then its own scan. Local `make ci` still runs both, unchanged.
#
# `fmt-check` is placed after `generate` on purpose — see that target's
# comment: it covers ./internal, which contains the gitignored internal/gen,
# so running it before codegen would check a different (smaller) set of files
# on a clean checkout than on a warm one.
# `gate-coverage` is placed AFTER every test target on purpose. It is a
# meta-check on the gate's own shape, not a test — running it last means its
# report describes the set of targets that just ran, and a genuine test
# failure surfaces before the structural complaint about coverage does.
ci-checks: generate tidy fmt-check lint test-domain test-platform vet-integration test-tools test-adapters test-cmd gate-coverage generate-client lint-web test-web build-web
	go build ./...

# The full local gate: every check CI runs, plus the vulnerability scan.
ci: ci-checks
	$(MAKE) security
	@echo
	@echo "make ci: OK — lint, unit tests, codegen, build, and the security gate all passed."
	@echo "The //go:build integration files ARE compiled here (vet-integration), but NOT executed."
	@echo "Run 'make ci-integration' on a machine with a Docker daemon to actually execute them."

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
