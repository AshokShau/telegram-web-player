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
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/net/websocket"
)

func streamHandler(w http.ResponseWriter, r *http.Request) {
	trackID := r.URL.Query().Get("track_id")

	chatID, err := strconv.ParseInt(r.URL.Query().Get("chat_id"), 10, 64)
	if err != nil || chatID == 0 || trackID == "" {
		http.Error(w, "invalid track", http.StatusBadRequest)
		return
	}

	track := cache.ChatCache.GetTrackIfExists(chatID, trackID)
	if track == nil || track.FilePath == "" {
		http.Error(w, "track (file Path) not found", http.StatusNotFound)
		return
	}

	filePath := track.FilePath
	if strings.HasPrefix(filePath, "http://") || strings.HasPrefix(filePath, "https://") {
		http.Redirect(w, r, filePath, http.StatusFound)
		return
	}

	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "track not found", http.StatusNotFound)
		return
	}
	
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		http.Error(w, "track not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Accept-Ranges", "bytes")
	contentType := mime.TypeByExtension(filepath.Ext(filePath))
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}

	http.ServeContent(w, r, info.Name(), info.ModTime(), file)
}
func RegisterRoutes() {
	http.Handle("/ws", websocket.Handler(handleWebSocket))
	http.HandleFunc("/", serveHomeHTML)
	http.HandleFunc("/stream", streamHandler)
	http.HandleFunc("/room", serveWebAppHTML)

	log.Info("[WebApp] Web App routes registered successfully")
	go http.ListenAndServe("0.0.0.0:"+config.Port, nil)
}
