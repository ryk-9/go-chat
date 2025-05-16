# Go Chat CLI

A real-time chat application built with Go that works across different WiFi networks. This application uses WebSockets for communication and features a simple command-line interface.

## Features

- Real-time messaging with WebSockets
- Works across different networks (as long as the server is accessible)
- Simple CLI interface
- Username identification
- Command system (/help, /users, /whisper, etc.)
- Private messaging between users
- Connection status monitoring

## Requirements

- Go 1.16 or higher
- Internet connectivity

## Installation

### Option 1: Clone the repository

```bash
git clone https://github.com/ryk-9/go-chat.git
cd go-chat
go mod tidy
```

### Option 2: Build from source

```bash
# Clone or create the project structure as shown
mkdir -p gochat/cmd/{client,server} go-chat/pkg/chat
# Copy all the files to their respective locations
# Run:
cd go-chat
go mod tidy
```

## Usage

### Building the Application

Use the provided Makefile to build the application:

```bash
# Build both server and client
make all

# Build only the server
make server

# Build only the client
make client
```

### Running the Server

```bash
# Using the Makefile
make run-server

# Or directly
./chat-server

# With a custom port
./chat-server -port 9000
```

### Running the Client

```bash
# Using the Makefile (will prompt for server and username)
make run-client

# Or directly
./chat-client -server example.com:8080 -user alice

# Or with positional arguments
./chat-client localhost:8080 bob
```

## Available Chat Commands

Once connected to the chat, you can use these commands:

- `/help` - Show available commands
- `/users` - List all connected users
- `/time` - Show current server time
- `/whisper <username> <message>` - Send a private message
- `/exit` - Exit the chat

## Deployment

### Server Deployment

To deploy the server on a public-facing machine:

1. Build the server binary:

   ```bash
   make server
   ```

2. Copy the binary to your server.

3. Run it (optionally in the background):

   ```bash
   ./chat-server &
   ```

4. Consider using a service manager like systemd for production deployments.

### Firewall Configuration

Make sure to open the server port (default: 8080) in your firewall:

```bash
# Using ufw (Ubuntu)
sudo ufw allow 8080/tcp

# Using iptables
sudo iptables -A INPUT -p tcp --dport 8080 -j ACCEPT
```

## Project Structure

```
gochat/
├── cmd/
│   ├── client/
│   │   └── main.go       # Client entry point
│   └── server/
│       └── main.go       # Server entry point
├── pkg/
│   └── chat/
│       ├── client.go     # Client implementation
│       └── server.go     # Server implementation
├── go.mod               # Go module file
├── go.sum               # Go dependencies
├── Makefile             # Build automation
└── README.md            # This file
```

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgments

- [Gorilla WebSocket](https://github.com/gorilla/websocket) for the WebSocket implementation
