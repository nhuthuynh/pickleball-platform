# Builds the Go API server. Vue static/SSG assets (docs/technology-options.md
# §6) are added to this image once the web app exists; for now this serves
# the gRPC + grpc-gateway REST API only.
FROM golang:1.24-bookworm AS build
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
