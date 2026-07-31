pipeline {
    agent {
        docker { image 'golang:1.24-bookworm' }
    }

    environment {
        GOFLAGS = '-mod=mod'
    }

    stages {
        stage('Domain tests') {
            steps {
                // Fast, dependency-free gate — matches HANDOFF.md's T0 check.
                sh 'make test-domain'
            }
        }

        stage('Generate') {
            steps {
                sh 'go install github.com/bufbuild/buf/cmd/buf@latest'
                sh 'go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest'
                sh 'make generate'
                sh 'make tidy'
            }
        }

        stage('Lint') {
            steps {
                sh 'go install github.com/golangci-lint/golangci-lint/cmd/golangci-lint@latest'
                sh 'make lint'
            }
        }

        stage('Full test suite') {
            steps {
                sh 'go install gotest.tools/gotestsum@latest'
                sh 'mkdir -p build'
                sh 'make test'
            }
            post {
                always {
                    junit 'build/junit.xml'
                }
            }
        }

        stage('Build image') {
            when { branch 'master' }
            steps {
                sh 'docker build -t pickleball-api:${GIT_COMMIT} .'
            }
        }
    }
}
