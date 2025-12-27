# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum* ./

# Download dependencies
RUN go mod download

# Copy source code
COPY *.go ./

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o octocatalog .

# Runtime stage
FROM scratch

# Copy the binary from builder
COPY --from=builder /app/octocatalog /octocatalog

# Copy the catalog configuration
COPY catalog.json /catalog.json

# Expose the port
EXPOSE 8080

# Run the application
ENTRYPOINT ["/octocatalog"]
