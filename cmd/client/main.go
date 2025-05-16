// cmd/client/main.go
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/ryk-9/go-chat/pkg/chat"
)

func main() {
	// Parse command-line flags
	serverAddr := flag.String("server", "", "Server address (host:port)")
	username := flag.String("user", "", "Your username")
	flag.Parse()

	// Check if server address was provided via flags or positional args
	if *serverAddr == "" {
		if flag.NArg() > 0 {
			*serverAddr = flag.Arg(0)
			if flag.NArg() > 1 {
				*username = flag.Arg(1)
			}
		}
	}

	// If server address is still empty, prompt for it
	if *serverAddr == "" {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter server address (e.g., localhost:8080): ")
		input, _ := reader.ReadString('\n')
		*serverAddr = strings.TrimSpace(input)

		if *serverAddr == "" {
			*serverAddr = "localhost:8080" // Default if empty
			fmt.Printf("Using default server: %s\n", *serverAddr)
		}
	}

	// If username is empty, prompt for it
	if *username == "" {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter your username: ")
		input, _ := reader.ReadString('\n')
		*username = strings.TrimSpace(input)

		// Keep prompting until we get a valid username
		for *username == "" || len(*username) < 2 || len(*username) > 20 ||
			strings.ContainsAny(*username, " \t\n/\\:") {
			fmt.Println("Username must be 2-20 characters without spaces or special chars (/, \\, :)")
			fmt.Print("Enter your username: ")
			input, _ := reader.ReadString('\n')
			*username = strings.TrimSpace(input)
		}
	}

	// Run the client
	fmt.Printf("Connecting as %s to %s...\n", *username, *serverAddr)
	err := chat.RunClient(*serverAddr, *username)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
