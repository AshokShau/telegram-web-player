package webapp

import (
	"ashokshau/tg-web/internal/config"
	"ashokshau/tg-web/internal/downloader"
	"encoding/json"
	"errors"
	"net/http"
)

func lyricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	data, valid := verifyTelegramInitData(r.Header.Get("X-Telegram-Init-Data"), config.Token)
	if !valid || data == nil || data.User == nil || data.User.ID <= 0 {
		http.Error(w, "valid Telegram authentication required", http.StatusUnauthorized)
		return
	}
	trackURL := r.URL.Query().Get("url")
	if trackURL == "" || len(trackURL) > 2048 {
		http.Error(w, "track URL required", http.StatusBadRequest)
		return
	}
	lyrics, err := downloader.FetchLyrics(r.Context(), trackURL)
	if errors.Is(err, downloader.ErrLyricsUnsupported) {
		http.Error(w, "lyrics currently support Spotify and YouTube tracks", http.StatusUnprocessableEntity)
		return
	}
	if err != nil {
		http.Error(w, "lyrics could not load; try again", http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(lyrics)
}
