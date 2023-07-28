@echo off

:: Build the Docker image
docker build -t ratelimiter .

:: Run the Docker container
docker run -d -p 8080:8080 --name ratelimiter-container ratelimiter

:: Display the container information
docker ps --filter "name=ratelimiter-container" --format "table {{.ID}}\t{{.Names}}\t{{.Status}}\t{{.Ports}}"
