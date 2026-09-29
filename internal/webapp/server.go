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
	"ashokshau/tg-web/internal/downloader"
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

type TelegramAuthRequest struct {
	IDToken string `json:"id_token"`
}

type TelegramAuthResponse struct {
	Success      bool        `json:"success"`
	SessionToken string      `json:"session_token,omitempty"`
	User         *WebAppUser `json:"user,omitempty"`
	Error        string      `json:"error,omitempty"`
}

func telegramAuthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(TelegramAuthResponse{Success: false, Error: "method not allowed"})
		return
	}

	var req TelegramAuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.IDToken) == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(TelegramAuthResponse{Success: false, Error: "id_token is required"})
		return
	}

	user, err := verifyTelegramIDToken(req.IDToken)
	if err != nil || user == nil || user.ID <= 0 {
		log.Warnf("[WebApp] Telegram ID token verification failed: %v", err)
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(TelegramAuthResponse{Success: false, Error: "invalid Telegram ID token"})
		return
	}

	sessionToken := CreateWebSession(user)

	_ = json.NewEncoder(w).Encode(TelegramAuthResponse{
		Success:      true,
		SessionToken: sessionToken,
		User:         user,
	})
}

func authMeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	authHeader := r.Header.Get("Authorization")
	sessionToken := strings.TrimPrefix(authHeader, "Bearer ")
	if sessionToken == "" {
		sessionToken = r.URL.Query().Get("session_token")
	}

	user := GetWebSession(sessionToken)
	if user == nil {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{"authenticated": false})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"authenticated": true,
		"user":          user,
	})
}

func ServeWebAppHTML(w http.ResponseWriter, r *http.Request) {
	content, err := staticFS.ReadFile("static/room.html")
	if err != nil {
		http.Error(w, "web app page not found", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(content)
}

func streamHandler(w http.ResponseWriter, r *http.Request) {
	initDataRaw := r.URL.Query().Get("init_data")
	if initDataRaw == "" {
		initDataRaw = r.Header.Get("X-Telegram-Init-Data")
	}

	initData, valid := verifyTelegramInitData(initDataRaw, config.Token)
	if !valid || initData == nil || initData.User == nil || initData.User.ID <= 0 {
		http.Error(w, "unauthorized: valid Telegram authentication required", http.StatusUnauthorized)
		return
	}

	trackID := strings.TrimSpace(r.URL.Query().Get("track_id"))
	chatID, err := strconv.ParseInt(r.URL.Query().Get("chat_id"), 10, 64)
	if err != nil || chatID == 0 || trackID == "" {
		http.Error(w, "invalid track or chat ID", http.StatusBadRequest)
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
	http.HandleFunc("/api/search", searchHandler)
	http.HandleFunc("/api/auth/telegram", telegramAuthHandler)
	http.HandleFunc("/api/auth/me", authMeHandler)

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
