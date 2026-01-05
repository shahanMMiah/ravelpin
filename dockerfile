# --- Stage 1: Build Environment ---
FROM golang:1.24-bookworm AS builder

# 1. Install TensorFlow C Library dependencies
RUN apt-get update && apt-get install -y \
    curl \
    tar \
    gcc \
    && rm -rf /var/lib/apt/lists/*

# 2. Download and extract the TensorFlow C library
# Current for 2026: Use the stable release from official storage

RUN curl -L "https://storage.googleapis.com/tensorflow/versions/2.18.0/libtensorflow-cpu-linux-x86_64.tar.gz" | \
    tar -C /usr/local -xz

# 3. Configure the linker to find the library
RUN ldconfig


# 4. Build the Go application
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Note: CGO is REQUIRED for TensorFlow bindings
RUN CGO_ENABLED=1 GOOS=linux go build -o main .

# --- Stage 2: Production Image ---
FROM debian:bookworm-slim

# 1. Copy the C library from the builder to the runtime image
COPY --from=builder /usr/local/lib/libtensorflow* /usr/local/lib/
RUN ldconfig

# 2. Copy the compiled Go binary
WORKDIR /root

COPY --from=builder /app/model .
COPY --from=builder /app/main .

# Ensure the ca-certificates package is installed
RUN apt-get update && apt-get install -y ca-certificates

# Use .crt extension specifically for the update-ca-certificates tool
COPY cert-ca.crt /usr/local/share/ca-certificates/cert-ca.crt

# Update the system trust store
RUN update-ca-certificates

# Start the application
CMD ["./main"]