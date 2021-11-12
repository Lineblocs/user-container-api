# Dockerfile References: https://docs.docker.com/engine/reference/builder/

# Start from the latest golang base image
FROM golang:1.17.1

# Add Maintainer Info
LABEL maintainer="Nadir Hamid <matrix.nad@gmail.com>"

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy the source from the current directory to the Working Directory inside the container
COPY . .

# Build the Go app
RUN go build -o main .

# Expose port 80 to the outside world
EXPOSE 80

# Command to run the executable
CMD ["./main"]