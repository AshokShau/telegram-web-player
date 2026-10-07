/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/telegram-web-player
 */

package webapp

import (
	"ashokshau/tg-web/internal/cache"
	"ashokshau/tg-web/internal/config"
	"ashokshau/tg-web/internal/db"
	"ashokshau/tg-web/internal/downloader"
	"ashokshau/tg-web/internal/utils"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	td "github.com/AshokShau/gotdbot"
	"golang.org/x/net/websocket"
)

type Client struct {
	mu              sync.RWMutex
	writeMu         sync.Mutex
	Conn            *websocket.Conn
	RoomID          int64
	UserID          int64
	IsAdmin         bool
	IsAuth          bool
	CanControl      bool
	CanPlay         bool
	AllowsWriteToPM bool
	InitData        *WebAppInitData
	TelegramContext TelegramWebAppContext
	outbox          chan string
	done            chan struct{}
	shutdown        chan struct{}
	requestID       string
	command         string
	sessionActive   bool
	sessionEnded    bool
}

// startWriter owns the connection's only writer. Bounded queues preserve event order
// and disconnect slow consumers instead of accumulating broadcast goroutines.
func (c *Client) startWriter() {
	c.outbox = make(chan string, 128)
	c.done = make(chan struct{})
	c.shutdown = make(chan struct{})
	go func() {
		defer close(c.done)
		for {
			select {
			case payload := <-c.outbox:
				if err := c.writeMessage(payload); err != nil {
					_ = c.Conn.Close()
					return
				}
			case <-c.shutdown:
				return
			}
		}
	}()
}

func (c *Client) writeMessage(payload string) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	if c.Conn == nil {
		return nil
	}
	_ = c.Conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	err := websocket.Message.Send(c.Conn, payload)
	_ = c.Conn.SetWriteDeadline(time.Time{})
	return err
}

func (c *Client) SendMessage(payload string) error {
	if c.outbox == nil {
		return c.writeMessage(payload)
	}
	select {
	case <-c.done:
		return errors.New("connection closed")
	default:
	}
	select {
	case c.outbox <- payload:
		return nil
	default:
		if c.Conn != nil {
			_ = c.Conn.Close()
		}
		return errors.New("client outbound queue is full")
	}
}

func (c *Client) GetInfo() (userID int64, isAdmin bool, canControl bool, canPlay bool, allowsWriteToPM bool, initData *WebAppInitData) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.UserID, c.IsAdmin, c.CanControl, c.CanPlay, c.AllowsWriteToPM, c.InitData
}

func (c *Client) SetPermissions(permissions roomPermissions) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.IsAdmin = permissions.IsAdmin
	c.IsAuth = permissions.IsAuth
	c.CanControl = permissions.CanControl
	c.CanPlay = permissions.CanPlay
}

func (c *Client) sendUserInfo() {
	c.mu.RLock()
	data := map[string]any{"userId": c.UserID, "isAdmin": c.IsAdmin, "isAuth": c.IsAuth,
		"canControl": c.CanControl, "canPlay": c.CanPlay, "allowsWriteToPM": c.AllowsWriteToPM}
	c.mu.RUnlock()
	payload, _ := json.Marshal(map[string]any{"event": "user_info", "data": data})
	_ = c.SendMessage(string(payload))
}

func (c *Client) SetUser(userID int64, initData *WebAppInitData, allowsWrite bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.UserID = userID
	c.InitData = initData
	c.TelegramContext = initData.telegramContext()
	c.AllowsWriteToPM = allowsWrite
}

func (c *Client) GetTelegramContext() TelegramWebAppContext {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.TelegramContext
}

type ListenerInfo struct {
	UserID    int64  `json:"userId"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName,omitempty"`
	Username  string `json:"username,omitempty"`
	PhotoURL  string `json:"photoUrl,omitempty"`
	IsAdmin   bool   `json:"isAdmin"`
}

type Hub struct {
	mu          sync.RWMutex
	broadcastMu sync.Mutex
	revision    uint64
	clients     map[int64][]*Client
}

var HubInstance = &Hub{
	clients: make(map[int64][]*Client),
}

// claimSession changes membership under one lock. Opening more tabs does not
// evict the current session unless the user explicitly chooses to take over.
func (h *Hub) claimSession(c *Client, takeOver bool) ([]*Client, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	c.mu.RLock()
	userID, ended := c.UserID, c.sessionEnded
	c.mu.RUnlock()
	if userID == 0 || ended {
		return nil, false
	}
	if slices.Contains(h.clients[c.RoomID], c) {
		return nil, true
	}
	var replaced []*Client
	for _, roomClients := range h.clients {
		for _, client := range roomClients {
			id, _, _, _, _, _ := client.GetInfo()
			if client != c && id == userID {
				if !takeOver {
					return nil, false
				}
				replaced = append(replaced, client)
			}
		}
	}
	for _, previous := range replaced {
		previous.mu.Lock()
		previous.sessionActive, previous.sessionEnded = false, true
		previous.mu.Unlock()
		roomClients := slices.DeleteFunc(h.clients[previous.RoomID], func(client *Client) bool { return client == previous })
		if len(roomClients) == 0 {
			delete(h.clients, previous.RoomID)
		} else {
			h.clients[previous.RoomID] = roomClients
		}
	}
	c.mu.Lock()
	c.sessionActive = true
	c.mu.Unlock()
	h.clients[c.RoomID] = append(h.clients[c.RoomID], c)
	return replaced, true
}

func (h *Hub) Register(bot *td.Client, c *Client, takeOver bool) bool {
	replaced, accepted := h.claimSession(c, takeOver)
	if !accepted {
		return false
	}
	for _, client := range replaced {
		payload, _ := json.Marshal(map[string]any{
			"event": "duplicate_session",
			"data":  "Listening switched to another tab or device. This session will stay disconnected.",
		})

		if client.Conn != nil {
			_ = client.writeMessage(string(payload))
			_ = client.Conn.Close()
		} else {
			_ = client.SendMessage(string(payload))
		}
		VCManagerInstance.LeaveVC(bot, client)
		Manager.CheckListenersCount(bot, client.RoomID)
		if client.RoomID != c.RoomID {
			h.BroadcastRoomState(bot, client.RoomID)
		}
	}
	Manager.CheckListenersCount(bot, c.RoomID)
	return true
}

func (h *Hub) Unregister(bot *td.Client, c *Client) bool {
	h.mu.Lock()

	roomClients, exists := h.clients[c.RoomID]
	if !exists {
		h.mu.Unlock()
		return false
	}

	removed := false
	for i, client := range roomClients {
		if client == c {
			removed = true
			c.mu.Lock()
			c.sessionActive = false
			c.mu.Unlock()
			copy(roomClients[i:], roomClients[i+1:])
			roomClients[len(roomClients)-1] = nil
			h.clients[c.RoomID] = roomClients[:len(roomClients)-1]
			log.Info("[WebApp] Client left room", "roomId", c.RoomID, "userID", c.UserID)
			break
		}
	}

	if len(h.clients[c.RoomID]) == 0 {
		delete(h.clients, c.RoomID)
	}
	h.mu.Unlock()

	if !removed {
		return false
	}
	VCManagerInstance.LeaveVC(bot, c)
	Manager.CheckListenersCount(bot, c.RoomID)
	return true
}

func (h *Hub) GetListeners(roomID int64) []ListenerInfo {
	h.mu.RLock()
	clients := make([]*Client, len(h.clients[roomID]))
	copy(clients, h.clients[roomID])
	h.mu.RUnlock()

	seen := make(map[int64]bool)
	listeners := make([]ListenerInfo, 0, len(clients))

	for _, client := range clients {
		userID, isAdmin, _, _, _, initData := client.GetInfo()
		if userID != 0 {
			if seen[userID] {
				continue
			}
			seen[userID] = true

			info := ListenerInfo{
				UserID:  userID,
				IsAdmin: isAdmin,
			}
			if initData != nil && initData.User != nil {
				u := initData.User
				info.FirstName = u.FirstName
				info.LastName = u.LastName
				info.Username = u.Username
				info.PhotoURL = u.PhotoURL
			}
			if info.FirstName == "" {
				info.FirstName = "Listener"
			}
			listeners = append(listeners, info)
		} else {
			listeners = append(listeners, ListenerInfo{
				UserID:    0,
				FirstName: "Anonymous Listener",
				IsAdmin:   false,
			})
		}
	}

	return listeners
}

func (h *Hub) BroadcastRoomState(c *td.Client, roomID int64) {
	h.broadcastMu.Lock()
	defer h.broadcastMu.Unlock()
	state := Manager.GetRoomStateData(c, roomID)
	h.revision++
	state.Revision = h.revision
	state.VC.Revision = h.revision
	msg := EventMessage{
		Event: "room_state",
		Data:  state,
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		log.Error("[WebApp] Error marshaling room state", "error", err)
		return
	}

	h.mu.RLock()
	clients := make([]*Client, len(h.clients[roomID]))
	copy(clients, h.clients[roomID])
	h.mu.RUnlock()

	for _, client := range clients {
		if client == nil {
			continue
		}
		_ = client.SendMessage(string(payload))
	}
}

func (h *Hub) BroadcastRoomMessage(roomID int64, payload string) {
	h.mu.RLock()
	clients := make([]*Client, len(h.clients[roomID]))
	copy(clients, h.clients[roomID])
	h.mu.RUnlock()

	for _, client := range clients {
		if client == nil {
			continue
		}
		_ = client.SendMessage(payload)
	}
}

type ClientMessage struct {
	Type            string     `json:"type"`
	RequestID       string     `json:"requestId,omitempty"`
	RoomID          string     `json:"roomId"`
	InitData        string     `json:"initData"`
	TakeOver        bool       `json:"takeOver,omitempty"`
	PositionSeconds float64    `json:"positionSeconds"`
	ClientTime      int64      `json:"clientTime"`
	Query           string     `json:"query,omitempty"`
	Track           *TrackData `json:"track,omitempty"`
	Force           bool       `json:"force,omitempty"`
	Count           int        `json:"count,omitempty"`
	Index           int        `json:"index,omitempty"`
	PlaylistID      string     `json:"playlistId,omitempty"`
	PlaylistName    string     `json:"playlistName,omitempty"`
	TrackID         string     `json:"trackId,omitempty"`
	Text            string     `json:"text,omitempty"`
	ChatEnabled     *bool      `json:"chatEnabled,omitempty"`
	ChatCooldown    *int       `json:"chatCooldown,omitempty"`
	SDP             string     `json:"sdp,omitempty"`
	Candidate       string     `json:"candidate,omitempty"`
	Muted           bool       `json:"muted,omitempty"`
	Speaking        bool       `json:"speaking,omitempty"`
	TargetUserID    int64      `json:"targetUserId,omitempty"`
}

func getClientUserName(c *Client) string {
	_, _, _, _, _, initData := c.GetInfo()
	if initData != nil && initData.User != nil {
		u := initData.User
		name := strings.TrimSpace(u.FirstName + " " + u.LastName)
		if name != "" {
			return name
		}
		if u.Username != "" {
			return "@" + u.Username
		}
	}
	return "Web User"
}

func sendError(c *Client, errMsg string) {
	sendErrorCode(c, errMsg, "")
}

func sendErrorCode(c *Client, errMsg, code string) {
	c.mu.RLock()
	requestID, command := c.requestID, c.command
	c.mu.RUnlock()
	errPayload := map[string]any{
		"requestId": requestID,
		"command":   command,
		"event":     "error",
		"data":      errMsg,
	}
	if code != "" {
		errPayload["code"] = code
	}

	payload, _ := json.Marshal(errPayload)
	_ = c.SendMessage(string(payload))
}

func handleWebSocket(bot *td.Client, ws *websocket.Conn) {
	defer ws.Close()

	req := ws.Request()
	roomStr := req.URL.Query().Get("room")
	roomID, _ := strconv.ParseInt(roomStr, 10, 64)
	if roomID == 0 {
		return
	}

	client := &Client{
		Conn:   ws,
		RoomID: roomID,
	}

	client.startWriter()
	defer func() {
		close(client.shutdown)
		if HubInstance.Unregister(bot, client) {
			HubInstance.BroadcastRoomState(bot, client.RoomID)
		}
	}()

	ws.MaxPayloadBytes = 128 * 1024
	_ = ws.SetReadDeadline(time.Now().Add(15 * time.Second))

	for {
		var msgStr string
		err := websocket.Message.Receive(ws, &msgStr)
		if err != nil {
			break
		}
		_ = ws.SetReadDeadline(time.Now().Add(60 * time.Second))

		var msg ClientMessage
		if err = json.Unmarshal([]byte(msgStr), &msg); err != nil {
			continue
		}

		handleClientMessage(bot, client, msg)
	}
}

func handleClientMessage(bot *td.Client, client *Client, msg ClientMessage) {
	client.mu.Lock()
	if client.sessionEnded {
		client.mu.Unlock()
		return
	}
	active := client.sessionActive
	client.requestID, client.command = msg.RequestID, msg.Type
	client.mu.Unlock()
	defer func() {
		if msg.RequestID != "" {
			payload, _ := json.Marshal(map[string]any{"event": "ack", "requestId": msg.RequestID, "command": msg.Type})
			_ = client.SendMessage(string(payload))
		}
	}()
	if msg.Type != "join" && msg.Type != "ping" {
		userID, _, _, _, _, _ := client.GetInfo()
		if userID == 0 {
			sendError(client, "Telegram authentication required.")
			return
		}
		if !active {
			sendErrorCode(client, "Choose Use this session before sending room commands.", "session_in_use")
			return
		}
	}
	switch msg.Type {
	case "join":
		if !initializeTelegramSession(client, msg, telegramRoomAccess{chatInstance: db.Instance.GetTelegramChatInstance}) {
			return
		}

		userID, _, _, _, allowsWriteToPM, _ := client.GetInfo()

		permissions := resolveRoomPermissions(bot, client.RoomID, userID)
		client.SetPermissions(permissions)

		if !HubInstance.Register(bot, client, msg.TakeOver) {
			sendErrorCode(client, "You’re already connected in another tab or device. Choose Use this session to switch here.", "session_in_use")
			return
		}

		log.Info("[WebApp] Client joined room", "roomId", client.RoomID, "userID", userID, "isAdmin", permissions.IsAdmin, "allowsWriteToPM", allowsWriteToPM)
		client.sendUserInfo()

		history := cache.ChatCache.GetChatHistory(client.RoomID)
		if history == nil {
			history = []*cache.ChatMessage{}
		}
		historyMsg := map[string]any{
			"event": "chat_history",
			"data": map[string]any{
				"messages": history,
			},
		}
		payload, _ := json.Marshal(historyMsg)
		_ = client.SendMessage(string(payload))

		HubInstance.BroadcastRoomState(bot, client.RoomID)
		VCManagerInstance.BroadcastVCState(bot, client.RoomID)

	case "vc_join":
		VCManagerInstance.JoinVC(bot, client)

	case "vc_leave":
		VCManagerInstance.LeaveVC(bot, client)

	case "vc_offer":
		VCManagerInstance.HandleClientOffer(bot, client, msg.SDP)

	case "vc_answer":
		VCManagerInstance.HandleClientAnswer(client, msg.SDP)

	case "vc_candidate":
		VCManagerInstance.HandleCandidate(client, msg.Candidate)

	case "vc_mute_self":
		VCManagerInstance.SetSelfMute(bot, client, msg.Muted)

	case "vc_speaking":
		VCManagerInstance.SetSpeaking(bot, client, msg.Speaking)

	case "vc_admin_mute":
		VCManagerInstance.AdminMuteUser(bot, client, msg.TargetUserID, msg.Muted)

	case "vc_admin_set_join_muted":
		VCManagerInstance.AdminSetJoinMuted(bot, client, msg.Muted)

	case "get_vc_state":
		VCManagerInstance.BroadcastVCState(bot, client.RoomID)

	case "write_access_granted":
		client.mu.Lock()
		client.AllowsWriteToPM = true
		if client.InitData != nil && client.InitData.User != nil {
			client.InitData.User.AllowsWriteToPM = true
		}
		userID := client.UserID
		log.Info("[WebApp] Bot write permission granted for user", "userID", userID)
		client.mu.Unlock()
		client.sendUserInfo()

	case "ping":
		pong := map[string]any{
			"event": "pong",
			"data": map[string]any{
				"clientTime": msg.ClientTime,
				"serverTime": time.Now().UnixMilli(),
			},
		}
		payload, _ := json.Marshal(pong)
		_ = client.SendMessage(string(payload))

	case "search":
		query := strings.TrimSpace(msg.Query)
		if query == "" {
			return
		}
		if len(query) > 200 {
			query = query[:200]
		}
		wrapper := downloader.NewDlWrapper(query)
		res, err := wrapper.Search()
		var tracks []*TrackData
		if err == nil && res != nil && len(res.Results) > 0 {
			for _, t := range res.Results {
				tracks = append(tracks, &TrackData{
					ID:        t.Id,
					Title:     t.Title,
					Artist:    t.Channel,
					Duration:  t.Duration,
					Thumbnail: t.Thumbnail,
					Platform:  t.Platform,
					URL:       t.Url,
				})
			}
		}
		searchResp := map[string]any{
			"event": "search_results",
			"data": map[string]any{
				"query":   query,
				"results": tracks,
			},
		}
		payload, _ := json.Marshal(searchResp)
		_ = client.SendMessage(string(payload))

	case "mix", "recommendations":
		query := strings.TrimSpace(msg.Query)
		seedTrackID := ""
		if msg.Track != nil && msg.Track.ID != "" {
			seedTrackID = msg.Track.ID
		} else if query == "" {
			playing := cache.ChatCache.GetPlayingTrack(client.RoomID)
			if playing != nil {
				seedTrackID = playing.TrackID
			}
		}

		if seedTrackID == "" && query == "" {
			sendError(client, "No current playing track or query provided for recommendations.")
			return
		}

		limit := 10
		if msg.Count > 0 && msg.Count <= 25 {
			limit = msg.Count
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		recTracks, err := downloader.GetYouTubeMix(ctx, query, seedTrackID, limit)
		cancel()

		history := cache.ChatCache.GetAutoplayHistory(client.RoomID)
		playingTrack := cache.ChatCache.GetPlayingTrack(client.RoomID)
		playingID := ""
		if playingTrack != nil {
			playingID = playingTrack.TrackID
		}

		var tracks []*TrackData
		if err == nil && len(recTracks) > 0 {
			for _, t := range recTracks {
				if t.Id == "" || t.Id == seedTrackID || t.Id == playingID || slices.Contains(history, t.Id) {
					continue
				}
				tracks = append(tracks, &TrackData{
					ID:        t.Id,
					Title:     t.Title,
					Artist:    t.Channel,
					Duration:  t.Duration,
					Thumbnail: t.Thumbnail,
					Platform:  t.Platform,
					URL:       t.Url,
				})
			}
		}

		recError := ""
		if err != nil {
			recError = "Recommendations are unavailable. Try again."
		}
		recResp := map[string]any{
			"event":     "recommendations_results",
			"requestId": msg.RequestID,
			"data": map[string]any{
				"error":   recError,
				"query":   query,
				"results": tracks,
			},
		}
		payload, _ := json.Marshal(recResp)
		_ = client.SendMessage(string(payload))

	case "play", "enqueue":
		userID, _, canControl, canPlay, allowsWriteToPM, _ := client.GetInfo()
		if userID == 0 {
			sendError(client, "Telegram authentication required to play music.")
			return
		}
		if !allowsWriteToPM {
			sendError(client, "Telegram bot write permission required before playback can be enabled.")
			return
		}

		if msg.Force {
			if !canControl {
				sendError(client, "Permission required to force play in this chat.")
				return
			}
		} else {
			if !canPlay {
				sendError(client, "Play mode is enabled. Permission required to play music in this chat.")
				return
			}
		}

		if queueLen := cache.ChatCache.GetQueueLength(client.RoomID); queueLen >= 10 {
			sendError(client, "Queue is full (max 10 tracks).")
			return
		}

		var saveCache *utils.PlayerCache
		if msg.Track != nil && msg.Track.ID != "" {
			if _track := cache.ChatCache.GetTrackIfExists(client.RoomID, msg.Track.ID); _track != nil {
				sendError(client, "Track is already in queue or playing.")
				return
			}
			if msg.Track.Duration > config.SongDurationLimit {
				sendError(client, fmt.Sprintf("Sorry, song exceeds max duration of %d minutes.", config.SongDurationLimit/60))
				return
			}
			url := msg.Track.URL
			if url == "" {
				url = "https://www.youtube.com/watch?v=" + msg.Track.ID
			}
			platform := msg.Track.Platform
			if platform == "" {
				platform = utils.YouTube
			}

			saveCache = &utils.PlayerCache{
				URL:       url,
				Name:      msg.Track.Title,
				User:      getClientUserName(client),
				Thumbnail: msg.Track.Thumbnail,
				TrackID:   msg.Track.ID,
				Duration:  msg.Track.Duration,
				Channel:   msg.Track.Artist,
				Platform:  platform,
			}
		} else if msg.PlaylistID != "" && msg.TrackID != "" {
			pl, err := db.Instance.GetPlaylist(msg.PlaylistID)
			if err != nil || pl == nil {
				sendError(client, "Playlist not found.")
				return
			}
			var found *db.Song
			for _, s := range pl.Songs {
				if s.TrackID == msg.TrackID {
					found = &s
					break
				}
			}
			if found == nil {
				sendError(client, "Song not found in playlist.")
				return
			}
			if _track := cache.ChatCache.GetTrackIfExists(client.RoomID, found.TrackID); _track != nil {
				sendError(client, "Track is already in queue or playing.")
				return
			}
			if found.Duration > config.SongDurationLimit {
				sendError(client, fmt.Sprintf("Sorry, song exceeds max duration of %d minutes.", config.SongDurationLimit/60))
				return
			}
			saveCache = &utils.PlayerCache{
				URL:       found.URL,
				Name:      found.Name,
				Channel:   found.Artist,
				Thumbnail: found.Thumbnail,
				User:      getClientUserName(client),
				TrackID:   found.TrackID,
				Duration:  found.Duration,
				Platform:  found.Platform,
			}
		} else if msg.Query != "" {
			wrapper := downloader.NewDlWrapper(msg.Query)
			searchResult, err := wrapper.Search()
			if err != nil || searchResult == nil || len(searchResult.Results) == 0 {
				sendError(client, "No results found for query.")
				return
			}
			song := searchResult.Results[0]
			if _track := cache.ChatCache.GetTrackIfExists(client.RoomID, song.Id); _track != nil {
				sendError(client, "Track is already in queue or playing.")
				return
			}
			if song.Duration > config.SongDurationLimit {
				sendError(client, fmt.Sprintf("Sorry, song exceeds max duration of %d minutes.", config.SongDurationLimit/60))
				return
			}
			saveCache = &utils.PlayerCache{
				URL:       song.Url,
				Name:      song.Title,
				User:      getClientUserName(client),
				Thumbnail: song.Thumbnail,
				TrackID:   song.Id,
				Duration:  song.Duration,
				Channel:   song.Channel,
				Views:     song.Views,
				Platform:  song.Platform,
			}
		} else {
			sendError(client, "Invalid track or query provided.")
			return
		}

		if msg.Force {
			room := Manager.getOrCreate(bot, client.RoomID)
			room.mu.Lock()
			room.isTransitioning = false
			room.mu.Unlock()

			qLen := cache.ChatCache.AddSongToFront(client.RoomID, saveCache)
			if qLen > 1 {
				_ = PlayNext(bot, client.RoomID)
			} else {
				go func() { _ = PlayTrack(bot, client.RoomID, saveCache) }()
			}
		} else {
			qLen := cache.ChatCache.AddSong(client.RoomID, saveCache)
			if qLen == 1 {
				go func() { _ = PlayTrack(bot, client.RoomID, saveCache) }()
			}
		}

		HubInstance.BroadcastRoomState(bot, client.RoomID)

	case "seek":
		userID, _, canControl, _, allowsWriteToPM, _ := client.GetInfo()
		if userID == 0 || !allowsWriteToPM {
			sendError(client, "Telegram bot write permission required to control playback.")
			return
		}
		if !canControl {
			sendError(client, "Permission required to seek")
			return
		}
		_, _ = Manager.SeekRoom(bot, client.RoomID, msg.PositionSeconds)

	case "pause":
		userID, _, canControl, _, allowsWriteToPM, _ := client.GetInfo()
		if userID == 0 || !allowsWriteToPM {
			sendError(client, "Telegram bot write permission required to control playback.")
			return
		}
		if !canControl {
			sendError(client, "Permission required to pause")
			return
		}
		_, _ = Manager.Pause(bot, client.RoomID)

	case "resume":
		userID, _, canControl, _, allowsWriteToPM, _ := client.GetInfo()
		if userID == 0 || !allowsWriteToPM {
			sendError(client, "Telegram bot write permission required to control playback.")
			return
		}
		if !canControl {
			sendError(client, "Permission required to resume")
			return
		}
		_, _ = Manager.Resume(bot, client.RoomID)

	case "skip":
		userID, _, canControl, _, allowsWriteToPM, _ := client.GetInfo()
		if userID == 0 || !allowsWriteToPM {
			sendError(client, "Telegram bot write permission required to control playback.")
			return
		}
		if !canControl {
			sendError(client, "Permission required to change track")
			return
		}
		_ = PlayNext(bot, client.RoomID)

	case "track_end":
		userID, _, canControl, _, allowsWriteToPM, _ := client.GetInfo()
		if userID == 0 || !allowsWriteToPM {
			sendError(client, "Telegram bot write permission required to control playback.")
			return
		}
		if !canControl {
			sendError(client, "Permission required to change track")
			return
		}
		_ = PlayNextForTrack(bot, client.RoomID, msg.TrackID)

	case "stop":
		userID, _, canControl, _, allowsWriteToPM, _ := client.GetInfo()
		if userID == 0 || !allowsWriteToPM {
			sendError(client, "Telegram bot write permission required to control playback.")
			return
		}
		if !canControl {
			sendError(client, "Permission required to stop")
			return
		}
		Manager.Stop(bot, client.RoomID)

	case "loop":
		userID, isAdmin, _, _, allowsWriteToPM, _ := client.GetInfo()
		if userID == 0 || !allowsWriteToPM {
			sendError(client, "Telegram bot write permission required to control playback.")
			return
		}
		if !isAdmin {
			sendError(client, "Admin permission required to change repeat settings.")
			return
		}
		if msg.Count < 0 || msg.Count > 10 {
			msg.Count = 0
		}
		cache.ChatCache.SetLoopCount(client.RoomID, msg.Count)
		HubInstance.BroadcastRoomState(bot, client.RoomID)

	case "autoplay":
		userID, isAdmin, _, _, allowsWriteToPM, _ := client.GetInfo()
		if userID == 0 || !allowsWriteToPM {
			sendError(client, "Telegram bot write permission required to control playback.")
			return
		}
		if !isAdmin {
			sendError(client, "Admin permission required to change autoplay settings.")
			return
		}
		if !cache.ChatCache.IsActive(client.RoomID) {
			sendError(client, "Bot is not streaming.")
			return
		}
		curr := cache.ChatCache.GetAutoplay(client.RoomID)
		cache.ChatCache.SetAutoplay(client.RoomID, !curr)
		HubInstance.BroadcastRoomState(bot, client.RoomID)

	case "remove":
		_, _, canControl, _, _, _ := client.GetInfo()
		if !canControl {
			sendError(client, "Permission required to remove tracks from queue.")
			return
		}
		if msg.Index > 0 {
			cache.ChatCache.RemoveTrack(client.RoomID, msg.Index)
			HubInstance.BroadcastRoomState(bot, client.RoomID)
		}

	case "clear_queue":
		_, _, canControl, _, _, _ := client.GetInfo()
		if !canControl {
			sendError(client, "Permission required to clear queue.")
			return
		}
		queue := cache.ChatCache.GetQueue(client.RoomID)
		for i := len(queue) - 1; i >= 1; i-- {
			cache.ChatCache.RemoveTrack(client.RoomID, i)
		}
		HubInstance.BroadcastRoomState(bot, client.RoomID)

	case "get_playlists":
		userID, _, _, _, _, _ := client.GetInfo()
		if userID == 0 {
			sendError(client, "Telegram login required to access playlists.")
			return
		}
		playlists, err := db.Instance.GetUserPlaylists(userID)
		if err != nil {
			sendError(client, "Failed to fetch playlists: "+err.Error())
			return
		}
		resp := map[string]any{
			"event": "user_playlists",
			"data": map[string]any{
				"playlists": playlists,
			},
		}
		payload, _ := json.Marshal(resp)
		_ = client.SendMessage(string(payload))

	case "rename_playlist":
		userID, _, _, _, _, _ := client.GetInfo()
		if userID == 0 {
			sendError(client, "Telegram login required to rename playlists.")
			return
		}
		if msg.PlaylistID == "" {
			sendError(client, "Playlist ID required.")
			return
		}
		newName := strings.TrimSpace(msg.PlaylistName)
		if newName == "" {
			sendError(client, "Playlist name cannot be empty.")
			return
		}
		if len([]rune(newName)) > 40 {
			newName = string([]rune(newName)[:40])
		}
		pl, err := db.Instance.GetPlaylist(msg.PlaylistID)
		if err != nil || pl == nil {
			sendError(client, "Playlist not found.")
			return
		}
		if pl.UserID != userID {
			sendError(client, "You do not own this playlist.")
			return
		}
		if err := db.Instance.RenamePlaylist(msg.PlaylistID, newName, userID); err != nil {
			sendError(client, "Failed to rename playlist: "+err.Error())
			return
		}
		updatedList, _ := db.Instance.GetUserPlaylists(userID)
		resp := map[string]any{
			"event": "playlist_renamed",
			"data": map[string]any{
				"playlistId": msg.PlaylistID,
				"name":       newName,
				"playlists":  updatedList,
			},
		}
		payload, _ := json.Marshal(resp)
		_ = client.SendMessage(string(payload))

	case "create_playlist":
		userID, _, _, _, _, _ := client.GetInfo()
		if userID == 0 {
			sendError(client, "Telegram login required to create playlists.")
			return
		}
		name := strings.TrimSpace(msg.PlaylistName)
		if name == "" {
			sendError(client, "Playlist name cannot be empty.")
			return
		}
		if len([]rune(name)) > 40 {
			name = string([]rune(name)[:40])
		}
		playlists, err := db.Instance.GetUserPlaylists(userID)
		if err == nil && len(playlists) >= 10 {
			sendError(client, "You have reached the limit of 10 playlists.")
			return
		}
		plID, err := db.Instance.CreatePlaylist(name, userID)
		if err != nil {
			sendError(client, "Failed to create playlist: "+err.Error())
			return
		}
		updatedList, _ := db.Instance.GetUserPlaylists(userID)
		resp := map[string]any{
			"event": "playlist_created",
			"data": map[string]any{
				"playlistId": plID,
				"name":       name,
				"playlists":  updatedList,
			},
		}
		payload, _ := json.Marshal(resp)
		_ = client.SendMessage(string(payload))

	case "delete_playlist":
		userID, _, _, _, _, _ := client.GetInfo()
		if userID == 0 {
			sendError(client, "Telegram login required to delete playlists.")
			return
		}
		if msg.PlaylistID == "" {
			sendError(client, "Playlist ID required.")
			return
		}
		pl, err := db.Instance.GetPlaylist(msg.PlaylistID)
		if err != nil || pl == nil {
			sendError(client, "Playlist not found.")
			return
		}
		if pl.UserID != userID {
			sendError(client, "You do not own this playlist.")
			return
		}
		if err := db.Instance.DeletePlaylist(msg.PlaylistID, userID); err != nil {
			sendError(client, "Failed to delete playlist: "+err.Error())
			return
		}
		updatedList, _ := db.Instance.GetUserPlaylists(userID)
		resp := map[string]any{
			"event": "playlist_deleted",
			"data": map[string]any{
				"playlistId": msg.PlaylistID,
				"playlists":  updatedList,
			},
		}
		payload, _ := json.Marshal(resp)
		_ = client.SendMessage(string(payload))

	case "add_to_playlist":
		userID, _, _, _, _, _ := client.GetInfo()
		if userID == 0 {
			sendError(client, "Telegram login required to add to playlist.")
			return
		}

		var song db.Song
		if msg.Track != nil && msg.Track.ID != "" {
			url := msg.Track.URL
			if url == "" {
				sendError(client, "Track URL cannot be empty.")
				return
			}
			platform := msg.Track.Platform
			if platform == "" {
				sendError(client, "Track Platform cannot be empty.")
				return
			}
			song = db.Song{
				URL:       url,
				Name:      msg.Track.Title,
				Artist:    msg.Track.Artist,
				Thumbnail: msg.Track.Thumbnail,
				TrackID:   msg.Track.ID,
				Duration:  msg.Track.Duration,
				Platform:  platform,
			}
		} else {
			playing := cache.ChatCache.GetPlayingTrack(client.RoomID)
			if playing == nil {
				sendError(client, "No active track playing to add to playlist.")
				return
			}
			song = db.Song{
				URL:       playing.URL,
				Name:      playing.Name,
				Artist:    playing.Channel,
				Thumbnail: playing.Thumbnail,
				TrackID:   playing.TrackID,
				Duration:  playing.Duration,
				Platform:  playing.Platform,
			}
		}

		playlists, err := db.Instance.GetUserPlaylists(userID)
		if err != nil {
			sendError(client, "Error fetching playlists.")
			return
		}

		var targetID string
		if msg.PlaylistID != "" {
			pl, err := db.Instance.GetPlaylist(msg.PlaylistID)
			if err != nil || pl == nil {
				sendError(client, "Playlist not found.")
				return
			}
			if pl.UserID != userID {
				sendError(client, "You do not own this playlist.")
				return
			}
			targetID = msg.PlaylistID
		} else {
			if len(playlists) == 0 {
				targetID, err = db.Instance.CreatePlaylist("My Playlist", userID)
				if err != nil {
					sendError(client, "Failed to create default playlist.")
					return
				}
			} else {
				targetID = playlists[0].ID
			}
		}

		if err := db.Instance.AddSongToPlaylist(targetID, song, userID); err != nil {
			sendError(client, "Failed to add song to playlist.")
			return
		}

		pl, _ := db.Instance.GetPlaylist(targetID)
		updatedList, _ := db.Instance.GetUserPlaylists(userID)

		resp := map[string]any{
			"event": "song_added_to_playlist",
			"data": map[string]any{
				"playlistId": targetID,
				"song":       song,
				"playlist":   pl,
				"playlists":  updatedList,
			},
		}
		payload, _ := json.Marshal(resp)
		_ = client.SendMessage(string(payload))

	case "remove_from_playlist":
		userID, _, _, _, _, _ := client.GetInfo()
		if userID == 0 {
			sendError(client, "Telegram login required to modify playlists.")
			return
		}
		if msg.PlaylistID == "" || msg.TrackID == "" {
			sendError(client, "Playlist ID and Track ID are required.")
			return
		}
		pl, err := db.Instance.GetPlaylist(msg.PlaylistID)
		if err != nil || pl == nil {
			sendError(client, "Playlist not found.")
			return
		}
		if pl.UserID != userID {
			sendError(client, "You do not own this playlist.")
			return
		}
		if err := db.Instance.RemoveSongFromPlaylist(msg.PlaylistID, msg.TrackID, userID); err != nil {
			sendError(client, "Failed to remove song: "+err.Error())
			return
		}
		pl, _ = db.Instance.GetPlaylist(msg.PlaylistID)
		updatedList, _ := db.Instance.GetUserPlaylists(userID)
		resp := map[string]any{
			"event": "song_removed_from_playlist",
			"data": map[string]any{
				"playlistId": msg.PlaylistID,
				"trackId":    msg.TrackID,
				"playlist":   pl,
				"playlists":  updatedList,
			},
		}
		payload, _ := json.Marshal(resp)
		_ = client.SendMessage(string(payload))

	case "play_playlist", "enqueue_playlist":
		userID, _, canControl, canPlay, allowsWriteToPM, _ := client.GetInfo()
		if userID == 0 {
			sendError(client, "Telegram authentication required to play music.")
			return
		}
		if !allowsWriteToPM {
			sendError(client, "Telegram bot write permission required before playback can be enabled.")
			return
		}

		if msg.Type == "play_playlist" {
			if !canControl {
				sendError(client, "Permission required to force play in this chat.")
				return
			}
		} else {
			if !canPlay {
				sendError(client, "Play mode is restricted in this chat.")
				return
			}
		}

		if msg.PlaylistID == "" {
			sendError(client, "Playlist ID required.")
			return
		}

		pl, err := db.Instance.GetPlaylist(msg.PlaylistID)
		if err != nil || pl == nil || len(pl.Songs) == 0 {
			sendError(client, "Playlist not found or empty.")
			return
		}

		addedCount := 0
		for idx, song := range pl.Songs {
			if cache.ChatCache.GetQueueLength(client.RoomID) >= 10 {
				break
			}
			if _track := cache.ChatCache.GetTrackIfExists(client.RoomID, song.TrackID); _track != nil {
				continue
			}
			if song.Duration > config.SongDurationLimit {
				continue
			}

			saveCache := &utils.PlayerCache{
				URL:       song.URL,
				Name:      song.Name,
				Channel:   song.Artist,
				Thumbnail: song.Thumbnail,
				User:      getClientUserName(client),
				TrackID:   song.TrackID,
				Duration:  song.Duration,
				Platform:  song.Platform,
			}

			if idx == 0 && msg.Type == "play_playlist" {
				room := Manager.getOrCreate(bot, client.RoomID)
				room.mu.Lock()
				room.isTransitioning = false
				room.mu.Unlock()

				qLen := cache.ChatCache.AddSongToFront(client.RoomID, saveCache)
				if qLen > 1 {
					_ = PlayNext(bot, client.RoomID)
				} else {
					go func() { _ = PlayTrack(bot, client.RoomID, saveCache) }()
				}
			} else {
				qLen := cache.ChatCache.AddSong(client.RoomID, saveCache)
				if qLen == 1 {
					go func() { _ = PlayTrack(bot, client.RoomID, saveCache) }()
				}
			}
			addedCount++
		}

		if addedCount == 0 {
			sendError(client, "No tracks from playlist could be added (queue full or duplicates).")
		} else {
			HubInstance.BroadcastRoomState(bot, client.RoomID)
		}

	case "chat_settings":
		userID, isAdmin, _, _, _, _ := client.GetInfo()
		if userID == 0 {
			sendError(client, "Telegram authentication required to change room settings.")
			return
		}
		if !isAdmin {
			sendError(client, "Admin permission required to change chat settings.")
			return
		}
		if msg.ChatEnabled != nil {
			_ = db.Instance.SetChatEnabled(client.RoomID, *msg.ChatEnabled)
		}
		if msg.ChatCooldown != nil {
			_ = db.Instance.SetChatCooldown(client.RoomID, *msg.ChatCooldown)
		}
		HubInstance.BroadcastRoomState(bot, client.RoomID)

	case "get_chat_history":
		history := cache.ChatCache.GetChatHistory(client.RoomID)
		if history == nil {
			history = []*cache.ChatMessage{}
		}
		resp := map[string]any{
			"event": "chat_history",
			"data": map[string]any{
				"messages": history,
			},
		}
		payload, _ := json.Marshal(resp)
		_ = client.SendMessage(string(payload))

	case "chat_message":
		userID, isAdmin, _, _, _, initData := client.GetInfo()
		if userID == 0 {
			sendError(client, "Telegram authentication required to send chat messages.")
			return
		}
		chatEnabled := db.Instance.GetChatEnabled(client.RoomID)
		if !chatEnabled {
			sendError(client, "Chat is currently disabled in this room.")
			return
		}
		text := strings.TrimSpace(msg.Text)
		if text == "" {
			return
		}
		if len([]rune(text)) > 500 {
			sendError(client, "Message is too long — 500 characters maximum.")
			return
		}

		chatCooldown := db.Instance.GetChatCooldown(client.RoomID)
		allowed, remSec := cache.ChatCache.CheckAndSetCooldown(client.RoomID, userID, chatEnabled, chatCooldown)
		if !allowed {
			errResp := map[string]any{
				"event": "chat_error",
				"data": map[string]any{
					"message":          fmt.Sprintf("Slow down! Please wait %.1fs before sending another message.", remSec),
					"remainingSeconds": remSec,
				},
			}
			payload, _ := json.Marshal(errResp)
			_ = client.SendMessage(string(payload))
			return
		}

		senderName := getClientUserName(client)
		var photoURL string
		if initData != nil && initData.User != nil {
			photoURL = initData.User.PhotoURL
		}

		msgObj := &cache.ChatMessage{
			ID:        fmt.Sprintf("msg_%d_%d", time.Now().UnixNano(), userID),
			UserID:    userID,
			Sender:    senderName,
			PhotoURL:  photoURL,
			Text:      text,
			Timestamp: time.Now().UnixMilli(),
			IsAdmin:   isAdmin,
		}

		cache.ChatCache.AddChatMessage(client.RoomID, msgObj)

		bcMsg := map[string]any{
			"event": "chat_message",
			"data":  msgObj,
		}
		payload, _ := json.Marshal(bcMsg)
		HubInstance.BroadcastRoomMessage(client.RoomID, string(payload))
	}
}
