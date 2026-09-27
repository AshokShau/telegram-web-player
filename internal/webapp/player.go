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
	"ashokshau/tg-web/internal/utils"
	"strconv"
	"strings"
	"sync"
	"time"

	td "github.com/AshokShau/gotdbot"
)

type RoomPlayback struct {
	Status     string  `json:"status"`     // "playing", "paused", "stopped"
	Position   float64 `json:"position"`   // position in seconds
	ServerTime int64   `json:"serverTime"` // epoch ms
}

type TrackData struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Artist    string `json:"artist"`
	Duration  int32  `json:"duration"`
	AudioURL  string `json:"audioUrl,omitempty"`
	Thumbnail string `json:"thumbnail"`
	Platform  string `json:"platform"`
	User      string `json:"user"`
	URL       string `json:"url"`
}

type RoomStateData struct {
	RoomID    int64          `json:"roomId"`
	Track     *TrackData     `json:"track"`
	Playback  RoomPlayback   `json:"playback"`
	Queue     []*TrackData   `json:"queue"`
	Listeners []ListenerInfo `json:"listeners"`
}

type EventMessage struct {
	Event string `json:"event"`
	Data  any    `json:"data"`
}

type RoomState struct {
	mu         sync.Mutex
	RoomID     int64
	Status     string  // "playing", "paused", "stopped"
	Position   float64 // position at ServerTime
	ServerTime int64   // epoch ms
	BotClient  *td.Client
}

type WebAppPlayerManager struct {
	mu    sync.RWMutex
	rooms map[int64]*RoomState
}

var Manager = &WebAppPlayerManager{
	rooms: make(map[int64]*RoomState),
}

func (m *WebAppPlayerManager) getOrCreate(chatID int64) *RoomState {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, ok := m.rooms[chatID]
	if !ok {
		room = &RoomState{
			RoomID:     chatID,
			Status:     "stopped",
			Position:   0,
			ServerTime: time.Now().UnixMilli(),
		}
		m.rooms[chatID] = room
	}
	return room
}

func (r *RoomState) GetCurrentPosition() float64 {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.Status != "playing" {
		return r.Position
	}

	track := cache.ChatCache.GetPlayingTrack(r.RoomID)
	if track != nil {
		isReady := track.FilePath != "" || track.Platform == utils.DirectLink || strings.HasPrefix(track.FilePath, "http://") || strings.HasPrefix(track.FilePath, "https://")
		if !isReady {
			return 0
		}
	}

	elapsed := float64(time.Now().UnixMilli()-r.ServerTime) / 1000.0
	currPos := r.Position + elapsed

	if track != nil && track.Duration > 0 && currPos >= float64(track.Duration) {
		currPos = float64(track.Duration)
	}

	return currPos
}

func (m *WebAppPlayerManager) PlayTrack(bot *td.Client, chatID int64, track *utils.PlayerCache) {
	room := m.getOrCreate(chatID)
	room.mu.Lock()
	room.Status = "playing"
	room.Position = 0
	room.ServerTime = time.Now().UnixMilli()
	if bot != nil {
		room.BotClient = bot
	}
	room.mu.Unlock()

	HubInstance.BroadcastRoomState(chatID)
}

func (m *WebAppPlayerManager) Pause(chatID int64) (float64, error) {
	room := m.getOrCreate(chatID)
	room.mu.Lock()
	pos := room.Position
	if room.Status == "playing" {
		elapsed := float64(time.Now().UnixMilli()-room.ServerTime) / 1000.0
		pos += elapsed
	}
	room.Status = "paused"
	room.Position = pos
	room.ServerTime = time.Now().UnixMilli()
	room.mu.Unlock()

	HubInstance.BroadcastRoomState(chatID)
	return pos, nil
}

func (m *WebAppPlayerManager) Resume(chatID int64) (float64, error) {
	room := m.getOrCreate(chatID)
	room.mu.Lock()
	room.Status = "playing"
	room.ServerTime = time.Now().UnixMilli()
	pos := room.Position
	room.mu.Unlock()

	HubInstance.BroadcastRoomState(chatID)
	return pos, nil
}

func (m *WebAppPlayerManager) SeekRoom(chatID int64, seconds float64) (float64, error) {
	room := m.getOrCreate(chatID)
	room.mu.Lock()
	if seconds < 0 {
		seconds = 0
	}
	track := cache.ChatCache.GetPlayingTrack(chatID)
	if track != nil && track.Duration > 0 && seconds > float64(track.Duration) {
		seconds = float64(track.Duration)
	}
	room.Position = seconds
	room.ServerTime = time.Now().UnixMilli()
	pos := room.Position
	room.mu.Unlock()

	HubInstance.BroadcastRoomState(chatID)
	return pos, nil
}

func (m *WebAppPlayerManager) Stop(chatID int64) {
	cache.ChatCache.ClearChat(chatID)

	room := m.getOrCreate(chatID)
	room.mu.Lock()
	room.Status = "stopped"
	room.Position = 0
	room.ServerTime = time.Now().UnixMilli()
	room.mu.Unlock()

	HubInstance.BroadcastRoomState(chatID)
}

func (m *WebAppPlayerManager) PlayedTime(chatID int64) (float64, error) {
	room := m.getOrCreate(chatID)
	return room.GetCurrentPosition(), nil
}

func (m *WebAppPlayerManager) GetRoomStateData(chatID int64) RoomStateData {
	room := m.getOrCreate(chatID)
	room.mu.Lock()
	status := room.Status
	pos := room.Position
	sTime := room.ServerTime
	if status == "playing" {
		elapsed := float64(time.Now().UnixMilli()-sTime) / 1000.0
		pos += elapsed
	}
	room.mu.Unlock()

	playingTrack := cache.ChatCache.GetPlayingTrack(chatID)
	var trackData *TrackData
	if playingTrack != nil {
		isReady := playingTrack.FilePath != "" || playingTrack.Platform == utils.DirectLink || strings.HasPrefix(playingTrack.FilePath, "http://") || strings.HasPrefix(playingTrack.FilePath, "https://")
		if !isReady {
			status = "stopped"
			pos = 0
			room.mu.Lock()
			room.Status = "stopped"
			room.Position = 0
			room.mu.Unlock()
		}

		audioURL := "/stream?track_id=" + playingTrack.TrackID + "&chat_id=" + strconv.FormatInt(chatID, 10)
		if playingTrack.FilePath != "" && (strings.HasPrefix(playingTrack.FilePath, "http://") || strings.HasPrefix(playingTrack.FilePath, "https://")) {
			audioURL = playingTrack.FilePath
		} else if playingTrack.Platform == utils.DirectLink && (strings.HasPrefix(playingTrack.URL, "http://") || strings.HasPrefix(playingTrack.URL, "https://")) {
			audioURL = playingTrack.URL
		}

		trackData = &TrackData{
			ID:        playingTrack.TrackID,
			Title:     playingTrack.Name,
			Artist:    playingTrack.Channel,
			Duration:  playingTrack.Duration,
			AudioURL:  audioURL,
			Thumbnail: playingTrack.Thumbnail,
			Platform:  playingTrack.Platform,
			User:      playingTrack.User,
			URL:       playingTrack.URL,
		}
	} else {
		status = "stopped"
		pos = 0
		room.mu.Lock()
		room.Status = "stopped"
		room.Position = 0
		room.mu.Unlock()
	}

	queueTracks := cache.ChatCache.GetQueue(chatID)
	var queueData []*TrackData
	if len(queueTracks) > 1 {
		for _, q := range queueTracks[1:] {
			if q == nil {
				continue
			}
			queueData = append(queueData, &TrackData{
				ID:        q.TrackID,
				Title:     q.Name,
				Artist:    q.Channel,
				Duration:  q.Duration,
				Thumbnail: q.Thumbnail,
				Platform:  q.Platform,
				User:      q.User,
				URL:       q.URL,
			})
		}
	}

	listenersList := HubInstance.GetListeners(chatID)

	return RoomStateData{
		RoomID: chatID,
		Track:  trackData,
		Playback: RoomPlayback{
			Status:     status,
			Position:   pos,
			ServerTime: time.Now().UnixMilli(),
		},
		Queue:     queueData,
		Listeners: listenersList,
	}
}
