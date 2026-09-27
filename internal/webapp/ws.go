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
	mu         sync.RWMutex
	writeMu    sync.Mutex
	Conn       *websocket.Conn
	RoomID     int64
	UserID     int64
	IsAdmin    bool
	CanControl bool
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

func (c *Client) GetInfo() (userID int64, isAdmin bool, canControl bool, initData *WebAppInitData) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.UserID, c.IsAdmin, c.CanControl, c.InitData
}

func (c *Client) SetPermissions(isAdmin, canControl bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.IsAdmin = isAdmin
	c.CanControl = canControl
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

func (h *Hub) Register(c *Client) {
	h.mu.Lock()
	roomClients := h.clients[c.RoomID]
	if slices.Contains(roomClients, c) {
		h.mu.Unlock()
		return
	}
	h.clients[c.RoomID] = append(h.clients[c.RoomID], c)
	h.mu.Unlock()

	log.Info("[WebApp] Client joined room", "roomId", c.RoomID, "userID", c.UserID, "isAdmin", c.IsAdmin)
	Manager.CheckListenersCount(c.RoomID)
}

func (h *Hub) Unregister(c *Client) {
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

	Manager.CheckListenersCount(c.RoomID)
}

func (h *Hub) GetListeners(roomID int64) []ListenerInfo {
	h.mu.RLock()
	clients := make([]*Client, len(h.clients[roomID]))
	copy(clients, h.clients[roomID])
	h.mu.RUnlock()

	seen := make(map[int64]bool)
	listeners := make([]ListenerInfo, 0, len(clients))

	for _, client := range clients {
		userID, isAdmin, _, initData := client.GetInfo()
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
		if client == nil {
			continue
		}
		go func(c *Client) {
			_ = c.SendMessage(string(payload))
		}(client)
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

func sendError(c *Client, errMsg string) {
	errPayload := map[string]any{
		"event": "error",
		"data":  errMsg,
	}

	payload, _ := json.Marshal(errPayload)
	_ = c.SendMessage(string(payload))
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
					if data.User != nil {
						client.SetUser(data.User.ID, data)
					} else {
						client.SetUser(0, data)
					}
				}
			}

			userID, _, _, _ := client.GetInfo()
			roomState := Manager.getOrCreate(client.RoomID)
			isAdmin := isUserChatAdmin(roomState.BotClient, client.RoomID, userID)
			canControl := canUserControl(roomState.BotClient, client.RoomID, userID)
			client.SetPermissions(isAdmin, canControl)

			userInfoMsg := map[string]any{
				"event": "user_info",
				"data": map[string]any{
					"userId":     userID,
					"isAdmin":    isAdmin,
					"canControl": canControl,
				},
			}
			payload, _ := json.Marshal(userInfoMsg)
			_ = client.SendMessage(string(payload))

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
			_ = client.SendMessage(string(payload))

		case "seek":
			userID, _, _, _ := client.GetInfo()
			roomState := Manager.getOrCreate(client.RoomID)
			if !canUserControl(roomState.BotClient, client.RoomID, userID) {
				sendError(client, "Permission required to seek")
				continue
			}
			_, _ = Manager.SeekRoom(client.RoomID, msg.PositionSeconds)

		case "pause":
			userID, _, _, _ := client.GetInfo()
			roomState := Manager.getOrCreate(client.RoomID)
			if !canUserControl(roomState.BotClient, client.RoomID, userID) {
				sendError(client, "Permission required to pause")
				continue
			}
			_, _ = Manager.Pause(client.RoomID)

		case "resume":
			userID, _, _, _ := client.GetInfo()
			roomState := Manager.getOrCreate(client.RoomID)
			if !canUserControl(roomState.BotClient, client.RoomID, userID) {
				sendError(client, "Permission required to resume")
				continue
			}
			_, _ = Manager.Resume(client.RoomID)

		case "skip", "track_end":
			userID, _, _, _ := client.GetInfo()
			roomState := Manager.getOrCreate(client.RoomID)
			if !canUserControl(roomState.BotClient, client.RoomID, userID) {
				sendError(client, "Permission required to change track")
				continue
			}
			if OnPlayNextHandler != nil {
				_ = OnPlayNextHandler(roomState.BotClient, client.RoomID)
			}

		case "stop":
			userID, _, _, _ := client.GetInfo()
			roomState := Manager.getOrCreate(client.RoomID)
			if !canUserControl(roomState.BotClient, client.RoomID, userID) {
				sendError(client, "Permission required to stop")
				continue
			}

			Manager.Stop(client.RoomID)
		}
	}
}
