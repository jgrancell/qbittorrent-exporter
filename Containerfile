# Stage 1: Build the Go binary
FROM docker.io/golang:1.23-alpine AS builder

# Install any dependencies
RUN apk add --no-cache git

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application code
COPY . .

# Build the Go application
RUN go build -o prometheus-qbittorrent-exporter

# Stage 2: Create the final container image
FROM alpine:3.16
LABEL org.opencontainers.image.source = "https://github.com/jgrancell/prometheus-qbittorrent-exporter" 

# Install certificates for HTTPS connections
RUN apk add --no-cache ca-certificates

# Set the working directory inside the container
WORKDIR /root/

# Copy the Go binary from the builder stage
COPY --from=builder /app/prometheus-qbittorrent-exporter .

# Expose port 8080 (or the port your exporter listens on)
EXPOSE 8080

# Command to run the Go binary
ENTRYPOINT ["./prometheus-qbittorrent-exporter"]
