// server.go - Run this on a server with a public IP or cloud service
package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type Client struct {
	conn     *websocket.Conn
	username string
	send     chan []byte
}

type Server struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mutex      sync.Mutex
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all connections
	},
}

func newServer() *Server {
	return &Server{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (s *Server) run() {
	for {
		select {
		case client := <-s.register:
			s.mutex.Lock()
			s.clients[client] = true
			s.mutex.Unlock()
			log.Printf("Client connected: %s", client.username)
			s.broadcast <- []byte(fmt.Sprintf("** %s joined the chat **", client.username))
		case client := <-s.unregister:
			s.mutex.Lock()
			if _, ok := s.clients[client]; ok {
				delete(s.clients, client)
				close(client.send)
				log.Printf("Client disconnected: %s", client.username)
				s.broadcast <- []byte(fmt.Sprintf("** %s left the chat **", client.username))
			}
			s.mutex.Unlock()
		case message := <-s.broadcast:
			s.mutex.Lock()
			for client := range s.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(s.clients, client)
				}
			}
			s.mutex.Unlock()
		}
	}
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
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

	client := &Client{
		conn:     conn,
		username: string(usernameMsg),
		send:     make(chan []byte, 256),
	}
	s.register <- client

	// Start goroutines for reading and writing
	go client.writePump(s)
	go client.readPump(s)
}

func (c *Client) readPump(s *Server) {
	defer func() {
		s.unregister <- c
		c.conn.Close()
	}()

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			log.Println("Read error:", err)
			break
		}
		formattedMsg := fmt.Sprintf("%s: %s", c.username, message)
		s.broadcast <- []byte(formattedMsg)
	}
}

func (c *Client) writePump(s *Server) {
	defer c.conn.Close()

	for {
		message, ok := <-c.send
		if !ok {
			c.conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		}

		if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
			log.Println("Write error:", err)
			return
		}
	}
}

func main() {
	server := newServer()
	go server.run()

	http.HandleFunc("/ws", server.handleWebSocket)

	port := ":8080"
	log.Printf("Server starting on %s", port)
	log.Fatal(http.ListenAndServe(port, nil))
}
