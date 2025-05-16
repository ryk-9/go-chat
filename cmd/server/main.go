// cmd/server/main.go
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/ryk-9/go-chat/pkg/chat"
)

func main() {
	// Parse command-line flags
	port := flag.Int("port", 8080, "Port to run the server on")
	flag.IntVar(port, "p", 8080, "Port to run the server on (shorthand)")
	flag.Parse()

	// Initialize the server
	server := chat.NewServer()
	go server.Run()

	// Set up WebSocket handler
	http.HandleFunc("/ws", server.HandleWebSocket)

	// Set up health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	// Set up graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Start HTTP server in a goroutine
	serverAddress := fmt.Sprintf(":%d", *port)
	srv := &http.Server{Addr: serverAddress}

	go func() {
		log.Printf("Chat server starting on %s", serverAddress)
		log.Printf("Press Ctrl+C to stop the server")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-stop
	log.Println("Shutting down server...")
}
