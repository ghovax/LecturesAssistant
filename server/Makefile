.PHONY: build run clean test deps

# Build the application
build:
	go build -o lectures ./cmd/server

# Run the application
run: build
	./lectures

# Clean build artifacts
clean:
	rm -f lectures
	go clean

# Run tests
test:
	go test -v ./...

# Download dependencies
deps:
	go mod download
	go mod tidy

# Development mode with auto-reload
dev:
	go run ./cmd/server

# Build for multiple platforms
build-all:
	GOOS=linux GOARCH=amd64 go build -o lectures-linux-amd64 ./cmd/server
	GOOS=darwin GOARCH=amd64 go build -o lectures-darwin-amd64 ./cmd/server
	GOOS=darwin GOARCH=arm64 go build -o lectures-darwin-arm64 ./cmd/server
	GOOS=windows GOARCH=amd64 go build -o lectures-windows-amd64.exe ./cmd/server
