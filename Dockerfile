# Builds the Go API server. Vue static/SSG assets (docs/technology-options.md
# §6) are added to this image once the web app exists; for now this serves
# the gRPC + grpc-gateway REST API only.
# The tag's minor version MUST match go.mod's `go` directive. It did not
# from the repository's first commit until 2026-10-17: go.mod has said
# `go 1.25.0` since T0 and this said 1.24, so `go mod download` failed with
# "go.mod requires go >= 1.25.0 (running go 1.24.13; GOTOOLCHAIN=local)" and
# **the server image has never built**. `make up` could not start the stack
# for 66 sprints, and nothing noticed because no gate builds this image.
# `make dockerfile-go-version` now compares the two.
FROM golang:1.25-bookworm AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
# internal/gen is produced by `make generate` on the host and copied in here
# rather than regenerated in the image, to avoid needing buf/sqlc + network
# access to buf.build during the image build.
RUN test -d internal/gen || (echo "internal/gen missing — run 'make generate' before 'docker build'" && exit 1)
RUN CGO_ENABLED=0 go build -o /out/server ./cmd/server

FROM gcr.io/distroless/static-debian12
COPY --from=build /out/server /server
EXPOSE 8080 8081
ENTRYPOINT ["/server"]
