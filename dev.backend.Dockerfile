FROM golang:1.21-bookworm AS builder

ARG OPENSSL_VERSION=3.2.2

WORKDIR /app

# Copy go mod files first for better layer caching
COPY backend/go.mod ./
# Copy go.sum if it exists (optional for first build)
COPY backend/go.sum* ./

# Download dependencies (cache layer if go.mod/go.sum unchanged)
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    pkg-config \
    ca-certificates \
    wget \
    perl \
    zlib1g-dev \
  && rm -rf /var/lib/apt/lists/*

# Build OpenSSL 3.2.x (required for Argon2id KDF)
RUN mkdir -p /tmp/openssl-src \
  && wget -q "https://www.openssl.org/source/openssl-${OPENSSL_VERSION}.tar.gz" -O /tmp/openssl.tar.gz \
  && tar -xzf /tmp/openssl.tar.gz -C /tmp/openssl-src --strip-components=1 \
  && cd /tmp/openssl-src \
  && ./config --prefix=/usr/local/openssl --openssldir=/usr/local/openssl shared zlib \
  && make -j"$(nproc)" \
  && make install_sw \
  && rm -rf /tmp/openssl-src /tmp/openssl.tar.gz

ENV PKG_CONFIG_PATH=/usr/local/openssl/lib/pkgconfig
ENV LD_LIBRARY_PATH=/usr/local/openssl/lib

RUN go mod download

# Copy source code
COPY backend/ .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -o main ./cmd/server

# Final stage
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    wget \
  && rm -rf /var/lib/apt/lists/*

WORKDIR /root/

# OpenSSL 3.2 runtime libraries
COPY --from=builder /usr/local/openssl /usr/local/openssl
ENV LD_LIBRARY_PATH=/usr/local/openssl/lib
ENV PATH=/usr/local/openssl/bin:$PATH

# Copy the binary from builder
COPY --from=builder /app/main .

EXPOSE 8080

CMD ["./main"]
