/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/FallenProjects/telegram-web-player
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
	mu         sync.RWMutex
	writeMu    sync.Mutex
	Conn       *websocket.Conn
	RoomID     int64
	UserID     int64
	IsAdmin    bool
	CanControl bool
	CanPlay    bool
	InitData   *WebAppInitData
}

func (c *Client) SendMessage(payload string) error {
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

func (c *Client) GetInfo() (userID int64, isAdmin bool, canControl bool, canPlay bool, initData *WebAppInitData) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.UserID, c.IsAdmin, c.CanControl, c.CanPlay, c.InitData
}

func (c *Client) SetPermissions(isAdmin, canControl, canPlay bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.IsAdmin = isAdmin
	c.CanControl = canControl
	c.CanPlay = canPlay
}

func (c *Client) SetUser(userID int64, initData *WebAppInitData) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.UserID = userID
	c.InitData = initData
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
	mu      sync.RWMutex
	clients map[int64][]*Client
}

var HubInstance = &Hub{
	clients: make(map[int64][]*Client),
}

func (h *Hub) Register(bot *td.Client, c *Client) {
	h.mu.Lock()
	roomClients := h.clients[c.RoomID]
	if slices.Contains(roomClients, c) {
		h.mu.Unlock()
		return
	}
	h.clients[c.RoomID] = append(h.clients[c.RoomID], c)
	h.mu.Unlock()

	if c.UserID != 0 {
		log.Info("[WebApp] Client joined room", "roomId", c.RoomID, "userID", c.UserID, "isAdmin", c.IsAdmin)
	}
	Manager.CheckListenersCount(bot, c.RoomID)
}

func (h *Hub) Unregister(bot *td.Client, c *Client) {
	h.mu.Lock()

	roomClients, exists := h.clients[c.RoomID]
	if !exists {
		h.mu.Unlock()
		return
	}

	for i, client := range roomClients {
		if client == c {
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

	Manager.CheckListenersCount(bot, c.RoomID)
}

func (h *Hub) GetListeners(roomID int64) []ListenerInfo {
	h.mu.RLock()
	clients := make([]*Client, len(h.clients[roomID]))
	copy(clients, h.clients[roomID])
	h.mu.RUnlock()

	seen := make(map[int64]bool)
	listeners := make([]ListenerInfo, 0, len(clients))

	for _, client := range clients {
		userID, isAdmin, _, _, initData := client.GetInfo()
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
	state := Manager.GetRoomStateData(c, roomID)
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
		go func(c *Client) {
			_ = c.SendMessage(string(payload))
		}(client)
	}
}

type ClientMessage struct {
	Type            string     `json:"type"`
	RoomID          string     `json:"roomId"`
	UserID          int64      `json:"userId"`
	InitData        string     `json:"initData"`
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
}

func getClientUserName(c *Client) string {
	_, _, _, _, initData := c.GetInfo()
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
	errPayload := map[string]any{
		"event": "error",
		"data":  errMsg,
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

	HubInstance.Register(bot, client)
	defer func() {
		HubInstance.Unregister(bot, client)
		HubInstance.BroadcastRoomState(bot, client.RoomID)
	}()

	HubInstance.BroadcastRoomState(bot, roomID)

	for {
		var msgStr string
		err := websocket.Message.Receive(ws, &msgStr)
		if err != nil {
			break
		}

		var msg ClientMessage
		if err = json.Unmarshal([]byte(msgStr), &msg); err != nil {
			continue
		}

		switch msg.Type {
		case "join":
			if msg.InitData != "" {
				if data, ok := verifyTelegramInitData(msg.InitData, config.Token); ok && data != nil {
					if data.User != nil {
						client.SetUser(data.User.ID, data)
					} else {
						client.SetUser(0, data)
					}
				}
			}

			userID, _, _, _, _ := client.GetInfo()
			isAdmin := isUserChatAdmin(bot, client.RoomID, userID)
			canControl := canUserControl(bot, client.RoomID, userID)
			canPlay := canUserPlay(bot, client.RoomID, userID)
			client.SetPermissions(isAdmin, canControl, canPlay)

			if msg.RoomID != "" {
				if rID, _ := strconv.ParseInt(msg.RoomID, 10, 64); rID != 0 && rID != client.RoomID {
					HubInstance.Unregister(bot, client)
					client.RoomID = rID
					HubInstance.Register(bot, client)
				}
			}

			log.Info("[WebApp] Client joined room", "roomId", client.RoomID, "userID", userID, "isAdmin", isAdmin)

			userInfoMsg := map[string]any{
				"event": "user_info",
				"data": map[string]any{
					"userId":     userID,
					"isAdmin":    isAdmin,
					"canControl": canControl,
					"canPlay":    canPlay,
				},
			}
			payload, _ := json.Marshal(userInfoMsg)
			_ = client.SendMessage(string(payload))

			HubInstance.BroadcastRoomState(bot, client.RoomID)

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
				continue
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
				continue
			}

			limit := 10
			if msg.Count > 0 && msg.Count <= 25 {
				limit = msg.Count
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			recTracks, err := downloader.GetYouTubeMix(ctx, query, seedTrackID, limit)
			cancel()

			var tracks []*TrackData
			if err == nil && len(recTracks) > 0 {
				for _, t := range recTracks {
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

			recResp := map[string]any{
				"event": "recommendations_results",
				"data": map[string]any{
					"query":   query,
					"results": tracks,
				},
			}
			payload, _ := json.Marshal(recResp)
			_ = client.SendMessage(string(payload))

		case "play", "enqueue":
			_, _, canControl, canPlay, _ := client.GetInfo()
			if msg.Force {
				if !canControl {
					sendError(client, "Permission required to force play in this chat.")
					continue
				}
			} else {
				if !canPlay {
					sendError(client, "Play mode is enabled. Permission required to play music in this chat.")
					continue
				}
			}

			if queueLen := cache.ChatCache.GetQueueLength(client.RoomID); queueLen >= 10 {
				sendError(client, "Queue is full (max 10 tracks).")
				continue
			}

			var saveCache *utils.PlayerCache
			if msg.Track != nil && msg.Track.ID != "" {
				if _track := cache.ChatCache.GetTrackIfExists(client.RoomID, msg.Track.ID); _track != nil {
					sendError(client, "Track is already in queue or playing.")
					continue
				}
				if msg.Track.Duration > config.SongDurationLimit {
					sendError(client, fmt.Sprintf("Sorry, song exceeds max duration of %d minutes.", config.SongDurationLimit/60))
					continue
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
			} else if msg.Query != "" {
				wrapper := downloader.NewDlWrapper(msg.Query)
				searchResult, err := wrapper.Search()
				if err != nil || searchResult == nil || len(searchResult.Results) == 0 {
					sendError(client, "No results found for query.")
					continue
				}
				song := searchResult.Results[0]
				if _track := cache.ChatCache.GetTrackIfExists(client.RoomID, song.Id); _track != nil {
					sendError(client, "Track is already in queue or playing.")
					continue
				}
				if song.Duration > config.SongDurationLimit {
					sendError(client, fmt.Sprintf("Sorry, song exceeds max duration of %d minutes.", config.SongDurationLimit/60))
					continue
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
				continue
			}

			if msg.Force {
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
			_, _, canControl, _, _ := client.GetInfo()
			if !canControl {
				sendError(client, "Permission required to seek")
				continue
			}
			_, _ = Manager.SeekRoom(bot, client.RoomID, msg.PositionSeconds)

		case "pause":
			_, _, canControl, _, _ := client.GetInfo()
			if !canControl {
				sendError(client, "Permission required to pause")
				continue
			}
			_, _ = Manager.Pause(bot, client.RoomID)

		case "resume":
			_, _, canControl, _, _ := client.GetInfo()
			if !canControl {
				sendError(client, "Permission required to resume")
				continue
			}
			_, _ = Manager.Resume(bot, client.RoomID)

		case "skip", "track_end":
			_, _, canControl, _, _ := client.GetInfo()
			if !canControl {
				sendError(client, "Permission required to change track")
				continue
			}
			_ = PlayNext(bot, client.RoomID)

		case "stop":
			_, _, canControl, _, _ := client.GetInfo()
			if !canControl {
				sendError(client, "Permission required to stop")
				continue
			}
			Manager.Stop(bot, client.RoomID)

		case "loop":
			_, _, canControl, _, _ := client.GetInfo()
			if !canControl {
				sendError(client, "Permission required to set loop count.")
				continue
			}
			if msg.Count < 0 || msg.Count > 10 {
				msg.Count = 0
			}
			cache.ChatCache.SetLoopCount(client.RoomID, msg.Count)
			HubInstance.BroadcastRoomState(bot, client.RoomID)

		case "autoplay":
			_, _, canControl, _, _ := client.GetInfo()
			if !canControl {
				sendError(client, "Permission required to toggle autoplay.")
				continue
			}
			curr := cache.ChatCache.GetAutoplay(client.RoomID)
			cache.ChatCache.SetAutoplay(client.RoomID, !curr)
			HubInstance.BroadcastRoomState(bot, client.RoomID)

		case "remove":
			_, _, canControl, _, _ := client.GetInfo()
			if !canControl {
				sendError(client, "Permission required to remove tracks from queue.")
				continue
			}
			if msg.Index > 0 {
				cache.ChatCache.RemoveTrack(client.RoomID, msg.Index)
				HubInstance.BroadcastRoomState(bot, client.RoomID)
			}

		case "clear_queue":
			_, _, canControl, _, _ := client.GetInfo()
			if !canControl {
				sendError(client, "Permission required to clear queue.")
				continue
			}
			queue := cache.ChatCache.GetQueue(client.RoomID)
			for i := len(queue) - 1; i >= 1; i-- {
				cache.ChatCache.RemoveTrack(client.RoomID, i)
			}
			HubInstance.BroadcastRoomState(bot, client.RoomID)

		case "get_playlists":
			userID, _, _, _, _ := client.GetInfo()
			if userID == 0 {
				sendError(client, "Telegram login required to access playlists.")
				continue
			}
			playlists, err := db.Instance.GetUserPlaylists(userID)
			if err != nil {
				sendError(client, "Failed to fetch playlists: "+err.Error())
				continue
			}
			resp := map[string]any{
				"event": "user_playlists",
				"data": map[string]any{
					"playlists": playlists,
				},
			}
			payload, _ := json.Marshal(resp)
			_ = client.SendMessage(string(payload))

		case "create_playlist":
			userID, _, _, _, _ := client.GetInfo()
			if userID == 0 {
				sendError(client, "Telegram login required to create playlists.")
				continue
			}
			name := strings.TrimSpace(msg.PlaylistName)
			if name == "" {
				sendError(client, "Playlist name cannot be empty.")
				continue
			}
			playlists, err := db.Instance.GetUserPlaylists(userID)
			if err == nil && len(playlists) >= 10 {
				sendError(client, "You have reached the limit of 10 playlists.")
				continue
			}
			plID, err := db.Instance.CreatePlaylist(name, userID)
			if err != nil {
				sendError(client, "Failed to create playlist: "+err.Error())
				continue
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
			userID, _, _, _, _ := client.GetInfo()
			if userID == 0 {
				sendError(client, "Telegram login required to delete playlists.")
				continue
			}
			if msg.PlaylistID == "" {
				sendError(client, "Playlist ID required.")
				continue
			}
			pl, err := db.Instance.GetPlaylist(msg.PlaylistID)
			if err != nil || pl == nil {
				sendError(client, "Playlist not found.")
				continue
			}
			if pl.UserID != userID {
				sendError(client, "You do not own this playlist.")
				continue
			}
			if err := db.Instance.DeletePlaylist(msg.PlaylistID, userID); err != nil {
				sendError(client, "Failed to delete playlist: "+err.Error())
				continue
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
			userID, _, _, _, _ := client.GetInfo()
			if userID == 0 {
				sendError(client, "Telegram login required to add to playlist.")
				continue
			}

			var song db.Song
			if msg.Track != nil && msg.Track.ID != "" {
				url := msg.Track.URL
				if url == "" {
					url = "https://www.youtube.com/watch?v=" + msg.Track.ID
				}
				platform := msg.Track.Platform
				if platform == "" {
					platform = utils.YouTube
				}
				song = db.Song{
					URL:      url,
					Name:     msg.Track.Title,
					TrackID:  msg.Track.ID,
					Duration: int32(msg.Track.Duration),
					Platform: platform,
				}
			} else {
				playing := cache.ChatCache.GetPlayingTrack(client.RoomID)
				if playing == nil {
					sendError(client, "No active track playing to add to playlist.")
					continue
				}
				song = db.Song{
					URL:      playing.URL,
					Name:     playing.Name,
					TrackID:  playing.TrackID,
					Duration: int32(playing.Duration),
					Platform: playing.Platform,
				}
			}

			playlists, err := db.Instance.GetUserPlaylists(userID)
			if err != nil {
				sendError(client, "Error fetching playlists.")
				continue
			}

			var targetID string
			if msg.PlaylistID != "" {
				targetID = msg.PlaylistID
			} else {
				if len(playlists) == 0 {
					targetID, err = db.Instance.CreatePlaylist("My Playlist", userID)
					if err != nil {
						sendError(client, "Failed to create default playlist.")
						continue
					}
				} else {
					targetID = playlists[0].ID
				}
			}

			if err := db.Instance.AddSongToPlaylist(targetID, song); err != nil {
				sendError(client, "Failed to add song to playlist.")
				continue
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
			userID, _, _, _, _ := client.GetInfo()
			if userID == 0 {
				sendError(client, "Telegram login required to modify playlists.")
				continue
			}
			if msg.PlaylistID == "" || msg.TrackID == "" {
				sendError(client, "Playlist ID and Track ID are required.")
				continue
			}
			pl, err := db.Instance.GetPlaylist(msg.PlaylistID)
			if err != nil || pl == nil {
				sendError(client, "Playlist not found.")
				continue
			}
			if pl.UserID != userID {
				sendError(client, "You do not own this playlist.")
				continue
			}
			if err := db.Instance.RemoveSongFromPlaylist(msg.PlaylistID, msg.TrackID); err != nil {
				sendError(client, "Failed to remove song: "+err.Error())
				continue
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
			_, _, canControl, canPlay, _ := client.GetInfo()
			if msg.Type == "play_playlist" {
				if !canControl {
					sendError(client, "Permission required to force play in this chat.")
					continue
				}
			} else {
				if !canPlay {
					sendError(client, "Play mode is restricted in this chat.")
					continue
				}
			}

			if msg.PlaylistID == "" {
				sendError(client, "Playlist ID required.")
				continue
			}

			pl, err := db.Instance.GetPlaylist(msg.PlaylistID)
			if err != nil || pl == nil || len(pl.Songs) == 0 {
				sendError(client, "Playlist not found or empty.")
				continue
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
					URL:      song.URL,
					Name:     song.Name,
					User:     getClientUserName(client),
					TrackID:  song.TrackID,
					Duration: song.Duration,
					Platform: song.Platform,
				}

				if idx == 0 && msg.Type == "play_playlist" {
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
		}
	}
}
