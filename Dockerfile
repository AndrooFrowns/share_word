# Build Stage
FROM golang:alpine AS builder

WORKDIR /app

# Install build tools and dependencies
RUN apk add --no-cache git build-base

# Install templ for UI component generation
RUN go install github.com/a-h/templ/cmd/templ@latest

# Copy dependency manifests and download modules
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Generate templ components and build the self-contained binary
RUN templ generate
RUN CGO_ENABLED=0 GOOS=linux go build -o shareword cmd/server/main.go

# Final Stage
FROM alpine:latest

WORKDIR /app

# Install CA certificates and create a non-root user
RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S appgroup && adduser -S appuser -G appgroup

# Copy binary from the builder stage
COPY --from=builder /app/shareword .

# Set ownership to the non-root user
RUN chown -R appuser:appgroup /app && \
    mkdir /data && chown -R appuser:appgroup /data

# Switch to the non-root user
USER appuser

# Production defaults
ENV PORT=8080
ENV DB_PATH=/data/shareword.db
ENV ENV=production

# The app listens on this port
EXPOSE 8080

# Run the app
CMD ["./shareword"]
