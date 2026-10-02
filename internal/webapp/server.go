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
	"ashokshau/tg-web/internal/downloader"
	"ashokshau/tg-web/internal/utils"
	"embed"
	"encoding/json"
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

//go:embed static/*
var staticFS embed.FS

func ServeHomeHTML(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	content, err := staticFS.ReadFile("static/home.html")
	if err != nil {
		http.Error(w, "home page not found", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cross-Origin-Opener-Policy", "same-origin-allow-popups")
	_, _ = w.Write(content)
}

func ServeWebAppHTML(w http.ResponseWriter, r *http.Request) {
	content, err := staticFS.ReadFile("static/room.html")
	if err != nil {
		http.Error(w, "web app page not found", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Permissions-Policy", "microphone=*")
	_, _ = w.Write(content)
}

func streamHandler(w http.ResponseWriter, r *http.Request) {
	trackID := strings.TrimSpace(r.URL.Query().Get("track_id"))
	chatID, err := strconv.ParseInt(r.URL.Query().Get("chat_id"), 10, 64)
	if err != nil || chatID == 0 || trackID == "" {
		http.Error(w, "invalid track or chat ID", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	timeout := time.After(6 * time.Second)

	var track *utils.PlayerCache
	for {
		track = cache.ChatCache.GetTrackIfExists(chatID, trackID)
		if track == nil {
			playing := cache.ChatCache.GetPlayingTrack(chatID)
			if playing != nil && playing.TrackID == trackID {
				track = playing
			}
		}

		if track != nil && track.FilePath != "" {
			break
		}

		select {
		case <-ctx.Done():
			return
		case <-timeout:
			if track == nil {
				http.Error(w, "track not found", http.StatusNotFound)
			} else {
				http.Error(w, "track file download timeout", http.StatusGatewayTimeout)
			}
			return
		case <-ticker.C:
		}
	}

	filePath := track.FilePath
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
		http.Error(w, "track info error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set(
		"Cache-Control",
		"public, max-age=31536000, immutable",
	)
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
	http.HandleFunc("/api/search", searchHandler)

	log.Info("[WebApp] Web App routes registered successfully")
	go http.ListenAndServe("0.0.0.0:"+config.Port, nil)
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
	initDataRaw := r.URL.Query().Get("init_data")
	if initDataRaw == "" {
		initDataRaw = r.Header.Get("X-Telegram-Init-Data")
	}

	initData, valid := verifyTelegramInitData(initDataRaw, config.Token)
	if !valid || initData == nil || initData.User == nil || initData.User.ID <= 0 {
		http.Error(w, "unauthorized: valid Telegram authentication required", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		http.Error(w, "query parameter 'q' required", http.StatusBadRequest)
		return
	}
	if len(query) > 200 {
		query = query[:200]
	}

	wrapper := downloader.NewDlWrapper(query)
	results, err := wrapper.Search()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error":   err.Error(),
			"results": []any{},
		})
		return
	}

	var tracks []*TrackData
	if results != nil && len(results.Results) > 0 {
		for _, t := range results.Results {
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

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"results": tracks,
	})
}
