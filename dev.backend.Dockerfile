FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files first for better layer caching
COPY backend/go.mod ./
# Copy go.sum if it exists (optional for first build)
COPY backend/go.sum* ./

# Download dependencies (cache layer if go.mod/go.sum unchanged)
RUN go mod download

# Copy source code
COPY backend/ .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/server

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates wget

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/main .

EXPOSE 8080

CMD ["./main"]
