# Use the official Go image to create a build artifact.
FROM golang:1.21rc2-bullseye as builder

# Copy local code to the container image.
WORKDIR /go/src/go-alpaca-streaming
COPY . .

# Build the command inside the container.
RUN go mod download
RUN go build -o alpaca-client ./cmd/

# Use a minimal image to run the compiled Go binary.
FROM debian:bullseye-slim

# Copy the binary from the builder stage.
COPY --from=builder /go/src/go-alpaca-streaming/alpaca-client /alpaca-client

# Run the web service on container startup.
CMD ["/alpaca-client"]