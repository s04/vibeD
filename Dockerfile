# Stage 1: Build React frontend
FROM node:22-bookworm-slim AS frontend

WORKDIR /app/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ .
COPY internal/frontend/static/ /app/internal/frontend/static/
RUN npm run build

# Stage 2: Build Go binary
FROM golang:1.23-bookworm AS builder

ENV GOTOOLCHAIN=auto
ENV GO111MODULE=on

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Copy built frontend assets from Stage 1
COPY --from=frontend /app/internal/frontend/static/ /app/internal/frontend/static/

RUN CGO_ENABLED=0 GOOS=linux go build -o /vibed ./cmd/vibed

# Stage 3: Runtime
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /vibed /vibed
COPY vibed.yaml /etc/vibed/vibed.yaml

EXPOSE 8080

ENTRYPOINT ["/vibed"]
CMD ["--config", "/etc/vibed/vibed.yaml", "--transport", "http"]
