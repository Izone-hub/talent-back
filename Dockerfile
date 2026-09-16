# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy dependency definitions
COPY go.mod go.sum ./
RUN go mod download

# Copy source files
COPY . .

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux go build -o main main.go

# Runtime stage
FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/main .
COPY --from=builder /app/templates ./templates

EXPOSE 8080

CMD ["./main"]
