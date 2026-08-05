# Builds the Vue web client and serves the static bundle with nginx.
#
# Mirrors the root Dockerfile's contract deliberately: just as that image
# refuses to build without `internal/gen` (produced by `make generate` on
# the host), this one refuses to build without `src/api/generated` — the
# typed OpenAPI client produced by `npm run generate:client`, which itself
# shells out to buf + sqlc. Generating inside the image would mean
# installing buf/sqlc/protoc-gen-* and reaching the BSR from the build
# sandbox, which CLAUDE.md's gotchas already rule out. `make up` runs the
# generate step first, so this guard should only ever fire on a manual
# `docker build`.
FROM node:22-alpine AS build
WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci

COPY . .
RUN test -d src/api/generated || (echo "web/src/api/generated missing — run 'npm --prefix web run generate:client' before 'docker build'" && exit 1)

# The API origin is inlined at build time by Vite (import.meta.env), so it
# is a build arg, not a runtime env var. The default matches the API port
# docker-compose publishes to the host — the browser talks to the API
# directly, not through this nginx.
ARG VITE_API_BASE_URL=http://localhost:8080
ENV VITE_API_BASE_URL=$VITE_API_BASE_URL

# `build-only` (vite build), not `build` (vite build + vue-tsc). Type
# checking is a CI stage in its own right (`make lint-web`); repeating it
# on every `make up` would slow the local stack down for no extra signal.
RUN npm run build-only

FROM nginx:1.27-alpine
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 80
