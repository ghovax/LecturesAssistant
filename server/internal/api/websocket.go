package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(httpRequest *http.Request) bool {
		// Strict check: only allow localhost in development
		origin := httpRequest.Header.Get("Origin")
		if origin == "" {
			return true
		}
		// Allow localhost and 127.0.0.1
		return strings.HasPrefix(origin, "http://localhost") || strings.HasPrefix(origin, "http://127.0.0.1")
	},
}

// Hub maintains the set of active clients and broadcasts messages
type Hub struct {
	clients    map[*WSClient]bool
	broadcast  chan WSMessage
	register   chan *WSClient
	unregister chan *WSClient
	mutex      sync.RWMutex
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
			hub.mutex.Lock()
			hub.clients[client] = true
			hub.mutex.Unlock()
		case client := <-hub.unregister:
			hub.mutex.Lock()
			if _, isRegistered := hub.clients[client]; isRegistered {
				delete(hub.clients, client)
				close(client.send)
			}
			hub.mutex.Unlock()
		case wsMessage := <-hub.broadcast:
			hub.mutex.RLock()
			for client := range hub.clients {
				if client.isSubscribed(wsMessage.Channel) {
					select {
					case client.send <- wsMessage:
					default:
						close(client.send)
						delete(hub.clients, client)
					}
				}
			}
			hub.mutex.RUnlock()
		}
	}
}

// WSClient represents a connected WebSocket client
type WSClient struct {
	hub           *Hub
	server        *Server
	connection    *websocket.Conn
	send          chan any
	subscriptions map[string]chan bool
	userID        string
	mutex         sync.Mutex
}

func (client *WSClient) isSubscribed(channel string) bool {
	client.mutex.Lock()
	defer client.mutex.Unlock()
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

	// Verify session and get user ID
	var userID string
	var expiresAt time.Time
	databaseError := server.database.QueryRow("SELECT user_id, expires_at FROM auth_sessions WHERE id = ?", sessionToken).Scan(&userID, &expiresAt)
	if databaseError != nil || time.Now().After(expiresAt) {
		server.writeError(responseWriter, http.StatusUnauthorized, "AUTH_ERROR", "Invalid or expired session", nil)
		return
	}

	// Correctly set user ID in context before upgrade if needed (though we'll use local variable)

	connection, upgradeError := upgrader.Upgrade(responseWriter, request, nil)
	if upgradeError != nil {
		slog.Error("WebSocket upgrade failed", "error", upgradeError)
		return
	}

	client := &WSClient{
		hub:           server.wsHub,
		server:        server,
		connection:    connection,
		send:          make(chan any, 256),
		subscriptions: make(map[string]chan bool),
		userID:        userID, // Add this field to WSClient
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
		_, message, err := client.connection.ReadMessage()
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
		client.connection.Close()
	}()

	for {
		select {
		case wsMessage, isAvailable := <-client.send:
			if !isAvailable {
				client.connection.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := client.connection.WriteJSON(wsMessage); err != nil {
				return
			}

		case <-ticker.C:
			if err := client.connection.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
				return
			}
			if err := client.connection.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (client *WSClient) handleSubscribe(channel string) {
	client.mutex.Lock()
	defer client.mutex.Unlock()

	if _, exists := client.subscriptions[channel]; exists {
		return
	}

	stopChannel := make(chan bool)
	client.subscriptions[channel] = stopChannel

	if len(channel) > 4 && channel[:4] == "job:" {
		jobID := channel[4:]
		go client.monitorJob(jobID, stopChannel)
	}

	if len(channel) > 8 && channel[:8] == "lecture:" {
		// Generic lecture channel, just used for broadcast
	}

	if len(channel) > 7 && channel[:7] == "course:" {
		// Generic course channel, just used for broadcast
	}

	client.send <- map[string]any{
		"type":      "subscribed",
		"channel":   channel,
		"timestamp": time.Now().Format(time.RFC3339),
	}
}

func (client *WSClient) handleUnsubscribe(channel string) {
	client.mutex.Lock()
	defer client.mutex.Unlock()

	if stopChannel, exists := client.subscriptions[channel]; exists {
		close(stopChannel)
		delete(client.subscriptions, channel)
	}
}

func (client *WSClient) monitorJob(jobID string, stopChannel chan bool) {
	// Verify job ownership
	job, err := client.server.jobQueue.GetJob(jobID)
	if err != nil || job.UserID != client.userID {
		slog.Warn("Unauthorized job subscription attempt", "jobID", jobID, "userID", client.userID)
		return
	}

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
	client.mutex.Lock()
	defer client.mutex.Unlock()

	for channel, stopChannel := range client.subscriptions {
		close(stopChannel)
		delete(client.subscriptions, channel)
	}
	client.connection.Close()
}
