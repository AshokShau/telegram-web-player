package webapp

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/websocket"
)

func TestUnauthenticatedCommandsStayOutsideRoom(t *testing.T) {
	for _, command := range []string{"join", "pause", "vc_join", "get_playlists"} {
		client := voiceTestClient(-99001, 0, false)
		handleClientMessage(nil, client, ClientMessage{Type: command, RequestID: "request-1"})
		for _, expected := range []string{"error", "ack"} {
			select {
			case raw := <-client.outbox:
				var event struct {
					Event     string `json:"event"`
					RequestID string `json:"requestId"`
					Command   string `json:"command"`
				}
				if err := json.Unmarshal([]byte(raw), &event); err != nil {
					t.Fatal(err)
				}
				if event.Event != expected || event.RequestID != "request-1" || event.Command != command {
					t.Fatalf("%s returned uncorrelated response: %+v", command, event)
				}
			default:
				t.Fatalf("%s did not return %s", command, expected)
			}
		}
		HubInstance.mu.RLock()
		registered := len(HubInstance.clients[client.RoomID])
		HubInstance.mu.RUnlock()
		if registered != 0 || HubInstance.Unregister(nil, client) {
			t.Fatal("unauthenticated client entered room membership")
		}
		Manager.mu.RLock()
		_, roomCreated := Manager.rooms[client.RoomID]
		Manager.mu.RUnlock()
		if roomCreated {
			t.Fatal("unauthenticated disconnect created a playback room")
		}
	}
}

func TestOutboundQueueIsBounded(t *testing.T) {
	client := voiceTestClient(-99002, 1, false)
	for i := 0; i < cap(client.outbox); i++ {
		if err := client.SendMessage("event"); err != nil {
			t.Fatal(err)
		}
	}
	if err := client.SendMessage("overflow"); err == nil {
		t.Fatal("slow consumer accepted an unbounded backlog")
	}
	close(client.done)
	if err := client.SendMessage("closed"); err == nil {
		t.Fatal("closed writer accepted an event")
	}
}

func TestSingleWriterPreservesOrderAndShutsDown(t *testing.T) {
	finished := make(chan error, 1)
	server := httptest.NewServer(websocket.Handler(func(ws *websocket.Conn) {
		client := &Client{Conn: ws}
		client.startWriter()
		for i := range 50 {
			if err := client.SendMessage(fmt.Sprint(i)); err != nil {
				finished <- err
				close(client.shutdown)
				return
			}
		}
		var acknowledgment string
		_ = ws.SetReadDeadline(time.Now().Add(5 * time.Second))
		err := websocket.Message.Receive(ws, &acknowledgment)
		close(client.shutdown)
		select {
		case <-client.done:
			if err == nil && client.SendMessage("after shutdown") == nil {
				err = fmt.Errorf("writer accepted a message after shutdown")
			}
		case <-time.After(5 * time.Second):
			err = fmt.Errorf("writer leaked after shutdown")
		}
		finished <- err
	}))
	defer server.Close()
	conn, err := websocket.Dial("ws"+strings.TrimPrefix(server.URL, "http"), "", server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	for i := range 50 {
		var message string
		if err := websocket.Message.Receive(conn, &message); err != nil {
			t.Fatal(err)
		}
		if message != fmt.Sprint(i) {
			t.Fatalf("event %d arrived as %q", i, message)
		}
	}
	if err := websocket.Message.Send(conn, "received"); err != nil {
		t.Fatal(err)
	}
	if err := <-finished; err != nil {
		t.Fatal(err)
	}
}
