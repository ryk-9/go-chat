// cmd/client/main.go
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ryk-9/go-chat/pkg/chat"
)

func main() {
	// Parse command-line flags
	serverAddr := flag.String("server", "", "Server address (host:port)")
	username := flag.String("user", "", "Your username")
	flag.Parse()

	// Check if server address was provided
	if *serverAddr == "" {
		if flag.NArg() > 0 {
			*serverAddr = flag.Arg(0)
			if flag.NArg() > 1 {
				*username = flag.Arg(1)
			}
		}
	}

	// Validate required parameters
	if *serverAddr == "" {
		fmt.Println("Error: Server address is required")
		fmt.Println("\nUsage:")
		fmt.Println("  go run main.go -server <host:port> -user <username>")
		fmt.Println("  go run main.go <host:port> <username>")
		fmt.Println("\nExample:")
		fmt.Println("  go run main.go -server localhost:8080 -user alice")
		fmt.Println("  go run main.go localhost:8080 alice")
		os.Exit(1)
	}

	// Ask for username if not provided
	if *username == "" {
		fmt.Print("Enter your username: ")
		fmt.Scanln(username)
	}

	// Run the client
	err := chat.RunClient(*serverAddr, *username)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
