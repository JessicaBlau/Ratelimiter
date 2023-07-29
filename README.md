# Ratelimiter

Ratelimiter is a simple API service written in GoLang that implements a basic rate limiter for multiple clients. It allows you to define a request limit for each client, and the rate limit resets every second. If a client exceeds its allowed number of requests within a second, subsequent requests will be blocked.

## Features

- Endpoint `/limit` for rate limiting.
- Endpoint `/custom` for custom limiting.
- Supports multiple clients, each with its own unique ID.
- Configuration file to define the request limit for each client.
- Requests are rate-limited and reset every second.
- Provides 204 HTTP status for allowed requests and 400 HTTP status for blocked requests.
- Concurrency handling to ensure thread-safety.
- Test coverage for basic functionality and concurrency scenarios.
- Dockerfile and deployment script for containerization.

### Custom Rate Limiter Endpoint (/custom)

#### Overview

The custom rate limiter endpoint (/custom) is an additional feature offered by the service to handle rate limiting for specific clients with unique requirements. It allows clients to make requests with their unique client ID included in the X-Client-ID header. The rate limiting logic for this endpoint differs from the standard endpoint, enabling customized rate limits based on individual client needs.

## How It Works

1. Clients make requests to the /custom endpoint and include their unique client ID in the X-Client-ID header.

2. The rate limiter performs the following steps to handle the request:
   - The rate limiter checks if the client's token bucket allows the request to proceed. The token bucket algorithm is a popular method used for rate limiting. It ensures that clients receive tokens at a certain rate and can only make requests if they have available tokens.
   - If the client has exceeded the rate limit (i.e., no more tokens available in their bucket), the service responds with a 400 Bad Request status code and the message "Request blocked. No more tokens."
   - If the request is allowed (i.e., the client has available tokens), the service responds with a 200 OK status code and the message "OK."

## Usage Example (Assuming Go Programming Language)

Assuming you have an instance of the RateLimiter struct named `rateLimiter`:

```go
// Handle the custom endpoint with custom rate limiter logic
http.HandleFunc("/custom", rateLimiter.handleCustom)
```

In this usage example, an instance of the `RateLimiter` struct named `rateLimiter` is utilized to handle the custom endpoint. The actual implementation of the `handleCustom` function within the `RateLimiter` struct will handle the rate limiting logic specific to the /custom endpoint and client ID provided in the X-Client-ID header.

## Benefits

The custom rate limiter endpoint offers more flexibility in managing different clients' request rates. It allows the service to accommodate specific client requirements by tailoring the rate limiting behavior to their needs. This feature is useful when certain clients need higher or lower rate limits compared to the standard rate limiting rules. By providing custom rate limits, the service can ensure a fair distribution of resources while meeting the varying demands of its clients.

## How to Use

### Prerequisites

- GoLang is installed on your machine.
- Docker is installed (optional, for containerization).

### Clone the Repository

```bash
git clone https://github.com/your-username/ratelimiter.git
cd ratelimiter
```

### Run the Application Locally

To install dependencies:

```sh
go mod download
```

To build and run the application:

```sh
go run ratelimiter.go
```

The rate limiter service will be running on http://localhost:8080/limit.

# Configuration

You can configure the request limit for each client in the `config.json` file. The file should have the following format:

```json
{
  "clients": [
    {
      "ID": "client1",
      "RequestMax": 5,
      "TokensPerSec": 10
    },
    {
      "ID": "client2",
      "RequestMax": 10,
      "TokensPerSec": 5
    },
    {
      "ID": "client3",
      "RequestMax": 20,
      "TokensPerSec": 3
    }
  ]
}
```

## API Documentation

### Rate Limit Endpoint

- **Endpoint:** `/limit`
- **Method:** GET
- **Headers:** Set the client ID as the header `X-Client-ID`.

**Response:**

- HTTP 204 No Content: Request is allowed within the rate limit.
- HTTP 400 Bad Request: Request is blocked due to exceeding the rate limit.

### Custom Endpoint

The custom endpoint uses a token bucket algorithm for rate limiting. It allows clients to request resources at a variable rate, as long as they have tokens available in their bucket.

- **Endpoint:** `/custom`
- **Method:** GET
- **Headers:** Set the client ID as the header `X-Client-ID`.

**Response:**

- HTTP 200 OK: Request is allowed within the rate limit, and a custom resource is served.
- HTTP 400 Bad Request: Request is blocked due to exceeding the rate limit or lack of available tokens.

This endpoint provides a more flexible rate limiting approach by utilizing the token bucket algorithm. Clients are assigned a specific number of tokens per second in the token bucket. Each request consumes one token from the client's token bucket. If a client's token bucket is empty, the request will be blocked until the bucket is refilled.

Note: The number of tokens per second for each client can be configured in the config.json file. The rate limiter uses the provided rate to determine if the client has enough tokens to proceed with the request.

## Testing

To run the tests, use the following command:

```sh
go test -v
```

## Docker Deployment

You can deploy the rate limiter service as a Docker container using the provided Dockerfile and deployment script.

To build the Docker image:

```sh
docker build -t ratelimiter .
```

## To run the Docker container:

```sh
docker run -d -p 8080:8080 --name ratelimiter-container ratelimiter
```

The rate limiter service will be accessible on http://localhost:8080/limit from your host machine.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.




