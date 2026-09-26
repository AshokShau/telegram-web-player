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
	"net/http"
	"os"
	"strconv"

	"golang.org/x/net/websocket"
)

func streamHandler(w http.ResponseWriter, r *http.Request) {
	trackID := r.URL.Query().Get("track_id")
	chatIDStr := r.URL.Query().Get("chat_id")
	chatID, _ := strconv.ParseInt(chatIDStr, 10, 64)

	var filePath string
	if chatID != 0 && trackID != "" {
		track := cache.ChatCache.GetTrackIfExists(chatID, trackID)
		if track != nil && track.FilePath != "" {
			filePath = track.FilePath
		}
	}

	if filePath == "" {
		http.Error(w, "Track media file not found", http.StatusNotFound)
		return
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "Track media file not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Accept-Ranges", "bytes")
	http.ServeFile(w, r, filePath)
}

func RegisterRoutes() {
	http.Handle("/ws", websocket.Handler(handleWebSocket))
	http.HandleFunc("/", serveHomeHTML)
	http.HandleFunc("/stream", streamHandler)
	http.HandleFunc("/room", serveWebAppHTML)

	log.Info("[WebApp] Web App routes registered successfully")
	go http.ListenAndServe("0.0.0.0:"+config.Port, nil)
}
