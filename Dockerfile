# Base image containing golang runtime image
FROM golang:latest

# Versioning
LABEL version="1.0.0"

# Directory containing application files inside the container
WORKDIR /app

# Copy files of current directory into the app directory of the container
COPY . /app

#  Command to run and its arguments
CMD ["go", "run", "/app/main.go"]
