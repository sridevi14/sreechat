package hub

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/sridhar/sreechat/internal/models"
)

type Client struct {
	ID     string
	UserID string
	RoomID string
	Conn   *websocket.Conn
	Send   chan []byte
}

type Hub struct {
	mu         sync.RWMutex
	rooms      map[string]map[string]*Client // roomID -> clientID -> Client
	Register   chan *Client
	Unregister chan *Client
	Broadcast  chan *RoomMessage
}

type RoomMessage struct {
	RoomID string
	Data   []byte
}

func NewHub() *Hub {
	return &Hub{
		rooms:      make(map[string]map[string]*Client),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Broadcast:  make(chan *RoomMessage, 256),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			if _, ok := h.rooms[client.RoomID]; !ok {
				h.rooms[client.RoomID] = make(map[string]*Client)
			}
			h.rooms[client.RoomID][client.ID] = client
			h.mu.Unlock()
			log.Printf("client %s joined room %s", client.UserID, client.RoomID)

		case client := <-h.Unregister:
			h.mu.Lock()
			if clients, ok := h.rooms[client.RoomID]; ok {
				if _, exists := clients[client.ID]; exists {
					delete(clients, client.ID)
					close(client.Send)
					if len(clients) == 0 {
						delete(h.rooms, client.RoomID)
					}
				}
			}
			h.mu.Unlock()
			log.Printf("client %s left room %s", client.UserID, client.RoomID)

		case msg := <-h.Broadcast:
			h.mu.RLock()
			if clients, ok := h.rooms[msg.RoomID]; ok {
				for _, client := range clients {
					select {
					case client.Send <- msg.Data:
					default:
						close(client.Send)
						delete(clients, client.ID)
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

// BroadcastToRoom sends a message to all local clients in a room.
// Called by the Redis subscriber when a message arrives via PubSub.
func (h *Hub) BroadcastToRoom(roomID string, msg *models.WSMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("hub: marshal error: %v", err)
		return
	}
	h.Broadcast <- &RoomMessage{RoomID: roomID, Data: data}
}

// WritePump drains the Send channel to the WebSocket connection.
func WritePump(client *Client) {
	defer client.Conn.Close()
	for msg := range client.Send {
		if err := client.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			return
		}
	}
}
