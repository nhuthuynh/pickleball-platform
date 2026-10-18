// Jenkins pipeline for the pickleball platform (SCRUM-6).
//
// ---------------------------------------------------------------------------
// THIS IS NOT THE PIPELINE THAT RUNS. `.github/workflows/ci.yml` IS.
// ---------------------------------------------------------------------------
// Added 2026-10-18. docs/technology-options.md picks Jenkins, and that
// decision stands for the eventual deployment pipeline — but this file has
// never executed, because it needs a controller, five plugins, a webhook and
// a GitHub credential that do not exist (see "STILL REQUIRED SERVER-SIDE"
// below, which has been true since the day it was written). Meanwhile this
// repository's pull requests showed ZERO checks for 66 sprints with a 20 KB
// pipeline sitting in the tree.
//
// So the per-PR gate now runs on GitHub Actions, which needs no server. Both
// files call the same make targets, which is what keeps them from drifting
// into two different gates — the Toolchain stage below calls
// `make tools-install` for exactly that reason. If a Jenkins controller ever
// appears, this file should still work; until then, treat Actions as the
// source of truth for "what CI runs" and this as the Jenkins port of it.
//
// Replaces the T0 bootstrap placeholder, which had never run: it assumed a
// Docker-capable agent, used `go install ...@latest` (unpinned), linted
// before generating the code being linted, and pointed at a golangci-lint
// module path that does not exist. All four are fixed below.
//
// ---------------------------------------------------------------------------
// AGENT ASSUMPTIONS — read before wiring this to a real Jenkins
// ---------------------------------------------------------------------------
// `agent any`, deliberately, NOT `agent { docker { ... } }`. The reason is
// that a Jenkins agent may not have a Docker daemon, and a docker-agent
// pipeline cannot even start there; this one runs every stage that does not
// intrinsically need Docker, and skips only the one that does, loudly.
//
// This comment used to justify the choice differently — "this project is
// developed in sandboxes that repeatedly turn out to have no Docker daemon".
// **That premise was false and was retired at T61**: the daemon was present
// the whole time and starts in about four seconds (CLAUDE.md's "Docker works
// here"). The decision survives its own rationale, which is why the rationale
// is corrected here rather than the decision reopened.
//
// The agent MUST provide:
//   - Go >= 1.26.9 on PATH (go.mod says 1.26.9 — four called net/http
//     HTTP/2 advisories are only fixed at go1.26.9; see the Dockerfile)
//   - Node >= 22.18 and npm on PATH (web/package.json "engines")
//   - git, make, curl
//
// The agent NEED NOT provide: buf, sqlc, gotestsum, golangci-lint,
// govulncheck, or the four protoc-gen-* plugins — the Toolchain stage
// installs all of them at pinned versions into a workspace-local GOPATH.
//
// The agent MAY provide a Docker daemon. If it does, the Integration tests
// stage runs the real testcontainers-backed concurrency suite. If it does
// not, that stage is SKIPPED and the build is marked UNSTABLE — never
// silently passed, because the no-double-booking invariant is the single
// most important thing this codebase asserts.
//
// ---------------------------------------------------------------------------
// STILL REQUIRED SERVER-SIDE (cannot be configured from this repo)
// ---------------------------------------------------------------------------
// This file is only half of "CI is wired". A Jenkins administrator must also:
//
//  1. Create a Multibranch Pipeline (or GitHub Organization) job pointing at
//     nhuthuynh/white-label, with "Discover pull requests from origin" and
//     "...from forks" enabled, so PR builds happen at all.
//  2. Install plugins this file uses: workflow-aggregator, git,
//     github-branch-source, junit, timestamper. (`warnError` needs
//     workflow-basic-steps >= 2.18.)
//  3. Add the GitHub webhook: in the repo's Settings > Webhooks, POST to
//     https://<jenkins>/github-webhook/ on "Pull requests" and "Pushes".
//     Equivalently, tick "GitHub hook trigger for GITScm polling" on the job
//     and let Jenkins register it with a credentialed GitHub App/PAT.
//  4. Provide a GitHub credential for checkout + status reporting, so PR
//     checks show up on the PR itself.
//  5. Optionally mark the branch protection rule on
//     claude/go-backend-pickleball-7up34j as requiring this check, which is
//     what actually turns the pipeline into a merge gate.
//
// NONE of the above was verifiable from this ticket — there is no Jenkins
// server reachable here. Everything in this file was checked by running the
// underlying commands directly (see the PR description for exactly which),
// not by observing a green Jenkins build. Per CLAUDE.md rule 10, treat the
// first real run as the actual verification.
//
// NO CREDENTIALS are referenced anywhere in this pipeline, by design (see
// the "Deploy" stage at the bottom, which is a documented stub rather than a
// half-working deploy).

pipeline {
    agent any

    options {
        timestamps()
        timeout(time: 45, unit: 'MINUTES')
        buildDiscarder(logRotator(numToKeepStr: '30'))
        // The repo is checked out explicitly in the Checkout stage so the
        // step is visible in the stage view rather than implicit.
        skipDefaultCheckout(true)
    }

    parameters {
        booleanParam(
            name: 'RUN_LOAD_TEST',
            defaultValue: false,
            description: 'Run the k6 load test. Off by default: it needs a full running stack and is not a per-PR gate — see loadtest/README.md.'
        )
        booleanParam(
            name: 'FAIL_ON_SKIPPED_INTEGRATION',
            defaultValue: false,
            description: 'Fail (rather than mark UNSTABLE) when no Docker daemon is available for the integration tests. Turn this ON for the agent that guards the shared branch.'
        )
    }

    triggers {
        // Nightly load-test run at 03:00 on the shared branch only. The
        // empty-string-for-no-schedule ternary is the standard multibranch
        // idiom, but it is one of the few things here that CANNOT be checked
        // without a live Jenkins (env.BRANCH_NAME is only populated by a
        // multibranch job). If your controller rejects the empty spec, drop
        // this block and set the schedule in the job configuration instead —
        // the Load test stage's `when` clause already gates on TimerTrigger,
        // so nothing else needs to change.
        cron(env.BRANCH_NAME == 'claude/go-backend-pickleball-7up34j' ? 'H 3 * * *' : '')
    }

    environment {
        GOFLAGS = '-mod=mod'
        // Workspace-local GOPATH so concurrent jobs on one agent cannot
        // fight over a shared module/binary cache.
        GOPATH = "${WORKSPACE}/.gopath"
        GOBIN = "${WORKSPACE}/.gopath/bin"
        PATH = "${WORKSPACE}/.gopath/bin:${PATH}"

        // The tool pins used to be duplicated here. They now live in
        // tools/versions.env, which `make tools-install` reads — so this
        // pipeline, the Actions workflow and a developer's machine install
        // the same versions from one file. The reasoning that put them here
        // is unchanged and is restated there: `@latest` in CI means a build
        // can break overnight with no commit behind it.
        GOVULNCHECK_VERSION = 'latest' // deliberately floating; see the Security stage
    }

    stages {

        stage('Checkout') {
            steps {
                checkout scm
                sh 'git log --oneline -1'
                sh 'mkdir -p build'
            }
        }

        stage('Toolchain') {
            steps {
                sh 'go version'
                sh 'node --version'
                sh 'npm --version'

                // One target, which a developer and the Actions workflow
                // also call, installing the pins from tools/versions.env.
                // This stage used to list eight `go install` lines, four of
                // them `@latest` — including the four codegen plugins
                // buf.gen.yaml invokes as LOCAL plugins (ADR-0003: the BSR
                // is not reachable everywhere), which therefore floated in
                // the one pipeline that pinned everything else.
                //
                // (The module path for the linter is
                // github.com/golangci/golangci-lint/v2 — the /v2 suffix is
                // required, and the T0 placeholder's
                // "github.com/golangci-lint/golangci-lint" does not exist.
                // That correction now lives in the Makefile target.)
                sh 'make tools-install'

                sh 'make tools-check'
            }
        }

        // node_modules must exist before the CI gate runs: `make ci-checks`
        // reaches generate-client, whose own guard would otherwise npm-ci for
        // us. Doing it explicitly here keeps it a clean, deterministic install
        // and keeps the dependency step visible in the stage view.
        stage('Web dependencies') {
            steps {
                sh 'npm --prefix web ci'
            }
        }

        // #129: this stage used to be four stages — Generate, Lint, Unit
        // tests, Build — each re-listing, by hand, a step that `make ci`
        // already listed. Two copies of one gate is one copy too many, and it
        // had already drifted: `make vet-integration` went into `make ci` in
        // T12.1 and was never added here, so the check that stops a broken
        // //go:build integration file from hiding was in the local gate and
        // NOT in CI. T13.4's `make test-platform` would have been the second
        // such step on the same day it was written.
        //
        // So the pipeline now calls the gate instead of reimplementing it.
        // `make ci-checks` is `make ci` minus the vulnerability scan (which
        // the Security stage below does better — see the Makefile comment),
        // and it covers, in this order: generate, tidy, lint, test-domain,
        // test-platform, vet-integration, test-tools, generate-client,
        // lint-web, test-web, build-web, go build ./...
        //
        // The old Build stage's bare `go vet ./...` is not lost: vet-integration
        // is `go vet -tags=integration ./...`, which vets every package plain
        // vet does and the integration-tagged files besides.
        //
        // Ordering is unchanged from ADR-0011 section 2 and still matters for
        // the same reason: internal/gen and web/src/api/generated are
        // gitignored, so generate MUST precede lint and build. That order now
        // lives in one place (the Makefile) rather than two.
        //
        // TRADE-OFF, stated because it is a real loss: the Lint/Unit/Build
        // stages ran their Go and Web halves in `parallel`, and one `sh` step
        // cannot. Wall-clock goes up, and a failure surfaces as "make ci-checks
        // failed" plus the failing target in the log rather than as a red box
        // in the stage view. Accepted: a pipeline that provably runs the same
        // checks as the local gate is worth more than a faster one that
        // silently runs fewer, which is precisely what #129 recorded.
        stage('CI gate') {
            steps {
                sh 'make ci-checks'
                sh 'git --no-pager diff --stat -- go.mod go.sum'
            }
        }

        // The web unit tests already ran inside the CI gate (`make test-web`).
        // This re-runs them through the JUnit reporter so Jenkins can publish
        // per-test results; `test:ci` is `vitest run` plus reporters, so it is
        // the same suite, not an additional one. Kept OUT of `make ci` on
        // purpose: writing a CI report file is not something a local gate
        // should do.
        stage('Web test report') {
            steps {
                sh 'make test-web-ci'
            }
            post {
                always {
                    junit allowEmptyResults: true, testResults: 'build/web-junit.xml'
                }
            }
        }

        // The only stage that intrinsically needs a Docker daemon: the
        // testcontainers-backed suite that proves the EXCLUDE constraint
        // holds under real concurrency (T4).
        //
        // Skipped-not-failed when Docker is absent, because this project's
        // own environments frequently lack it — but the build is marked
        // UNSTABLE so a skip is visible on the PR rather than looking green.
        // Set FAIL_ON_SKIPPED_INTEGRATION on the agent that guards the
        // shared branch to make it a hard failure there.
        stage('Integration tests') {
            steps {
                script {
                    def hasDocker = sh(script: 'docker info >/dev/null 2>&1', returnStatus: true) == 0
                    if (hasDocker) {
                        echo 'Docker daemon reachable — running the full suite (make test).'
                        sh 'make test'
                    } else {
                        def msg = 'No Docker daemon on this agent: the integration/concurrency ' +
                                  'tests did NOT run. The no-double-booking invariant is therefore ' +
                                  'UNVERIFIED by this build. Fix the agent (provision Docker, or ' +
                                  'start the daemon) rather than accepting this as expected — ' +
                                  'T61 found three live production defects behind exactly this ' +
                                  'disclaimer. See CLAUDE.md\'s "Docker works here" gotcha.'
                        if (params.FAIL_ON_SKIPPED_INTEGRATION) {
                            error(msg)
                        }
                        unstable(msg)
                    }
                }
            }
            post {
                always {
                    junit allowEmptyResults: true, testResults: 'build/junit.xml'
                }
            }
        }

        stage('Security scan') {
            steps {
                // govulncheck floats at @latest on purpose, unlike the other
                // tools: a pinned scanner is a scanner that stops learning
                // about new vulnerabilities. The vulnerability DATA is
                // fetched at run time regardless.
                //
                // warnError, not sh: if vuln.go.dev is unreachable (egress
                // policy, offline agent) that is an infrastructure problem,
                // and it must not masquerade as "no vulnerabilities found".
                // vulngate below is what decides pass/fail, and it gates on
                // whatever reports actually exist.
                //
                // The exit code, not the file's presence or size, is what
                // decides whether build/govulncheck.json is usable. A run
                // that dies partway through (e.g. can't reach vuln.go.dev)
                // still writes a well-formed `config`/SBOM preamble before
                // failing — a real, non-empty, valid-JSON file with zero
                // findings in it — so `[ -s ... ]` alone reads a failed scan
                // as a clean one. PR #95 review caught this: the gate
                // printed "PASS: no new gating findings" for a scan that
                // never actually ran. build/govulncheck.exit is the real
                // signal; see tools/vulngate/gate_test.go for the case this
                // guards (a truncated-but-nonempty report must not gate as
                // "no findings").
                warnError('Go vulnerability scan could not complete — findings UNKNOWN, not "none"') {
                    sh '''
                        set +e
                        go run golang.org/x/vuln/cmd/govulncheck@$GOVULNCHECK_VERSION -format json ./... > build/govulncheck.json
                        status=$?
                        echo "$status" > build/govulncheck.exit
                        exit "$status"
                    '''
                }

                // npm audit exits non-zero when it finds something; that is a
                // finding, not a tool failure. vulngate makes the decision.
                sh 'cd web && npm audit --json > ../build/npm-audit.json || true'

                sh '''
                    set -e
                    ARGS="-baseline security/vuln-baseline.json -npm build/npm-audit.json"
                    if [ -f build/govulncheck.exit ] && [ "$(cat build/govulncheck.exit)" = "0" ]; then
                        ARGS="$ARGS -go build/govulncheck.json"
                    else
                        echo "NOTE: Go vulnerability scan did not complete successfully (see build/govulncheck.exit) — gating on npm findings ONLY."
                    fi
                    go run ./cmd/vulngate $ARGS
                '''
            }
            post {
                always {
                    archiveArtifacts artifacts: 'build/govulncheck.json,build/govulncheck.exit,build/npm-audit.json',
                                     allowEmptyArchive: true, fingerprint: false
                }
            }
        }

        // Opt-in. Not a per-PR gate — see loadtest/README.md for why.
        // Needs a full running stack, hence Docker; skipped with a clear
        // reason rather than failing when that is unavailable.
        stage('Load test') {
            when {
                anyOf {
                    expression { params.RUN_LOAD_TEST }
                    triggeredBy 'TimerTrigger'
                }
            }
            steps {
                script {
                    def hasDocker = sh(script: 'docker info >/dev/null 2>&1', returnStatus: true) == 0
                    def hasK6 = sh(script: 'command -v k6 >/dev/null 2>&1', returnStatus: true) == 0

                    if (!hasDocker) {
                        unstable('Load test requested but no Docker daemon is available to run the stack — skipped.')
                        return
                    }
                    if (!hasK6) {
                        unstable('Load test requested but k6 is not installed on this agent — skipped. See loadtest/README.md.')
                        return
                    }

                    try {
                        sh 'make up'
                        // Give the API a moment to come up behind the DB
                        // healthcheck before driving load at it.
                        sh '''
                            for i in $(seq 1 30); do
                                if curl -fsS -o /dev/null "http://localhost:8080/v1/courts/11111111-1111-1111-1111-111111111111/bookings?from=2026-08-10T00:00:00Z&to=2026-08-11T00:00:00Z"; then
                                    echo "API is up"; exit 0
                                fi
                                sleep 2
                            done
                            echo "API did not become ready in 60s"; exit 1
                        '''
                        sh 'k6 run --summary-export=build/k6-summary.json loadtest/booking-quote.js'
                    } finally {
                        sh 'make down || true'
                    }
                }
            }
            post {
                always {
                    archiveArtifacts artifacts: 'build/k6-summary.json',
                                     allowEmptyArchive: true, fingerprint: false
                }
            }
        }

        stage('Deploy') {
            when { expression { false } }
            steps {
                // INTENTIONALLY DISABLED — needs a credential that does not
                // exist in this environment.
                //
                // SCRUM-6 explicitly forbids introducing a stage that
                // requires secrets not already present, so rather than
                // hardcode a fake registry/token or let a deploy fail
                // mysteriously, the stage is stubbed out and the requirement
                // written down:
                //
                //   - a container registry credential (Jenkins credential ID,
                //     e.g. 'registry-creds') for `docker push`
                //   - the target registry host
                //   - a deploy target (compose host / k8s context) and its
                //     own credential
                //
                // Enable by replacing the `when` above with a real branch
                // condition and wiring withCredentials(...) here.
                echo 'Deploy stage is a documented stub: needs a registry credential, not configured here.'
            }
        }
    }

    post {
        always {
            archiveArtifacts artifacts: 'build/coverage.out', allowEmptyArchive: true, fingerprint: false
            echo "Result: ${currentBuild.currentResult}"
        }
        unstable {
            echo 'UNSTABLE — a non-gating stage was skipped or degraded. Check the log for which; do not read this as a clean pass.'
        }
        cleanup {
            // Never leave a stack running on a shared agent.
            sh 'docker compose down -v >/dev/null 2>&1 || true'
        }
    }
}
