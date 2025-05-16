// pkg/chat/server.go
package chat

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Client represents a connected chat user
type Client struct {
	Conn     *websocket.Conn
	Username string
	Server   *Server
}

// Server manages all active clients
type Server struct {
	// Map of active clients
	Clients map[*Client]bool

	// Mutex to protect clients map from concurrent access
	Mutex sync.Mutex

	// Keep track of when clients joined
	ClientJoinTime map[*Client]time.Time
}

// Upgrader converts HTTP connections to WebSocket connections
var Upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all connections for simplicity
	},
}

// NewServer creates a new chat server instance
func NewServer() *Server {
	return &Server{
		Clients:        make(map[*Client]bool),
		ClientJoinTime: make(map[*Client]time.Time),
	}
}

// Run starts the server's main process (keeping for compatibility)
func (s *Server) Run() {
	log.Println("Server running and ready for connections")
	// This method is now mostly for compatibility - the real work happens in the WebSocket handlers
}

// broadcastMessage sends a message to all connected clients
func (s *Server) broadcastMessage(message string) {
	log.Printf("Broadcasting: %s", message)

	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	for client := range s.Clients {
		err := client.Conn.WriteMessage(websocket.TextMessage, []byte(message))
		if err != nil {
			log.Printf("Error sending to client %s: %v", client.Username, err)
			// Will be removed in ReadPump when connection error is detected
		}
	}
}

// HandleWebSocket upgrades HTTP connections to WebSocket
func (s *Server) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := Upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error upgrading connection:", err)
		return
	}

	// Get username first
	_, usernameMsg, err := conn.ReadMessage()
	if err != nil {
		conn.Close()
		log.Println("Error reading username:", err)
		return
	}

	username := string(usernameMsg)
	log.Printf("User connecting: %s", username)

	// Check if username is already taken
	s.Mutex.Lock()
	usernameTaken := false
	for client := range s.Clients {
		if strings.EqualFold(client.Username, username) {
			usernameTaken = true
			break
		}
	}
	s.Mutex.Unlock()

	if usernameTaken {
		// Notify client that username is taken
		conn.WriteMessage(websocket.TextMessage, []byte("ERROR: Username already taken. Please try again with a different name."))
		conn.Close()
		return
	}

	client := &Client{
		Conn:     conn,
		Username: username,
		Server:   s,
	}

	// Register client
	s.Mutex.Lock()
	s.Clients[client] = true
	s.ClientJoinTime[client] = time.Now()
	s.Mutex.Unlock()

	log.Printf("Client connected: %s", client.Username)

	// Send welcome message
	welcomeMsg := fmt.Sprintf("Welcome %s! There are %d users online. Type /help for available commands.",
		client.Username, len(s.Clients))
	client.Conn.WriteMessage(websocket.TextMessage, []byte(welcomeMsg))

	// Broadcast join notification
	s.broadcastMessage(fmt.Sprintf("*** %s joined the chat ***", client.Username))

	// Start the reading goroutine
	go client.ReadPump()
}

// GetClientList returns a list of all connected usernames
func (s *Server) GetClientList() []string {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	users := make([]string, 0, len(s.Clients))
	for client := range s.Clients {
		duration := time.Since(s.ClientJoinTime[client]).Round(time.Second)
		users = append(users, fmt.Sprintf("%s (connected for %s)", client.Username, duration))
	}
	return users
}

// ReadPump reads messages from the client connection
func (c *Client) ReadPump() {
	defer func() {
		// Unregister client on disconnect
		c.Server.Mutex.Lock()
		delete(c.Server.Clients, c)
		delete(c.Server.ClientJoinTime, c)
		c.Server.Mutex.Unlock()

		log.Printf("Client disconnected: %s", c.Username)
		c.Server.broadcastMessage(fmt.Sprintf("*** %s left the chat ***", c.Username))
		c.Conn.Close()
	}()

	// Setup ping/pong for keeping connection alive
	c.Conn.SetReadDeadline(time.Now().Add(10 * time.Minute))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(10 * time.Minute))
		return nil
	})

	// Start a ticker to send pings
	ticker := time.NewTicker(30 * time.Second)
	go func() {
		defer ticker.Stop()
		for range ticker.C {
			if err := c.Conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second)); err != nil {
				return
			}
		}
	}()

	// Main message loop
	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Error: %v", err)
			}
			break
		}

		msgText := string(message)
		log.Printf("Received from %s: %s", c.Username, msgText)

		// Handle commands
		if strings.HasPrefix(msgText, "/") {
			c.handleCommand(msgText)
			continue
		}

		// Regular message
		formattedMsg := fmt.Sprintf("%s: %s", c.Username, msgText)
		c.Server.broadcastMessage(formattedMsg)
	}
}

// handleCommand processes client commands like /help, /users, etc.
func (c *Client) handleCommand(cmd string) {
	log.Printf("Command from %s: %s", c.Username, cmd)

	if cmd == "/help" {
		helpMsg := `
Available commands:
/help - Show this help message
/users - List all connected users
/time - Show current server time
/exit - Exit the chat
/whisper <username> <message> - Send private message to a user
`
		c.Conn.WriteMessage(websocket.TextMessage, []byte(helpMsg))
	} else if cmd == "/users" {
		users := c.Server.GetClientList()
		usersMsg := fmt.Sprintf("Connected users (%d):\n", len(users))
		for i, user := range users {
			usersMsg += fmt.Sprintf("%d. %s\n", i+1, user)
		}
		c.Conn.WriteMessage(websocket.TextMessage, []byte(usersMsg))
	} else if cmd == "/time" {
		c.Conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Server time: %s", time.Now().Format(time.RFC1123))))
	} else if strings.HasPrefix(cmd, "/whisper ") {
		parts := strings.SplitN(cmd[9:], " ", 2)
		if len(parts) != 2 {
			c.Conn.WriteMessage(websocket.TextMessage, []byte("Usage: /whisper <username> <message>"))
			return
		}

		targetUsername := strings.TrimSpace(parts[0])
		message := parts[1]

		c.Server.Mutex.Lock()
		var targetClient *Client
		for client := range c.Server.Clients {
			if strings.EqualFold(client.Username, targetUsername) {
				targetClient = client
				break
			}
		}
		c.Server.Mutex.Unlock()

		if targetClient == nil {
			c.Conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("User '%s' not found", targetUsername)))
			return
		}

		// Send to recipient
		targetClient.Conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("[PM from %s]: %s", c.Username, message)))
		// Confirmation to sender
		c.Conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("[PM to %s]: %s", targetUsername, message)))
	} else {
		c.Conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Unknown command: %s. Type /help for available commands.", cmd)))
	}
}
