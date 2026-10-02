/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/telegram-web-player
 */

package webapp

import (
	"encoding/json"
	"fmt"
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
	CanSpeak       bool   `json:"canSpeak"`
	AllowedToSpeak bool   `json:"allowedToSpeak"`
}

type VCRoomState struct {
	RoomID             int64               `json:"roomId"`
	AllowEveryoneSpeak bool                `json:"allowEveryoneSpeak"`
	Participants       []VCParticipantInfo `json:"participants"`
}

type VCParticipant struct {
	UserID         int64
	FirstName      string
	LastName       string
	Username       string
	PhotoURL       string
	IsAdmin        bool
	IsSelfMuted    bool
	IsAdminMuted   bool
	IsSpeaking     bool
	PeerConnection *webrtc.PeerConnection
	AudioTrack     *webrtc.TrackLocalStaticRTP
	Client         *Client
}

func (p *VCParticipant) AllowedToSpeak(allowEveryoneSpeak bool) bool {
	if p.IsAdminMuted {
		return false
	}
	if p.IsAdmin {
		return true
	}
	return allowEveryoneSpeak
}

func (p *VCParticipant) CanSpeak(allowEveryoneSpeak bool) bool {
	if p.IsSelfMuted {
		return false
	}
	return p.AllowedToSpeak(allowEveryoneSpeak)
}

func (p *VCParticipant) ToInfo(allowEveryoneSpeak bool) VCParticipantInfo {
	return VCParticipantInfo{
		UserID:         p.UserID,
		FirstName:      p.FirstName,
		LastName:       p.LastName,
		Username:       p.Username,
		PhotoURL:       p.PhotoURL,
		IsAdmin:        p.IsAdmin,
		IsSelfMuted:    p.IsSelfMuted,
		IsAdminMuted:   p.IsAdminMuted,
		IsSpeaking:     p.IsSpeaking,
		CanSpeak:       p.CanSpeak(allowEveryoneSpeak),
		AllowedToSpeak: p.AllowedToSpeak(allowEveryoneSpeak),
	}
}

type VCRoom struct {
	mu                 sync.RWMutex
	RoomID             int64
	AllowEveryoneSpeak bool
	Participants       map[int64]*VCParticipant
}

type VCManager struct {
	mu        sync.RWMutex
	rooms     map[int64]*VCRoom
	webrtcAPI *webrtc.API
}

var VCManagerInstance = newVCManager()

func newVCManager() *VCManager {
	m := &webrtc.MediaEngine{}
	if err := m.RegisterDefaultCodecs(); err != nil {
		log.Error("[VCManager] Failed to register default codecs", "error", err)
	}
	api := webrtc.NewAPI(webrtc.WithMediaEngine(m))
	return &VCManager{
		rooms:     make(map[int64]*VCRoom),
		webrtcAPI: api,
	}
}

func (m *VCManager) GetOrCreateRoom(roomID int64) *VCRoom {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, ok := m.rooms[roomID]
	if !ok {
		room = &VCRoom{
			RoomID:             roomID,
			AllowEveryoneSpeak: true,
			Participants:       make(map[int64]*VCParticipant),
		}
		m.rooms[roomID] = room
	}
	return room
}

func (r *VCRoom) GetState() VCRoomState {
	r.mu.RLock()
	defer r.mu.RUnlock()

	parts := make([]VCParticipantInfo, 0, len(r.Participants))
	for _, p := range r.Participants {
		parts = append(parts, p.ToInfo(r.AllowEveryoneSpeak))
	}

	return VCRoomState{
		RoomID:             r.RoomID,
		AllowEveryoneSpeak: r.AllowEveryoneSpeak,
		Participants:       parts,
	}
}

func (m *VCManager) BroadcastVCState(bot *td.Client, roomID int64) {
	room := m.GetOrCreateRoom(roomID)
	state := room.GetState()

	msg := EventMessage{
		Event: "vc_state",
		Data:  state,
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		return
	}

	HubInstance.BroadcastRoomMessage(roomID, string(payload))
}

func (m *VCManager) JoinVC(bot *td.Client, client *Client) {
	userID, isAdmin, _, _, _, initData := client.GetInfo()
	if userID == 0 {
		sendError(client, "Telegram authentication required to join Voice Chat.")
		return
	}

	room := m.GetOrCreateRoom(client.RoomID)
	room.mu.Lock()

	existing, exists := room.Participants[userID]
	if exists {
		if existing.PeerConnection != nil {
			_ = existing.PeerConnection.Close()
		}
	}

	audioTrack, err := webrtc.NewTrackLocalStaticRTP(
		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus},
		"audio",
		fmt.Sprintf("audio_%d", userID),
	)
	if err != nil {
		room.mu.Unlock()
		log.Error("[VCManager] Failed to create local RTP track", "error", err)
		sendError(client, "Failed to create voice track.")
		return
	}

	participant := &VCParticipant{
		UserID:       userID,
		IsAdmin:      isAdmin,
		IsSelfMuted:  true,
		IsAdminMuted: false,
		IsSpeaking:   false,
		AudioTrack:   audioTrack,
		Client:       client,
	}

	if initData != nil && initData.User != nil {
		u := initData.User
		participant.FirstName = u.FirstName
		participant.LastName = u.LastName
		participant.Username = u.Username
		participant.PhotoURL = u.PhotoURL
	}
	if participant.FirstName == "" {
		participant.FirstName = getClientUserName(client)
	}

	room.Participants[userID] = participant
	room.mu.Unlock()

	log.Info("[VCManager] User joined VC", "roomID", client.RoomID, "userID", userID)
	m.BroadcastVCState(bot, client.RoomID)
}

func (m *VCManager) LeaveVC(bot *td.Client, client *Client) {
	userID, _, _, _, _, _ := client.GetInfo()
	if userID == 0 {
		return
	}

	room := m.GetOrCreateRoom(client.RoomID)
	room.mu.Lock()

	p, exists := room.Participants[userID]
	var remaining []*VCParticipant
	if exists {
		leavingTrack := p.AudioTrack
		if p.PeerConnection != nil {
			_ = p.PeerConnection.Close()
		}
		delete(room.Participants, userID)
		log.Info("[VCManager] User left VC", "roomID", client.RoomID, "userID", userID)

		for _, otherP := range room.Participants {
			if otherP.PeerConnection != nil && leavingTrack != nil {
				for _, sender := range otherP.PeerConnection.GetSenders() {
					if sender.Track() == leavingTrack {
						_ = otherP.PeerConnection.RemoveTrack(sender)
					}
				}
				remaining = append(remaining, otherP)
			}
		}
	}

	if len(room.Participants) == 0 {
		m.mu.Lock()
		delete(m.rooms, client.RoomID)
		m.mu.Unlock()
	}
	room.mu.Unlock()

	for _, otherP := range remaining {
		go m.renegotiateUser(otherP, nil)
	}

	m.BroadcastVCState(bot, client.RoomID)
}

func (m *VCManager) HandleClientOffer(bot *td.Client, client *Client, sdp string) {
	userID, _, _, _, _, _ := client.GetInfo()
	if userID == 0 {
		return
	}

	room := m.GetOrCreateRoom(client.RoomID)
	room.mu.Lock()
	p, exists := room.Participants[userID]
	if !exists {
		room.mu.Unlock()
		return
	}

	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	if p.PeerConnection != nil {
		_ = p.PeerConnection.Close()
	}

	pc, err := m.webrtcAPI.NewPeerConnection(config)
	if err != nil {
		room.mu.Unlock()
		log.Error("[VCManager] Failed to create PeerConnection", "error", err)
		return
	}
	p.PeerConnection = pc

	// Add existing participants' tracks to this user's PeerConnection
	for otherID, otherP := range room.Participants {
		if otherID != userID && otherP.AudioTrack != nil {
			_, _ = pc.AddTrack(otherP.AudioTrack)
		}
	}

	room.mu.Unlock()

	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Info("[VCManager] PeerConnection state changed", "userID", userID, "state", state.String())
	})

	pc.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Info("[VCManager] ICE connection state changed", "userID", userID, "state", state.String())
		if state == webrtc.ICEConnectionStateFailed || state == webrtc.ICEConnectionStateDisconnected {
			log.Warn("[VCManager] ICE connection issue", "userID", userID, "state", state.String())
		}
	})

	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}
		candJSON, err := json.Marshal(c.ToJSON())
		if err != nil {
			log.Error("[VCManager] Failed to marshal ICE candidate", "userID", userID, "error", err)
			return
		}
		msg := map[string]any{
			"event": "vc_candidate",
			"data": map[string]any{
				"candidate": string(candJSON),
			},
		}
		payload, _ := json.Marshal(msg)
		_ = client.SendMessage(string(payload))
	})

	pc.OnTrack(func(remoteTrack *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		log.Info("[VCManager] OnTrack received for user", "userID", userID, "codec", remoteTrack.Codec().MimeType, "ssrc", remoteTrack.SSRC())
		for {
			pkt, _, err := remoteTrack.ReadRTP()
			if err != nil {
				log.Error("[VCManager] Error reading RTP from remote track", "userID", userID, "error", err)
				return
			}

			room.mu.RLock()
			canSpeak := p.CanSpeak(room.AllowEveryoneSpeak)
			if !canSpeak {
				room.mu.RUnlock()
				continue
			}

			if p.AudioTrack != nil {
				if writeErr := p.AudioTrack.WriteRTP(pkt); writeErr != nil {
					log.Error("[VCManager] Error writing RTP to AudioTrack", "userID", userID, "error", writeErr)
				}
			}
			room.mu.RUnlock()
		}
	})

	offer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  sdp,
	}

	if err := pc.SetRemoteDescription(offer); err != nil {
		log.Error("[VCManager] Failed to set remote description", "error", err)
		return
	}

	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		log.Error("[VCManager] Failed to create answer", "error", err)
		return
	}

	if err := pc.SetLocalDescription(answer); err != nil {
		log.Error("[VCManager] Failed to set local description", "error", err)
		return
	}

	ansMsg := map[string]any{
		"event": "vc_answer",
		"data": map[string]any{
			"sdp": answer.SDP,
		},
	}
	payload, _ := json.Marshal(ansMsg)
	_ = client.SendMessage(string(payload))

	// Notify existing participants of new track via renegotiation
	room.mu.RLock()
	for otherID, otherP := range room.Participants {
		if otherID != userID && otherP.PeerConnection != nil && otherP.Client != nil {
			go m.renegotiateUser(otherP, p.AudioTrack)
		}
	}
	room.mu.RUnlock()
}

func (m *VCManager) renegotiateUser(target *VCParticipant, newTrack *webrtc.TrackLocalStaticRTP) {
	if target.PeerConnection == nil || target.Client == nil {
		return
	}

	if target.PeerConnection.SignalingState() != webrtc.SignalingStateStable {
		log.Warn("[VCManager] Skipping renegotiation, SignalingState not stable", "userID", target.UserID, "state", target.PeerConnection.SignalingState().String())
		return
	}

	if newTrack != nil {
		alreadyHasTrack := false
		for _, sender := range target.PeerConnection.GetSenders() {
			if sender.Track() == newTrack {
				alreadyHasTrack = true
				break
			}
		}
		if !alreadyHasTrack {
			if _, err := target.PeerConnection.AddTrack(newTrack); err != nil {
				log.Error("[VCManager] Failed to add track during renegotiation", "userID", target.UserID, "error", err)
				return
			}
		}
	}

	offer, err := target.PeerConnection.CreateOffer(nil)
	if err != nil {
		log.Error("[VCManager] Failed to create offer during renegotiation", "userID", target.UserID, "error", err)
		return
	}

	if err := target.PeerConnection.SetLocalDescription(offer); err != nil {
		log.Error("[VCManager] Failed to set local description during renegotiation", "userID", target.UserID, "error", err)
		return
	}

	msg := map[string]any{
		"event": "vc_offer",
		"data": map[string]any{
			"sdp": offer.SDP,
		},
	}
	payload, _ := json.Marshal(msg)
	_ = target.Client.SendMessage(string(payload))
}

func (m *VCManager) HandleClientAnswer(client *Client, sdp string) {
	userID, _, _, _, _, _ := client.GetInfo()
	if userID == 0 {
		return
	}

	room := m.GetOrCreateRoom(client.RoomID)
	room.mu.RLock()
	p, exists := room.Participants[userID]
	room.mu.RUnlock()

	if !exists || p.PeerConnection == nil {
		return
	}

	answer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  sdp,
	}
	_ = p.PeerConnection.SetRemoteDescription(answer)
}

func (m *VCManager) HandleCandidate(client *Client, candStr string) {
	userID, _, _, _, _, _ := client.GetInfo()
	if userID == 0 {
		return
	}

	room := m.GetOrCreateRoom(client.RoomID)
	room.mu.RLock()
	p, exists := room.Participants[userID]
	room.mu.RUnlock()

	if !exists || p.PeerConnection == nil {
		log.Warn("[VCManager] HandleCandidate: participant or PeerConnection not found", "userID", userID)
		return
	}

	var cand webrtc.ICECandidateInit
	if err := json.Unmarshal([]byte(candStr), &cand); err != nil {
		log.Error("[VCManager] Failed to unmarshal ICE candidate", "userID", userID, "error", err)
		return
	}
	if err := p.PeerConnection.AddICECandidate(cand); err != nil {
		log.Error("[VCManager] Failed to add ICE candidate", "userID", userID, "error", err)
	}
}

func (m *VCManager) SetSelfMute(bot *td.Client, client *Client, muted bool) {
	userID, _, _, _, _, _ := client.GetInfo()
	if userID == 0 {
		return
	}

	room := m.GetOrCreateRoom(client.RoomID)
	room.mu.Lock()
	p, exists := room.Participants[userID]
	if exists {
		p.IsSelfMuted = muted
		if muted {
			p.IsSpeaking = false
		}
	}
	room.mu.Unlock()

	m.BroadcastVCState(bot, client.RoomID)
}

func (m *VCManager) SetSpeaking(bot *td.Client, client *Client, speaking bool) {
	userID, _, _, _, _, _ := client.GetInfo()
	if userID == 0 {
		return
	}

	room := m.GetOrCreateRoom(client.RoomID)
	room.mu.Lock()
	p, exists := room.Participants[userID]
	changed := false
	if exists {
		canSpeak := p.CanSpeak(room.AllowEveryoneSpeak)
		if !canSpeak {
			speaking = false
		}
		if p.IsSpeaking != speaking {
			p.IsSpeaking = speaking
			changed = true
		}
	}
	room.mu.Unlock()

	if changed {
		spkMsg := map[string]any{
			"event": "vc_user_speaking",
			"data": map[string]any{
				"userId":     userID,
				"isSpeaking": speaking,
			},
		}
		payload, _ := json.Marshal(spkMsg)
		HubInstance.BroadcastRoomMessage(client.RoomID, string(payload))
	}
}

func (m *VCManager) AdminMuteUser(bot *td.Client, client *Client, targetUserID int64, muted bool) {
	userID, isAdmin, _, _, _, _ := client.GetInfo()
	if userID == 0 || !isAdmin {
		sendError(client, "Admin permission required to mute users in VC.")
		return
	}

	room := m.GetOrCreateRoom(client.RoomID)
	room.mu.Lock()
	p, exists := room.Participants[targetUserID]
	if exists {
		p.IsAdminMuted = muted
		if muted {
			p.IsSpeaking = false
		}
	}
	room.mu.Unlock()

	m.BroadcastVCState(bot, client.RoomID)
}

func (m *VCManager) AdminMuteAll(bot *td.Client, client *Client, muted bool) {
	userID, isAdmin, _, _, _, _ := client.GetInfo()
	if userID == 0 || !isAdmin {
		sendError(client, "Admin permission required to manage VC users.")
		return
	}

	room := m.GetOrCreateRoom(client.RoomID)
	room.mu.Lock()
	for _, p := range room.Participants {
		if !p.IsAdmin {
			p.IsAdminMuted = muted
			if muted {
				p.IsSpeaking = false
			}
		}
	}
	room.mu.Unlock()

	m.BroadcastVCState(bot, client.RoomID)
}

func (m *VCManager) AdminSetPermission(bot *td.Client, client *Client, allowEveryone bool) {
	userID, isAdmin, _, _, _, _ := client.GetInfo()
	if userID == 0 || !isAdmin {
		sendError(client, "Admin permission required to change VC room settings.")
		return
	}

	room := m.GetOrCreateRoom(client.RoomID)
	room.mu.Lock()
	room.AllowEveryoneSpeak = allowEveryone
	if !allowEveryone {
		for _, p := range room.Participants {
			if !p.IsAdmin {
				p.IsSpeaking = false
			}
		}
	}
	room.mu.Unlock()

	m.BroadcastVCState(bot, client.RoomID)
}
