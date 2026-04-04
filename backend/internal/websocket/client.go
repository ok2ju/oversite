package websocket

import (
	"log/slog"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = 54 * time.Second
	maxMessageSize = 512 * 1024 // 512 KB
	sendBufferSize = 256
)

// Client represents a single WebSocket connection to a board room.
type Client struct {
	hub     *Hub
	conn    *websocket.Conn
	boardID string
	userID  string
	send    chan []byte
}

// NewClient creates a new Client.
func NewClient(hub *Hub, conn *websocket.Conn, boardID, userID string) *Client {
	return &Client{
		hub:     hub,
		conn:    conn,
		boardID: boardID,
		userID:  userID,
		send:    make(chan []byte, sendBufferSize),
	}
}

// ReadPump reads binary messages from the WebSocket connection and pushes
// them to the hub for broadcast. It runs in its own goroutine per client.
func (c *Client) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		_ = c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		msgType, data, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				slog.Warn("websocket read error", "board_id", c.boardID, "user_id", c.userID, "error", err)
			}
			return
		}

		// Only relay binary messages (Yjs protocol uses binary frames).
		if msgType != websocket.BinaryMessage {
			continue
		}

		c.hub.broadcast <- &Message{
			sender:  c,
			boardID: c.boardID,
			data:    data,
		}
	}
}

// WritePump pumps messages from the send channel to the WebSocket connection.
// It runs in its own goroutine per client.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel.
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.BinaryMessage, msg); err != nil {
				return
			}

		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
