package webapp

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/net/websocket"
)

func TestSessionClaimsRequireExplicitSwitch(t *testing.T) {
	hub := &Hub{clients: make(map[int64][]*Client)}
	current := voiceTestClient(-99010, 10, false)
	if _, accepted := hub.claimSession(current, false); !accepted {
		t.Fatal("first session was not accepted")
	}
	otherUser := voiceTestClient(-99010, 20, false)
	if _, accepted := hub.claimSession(otherUser, false); !accepted {
		t.Fatal("another user's session was blocked")
	}
	var waiting *Client
	for i := 0; i < 8; i++ {
		waiting = voiceTestClient(-99010-int64(i%2), 10, false)
		if _, accepted := hub.claimSession(waiting, false); accepted || waiting.sessionActive {
			t.Fatal("an additional tab stole the current session")
		}
	}
	replaced, accepted := hub.claimSession(waiting, true)
	if !accepted || len(replaced) != 1 || replaced[0] != current || !waiting.sessionActive || !current.sessionEnded {
		t.Fatal("explicit switch did not replace the previous session")
	}
	if hub.Unregister(nil, current) {
		t.Fatal("late old-session cleanup altered active membership")
	}
	if _, accepted := hub.claimSession(current, true); accepted {
		t.Fatal("a displaced connection reclaimed membership")
	}
	if replaced, accepted := hub.claimSession(waiting, false); !accepted || len(replaced) != 0 {
		t.Fatal("repeated join was not idempotent")
	}
	if !otherUser.sessionActive || len(hub.GetListeners(waiting.RoomID)) != 1 {
		t.Fatal("switch affected unrelated users or duplicated presence")
	}
}

func TestConcurrentSessionClaimsHaveOneOwner(t *testing.T) {
	for _, takeOver := range []bool{false, true} {
		hub := &Hub{clients: make(map[int64][]*Client)}
		clients := make([]*Client, 32)
		var group sync.WaitGroup
		start := make(chan struct{})
		for i := range clients {
			clients[i] = voiceTestClient(-99020-int64(i%3), 42, false)
			group.Add(1)
			go func(client *Client) {
				defer group.Done()
				<-start
				hub.claimSession(client, takeOver)
			}(clients[i])
		}
		close(start)
		group.Wait()
		active, registered := 0, 0
		for _, client := range clients {
			if client.sessionActive {
				active++
				if client.sessionEnded {
					t.Fatal("active session was marked ended")
				}
			}
		}
		for _, roomClients := range hub.clients {
			registered += len(roomClients)
		}
		if active != 1 || registered != 1 {
			t.Fatalf("takeOver=%t left %d active and %d registered sessions", takeOver, active, registered)
		}
	}
}

func TestTakeoverRemovesAllLegacySessions(t *testing.T) {
	hub := &Hub{clients: make(map[int64][]*Client)}
	for i := 0; i < 4; i++ {
		client := voiceTestClient(-99030-int64(i%2), 42, false)
		client.sessionActive = true
		hub.clients[client.RoomID] = append(hub.clients[client.RoomID], client)
	}
	current := voiceTestClient(-99030, 42, false)
	replaced, accepted := hub.claimSession(current, true)
	if !accepted || len(replaced) != 4 || len(hub.clients) != 1 || len(hub.clients[current.RoomID]) != 1 {
		t.Fatal("takeover left old sessions registered")
	}
	for _, client := range replaced {
		if client.sessionActive || !client.sessionEnded {
			t.Fatal("an old session can still send room commands")
		}
	}
}

func TestWaitingSessionsCannotSendRoomCommands(t *testing.T) {
	for _, command := range []string{"pause", "vc_join", "get_playlists"} {
		client := voiceTestClient(-99040, 42, true)
		client.CanControl, client.AllowsWriteToPM = true, true
		handleClientMessage(nil, client, ClientMessage{Type: command})
		var event struct{ Event, Code string }
		if err := json.Unmarshal([]byte(<-client.outbox), &event); err != nil {
			t.Fatal(err)
		}
		if event.Event != "error" || event.Code != "session_in_use" {
			t.Fatalf("waiting session sent %s: %+v", command, event)
		}
	}
}

func TestReplacementNoticeArrivesBeforeSocketClose(t *testing.T) {
	hub := &Hub{clients: make(map[int64][]*Client)}
	ready := make(chan *Client, 1)
	finished := make(chan struct{})
	server := httptest.NewServer(websocket.Handler(func(ws *websocket.Conn) {
		defer close(finished)
		client := &Client{Conn: ws, RoomID: -99050, UserID: 42}
		client.startWriter()
		defer close(client.shutdown)
		hub.claimSession(client, false)
		ready <- client
		var message string
		_ = websocket.Message.Receive(ws, &message)
		hub.Unregister(nil, client)
	}))
	defer server.Close()
	conn, err := websocket.Dial("ws"+strings.TrimPrefix(server.URL, "http"), "", server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	<-ready
	replacement := voiceTestClient(-99050, 42, false)
	if !hub.Register(nil, replacement, true) {
		t.Fatal("explicit switch was rejected")
	}
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	var raw string
	if err := websocket.Message.Receive(conn, &raw); err != nil {
		t.Fatalf("socket closed before replacement notice: %v", err)
	}
	var event struct{ Event string }
	if err := json.Unmarshal([]byte(raw), &event); err != nil || event.Event != "duplicate_session" {
		t.Fatalf("replacement notice was lost: %q, %v", raw, err)
	}
	select {
	case <-finished:
	case <-time.After(5 * time.Second):
		t.Fatal("old socket stayed open")
	}
	if len(hub.GetListeners(replacement.RoomID)) != 1 || !replacement.sessionActive {
		t.Fatal("old disconnect removed the replacement session")
	}
}

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
		for i := 0; i < 50; i++ {
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
	for i := 0; i < 50; i++ {
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
