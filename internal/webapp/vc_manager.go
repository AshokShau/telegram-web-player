/*
 * TgMusicBot - Telegram Music Bot
 * Copyright (c) 2025-2026 Ashok Shau
 * Licensed under GNU GPL v3
 * See https://github.com/AshokShau/telegram-web-player
 */
package webapp

import (
	"encoding/json"
	"fmt"
	"slices"
	"sync"

	td "github.com/AshokShau/gotdbot"
	"github.com/pion/webrtc/v4"
)

type VCParticipantInfo struct {
	UserID         int64  `json:"userId"`
	FirstName      string `json:"firstName"`
	LastName       string `json:"lastName,omitempty"`
	Username       string `json:"username,omitempty"`
	PhotoURL       string `json:"photoUrl,omitempty"`
	IsAdmin        bool   `json:"isAdmin"`
	IsSelfMuted    bool   `json:"isSelfMuted"`
	IsAdminMuted   bool   `json:"isAdminMuted"`
	IsSpeaking     bool   `json:"isSpeaking"`
	AllowedToSpeak bool   `json:"allowedToSpeak"`
}

type VCRoomState struct {
	Revision            uint64              `json:"revision"`
	RoomID              int64               `json:"roomId"`
	MuteNewParticipants bool                `json:"muteNewParticipants"`
	Participants        []VCParticipantInfo `json:"participants"`
}

// Room.mu owns participant identity and permission fields. signalMu owns all
// signaling resources. Membership writers release Room.mu before acquiring
// signalMu; signaling may read membership while holding signalMu.
type VCParticipant struct {
	UserID             int64
	FirstName          string
	LastName           string
	Username           string
	PhotoURL           string
	IsAdmin            bool
	IsSelfMuted        bool
	IsAdminMuted       bool
	IsSpeaking         bool
	AudioTrack         *webrtc.TrackLocalStaticRTP
	Client             *Client
	signalMu           sync.Mutex
	PeerConnection     *webrtc.PeerConnection
	pendingCandidates  []webrtc.ICECandidateInit
	senders            map[*webrtc.TrackLocalStaticRTP]*webrtc.RTPSender
	negotiationPending bool
	closed             bool
}

func (p *VCParticipant) AllowedToSpeak() bool {
	return !p.IsAdminMuted
}
func (p *VCParticipant) CanSpeak() bool {
	return !p.IsSelfMuted && p.AllowedToSpeak()
}
func (p *VCParticipant) ToInfo() VCParticipantInfo {
	return VCParticipantInfo{UserID: p.UserID, FirstName: p.FirstName, LastName: p.LastName, Username: p.Username, PhotoURL: p.PhotoURL, IsAdmin: p.IsAdmin, IsSelfMuted: p.IsSelfMuted, IsAdminMuted: p.IsAdminMuted, IsSpeaking: p.IsSpeaking, AllowedToSpeak: p.AllowedToSpeak()}
}

type VCRoom struct {
	mu                  sync.RWMutex
	RoomID              int64
	MuteNewParticipants bool
	Participants        map[int64]*VCParticipant
}
type VCManager struct {
	mu        sync.RWMutex
	rooms     map[int64]*VCRoom
	webrtcAPI *webrtc.API
}

var VCManagerInstance = newVCManager()

func newVCManager() *VCManager {
	engine := &webrtc.MediaEngine{}
	if err := engine.RegisterDefaultCodecs(); err != nil {
		log.Error("[VC] Register codecs", "error", err)
	}
	return &VCManager{rooms: make(map[int64]*VCRoom), webrtcAPI: webrtc.NewAPI(webrtc.WithMediaEngine(engine))}
}
func (m *VCManager) GetOrCreateRoom(roomID int64) *VCRoom {
	m.mu.Lock()
	defer m.mu.Unlock()
	room := m.rooms[roomID]
	if room == nil {
		room = &VCRoom{RoomID: roomID, Participants: make(map[int64]*VCParticipant)}
		m.rooms[roomID] = room
	}
	return room
}
func (r *VCRoom) GetState() VCRoomState {
	r.mu.RLock()
	defer r.mu.RUnlock()
	participants := make([]VCParticipantInfo, 0, len(r.Participants))
	for _, p := range r.Participants {
		participants = append(participants, p.ToInfo())
	}
	slices.SortFunc(participants, func(a, b VCParticipantInfo) int {
		if a.UserID < b.UserID {
			return -1
		}
		if a.UserID > b.UserID {
			return 1
		}
		return 0
	})
	return VCRoomState{RoomID: r.RoomID, MuteNewParticipants: r.MuteNewParticipants, Participants: participants}
}
func (m *VCManager) BroadcastVCState(bot *td.Client, roomID int64) {
	HubInstance.broadcastMu.Lock()
	defer HubInstance.broadcastMu.Unlock()
	state := m.GetOrCreateRoom(roomID).GetState()
	HubInstance.revision++
	state.Revision = HubInstance.revision
	payload, err := json.Marshal(EventMessage{Event: "vc_state", Data: state})
	if err == nil {
		HubInstance.BroadcastRoomMessage(roomID, string(payload))
	}
}
func (m *VCManager) participant(client *Client) (*VCRoom, *VCParticipant) {
	userID, _, _, _, _, _ := client.GetInfo()
	room := m.GetOrCreateRoom(client.RoomID)
	room.mu.RLock()
	p := room.Participants[userID]
	room.mu.RUnlock()
	if p == nil || p.Client != client {
		return room, nil
	}
	return room, p
}
func (p *VCParticipant) close() {
	p.signalMu.Lock()
	defer p.signalMu.Unlock()
	p.closed = true
	if p.PeerConnection != nil {
		_ = p.PeerConnection.Close()
		p.PeerConnection = nil
	}
	p.pendingCandidates = nil
	p.senders = nil
}
func (m *VCManager) JoinVC(bot *td.Client, client *Client) {
	userID, admin, _, _, _, data := client.GetInfo()
	if userID == 0 {
		sendError(client, "Telegram authentication required to join voice chat.")
		return
	}
	room := m.GetOrCreateRoom(client.RoomID)
	track, err := webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus}, fmt.Sprintf("voice_%d", userID), fmt.Sprintf("user_%d", userID))
	if err != nil {
		sendError(client, "Could not create a voice track.")
		return
	}
	p := &VCParticipant{UserID: userID, IsAdmin: admin, IsSelfMuted: true, AudioTrack: track, Client: client, senders: make(map[*webrtc.TrackLocalStaticRTP]*webrtc.RTPSender)}
	if data != nil && data.User != nil {
		p.FirstName = data.User.FirstName
		p.LastName = data.User.LastName
		p.Username = data.User.Username
		p.PhotoURL = data.User.PhotoURL
	}
	if p.FirstName == "" {
		p.FirstName = "Listener"
	}
	room.mu.Lock()
	p.IsAdminMuted = room.MuteNewParticipants && !p.IsAdmin
	previous := room.Participants[userID]
	room.Participants[userID] = p
	room.mu.Unlock()
	if previous != nil {
		previous.close()
	}
	m.reconcileRoom(room)
	m.BroadcastVCState(bot, client.RoomID)
}
func (m *VCManager) LeaveVC(bot *td.Client, client *Client) {
	room, p := m.participant(client)
	if p == nil {
		return
	}
	m.leaveParticipant(bot, room, p)
}

func (m *VCManager) leaveParticipant(bot *td.Client, room *VCRoom, p *VCParticipant) {
	room.mu.Lock()
	if room.Participants[p.UserID] != p {
		room.mu.Unlock()
		return
	}
	delete(room.Participants, p.UserID)
	room.mu.Unlock()
	p.close()
	m.reconcileRoom(room)
	// Keep empty room permissions; a subsequent join must not reset an admin's decision.
	m.BroadcastVCState(bot, room.RoomID)
}
func (m *VCManager) reconcileRoom(room *VCRoom) {
	room.mu.RLock()
	participants := make([]*VCParticipant, 0, len(room.Participants))
	for _, p := range room.Participants {
		participants = append(participants, p)
	}
	room.mu.RUnlock()
	for _, p := range participants {
		m.renegotiateUser(room, p)
	}
}
func voiceMessage(client *Client, event string, data any) {
	payload, err := json.Marshal(EventMessage{Event: event, Data: data})
	if err == nil {
		_ = client.SendMessage(string(payload))
	}
}

// reconcileSendersLocked works against the latest complete room track set, so
// joins/leaves during an outstanding offer are coalesced rather than skipped.
func (m *VCManager) renegotiateUser(room *VCRoom, p *VCParticipant) {
	p.signalMu.Lock()
	defer p.signalMu.Unlock()
	room.mu.RLock()
	if room.Participants[p.UserID] != p {
		room.mu.RUnlock()
		return
	}
	tracks := make([]*webrtc.TrackLocalStaticRTP, 0, len(room.Participants))
	for id, other := range room.Participants {
		if id != p.UserID {
			tracks = append(tracks, other.AudioTrack)
		}
	}
	room.mu.RUnlock()
	pc := p.PeerConnection
	if p.closed || pc == nil {
		return
	}
	if pc.SignalingState() != webrtc.SignalingStateStable {
		p.negotiationPending = true
		return
	}
	changed := false
	for track, sender := range p.senders {
		if !slices.Contains(tracks, track) {
			if pc.RemoveTrack(sender) == nil {
				delete(p.senders, track)
				changed = true
			}
		}
	}
	for _, track := range tracks {
		if _, exists := p.senders[track]; exists {
			continue
		}
		sender, err := pc.AddTrack(track)
		if err != nil {
			log.Error("[VC] Add sender", "error", err)
			continue
		}
		p.senders[track] = sender
		changed = true
		go drainRTCP(sender)
	}
	if !changed && !p.negotiationPending {
		return
	}
	p.negotiationPending = false
	offer, err := pc.CreateOffer(nil)
	if err == nil {
		err = pc.SetLocalDescription(offer)
	}
	if err != nil {
		p.negotiationPending = true
		log.Error("[VC] Create offer", "error", err)
		return
	}
	voiceMessage(p.Client, "vc_offer", map[string]any{"sdp": offer.SDP})
}
func drainRTCP(sender *webrtc.RTPSender) {
	// Pion requires feedback to be consumed. Closing the peer ends this reader.
	buffer := make([]byte, 1500)
	for {
		if _, _, err := sender.Read(buffer); err != nil {
			return
		}
	}
}
func (m *VCManager) HandleClientOffer(bot *td.Client, client *Client, sdp string) {
	room, p := m.participant(client)
	if p == nil {
		sendError(client, "Join voice chat before sending an offer.")
		return
	}
	p.signalMu.Lock()
	if p.closed {
		p.signalMu.Unlock()
		return
	}
	pc := p.PeerConnection
	if pc == nil {
		var err error
		pc, err = m.webrtcAPI.NewPeerConnection(webrtc.Configuration{ICEServers: []webrtc.ICEServer{{URLs: []string{"stun:stun.l.google.com:19302"}}}})
		if err != nil {
			p.signalMu.Unlock()
			sendError(client, "Could not create a voice connection.")
			return
		}
		p.PeerConnection = pc
		pc.OnICECandidate(func(candidate *webrtc.ICECandidate) {
			if candidate == nil {
				return
			}
			value, err := json.Marshal(candidate.ToJSON())
			if err == nil {
				voiceMessage(client, "vc_candidate", map[string]any{"candidate": string(value)})
			}
		})
		pc.OnConnectionStateChange(func(connection webrtc.PeerConnectionState) {
			if connection == webrtc.PeerConnectionStateFailed {
				// Ownership check in LeaveVC protects a newer session from old callbacks.
				go m.leaveParticipant(bot, room, p)
			}
		})
		pc.OnTrack(func(remote *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
			if remote.Kind() != webrtc.RTPCodecTypeAudio {
				return
			}
			for {
				packet, _, err := remote.ReadRTP()
				if err != nil {
					return
				}
				room.mu.RLock()
				permitted := room.Participants[p.UserID] == p && p.CanSpeak()
				room.mu.RUnlock()
				if permitted {
					_ = p.AudioTrack.WriteRTP(packet)
				}
			}
		})
	}
	err := pc.SetRemoteDescription(webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: sdp})
	if err == nil {
		for _, candidate := range p.pendingCandidates {
			_ = pc.AddICECandidate(candidate)
		}
		p.pendingCandidates = nil
		var answer webrtc.SessionDescription
		answer, err = pc.CreateAnswer(nil)
		if err == nil {
			err = pc.SetLocalDescription(answer)
		}
		if err == nil {
			voiceMessage(client, "vc_answer", map[string]any{"sdp": answer.SDP})
		}
	}
	p.signalMu.Unlock()
	if err != nil {
		sendError(client, "Voice signaling failed. Leave and join again.")
		m.LeaveVC(bot, client)
		return
	}
	m.reconcileRoom(room)
}
func (m *VCManager) HandleClientAnswer(client *Client, sdp string) {
	room, p := m.participant(client)
	if p == nil {
		return
	}
	p.signalMu.Lock()
	if p.closed || p.PeerConnection == nil {
		p.signalMu.Unlock()
		return
	}
	err := p.PeerConnection.SetRemoteDescription(webrtc.SessionDescription{Type: webrtc.SDPTypeAnswer, SDP: sdp})
	if err == nil {
		for _, candidate := range p.pendingCandidates {
			_ = p.PeerConnection.AddICECandidate(candidate)
		}
		p.pendingCandidates = nil
	}
	pending := p.negotiationPending
	p.signalMu.Unlock()
	if err != nil {
		sendError(client, "Could not apply the voice answer. Leave and join again.")
		return
	}
	if pending {
		m.renegotiateUser(room, p)
	}
}
func (m *VCManager) HandleCandidate(client *Client, value string) {
	_, p := m.participant(client)
	if p == nil {
		return
	}
	var candidate webrtc.ICECandidateInit
	if json.Unmarshal([]byte(value), &candidate) != nil {
		return
	}
	p.signalMu.Lock()
	defer p.signalMu.Unlock()
	if p.closed {
		return
	}
	if p.PeerConnection == nil || p.PeerConnection.RemoteDescription() == nil {
		if len(p.pendingCandidates) < 64 {
			p.pendingCandidates = append(p.pendingCandidates, candidate)
		}
		return
	}
	if err := p.PeerConnection.AddICECandidate(candidate); err != nil {
		log.Warn("[VC] ICE candidate", "error", err)
	}
}
func (m *VCManager) SetSelfMute(bot *td.Client, client *Client, muted bool) {
	room, p := m.participant(client)
	if p == nil {
		return
	}
	room.mu.Lock()
	p.IsSelfMuted = muted || !p.AllowedToSpeak()
	if p.IsSelfMuted {
		p.IsSpeaking = false
	}
	room.mu.Unlock()
	m.BroadcastVCState(bot, client.RoomID)
}
func (m *VCManager) SetSpeaking(bot *td.Client, client *Client, speaking bool) {
	room, p := m.participant(client)
	if p == nil {
		return
	}
	room.mu.Lock()
	speaking = speaking && p.CanSpeak()
	changed := p.IsSpeaking != speaking
	p.IsSpeaking = speaking
	room.mu.Unlock()
	if changed {
		payload, _ := json.Marshal(EventMessage{Event: "vc_user_speaking", Data: map[string]any{"userId": p.UserID, "isSpeaking": speaking}})
		HubInstance.BroadcastRoomMessage(client.RoomID, string(payload))
	}
}
func voiceAdmin(client *Client) bool {
	id, admin, _, _, _, _ := client.GetInfo()
	if id == 0 || !admin {
		sendError(client, "Admin permission required to manage voice chat.")
		return false
	}
	return true
}
func (m *VCManager) AdminMuteUser(bot *td.Client, client *Client, target int64, muted bool) {
	if !voiceAdmin(client) {
		return
	}
	room := m.GetOrCreateRoom(client.RoomID)
	room.mu.Lock()
	if p := room.Participants[target]; p != nil {
		p.IsAdminMuted = muted
		if muted {
			p.IsSpeaking = false
		}
	}
	room.mu.Unlock()
	m.BroadcastVCState(bot, client.RoomID)
}

// AdminSetJoinMuted governs future joins. Disabling the rule also removes
// existing admin mutes; each listener still chooses whether to enable their mic.
func (m *VCManager) AdminSetJoinMuted(bot *td.Client, client *Client, muted bool) {
	if !voiceAdmin(client) {
		return
	}
	room := m.GetOrCreateRoom(client.RoomID)
	room.mu.Lock()
	room.MuteNewParticipants = muted
	if !muted {
		for _, p := range room.Participants {
			if !p.IsAdmin {
				p.IsAdminMuted = false
			}
		}
	}
	room.mu.Unlock()
	m.BroadcastVCState(bot, client.RoomID)
}
