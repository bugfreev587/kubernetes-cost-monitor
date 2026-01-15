# ----------------------
# Build stage
# ----------------------
FROM golang:1.25-alpine AS builder

# Install git and ca-certificates for Go modules
RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Copy Go modules manifests
COPY go.mod go.sum ./
RUN go mod download

# Copy all source code
COPY . .

# Build the binary for the target architecture (buildx will set this automatically)
# Binary output: /bin/agent
ARG TARGETARCH
ARG TARGETOS=linux
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags="-s -w" -o /bin/agent ./main.go

# ----------------------
# Final stage
# ----------------------
FROM alpine:3.18

# Add CA certificates for HTTPS requests, vim for debugging, and wget for test scripts
RUN apk add --no-cache ca-certificates vim wget

# Copy the compiled binary from builder
COPY --from=builder /bin/agent /bin/agent

# Copy test scripts before switching to non-root user
COPY --from=builder /app/scripts/test-api.sh /usr/local/bin/test-api.sh

# Ensure executable permissions
RUN chmod +x /bin/agent && \
    chmod +x /usr/local/bin/test-api.sh

# Use a non-root user (production best practice)
RUN addgroup -S agent && adduser -S agent -G agent

# Create scripts directory accessible by agent user and copy to /tmp
RUN mkdir -p /home/agent/scripts && \
    cp /usr/local/bin/test-api.sh /home/agent/scripts/test-api.sh && \
    cp /usr/local/bin/test-api.sh /tmp/test-api.sh && \
    chown -R agent:agent /home/agent/scripts && \
    chmod +x /home/agent/scripts/test-api.sh && \
    chmod +x /tmp/test-api.sh

USER agent

# Entrypoint
ENTRYPOINT ["/bin/agent"]