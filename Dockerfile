# Stage 1: Build React frontend (architecture-independent)
FROM --platform=$BUILDPLATFORM node:22-bookworm-slim AS frontend

WORKDIR /app/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ .
COPY internal/api/static/ /app/internal/api/static/
RUN npm run build

# Stage 2: Build Go binary (cross-compile natively, no QEMU needed)
FROM --platform=$BUILDPLATFORM golang:1.23-bookworm AS builder

ARG TARGETOS=linux
ARG TARGETARCH

ENV GOTOOLCHAIN=auto
ENV GO111MODULE=on

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Copy built frontend assets from Stage 1
COPY --from=frontend /app/internal/api/static/ /app/internal/api/static/

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o /vibed ./cmd/vibed

# Stage 3: Runtime
FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates git \
    && rm -rf /var/lib/apt/lists/* \
    && useradd --system --create-home --home-dir /home/vibed --shell /usr/sbin/nologin vibed

COPY --from=builder /vibed /vibed
COPY vibed.yaml /etc/vibed/vibed.yaml

EXPOSE 8080

USER vibed

ENTRYPOINT ["/vibed"]
CMD ["--config", "/etc/vibed/vibed.yaml", "--transport", "http"]
