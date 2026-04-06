package websocket

import (
	"bytes"
	"testing"
	"time"
)

// newTestClient creates a Client with nil conn and a buffered send channel for testing.
func newTestClient(hub *Hub, boardID string) *Client {
	return &Client{
		hub:     hub,
		conn:    nil,
		boardID: boardID,
		userID:  "test-user",
		send:    make(chan []byte, sendBufferSize),
	}
}

// waitForHub polls the predicate with a short deadline for hub goroutine synchronization.
func waitForHub(t *testing.T, predicate func() bool) {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		if predicate() {
			return
		}
		select {
		case <-deadline:
			t.Fatal("timed out waiting for hub state")
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}
}

func TestHub_RegisterClient(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	c := newTestClient(hub, "board-1")
	hub.register <- c

	waitForHub(t, func() bool { return hub.ClientsInRoom("board-1") == 1 })

	if got := hub.ClientsInRoom("board-1"); got != 1 {
		t.Errorf("ClientsInRoom = %d, want 1", got)
	}
}

func TestHub_RegisterMultipleClients_SameRoom(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	c1 := newTestClient(hub, "board-1")
	c2 := newTestClient(hub, "board-1")
	hub.register <- c1
	hub.register <- c2

	waitForHub(t, func() bool { return hub.ClientsInRoom("board-1") == 2 })

	if got := hub.ClientsInRoom("board-1"); got != 2 {
		t.Errorf("ClientsInRoom = %d, want 2", got)
	}
}

func TestHub_RegisterClients_DifferentRooms(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	c1 := newTestClient(hub, "board-1")
	c2 := newTestClient(hub, "board-2")
	hub.register <- c1
	hub.register <- c2

	waitForHub(t, func() bool {
		return hub.ClientsInRoom("board-1") == 1 && hub.ClientsInRoom("board-2") == 1
	})

	if got := hub.ClientsInRoom("board-1"); got != 1 {
		t.Errorf("ClientsInRoom(board-1) = %d, want 1", got)
	}
	if got := hub.ClientsInRoom("board-2"); got != 1 {
		t.Errorf("ClientsInRoom(board-2) = %d, want 1", got)
	}
	if got := hub.RoomCount(); got != 2 {
		t.Errorf("RoomCount = %d, want 2", got)
	}
}

func TestHub_UnregisterClient(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	c1 := newTestClient(hub, "board-1")
	c2 := newTestClient(hub, "board-1")
	hub.register <- c1
	hub.register <- c2
	waitForHub(t, func() bool { return hub.ClientsInRoom("board-1") == 2 })

	hub.unregister <- c1
	waitForHub(t, func() bool { return hub.ClientsInRoom("board-1") == 1 })

	if got := hub.ClientsInRoom("board-1"); got != 1 {
		t.Errorf("ClientsInRoom = %d, want 1", got)
	}
}

func TestHub_UnregisterLastClient_CleansUpRoom(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	c := newTestClient(hub, "board-1")
	hub.register <- c
	waitForHub(t, func() bool { return hub.ClientsInRoom("board-1") == 1 })

	hub.unregister <- c
	waitForHub(t, func() bool { return hub.RoomCount() == 0 })

	if got := hub.RoomCount(); got != 0 {
		t.Errorf("RoomCount = %d, want 0", got)
	}
}

func TestHub_UnregisterUnknownClient(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Register a real client first so the hub is active.
	real := newTestClient(hub, "board-1")
	hub.register <- real
	waitForHub(t, func() bool { return hub.ClientsInRoom("board-1") == 1 })

	// Unregister a client that was never registered — should not panic.
	unknown := newTestClient(hub, "board-99")
	hub.unregister <- unknown

	// Verify the real client is still there.
	waitForHub(t, func() bool { return hub.ClientsInRoom("board-1") == 1 })
	if got := hub.ClientsInRoom("board-1"); got != 1 {
		t.Errorf("ClientsInRoom = %d, want 1", got)
	}
}

func TestHub_Broadcast_ExcludesSender(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	sender := newTestClient(hub, "board-1")
	receiver1 := newTestClient(hub, "board-1")
	receiver2 := newTestClient(hub, "board-1")
	hub.register <- sender
	hub.register <- receiver1
	hub.register <- receiver2
	waitForHub(t, func() bool { return hub.ClientsInRoom("board-1") == 3 })

	data := []byte{0x01, 0x02, 0x03}
	hub.broadcast <- &Message{sender: sender, boardID: "board-1", data: data}

	// Receivers should get the message.
	select {
	case msg := <-receiver1.send:
		if string(msg) != string(data) {
			t.Errorf("receiver1 got %v, want %v", msg, data)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("receiver1 timed out")
	}

	select {
	case msg := <-receiver2.send:
		if string(msg) != string(data) {
			t.Errorf("receiver2 got %v, want %v", msg, data)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("receiver2 timed out")
	}

	// Sender should NOT receive.
	select {
	case msg := <-sender.send:
		t.Errorf("sender should not receive, got %v", msg)
	case <-time.After(50 * time.Millisecond):
		// Expected — no message for sender.
	}
}

func TestHub_Broadcast_OnlyToSameRoom(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	sender := newTestClient(hub, "board-1")
	sameRoom := newTestClient(hub, "board-1")
	otherRoom := newTestClient(hub, "board-2")
	hub.register <- sender
	hub.register <- sameRoom
	hub.register <- otherRoom
	waitForHub(t, func() bool {
		return hub.ClientsInRoom("board-1") == 2 && hub.ClientsInRoom("board-2") == 1
	})

	data := []byte{0xAA, 0xBB}
	hub.broadcast <- &Message{sender: sender, boardID: "board-1", data: data}

	select {
	case msg := <-sameRoom.send:
		if string(msg) != string(data) {
			t.Errorf("sameRoom got %v, want %v", msg, data)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("sameRoom timed out")
	}

	// Other room should NOT receive.
	select {
	case msg := <-otherRoom.send:
		t.Errorf("otherRoom should not receive, got %v", msg)
	case <-time.After(50 * time.Millisecond):
		// Expected.
	}
}

func TestHub_Broadcast_BinaryMessage(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	sender := newTestClient(hub, "board-1")
	receiver := newTestClient(hub, "board-1")
	hub.register <- sender
	hub.register <- receiver
	waitForHub(t, func() bool { return hub.ClientsInRoom("board-1") == 2 })

	// Simulate a realistic Yjs binary update.
	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte(i % 256)
	}

	hub.broadcast <- &Message{sender: sender, boardID: "board-1", data: data}

	select {
	case msg := <-receiver.send:
		if len(msg) != len(data) {
			t.Fatalf("received length %d, want %d", len(msg), len(data))
		}
		for i := range msg {
			if msg[i] != data[i] {
				t.Errorf("byte[%d] = %d, want %d", i, msg[i], data[i])
				break
			}
		}
	case <-time.After(2 * time.Second):
		t.Fatal("receiver timed out")
	}
}

// --- Hub + Relay integration tests ---

func TestHub_WithRelay_FirstClientReceivesState(t *testing.T) {
	ms := newMockStateStore()
	original := [][]byte{
		{yjsMsgSync, 0x01, 0x02},
		{yjsMsgSync, 0x03, 0x04},
	}
	ms.state["board-1"] = EncodeUpdates(original)

	relay := NewYjsRelay(ms, time.Hour)
	hub := NewHubWithRelay(relay)
	go hub.Run()

	client := newTestClient(hub, "board-1")
	hub.register <- client
	waitForHub(t, func() bool { return hub.ClientsInRoom("board-1") == 1 })

	// Client should receive 2 state messages.
	for i := 0; i < 2; i++ {
		select {
		case msg := <-client.send:
			if !bytes.Equal(msg, original[i]) {
				t.Errorf("message[%d] mismatch: got %v, want %v", i, msg, original[i])
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for message %d", i)
		}
	}

	// Clean up relay room.
	hub.unregister <- client
	waitForHub(t, func() bool { return hub.RoomCount() == 0 })
}

func TestHub_WithRelay_UnknownTypeDropped(t *testing.T) {
	ms := newMockStateStore()
	relay := NewYjsRelay(ms, time.Hour)
	hub := NewHubWithRelay(relay)
	go hub.Run()

	sender := newTestClient(hub, "board-1")
	receiver := newTestClient(hub, "board-1")
	hub.register <- sender
	hub.register <- receiver
	waitForHub(t, func() bool { return hub.ClientsInRoom("board-1") == 2 })

	// Unknown type (2) should be dropped by relay.
	hub.broadcast <- &Message{sender: sender, boardID: "board-1", data: []byte{0x02, 0xAA}}

	select {
	case msg := <-receiver.send:
		t.Errorf("receiver should not get unknown type, got %v", msg)
	case <-time.After(100 * time.Millisecond):
		// Expected — message dropped.
	}

	// Clean up.
	hub.unregister <- sender
	hub.unregister <- receiver
	waitForHub(t, func() bool { return hub.RoomCount() == 0 })
}

func TestHub_WithRelay_SyncRelayed(t *testing.T) {
	ms := newMockStateStore()
	relay := NewYjsRelay(ms, time.Hour)
	hub := NewHubWithRelay(relay)
	go hub.Run()

	sender := newTestClient(hub, "board-1")
	receiver := newTestClient(hub, "board-1")
	hub.register <- sender
	hub.register <- receiver
	waitForHub(t, func() bool { return hub.ClientsInRoom("board-1") == 2 })

	data := []byte{yjsMsgSync, 0x01, 0x02}
	hub.broadcast <- &Message{sender: sender, boardID: "board-1", data: data}

	select {
	case msg := <-receiver.send:
		if !bytes.Equal(msg, data) {
			t.Errorf("got %v, want %v", msg, data)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("receiver timed out")
	}

	// Clean up.
	hub.unregister <- sender
	hub.unregister <- receiver
	waitForHub(t, func() bool { return hub.RoomCount() == 0 })
}

func TestHub_WithoutRelay_BackwardCompat(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	sender := newTestClient(hub, "board-1")
	receiver := newTestClient(hub, "board-1")
	hub.register <- sender
	hub.register <- receiver
	waitForHub(t, func() bool { return hub.ClientsInRoom("board-1") == 2 })

	// Any message type should pass through without relay filtering.
	data := []byte{0x02, 0xAA, 0xBB}
	hub.broadcast <- &Message{sender: sender, boardID: "board-1", data: data}

	select {
	case msg := <-receiver.send:
		if !bytes.Equal(msg, data) {
			t.Errorf("got %v, want %v", msg, data)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("receiver timed out")
	}
}
