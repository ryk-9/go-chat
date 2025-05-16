// pkg/chat/client.go
package chat

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

// RunClient connects to a chat server and handles the chat session
func RunClient(serverAddr, username string) error {
	// Validate username
	if len(username) < 2 || len(username) > 20 {
		return fmt.Errorf("username must be between 2 and 20 characters")
	}

	if strings.ContainsAny(username, " \t\n/\\:") {
		return fmt.Errorf("username cannot contain spaces or special characters (/, \\, :)")
	}

	// Construct websocket URL
	u := url.URL{Scheme: "ws", Host: serverAddr, Path: "/ws"}
	fmt.Printf("Connecting to %s...\n", u.String())

	// Connect to the WebSocket server
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("connection error: %w", err)
	}
	defer conn.Close()

	// Send username as the first message
	if err := conn.WriteMessage(websocket.TextMessage, []byte(username)); err != nil {
		return fmt.Errorf("error sending username: %w", err)
	}

	// Setup channels
	done := make(chan struct{})
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	// Clear the screen and show welcome message
	fmt.Print("\033[H\033[2J") // Clear screen
	fmt.Println("=== Go Chat CLI ===")
	fmt.Println("Type /help for available commands")
	fmt.Println("Press Ctrl+C to exit")
	fmt.Println("====================")

	// Goroutine to receive messages
	go func() {
		defer close(done)
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					fmt.Printf("\rConnection closed: %v\n", err)
				}
				return
			}

			msgText := string(message)
			fmt.Printf("\r%s\n", msgText)
			fmt.Print("> ")
		}
	}()

	// User input loop with command history
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("> ")
	go func() {
		for scanner.Scan() {
			message := scanner.Text()

			// Skip empty messages
			if strings.TrimSpace(message) == "" {
				fmt.Print("> ")
				continue
			}

			// Handle client-side exit command
			if message == "/exit" {
				fmt.Println("Exiting chat...")
				// Send close message
				conn.WriteMessage(
					websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
				)
				return
			}

			// Send the message
			err := conn.WriteMessage(websocket.TextMessage, []byte(message))
			if err != nil {
				fmt.Printf("Error sending message: %v\n", err)
				return
			}

			fmt.Print("> ")
		}
	}()

	// Wait for termination
	for {
		select {
		case <-done:
			return nil
		case <-interrupt:
			fmt.Println("\rInterrupted, closing connection...")

			// Gracefully close WebSocket
			err := conn.WriteMessage(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
			)
			if err != nil {
				return fmt.Errorf("write close error: %w", err)
			}

			// Wait for server to close connection or timeout
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return nil
		}
	}
}
