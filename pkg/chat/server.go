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
	Send     chan []byte
	Server   *Server
}

// Server manages all active clients and broadcasts messages
type Server struct {
	// Map of active clients
	Clients map[*Client]bool

	// Channel for outbound messages to broadcast
	Broadcast chan []byte

	// Channel for registering clients
	Register chan *Client

	// Channel for unregistering clients
	Unregister chan *Client

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
		Broadcast:      make(chan []byte),
		Register:       make(chan *Client),
		Unregister:     make(chan *Client),
	}
}

// Run starts the server's main event loop
func (s *Server) Run() {
	for {
		select {
		case client := <-s.Register:
			s.registerClient(client)
		case client := <-s.Unregister:
			s.unregisterClient(client)
		case message := <-s.Broadcast:
			s.broadcastMessage(message)
		}
	}
}

// registerClient adds a new client to the server
func (s *Server) registerClient(client *Client) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	s.Clients[client] = true
	s.ClientJoinTime[client] = time.Now()
	log.Printf("Client connected: %s", client.Username)

	// Welcome message
	welcomeMsg := fmt.Sprintf("Welcome %s! There are %d users online. Type /help for available commands.",
		client.Username, len(s.Clients))
	client.Send <- []byte(welcomeMsg)

	// Broadcast join notification
	s.Broadcast <- []byte(fmt.Sprintf("*** %s joined the chat ***", client.Username))
}

// unregisterClient removes a client from the server
func (s *Server) unregisterClient(client *Client) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	if _, ok := s.Clients[client]; ok {
		delete(s.Clients, client)
		delete(s.ClientJoinTime, client)
		close(client.Send)
		log.Printf("Client disconnected: %s", client.Username)

		// Only broadcast if there was actually a client that left
		s.Broadcast <- []byte(fmt.Sprintf("*** %s left the chat ***", client.Username))
	}
}

// broadcastMessage sends a message to all connected clients
func (s *Server) broadcastMessage(message []byte) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	for client := range s.Clients {
		select {
		case client.Send <- message:
			// Message sent successfully
		default:
			// Failed to send, client may be unresponsive
			close(client.Send)
			delete(s.Clients, client)
			delete(s.ClientJoinTime, client)
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
		Send:     make(chan []byte, 256),
		Server:   s,
	}

	s.Register <- client

	// Start goroutines for reading and writing
	go client.WritePump()
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
		c.Server.Unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadDeadline(time.Now().Add(10 * time.Minute))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(10 * time.Minute))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Error: %v", err)
			}
			break
		}

		msgText := string(message)

		// Handle commands
		if strings.HasPrefix(msgText, "/") {
			c.handleCommand(msgText)
			continue
		}

		// Regular message
		formattedMsg := fmt.Sprintf("%s: %s", c.Username, msgText)
		c.Server.Broadcast <- []byte(formattedMsg)
	}
}

// WritePump sends messages to the client connection
func (c *Client) WritePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// Channel closed, server closed the connection
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add any queued messages
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			// Send ping to keep connection alive
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleCommand processes client commands like /help, /users, etc.
func (c *Client) handleCommand(cmd string) {
	if cmd == "/help" {
		helpMsg := `
Available commands:
/help - Show this help message
/users - List all connected users
/time - Show current server time
/exit - Exit the chat
/whisper <username> <message> - Send private message to a user
`
		c.Send <- []byte(helpMsg)
	} else if cmd == "/users" {
		users := c.Server.GetClientList()
		usersMsg := fmt.Sprintf("Connected users (%d):\n", len(users))
		for i, user := range users {
			usersMsg += fmt.Sprintf("%d. %s\n", i+1, user)
		}
		c.Send <- []byte(usersMsg)
	} else if cmd == "/time" {
		c.Send <- []byte(fmt.Sprintf("Server time: %s", time.Now().Format(time.RFC1123)))
	} else if strings.HasPrefix(cmd, "/whisper ") {
		parts := strings.SplitN(cmd[9:], " ", 2)
		if len(parts) != 2 {
			c.Send <- []byte("Usage: /whisper <username> <message>")
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
			c.Send <- []byte(fmt.Sprintf("User '%s' not found", targetUsername))
			return
		}

		// Send to recipient
		targetClient.Send <- []byte(fmt.Sprintf("[PM from %s]: %s", c.Username, message))
		// Confirmation to sender
		c.Send <- []byte(fmt.Sprintf("[PM to %s]: %s", targetUsername, message))
	} else {
		c.Send <- []byte(fmt.Sprintf("Unknown command: %s. Type /help for available commands.", cmd))
	}
}
