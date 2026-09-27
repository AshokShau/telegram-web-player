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
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	td "github.com/AshokShau/gotdbot"
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
	if track == nil {
		playing := cache.ChatCache.GetPlayingTrack(chatID)
		if playing != nil && playing.TrackID == trackID {
			track = playing
		}
	}

	if track == nil {
		http.Error(w, "track not found", http.StatusNotFound)
		return
	}

	filePath := track.FilePath
	if filePath == "" && (track.Platform != "") {
		for range 30 {
			time.Sleep(100 * time.Millisecond)
			t := cache.ChatCache.GetTrackIfExists(chatID, trackID)
			if t == nil {
				t = cache.ChatCache.GetPlayingTrack(chatID)
			}
			if t != nil && t.FilePath != "" {
				track = t
				filePath = t.FilePath
				break
			}
		}
	}

	if filePath == "" {
		http.Error(w, "track (file Path) not found", http.StatusNotFound)
		return
	}

	if strings.HasPrefix(filePath, "http://") || strings.HasPrefix(filePath, "https://") {
		http.Redirect(w, r, filePath, http.StatusFound)
		return
	}

	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "track file not found on server", http.StatusNotFound)
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

func RegisterRoutes(bot *td.Client) {
	staticSub, err := fs.Sub(staticFS, "static")
	if err == nil {
		http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticSub))))
	}

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		websocket.Handler(func(ws *websocket.Conn) {
			handleWebSocket(bot, ws)
		}).ServeHTTP(w, r)
	})
	http.HandleFunc("/", ServeHomeHTML)
	http.HandleFunc("/stream", streamHandler)
	http.HandleFunc("/room", ServeWebAppHTML)

	log.Info("[WebApp] Web App routes registered successfully")
	go http.ListenAndServe("0.0.0.0:"+config.Port, nil)
}
