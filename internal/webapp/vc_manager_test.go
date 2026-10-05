package webapp

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/pion/webrtc/v4"
)

func voiceTestClient(room, user int64, admin bool) *Client {
	return &Client{RoomID: room, UserID: user, IsAdmin: admin, outbox: make(chan string, 128), done: make(chan struct{})}
}
func TestVoiceJoinMutePolicy(t *testing.T) {
	manager := newVCManager()
	admin := voiceTestClient(-903, 1, true)
	existing := voiceTestClient(-903, 2, false)
	manager.JoinVC(nil, admin)
	manager.JoinVC(nil, existing)
	manager.AdminSetJoinMuted(nil, existing, true)
	if manager.GetOrCreateRoom(-903).GetState().MuteNewParticipants {
		t.Fatal("non-admin changed join policy")
	}
	manager.AdminSetJoinMuted(nil, admin, true)
	newListener := voiceTestClient(-903, 3, false)
	manager.JoinVC(nil, newListener)
	manager.SetSelfMute(nil, newListener, false)
	for _, p := range manager.GetOrCreateRoom(-903).GetState().Participants {
		if p.UserID == 3 && (!p.IsAdminMuted || !p.IsSelfMuted || p.AllowedToSpeak) {
			t.Fatal("new listener bypassed join mute")
		}
		if p.UserID == 2 && p.IsAdminMuted {
			t.Fatal("future-join policy changed existing listener")
		}
		if p.IsAdmin && p.IsAdminMuted {
			t.Fatal("join policy muted an admin")
		}
	}
	manager.AdminMuteUser(nil, admin, 3, false)
	manager.SetSelfMute(nil, newListener, false)
	for _, p := range manager.GetOrCreateRoom(-903).GetState().Participants {
		if p.UserID == 3 && (!p.AllowedToSpeak || p.IsSelfMuted) {
			t.Fatal("individual admin permission did not allow speaking")
		}
	}
	manager.AdminMuteUser(nil, admin, 2, true)
	manager.AdminSetJoinMuted(nil, admin, false)
	state := manager.GetOrCreateRoom(-903).GetState()
	if state.MuteNewParticipants {
		t.Fatal("unmute-all did not release room policy")
	}
	for _, p := range state.Participants {
		if p.IsAdminMuted {
			t.Fatal("unmute-all left an admin mute")
		}
	}
	manager.LeaveVC(nil, newListener)
	manager.LeaveVC(nil, existing)
	manager.LeaveVC(nil, admin)
	manager.AdminSetJoinMuted(nil, admin, true)
	manager.JoinVC(nil, newListener)
	state = manager.GetOrCreateRoom(-903).GetState()
	if !state.MuteNewParticipants || !state.Participants[0].IsAdminMuted {
		t.Fatal("empty room lost join policy")
	}
	manager.LeaveVC(nil, newListener)
}
func TestVoiceMuteAndConnectionOwnership(t *testing.T) {
	manager := newVCManager()
	admin := voiceTestClient(-900, 1, true)
	listener := voiceTestClient(-900, 2, false)
	manager.JoinVC(nil, admin)
	manager.JoinVC(nil, listener)
	manager.AdminMuteUser(nil, admin, 2, true)
	manager.SetSelfMute(nil, listener, false)
	manager.SetSpeaking(nil, listener, true)
	room := manager.GetOrCreateRoom(-900)
	info := room.GetState()
	for _, p := range info.Participants {
		if p.UserID == 2 && (!p.IsSelfMuted || p.IsSpeaking || p.AllowedToSpeak) {
			t.Fatal("restricted listener can speak")
		}
	}
	manager.AdminMuteUser(nil, admin, 2, false)
	manager.SetSelfMute(nil, listener, false)
	manager.SetSpeaking(nil, listener, true)
	for _, p := range room.GetState().Participants {
		if p.UserID == 2 && (p.IsAdminMuted || !p.AllowedToSpeak || p.IsSelfMuted || !p.IsSpeaking) {
			t.Fatal("removing the admin mute did not restore speaking")
		}
		if p.UserID == 1 && p.IsAdminMuted {
			t.Fatal("mute-all muted an admin")
		}
	}
	replacement := voiceTestClient(-900, 2, false)
	manager.JoinVC(nil, replacement)
	manager.LeaveVC(nil, listener)
	if len(room.GetState().Participants) != 2 {
		t.Fatal("old connection removed the replacement participant")
	}
	manager.AdminSetJoinMuted(nil, admin, true)
	manager.LeaveVC(nil, replacement)
	manager.LeaveVC(nil, admin)
	if !room.GetState().MuteNewParticipants {
		t.Fatal("empty room reset the new-participant mute policy")
	}
	if len(room.GetState().Participants) != 0 {
		t.Fatal("participants leaked after leaving")
	}
}

func TestConcurrentVoiceLifecycle(t *testing.T) {
	manager := newVCManager()
	admin := voiceTestClient(-901, 1, true)
	manager.JoinVC(nil, admin)
	var wg sync.WaitGroup
	for i := int64(2); i < 8; i++ {
		wg.Add(1)
		go func(user int64) {
			defer wg.Done()
			client := voiceTestClient(-901, user, false)
			for range 10 {
				manager.JoinVC(nil, client)
				manager.SetSelfMute(nil, client, false)
				manager.SetSpeaking(nil, client, true)
				manager.GetOrCreateRoom(-901).GetState()
				manager.LeaveVC(nil, client)
			}
		}(i)
	}
	wg.Go(func() {
		for i := range 30 {
			manager.AdminSetJoinMuted(nil, admin, i%2 == 0)
		}
	})
	wg.Wait()
	manager.LeaveVC(nil, admin)
	if len(manager.GetOrCreateRoom(-901).GetState().Participants) != 0 {
		t.Fatal("participants leaked")
	}
}

func TestVoiceNegotiationQueuesRoomChanges(t *testing.T) {
	manager := newVCManager()
	client := voiceTestClient(-902, 1, true)
	manager.JoinVC(nil, client)
	peer, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		t.Fatal(err)
	}
	defer peer.Close()
	_, err = peer.AddTransceiverFromKind(webrtc.RTPCodecTypeAudio, webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionSendrecv})
	if err != nil {
		t.Fatal(err)
	}
	offer, err := peer.CreateOffer(nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = peer.SetLocalDescription(offer); err != nil {
		t.Fatal(err)
	}
	manager.HandleClientOffer(nil, client, offer.SDP)
	// Initial answer must use the existing peer; it is not recreated on further signaling.
	room, p := manager.participant(client)
	if p == nil || p.PeerConnection == nil {
		t.Fatal("server peer was not created")
	}
	original := p.PeerConnection
	var answer string
	deadline := time.After(3 * time.Second)
	for answer == "" {
		select {
		case raw := <-client.outbox:
			var message struct {
				Event string `json:"event"`
				Data  struct {
					SDP string `json:"sdp"`
				} `json:"data"`
			}
			if json.Unmarshal([]byte(raw), &message) == nil && message.Event == "vc_answer" {
				answer = message.Data.SDP
			}
		case <-deadline:
			t.Fatal("no initial answer")
		}
	}
	if err = peer.SetRemoteDescription(webrtc.SessionDescription{Type: webrtc.SDPTypeAnswer, SDP: answer}); err != nil {
		t.Fatal(err)
	}
	other := voiceTestClient(-902, 2, false)
	manager.JoinVC(nil, other)
	p.signalMu.Lock()
	state := p.PeerConnection.SignalingState()
	p.signalMu.Unlock()
	if state != webrtc.SignalingStateHaveLocalOffer {
		t.Fatalf("expected an offer for new participant, got %s", state)
	}
	third := voiceTestClient(-902, 3, false)
	manager.JoinVC(nil, third)
	p.signalMu.Lock()
	pending := p.negotiationPending
	p.signalMu.Unlock()
	if !pending {
		t.Fatal("join during outstanding offer was dropped")
	}
	manager.LeaveVC(nil, other)
	manager.LeaveVC(nil, third)
	if room.GetState().RoomID != -902 {
		t.Fatal("room identity changed")
	}
	p.signalMu.Lock()
	if p.PeerConnection != original {
		t.Fatal("signaling replaced the peer connection")
	}
	p.signalMu.Unlock()
	manager.LeaveVC(nil, client)
	p.signalMu.Lock()
	defer p.signalMu.Unlock()
	if !p.closed || p.PeerConnection != nil {
		t.Fatal("peer resources leaked after leave")
	}
}
