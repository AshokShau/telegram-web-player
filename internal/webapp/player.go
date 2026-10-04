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
	"ashokshau/tg-web/internal/db"
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
	IsVideo   bool   `json:"isVideo"`
}

type RoomStateData struct {
	RoomID       int64          `json:"roomId"`
	Track        *TrackData     `json:"track"`
	Playback     RoomPlayback   `json:"playback"`
	Queue        []*TrackData   `json:"queue"`
	Listeners    []ListenerInfo `json:"listeners"`
	Loop         int            `json:"loop"`
	Autoplay     bool           `json:"autoplay"`
	ChatEnabled  bool           `json:"chatEnabled"`
	ChatCooldown int            `json:"chatCooldown"`
	VC           VCRoomState    `json:"vc"`
}

type EventMessage struct {
	Event string `json:"event"`
	Data  any    `json:"data"`
}

type RoomState struct {
	mu              sync.Mutex
	RoomID          int64
	Status          string  // "playing", "paused", "stopped"
	Position        float64 // position at ServerTime
	ServerTime      int64   // epoch ms
	BotClient       *td.Client
	graceTimer      *time.Timer
	trackEndTimer   *time.Timer
	currentTrackID  string
	isTransitioning bool
}

type WebAppPlayerManager struct {
	mu    sync.RWMutex
	rooms map[int64]*RoomState
}

var Manager = &WebAppPlayerManager{
	rooms: make(map[int64]*RoomState),
}

func (m *WebAppPlayerManager) getOrCreate(bot *td.Client, chatID int64) *RoomState {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, ok := m.rooms[chatID]
	if !ok {
		room = &RoomState{
			RoomID:     chatID,
			Status:     "stopped",
			Position:   0,
			ServerTime: time.Now().UnixMilli(),
			BotClient:  bot,
		}
		m.rooms[chatID] = room
	} else if room.BotClient == nil && bot != nil {
		room.BotClient = bot
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
	room := m.getOrCreate(bot, chatID)
	room.mu.Lock()
	room.cancelGraceTimerLocked()
	room.Status = "playing"
	room.Position = 0
	room.ServerTime = time.Now().UnixMilli()
	room.isTransitioning = false

	if track != nil {
		room.currentTrackID = track.TrackID
		room.scheduleTrackEndTimerLocked(track.Duration, 0)
	}

	room.mu.Unlock()
	HubInstance.BroadcastRoomState(bot, chatID)
}

func (r *RoomState) cancelGraceTimerLocked() {
	if r.graceTimer != nil {
		r.graceTimer.Stop()
		r.graceTimer = nil
	}
}

func (r *RoomState) cancelTrackEndTimerLocked() {
	if r.trackEndTimer != nil {
		r.trackEndTimer.Stop()
		r.trackEndTimer = nil
	}
}

func (r *RoomState) scheduleTrackEndTimerLocked(durationSec int32, startPosSec float64) {
	r.cancelTrackEndTimerLocked()
	if durationSec <= 0 {
		return
	}

	remSec := float64(durationSec) - startPosSec
	if remSec <= 0 {
		remSec = 0.1
	}

	chatID := r.RoomID
	trackID := r.currentTrackID
	d := time.Duration(remSec * float64(time.Second))

	r.trackEndTimer = time.AfterFunc(d, func() {
		r.mu.Lock()
		if r.Status != "playing" || r.currentTrackID != trackID {
			r.mu.Unlock()
			return
		}
		bot := r.BotClient
		r.mu.Unlock()

		_ = PlayNextForTrack(bot, chatID, trackID)
	})
}

func (m *WebAppPlayerManager) CheckListenersCount(bot *td.Client, chatID int64) {
	room := m.getOrCreate(bot, chatID)
	room.mu.Lock()
	defer room.mu.Unlock()

	if room.Status == "stopped" {
		return
	}

	listenerCount := len(HubInstance.GetListeners(chatID))
	if listenerCount == 0 {
		if room.graceTimer == nil {
			log.Debug("[WebApp] VC empty, starting 40s grace timer", "chatID", chatID)
			room.graceTimer = time.AfterFunc(40*time.Second, func() {
				room.mu.Lock()
				if room.graceTimer == nil {
					room.mu.Unlock()
					return
				}
				room.graceTimer = nil
				b := room.BotClient
				status := room.Status
				room.mu.Unlock()

				log.Debug("[WebApp] 40s grace timer expired, stopping playback session", "chatID", chatID)
				if status != "stopped" {
					_ = handleNoSong(b, chatID)
				}
			})
		}
	} else {
		if room.graceTimer != nil {
			log.Info("[WebApp] Listener rejoined, canceling grace timer", "chatID", chatID)
			room.cancelGraceTimerLocked()
		}
	}
}

func (m *WebAppPlayerManager) Pause(c *td.Client, chatID int64) (float64, error) {
	room := m.getOrCreate(c, chatID)
	room.mu.Lock()
	pos := room.Position
	if room.Status == "playing" {
		elapsed := float64(time.Now().UnixMilli()-room.ServerTime) / 1000.0
		pos += elapsed
	}
	room.Status = "paused"
	room.Position = pos
	room.ServerTime = time.Now().UnixMilli()
	room.cancelTrackEndTimerLocked()
	room.mu.Unlock()

	HubInstance.BroadcastRoomState(c, chatID)
	return pos, nil
}

func (m *WebAppPlayerManager) Resume(c *td.Client, chatID int64) (float64, error) {
	room := m.getOrCreate(c, chatID)
	room.mu.Lock()
	room.Status = "playing"
	room.ServerTime = time.Now().UnixMilli()
	pos := room.Position
	track := cache.ChatCache.GetPlayingTrack(chatID)
	if track != nil {
		room.currentTrackID = track.TrackID
		room.scheduleTrackEndTimerLocked(track.Duration, pos)
	}
	room.mu.Unlock()

	HubInstance.BroadcastRoomState(c, chatID)
	return pos, nil
}

func (m *WebAppPlayerManager) SeekRoom(c *td.Client, chatID int64, seconds float64) (float64, error) {
	room := m.getOrCreate(c, chatID)
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
	if room.Status == "playing" && track != nil {
		room.currentTrackID = track.TrackID
		room.scheduleTrackEndTimerLocked(track.Duration, pos)
	}
	room.mu.Unlock()

	HubInstance.BroadcastRoomState(c, chatID)
	return pos, nil
}

func (m *WebAppPlayerManager) Stop(c *td.Client, chatID int64) {
	cache.ChatCache.ClearChat(chatID)

	room := m.getOrCreate(c, chatID)
	room.mu.Lock()
	room.cancelGraceTimerLocked()
	room.cancelTrackEndTimerLocked()
	room.Status = "stopped"
	room.Position = 0
	room.ServerTime = time.Now().UnixMilli()
	room.currentTrackID = ""
	room.isTransitioning = false
	room.mu.Unlock()

	HubInstance.BroadcastRoomState(c, chatID)
}

func (m *WebAppPlayerManager) PlayedTime(c *td.Client, chatID int64) (float64, error) {
	room := m.getOrCreate(c, chatID)
	return room.GetCurrentPosition(), nil
}

func (m *WebAppPlayerManager) GetRoomStateData(c *td.Client, chatID int64) RoomStateData {
	room := m.getOrCreate(c, chatID)
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
			pos = 0
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
			IsVideo:   playingTrack.IsVideo,
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
				IsVideo:   q.IsVideo,
			})
		}
	}

	listenersList := HubInstance.GetListeners(chatID)
	loopCount := cache.ChatCache.GetLoopCount(chatID)
	autoplayState := cache.ChatCache.GetAutoplay(chatID)
	chatEnabled := db.Instance.GetChatEnabled(chatID)
	chatCooldown := db.Instance.GetChatCooldown(chatID)

	vcRoom := VCManagerInstance.GetOrCreateRoom(chatID)
	vcState := vcRoom.GetState()

	return RoomStateData{
		RoomID: chatID,
		Track:  trackData,
		Playback: RoomPlayback{
			Status:     status,
			Position:   pos,
			ServerTime: time.Now().UnixMilli(),
		},
		Queue:        queueData,
		Listeners:    listenersList,
		Loop:         loopCount,
		Autoplay:     autoplayState,
		ChatEnabled:  chatEnabled,
		ChatCooldown: chatCooldown,
		VC:           vcState,
	}
}
