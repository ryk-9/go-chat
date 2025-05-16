.PHONY: all build clean server client

# Build settings
BINARY_SERVER=chat-server
BINARY_CLIENT=chat-client
MAIN_SERVER=./cmd/server
MAIN_CLIENT=./cmd/client

# Default target: build both server and client
all: server client

# Build server binary
server:
	go build -o $(BINARY_SERVER) $(MAIN_SERVER)

# Build client binary
client:
	go build -o $(BINARY_CLIENT) $(MAIN_CLIENT)

# Remove built binaries
clean:
	rm -f $(BINARY_SERVER) $(BINARY_CLIENT)

# Run the server
run-server: server
	./$(BINARY_SERVER)

# Run client (requires server address and username)
run-client: client
	./$(BINARY_CLIENT)

# Install dependencies
deps:
	go mod tidy

# Build for multiple platforms
build-all: clean
	# Linux
	GOOS=linux GOARCH=amd64 go build -o $(BINARY_SERVER)-linux-amd64 $(MAIN_SERVER)
	GOOS=linux GOARCH=amd64 go build -o $(BINARY_CLIENT)-linux-amd64 $(MAIN_CLIENT)
	# macOS
	GOOS=darwin GOARCH=amd64 go build -o $(BINARY_SERVER)-darwin-amd64 $(MAIN_SERVER)
	GOOS=darwin GOARCH=amd64 go build -o $(BINARY_CLIENT)-darwin-amd64 $(MAIN_CLIENT)
	# Windows
	GOOS=windows GOARCH=amd64 go build -o $(BINARY_SERVER)-windows-amd64.exe $(MAIN_SERVER)
	GOOS=windows GOARCH=amd64 go build -o $(BINARY_CLIENT)-windows-amd64.exe $(MAIN_CLIENT)

# Run tests
test:
	go test -v ./...