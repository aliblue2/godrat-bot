# Use the official Go 1.24 image as the build stage
FROM golang:1.24-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files to download dependencies
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY main.go .
COPY models/ models/

# Build the Go binary
RUN CGO_ENABLED=0 GOOS=linux go build -o godrat-bot

# Use a minimal alpine image for the final stage
FROM alpine:latest

# Set the working directory
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/godrat-bot .


# Install ca-certificates for HTTPS requests (needed for Supabase/Telegram APIs)
RUN apk --no-cache add ca-certificates

# Expose no ports (Telegram bot uses outbound HTTPS requests)
# Set the entrypoint to run the bot
CMD ["./godrat-bot"]