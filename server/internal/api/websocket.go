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
		origin := httpRequest.Header.Get("Origin")
		if origin == "" {
			return true
		}
		// Allow any port on localhost or 127.0.0.1
		return strings.Contains(origin, "://localhost") || strings.Contains(origin, "://127.0.0.1")
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
		broadcast:  make(chan WSMessage, 1024), // Buffered to prevent blocking
		register:   make(chan *WSClient),
		unregister: make(chan *WSClient),
	}
}

// Broadcast sends a message to the hub for broadcasting
func (hub *Hub) Broadcast(message WSMessage) {
	// Use a non-blocking send or a timeout to prevent deadlocking the entire server if the hub is stuck
	select {
	case hub.broadcast <- message:
	case <-time.After(5 * time.Second):
		slog.Error("Hub broadcast channel full, message dropped", "type", message.Type, "channel", message.Channel)
	}
}

// Run starts the hub loop
func (hub *Hub) Run() {
	for {
		select {
		case client := <-hub.register:
			hub.mutex.Lock()
			hub.clients[client] = true
			hub.mutex.Unlock()
			slog.Debug("WS client registered", "total", len(hub.clients))

		case client := <-hub.unregister:
			hub.mutex.Lock()
			if _, ok := hub.clients[client]; ok {
				delete(hub.clients, client)
				close(client.send)
			}
			hub.mutex.Unlock()
			slog.Debug("WS client unregistered", "total", len(hub.clients))

		case wsMessage := <-hub.broadcast:
			hub.mutex.RLock()
			var toRemove []*WSClient
			sentCount := 0

			for client := range hub.clients {
				if client.isSubscribed(wsMessage.Channel) {
					select {
					case client.send <- wsMessage:
						sentCount++
					default:
						// If send channel is full, mark client for removal
						toRemove = append(toRemove, client)
					}
				}
			}
			hub.mutex.RUnlock()

			// Perform removals outside of RLock to avoid concurrent map modification
			if len(toRemove) > 0 {
				hub.mutex.Lock()
				for _, client := range toRemove {
					if _, ok := hub.clients[client]; ok {
						delete(hub.clients, client)
						close(client.send)
						slog.Warn("WS client disconnected due to full buffer", "userID", client.userID)
					}
				}
				hub.mutex.Unlock()
			}

			if sentCount > 0 || len(toRemove) > 0 {
				slog.Debug("Broadcast delivered", "type", wsMessage.Type, "channel", wsMessage.Channel, "sent", sentCount, "dropped", len(toRemove))
			}
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
		server.writeError(responseWriter, http.StatusUnauthorized, "AUTHENTICATION_ERROR", "Authentication required", nil)
		return
	}

	var userID string
	var expiresAt time.Time
	databaseError := server.database.QueryRow("SELECT user_id, expires_at FROM auth_sessions WHERE id = ?", sessionToken).Scan(&userID, &expiresAt)
	if databaseError != nil || time.Now().After(expiresAt) {
		server.writeError(responseWriter, http.StatusUnauthorized, "AUTHENTICATION_ERROR", "Invalid or expired session", nil)
		return
	}

	connection, upgradeError := upgrader.Upgrade(responseWriter, request, nil)
	if upgradeError != nil {
		slog.Error("WebSocket upgrade failed", "error", upgradeError)
		return
	}

	client := &WSClient{
		hub:           server.wsHub,
		server:        server,
		connection:    connection,
		send:          make(chan any, 512), // Larger buffer
		subscriptions: make(map[string]chan bool),
		userID:        userID,
	}

	// Auto-subscribe to chat session if provided in query
	if autoChatID := request.URL.Query().Get("subscribe_chat"); autoChatID != "" {
		var exists bool
		server.database.QueryRow("SELECT EXISTS(SELECT 1 FROM chat_sessions JOIN exams ON chat_sessions.exam_id = exams.id WHERE chat_sessions.id = ? AND exams.user_id = ?)", autoChatID, userID).Scan(&exists)
		if exists {
			slog.Info("Auto-subscribing to chat", "sessionID", autoChatID, "userID", userID)
			client.subscriptions["chat:"+autoChatID] = make(chan bool)
		}
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
			// Set a write deadline for every message
			client.connection.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !isAvailable {
				client.connection.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := client.connection.WriteJSON(wsMessage); err != nil {
				// Don't log normal closure errors
				if !websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					return
				}
				slog.Error("WS write error", "userID", client.userID, "error", err)
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

	// Security Check: Ensure user owns the resource they are subscribing to
	if strings.HasPrefix(channel, "lecture:") {
		lectureID := strings.TrimPrefix(channel, "lecture:")
		var exists bool
		client.server.database.QueryRow(`
			SELECT EXISTS(
				SELECT 1 FROM lectures 
				JOIN exams ON lectures.exam_id = exams.id 
				WHERE lectures.id = ? AND exams.user_id = ?
			)
		`, lectureID, client.userID).Scan(&exists)
		if !exists {
			slog.Warn("Unauthorized subscription attempt to lecture", "userID", client.userID, "lectureID", lectureID)
			return
		}
	} else if strings.HasPrefix(channel, "course:") {
		courseID := strings.TrimPrefix(channel, "course:")
		var exists bool
		client.server.database.QueryRow("SELECT EXISTS(SELECT 1 FROM exams WHERE id = ? AND user_id = ?)", courseID, client.userID).Scan(&exists)
		if !exists {
			slog.Warn("Unauthorized subscription attempt to course", "userID", client.userID, "courseID", courseID)
			return
		}
	} else if strings.HasPrefix(channel, "job:") {
		jobID := strings.TrimPrefix(channel, "job:")
		job, err := client.server.jobQueue.GetJob(jobID)
		if err != nil || job.UserID != client.userID {
			slog.Warn("Unauthorized subscription attempt to job", "userID", client.userID, "jobID", jobID)
			return
		}
	} else if strings.HasPrefix(channel, "chat:") {
		chatID := strings.TrimPrefix(channel, "chat:")
		var exists bool
		client.server.database.QueryRow("SELECT EXISTS(SELECT 1 FROM chat_sessions JOIN exams ON chat_sessions.exam_id = exams.id WHERE chat_sessions.id = ? AND exams.user_id = ?)", chatID, client.userID).Scan(&exists)
		if !exists {
			slog.Warn("Unauthorized subscription attempt to chat", "userID", client.userID, "chatID", chatID)
			return
		}
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
	client.mutex.Lock()
	defer client.mutex.Unlock()

	if stopChannel, exists := client.subscriptions[channel]; exists {
		close(stopChannel)
		delete(client.subscriptions, channel)
	}
}

func (client *WSClient) monitorJob(jobID string, stopChannel chan bool) {
	job, err := client.server.jobQueue.GetJob(jobID)
	if err != nil || job.UserID != client.userID {
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
