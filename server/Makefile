.PHONY: build run clean test test-integration deps fmt

# Build the application
build: fmt
	CGO_ENABLED=0 go build -o lectures ./cmd/server

# Run the application
run: build
	./lectures

# Clean build artifacts
clean:
	rm -f lectures
	go clean

# Run tests (can take TEST argument for -run, and TAGS for build tags like integration)
test: fmt
	CGO_ENABLED=0 go test -v $(if $(TAGS),-tags=$(TAGS)) ./... $(if $(TEST),-run $(TEST))

# Format the code
fmt:
	go fmt ./...

# Download dependencies
deps:
	go mod download
	go mod tidy

# Development mode with auto-reload
dev:
	CGO_ENABLED=0 go run ./cmd/server

# Build for multiple platforms
build-all:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o lectures-linux-amd64 ./cmd/server
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o lectures-darwin-amd64 ./cmd/server
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o lectures-darwin-arm64 ./cmd/server
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o lectures-windows-amd64.exe ./cmd/server
