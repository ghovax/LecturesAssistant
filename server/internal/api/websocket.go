package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Hub maintains the set of active clients and broadcasts messages
type Hub struct {
	clients    map[*WSClient]bool
	broadcast  chan WSMessage
	register   chan *WSClient
	unregister chan *WSClient
	mu         sync.RWMutex
}

// WSMessage represents a message to be broadcast
type WSMessage struct {
	Type      string `json:"type"`
	Channel   string `json:"channel"`
	Payload   any    `json:"payload"`
	Timestamp string `json:"timestamp"`
}

// NewHub creates a new WebSocket hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*WSClient]bool),
		broadcast:  make(chan WSMessage),
		register:   make(chan *WSClient),
		unregister: make(chan *WSClient),
	}
}

// Broadcast sends a message to the hub for broadcasting
func (hub *Hub) Broadcast(message WSMessage) {
	hub.broadcast <- message
}

// Run starts the hub loop
func (hub *Hub) Run() {
	for {
		select {
		case client := <-hub.register:
			hub.mu.Lock()
			hub.clients[client] = true
			hub.mu.Unlock()
		case client := <-hub.unregister:
			hub.mu.Lock()
			if _, ok := hub.clients[client]; ok {
				delete(hub.clients, client)
				close(client.send)
			}
			hub.mu.Unlock()
		case message := <-hub.broadcast:
			hub.mu.RLock()
			for client := range hub.clients {
				if client.isSubscribed(message.Channel) {
					select {
					case client.send <- message:
					default:
						close(client.send)
						delete(hub.clients, client)
					}
				}
			}
			hub.mu.RUnlock()
		}
	}
}

// WSClient represents a connected WebSocket client
type WSClient struct {
	hub           *Hub
	server        *Server
	conn          *websocket.Conn
	send          chan any
	subscriptions map[string]chan bool
	mu            sync.Mutex
}

func (client *WSClient) isSubscribed(channel string) bool {
	client.mu.Lock()
	defer client.mu.Unlock()
	_, exists := client.subscriptions[channel]
	return exists
}

// handleWebSocket handles the WebSocket connection upgrade
func (server *Server) handleWebSocket(responseWriter http.ResponseWriter, request *http.Request) {
	sessionToken := server.getSessionToken(request)
	if sessionToken == "" {
		server.writeError(responseWriter, http.StatusUnauthorized, "AUTH_ERROR", "Authentication required", nil)
		return
	}

	// Verify session
	var expiresAt time.Time
	databaseError := server.database.QueryRow("SELECT expires_at FROM auth_sessions WHERE id = ?", sessionToken).Scan(&expiresAt)
	if databaseError != nil || time.Now().After(expiresAt) {
		server.writeError(responseWriter, http.StatusUnauthorized, "AUTH_ERROR", "Invalid or expired session", nil)
		return
	}

	conn, upgradeError := upgrader.Upgrade(responseWriter, request, nil)
	if upgradeError != nil {
		slog.Error("WebSocket upgrade failed", "error", upgradeError)
		return
	}

	client := &WSClient{
		hub:           server.wsHub,
		server:        server,
		conn:          conn,
		send:          make(chan any, 256),
		subscriptions: make(map[string]chan bool),
	}
	client.hub.register <- client

	// Send handshake
	client.send <- map[string]any{
		"type":           "connected",
		"timestamp":      time.Now().Format(time.RFC3339),
		"server_version": "1.0.0",
	}

	go client.writePump()
	go client.readPump()
}

func (client *WSClient) readPump() {
	defer func() {
		client.hub.unregister <- client
		client.close()
	}()

	for {
		_, message, err := client.conn.ReadMessage()
		if err != nil {
			break
		}

		var envelope struct {
			Type    string `json:"type"`
			Channel string `json:"channel"`
			Payload any    `json:"payload"`
		}

		if err := json.Unmarshal(message, &envelope); err != nil {
			continue
		}

		switch envelope.Type {
		case "subscribe":
			client.handleSubscribe(envelope.Channel)
		case "unsubscribe":
			client.handleUnsubscribe(envelope.Channel)
		}
	}
}

func (client *WSClient) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		client.conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.send:
			if !ok {
				client.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := client.conn.WriteJSON(message); err != nil {
				return
			}

		case <-ticker.C:
			if err := client.conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
				return
			}
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (client *WSClient) handleSubscribe(channel string) {
	client.mu.Lock()
	defer client.mu.Unlock()

	if _, exists := client.subscriptions[channel]; exists {
		return
	}

	stopChannel := make(chan bool)
	client.subscriptions[channel] = stopChannel

	if len(channel) > 4 && channel[:4] == "job:" {
		jobID := channel[4:]
		go client.monitorJob(jobID, stopChannel)
	}

	client.send <- map[string]any{
		"type":      "subscribed",
		"channel":   channel,
		"timestamp": time.Now().Format(time.RFC3339),
	}
}

func (client *WSClient) handleUnsubscribe(channel string) {
	client.mu.Lock()
	defer client.mu.Unlock()

	if stopChannel, exists := client.subscriptions[channel]; exists {
		close(stopChannel)
		delete(client.subscriptions, channel)
	}
}

func (client *WSClient) monitorJob(jobID string, stopChannel chan bool) {
	jobUpdates := client.server.jobQueue.Subscribe(jobID)
	defer client.server.jobQueue.Unsubscribe(jobID, jobUpdates)

	for {
		select {
		case <-stopChannel:
			return
		case update, ok := <-jobUpdates:
			if !ok {
				return
			}
			client.send <- WSMessage{
				Type:      "job:progress",
				Channel:   "job:" + jobID,
				Payload:   update,
				Timestamp: time.Now().Format(time.RFC3339),
			}
			if update.Status == "COMPLETED" || update.Status == "FAILED" || update.Status == "CANCELLED" {
				return
			}
		}
	}
}

func (client *WSClient) close() {
	client.mu.Lock()
	defer client.mu.Unlock()

	for channel, stopChannel := range client.subscriptions {
		close(stopChannel)
		delete(client.subscriptions, channel)
	}
	client.conn.Close()
}
