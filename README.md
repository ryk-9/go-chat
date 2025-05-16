# Go Chat CLI

A simple real-time chat application built with Go that works across different WiFi networks.

## Features

- Real-time messaging using WebSockets
- Simple CLI interface
- Works across different networks
- Username identification

## Setup

1. Install dependencies:
   ```
   go mod tidy
   ```

2. Run the server:
   ```
   go run server.go
   ```

3. Connect with clients:
   ```
   go run client.go <server-address> <username>
   ```
   Example: `go run client.go example.com:8080 alice`

## Commands

- Type messages and press Enter to send
- Use `/exit` to quit the application

## Requirements

- Go 1.16+
- gorilla/websocket package
