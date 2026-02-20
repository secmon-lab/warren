# Frontend build stage
FROM node:18-alpine AS build-frontend
WORKDIR /app/frontend

# Install pnpm
RUN corepack enable && corepack prepare pnpm@10 --activate

# Copy package files first for better caching
COPY frontend/package.json frontend/pnpm-lock.yaml frontend/.npmrc ./
RUN pnpm install --frozen-lockfile

# Copy source files and build
COPY frontend/ ./
RUN pnpm run build

# Go build stage
FROM golang:1.26-alpine AS build-go
ENV CGO_ENABLED=0
ARG BUILD_VERSION

# Install git for go mod operations
RUN apk add --no-cache git

WORKDIR /app

# Set up Go module cache directory
ENV GOCACHE=/root/.cache/go-build
ENV GOMODCACHE=/root/.cache/go-mod

# Copy go.mod and go.sum first for dependency caching
COPY go.mod go.sum ./

# Download dependencies with cache mount
RUN --mount=type=cache,target=/root/.cache/go-mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

# Copy source code
COPY . /app

# Copy frontend build output
COPY --from=build-frontend /app/frontend/dist /app/frontend/dist

# Build the application with cache mounts
RUN --mount=type=cache,target=/root/.cache/go-mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -ldflags="-w -s" -o warren

# Final stage
FROM gcr.io/distroless/base:nonroot
USER nonroot
COPY --from=build-go /app/warren /warren

ENTRYPOINT ["/warren"]
