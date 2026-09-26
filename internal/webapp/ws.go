/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/FallenProjects/telegram-web-player
 */

package webapp

import (
	"ashokshau/tg-web/internal/config"
	"encoding/json"
	"slices"
	"strconv"
	"sync"
	"time"

	td "github.com/AshokShau/gotdbot"
	"golang.org/x/net/websocket"
)

type Client struct {
	Conn       *websocket.Conn
	RoomID     int64
	UserID     int64
	IsAdmin    bool
	CanControl bool
	InitData   *WebAppInitData
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

func (h *Hub) Register(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	roomClients := h.clients[c.RoomID]
	if slices.Contains(roomClients, c) {
		return
	}
	h.clients[c.RoomID] = append(h.clients[c.RoomID], c)

	log.Info("[WebApp] Client joined room", "roomId", c.RoomID, "userID", c.UserID, "isAdmin", c.IsAdmin)
}

func (h *Hub) Unregister(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	roomClients, exists := h.clients[c.RoomID]
	if !exists {
		return
	}

	for i, client := range roomClients {
		if client == c {
			h.clients[c.RoomID] = append(roomClients[:i], roomClients[i+1:]...)
			log.Info("[WebApp] Client left room", "roomId", c.RoomID, "userID", c.UserID)
			break
		}
	}

	if len(h.clients[c.RoomID]) == 0 {
		delete(h.clients, c.RoomID)
	}
}

func (h *Hub) GetListeners(roomID int64) []ListenerInfo {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clients := h.clients[roomID]
	seen := make(map[int64]bool)
	listeners := make([]ListenerInfo, 0, len(clients))

	for _, client := range clients {
		if client.UserID != 0 {
			if seen[client.UserID] {
				continue
			}
			seen[client.UserID] = true

			info := ListenerInfo{
				UserID:  client.UserID,
				IsAdmin: client.IsAdmin,
			}
			if client.InitData != nil && client.InitData.User != nil {
				u := client.InitData.User
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

func (h *Hub) BroadcastRoomState(roomID int64) {
	state := Manager.GetRoomStateData(roomID)
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
		_ = websocket.Message.Send(client.Conn, string(payload))
	}
}

type ClientMessage struct {
	Type            string  `json:"type"`
	RoomID          string  `json:"roomId"`
	UserID          int64   `json:"userId"`
	InitData        string  `json:"initData"`
	PositionSeconds float64 `json:"positionSeconds"`
	ClientTime      int64   `json:"clientTime"`
}

type CallbackFunc func(bot *td.Client, chatID int64) error

var OnPlayNextHandler CallbackFunc

func sendError(ws *websocket.Conn, errMsg string) {
	errPayload := map[string]any{
		"event": "error",
		"data":  errMsg,
	}

	payload, _ := json.Marshal(errPayload)
	_ = websocket.Message.Send(ws, string(payload))
}

func handleWebSocket(ws *websocket.Conn) {
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

	HubInstance.Register(client)
	defer func() {
		HubInstance.Unregister(client)
		HubInstance.BroadcastRoomState(client.RoomID)
	}()

	HubInstance.BroadcastRoomState(roomID)

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
			if msg.RoomID != "" {
				if rID, _ := strconv.ParseInt(msg.RoomID, 10, 64); rID != 0 && rID != client.RoomID {
					HubInstance.Unregister(client)
					client.RoomID = rID
					HubInstance.Register(client)
				}
			}

			if msg.InitData != "" {
				if data, ok := verifyTelegramInitData(msg.InitData, config.Token); ok && data != nil {
					client.InitData = data
					if data.User != nil {
						client.UserID = data.User.ID
					}
				}
			}
			if client.UserID == 0 && msg.UserID != 0 {
				client.UserID = msg.UserID
			}

			roomState := Manager.getOrCreate(client.RoomID)
			client.IsAdmin = isUserChatAdmin(roomState.BotClient, client.RoomID, client.UserID)
			client.CanControl = canUserControl(roomState.BotClient, client.RoomID, client.UserID)

			userInfoMsg := map[string]any{
				"event": "user_info",
				"data": map[string]any{
					"userId":     client.UserID,
					"isAdmin":    client.IsAdmin,
					"canControl": client.CanControl,
				},
			}
			payload, _ := json.Marshal(userInfoMsg)
			_ = websocket.Message.Send(ws, string(payload))

			HubInstance.BroadcastRoomState(client.RoomID)

		case "ping":
			pong := map[string]any{
				"event": "pong",
				"data": map[string]any{
					"clientTime": msg.ClientTime,
					"serverTime": time.Now().UnixMilli(),
				},
			}
			payload, _ := json.Marshal(pong)
			_ = websocket.Message.Send(ws, string(payload))

		case "seek":
			if !client.CanControl {
				sendError(ws, "Permission required to seek")
				continue
			}
			_, _ = Manager.SeekRoom(client.RoomID, msg.PositionSeconds)

		case "pause":
			if !client.CanControl {
				sendError(ws, "Permission required to pause")
				continue
			}
			_, _ = Manager.Pause(client.RoomID)

		case "resume":
			if !client.CanControl {
				sendError(ws, "Permission required to resume")
				continue
			}
			_, _ = Manager.Resume(client.RoomID)

		case "skip", "track_end":
			if msg.Type == "skip" && !client.CanControl {
				sendError(ws, "Permission required to skip")
				continue
			}
			if OnPlayNextHandler != nil {
				roomState := Manager.getOrCreate(client.RoomID)
				_ = OnPlayNextHandler(roomState.BotClient, client.RoomID)
			}

		case "stop":
			if !client.CanControl {
				sendError(ws, "Permission required to stop")
				continue
			}

			Manager.Stop(client.RoomID)
		}
	}
}
