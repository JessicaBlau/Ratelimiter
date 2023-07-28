# Use the official Golang image as the base image
FROM golang:1.16-alpine

# Set the working directory inside the container
WORKDIR /app

# Copy the Go modules and download the dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the Go application inside the container
RUN go build -o ratelimiter .

# Set the entry point command to run the Go application
CMD ["./ratelimiter"]
