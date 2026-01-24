# test.Dockerfile - Test image with OpenSSL 3.2.x

FROM golang:1.21-bookworm

ARG OPENSSL_VERSION=3.2.2

WORKDIR /app

# Build dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential pkg-config ca-certificates wget perl zlib1g-dev \
  && rm -rf /var/lib/apt/lists/*

# OpenSSL 3.2.x
RUN mkdir -p /tmp/openssl-src \
  && wget -q "https://www.openssl.org/source/openssl-${OPENSSL_VERSION}.tar.gz" -O /tmp/openssl.tar.gz \
  && tar -xzf /tmp/openssl.tar.gz -C /tmp/openssl-src --strip-components=1 \
  && cd /tmp/openssl-src \
  && ./config --prefix=/usr/local/openssl --openssldir=/usr/local/openssl shared zlib \
  && make -j"$(nproc)" && make install_sw \
  && rm -rf /tmp/openssl-src /tmp/openssl.tar.gz

ENV PKG_CONFIG_PATH=/usr/local/openssl/lib/pkgconfig
ENV LD_LIBRARY_PATH=/usr/local/openssl/lib

COPY backend/go.mod backend/go.sum* ./
RUN go mod download

COPY backend/ .

RUN mkdir -p /test-results

# Run unit tests (integration tests require DB init)
CMD ["bash", "-c", "go test -tags=test -v ./internal/pow/... ./tests/handlers/... ./tests/utils/... ./tests/middleware/... ./tests/encryption/... ./tests/auth/... 2>&1 | tee /test-results/output.log; exit ${PIPESTATUS[0]}"]
