package service

import (
	"encoding/json"
	"github.com/shiva/ai-match/pkg/logger"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/shiva/ai-match/internal/models"
	"github.com/shiva/ai-match/internal/repository"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 4096
)

// Client represents a connected WebSocket user.
type Client struct {
	Hub      *Hub
	Conn     *websocket.Conn
	Send     chan []byte
	Username string
	RoomID   string
}

// Hub manages all WebSocket connections and message broadcasting.
type Hub struct {
	mu         sync.RWMutex
	Clients    map[*Client]bool
	Broadcast  chan []byte
	Register   chan *Client
	Unregister chan *Client
	Rooms      map[string]map[*Client]bool // roomID -> set of clients
	FakeDB     *repository.FakeDB
	PgDB       *repository.PostgresDB
	usePostgres bool
}

// NewHub creates a new WebSocket hub.
func NewHub(fakeDB *repository.FakeDB, pgDB *repository.PostgresDB) *Hub {
	return &Hub{
		Clients:    make(map[*Client]bool),
		Broadcast:  make(chan []byte, 256),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Rooms:      make(map[string]map[*Client]bool),
		FakeDB:     fakeDB,
		PgDB:       pgDB,
		usePostgres: pgDB != nil,
	}
}

// Run starts the hub's main event loop.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			h.Clients[client] = true
			if _, ok := h.Rooms[client.RoomID]; !ok {
				h.Rooms[client.RoomID] = make(map[*Client]bool)
			}
			h.Rooms[client.RoomID][client] = true
			h.mu.Unlock()

			// Broadcast join message
			joinMsg := models.WSMessage{
				Type:     "system",
				Content:  client.Username + " joined the room 🎉",
				Username: "system",
				RoomID:   client.RoomID,
			}
			data, _ := json.Marshal(joinMsg)
			h.BroadcastToRoom(client.RoomID, data)

			logger.Printf("✅ %s joined room %s", client.Username, client.RoomID)

		case client := <-h.Unregister:
			h.mu.Lock()
			if _, ok := h.Clients[client]; ok {
				delete(h.Clients, client)
				if roomClients, ok := h.Rooms[client.RoomID]; ok {
					delete(roomClients, client)
				}
				close(client.Send)
			}
			h.mu.Unlock()

			// Broadcast leave message
			leaveMsg := models.WSMessage{
				Type:     "system",
				Content:  client.Username + " left the room 👋",
				Username: "system",
				RoomID:   client.RoomID,
			}
			data, _ := json.Marshal(leaveMsg)
			h.BroadcastToRoom(client.RoomID, data)

			logger.Printf("❌ %s left room %s", client.Username, client.RoomID)

		case message := <-h.Broadcast:
			var wsMsg models.WSMessage
			if err := json.Unmarshal(message, &wsMsg); err == nil {
				h.BroadcastToRoom(wsMsg.RoomID, message)
			}
		}
	}
}

// BroadcastToRoom sends a message to all clients in a room.
func (h *Hub) BroadcastToRoom(roomID string, message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if clients, ok := h.Rooms[roomID]; ok {
		for client := range clients {
			select {
			case client.Send <- message:
			default:
				close(client.Send)
				delete(clients, client)
				delete(h.Clients, client)
			}
		}
	}
}

// GetOnlineCount returns the number of connected clients.
func (h *Hub) GetOnlineCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.Clients)
}

// GetRoomCount returns the number of clients in a specific room.
func (h *Hub) GetRoomCount(roomID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if clients, ok := h.Rooms[roomID]; ok {
		return len(clients)
	}
	return 0
}

// ReadPump pumps messages from the WebSocket connection to the hub.
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Parse the incoming message
		var wsMsg models.WSMessage
		if err := json.Unmarshal(message, &wsMsg); err != nil {
			continue
		}

		wsMsg.Username = c.Username
		wsMsg.RoomID = c.RoomID

		// Save message to DB
		dbMsg := models.Message{
			SenderID: c.Username,
			Username: c.Username,
			Content:  wsMsg.Content,
			RoomID:   c.RoomID,
		}
		
		if c.Hub.usePostgres {
			c.Hub.PgDB.SaveMessage(dbMsg)
		} else {
			c.Hub.FakeDB.SaveMessage(dbMsg)
		}

		// Re-marshal and broadcast
		data, _ := json.Marshal(wsMsg)
		c.Hub.Broadcast <- data
	}
}

// WritePump pumps messages from the hub to the WebSocket connection.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
