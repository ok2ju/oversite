package websocket

import (
	"sync"
)

// Message represents a broadcast message from a client.
type Message struct {
	sender  *Client
	boardID string
	data    []byte
}

// Room holds clients connected to the same board.
type Room struct {
	clients map[*Client]bool
}

func newRoom() *Room {
	return &Room{clients: make(map[*Client]bool)}
}

func (r *Room) add(c *Client) {
	r.clients[c] = true
}

func (r *Room) remove(c *Client) {
	delete(r.clients, c)
}

func (r *Room) isEmpty() bool {
	return len(r.clients) == 0
}

func (r *Room) count() int {
	return len(r.clients)
}

func (r *Room) broadcastExcept(sender *Client, data []byte) {
	for c := range r.clients {
		if c == sender {
			continue
		}
		select {
		case c.send <- data:
		default:
			// Client send buffer full — drop message to avoid blocking.
		}
	}
}

// Hub maintains the set of active clients and broadcasts messages to rooms.
type Hub struct {
	rooms      map[string]*Room
	register   chan *Client
	unregister chan *Client
	broadcast  chan *Message
	mu         sync.RWMutex
}

// NewHub creates a new Hub.
func NewHub() *Hub {
	return &Hub{
		rooms:      make(map[string]*Room),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *Message),
	}
}

// Run starts the hub event loop. It should be launched in a goroutine.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			room, ok := h.rooms[client.boardID]
			if !ok {
				room = newRoom()
				h.rooms[client.boardID] = room
			}
			room.add(client)
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if room, ok := h.rooms[client.boardID]; ok {
				if room.clients[client] {
					room.remove(client)
					close(client.send)
					if room.isEmpty() {
						delete(h.rooms, client.boardID)
					}
				}
			}
			h.mu.Unlock()

		case msg := <-h.broadcast:
			h.mu.RLock()
			if room, ok := h.rooms[msg.boardID]; ok {
				room.broadcastExcept(msg.sender, msg.data)
			}
			h.mu.RUnlock()
		}
	}
}

// ClientsInRoom returns the number of clients in the given room.
func (h *Hub) ClientsInRoom(boardID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if room, ok := h.rooms[boardID]; ok {
		return room.count()
	}
	return 0
}

// RoomCount returns the number of active rooms.
func (h *Hub) RoomCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.rooms)
}
