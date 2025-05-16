// debug.go - Place this in the root directory
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Connected clients
var clients = make(map[*websocket.Conn]string)

// Main handler
func handleConnections(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP request to WebSocket
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	defer ws.Close()

	log.Println("New client connected")

	// First message should be username
	_, userMsg, err := ws.ReadMessage()
	if err != nil {
		log.Println("Read error:", err)
		return
	}
	username := string(userMsg)
	log.Printf("Client registered: %s", username)

	// Register client
	clients[ws] = username

	// Send welcome message
	welcomeMsg := fmt.Sprintf("Welcome, %s! There are %d users online.", username, len(clients))
	ws.WriteMessage(websocket.TextMessage, []byte(welcomeMsg))

	// Broadcast join message to all clients
	broadcastMessage(fmt.Sprintf("*** %s joined the chat ***", username))

	// Listen for messages from this client
	for {
		_, msg, err := ws.ReadMessage()
		if err != nil {
			log.Printf("Read error: %v", err)
			delete(clients, ws)
			broadcastMessage(fmt.Sprintf("*** %s left the chat ***", username))
			break
		}

		message := string(msg)
		log.Printf("Message from %s: %s", username, message)

		// Handle commands
		if len(message) > 0 && message[0] == '/' {
			handleCommand(ws, username, message)
			continue
		}

		// Broadcast the message
		broadcastMessage(fmt.Sprintf("%s: %s", username, message))
	}
}

// Send a message to all clients
func broadcastMessage(message string) {
	log.Printf("Broadcasting: %s", message)

	for client := range clients {
		err := client.WriteMessage(websocket.TextMessage, []byte(message))
		if err != nil {
			log.Printf("Write error: %v", err)
			client.Close()
			delete(clients, client)
		}
	}
}

// Handle commands
func handleCommand(conn *websocket.Conn, username, cmd string) {
	log.Printf("Command from %s: %s", username, cmd)

	switch cmd {
	case "/help":
		helpMsg := `
Available commands:
/help - Show this help message
/users - List all connected users
/time - Show current server time
`
		conn.WriteMessage(websocket.TextMessage, []byte(helpMsg))

	case "/users":
		userMsg := fmt.Sprintf("Connected users (%d):", len(clients))
		i := 1
		for _, name := range clients {
			userMsg += fmt.Sprintf("\n%d. %s", i, name)
			i++
		}
		conn.WriteMessage(websocket.TextMessage, []byte(userMsg))

	case "/time":
		timeMsg := fmt.Sprintf("Server time: %s", time.Now().Format(time.RFC1123))
		conn.WriteMessage(websocket.TextMessage, []byte(timeMsg))

	default:
		conn.WriteMessage(websocket.TextMessage, []byte("Unknown command. Type /help for available commands."))
	}
}

func main() {
	// Configure logging
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	// Create a simple file server for the web client
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	// WebSocket route
	http.HandleFunc("/ws", handleConnections)

	// Determine port to listen on
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Start the server
	log.Printf("Server starting on :%s", port)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
