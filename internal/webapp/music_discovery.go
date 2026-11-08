package webapp

import (
	"ashokshau/tg-web/internal/config"
	"ashokshau/tg-web/internal/downloader"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
)

var musicLocaleRegion = regexp.MustCompile(`(?i)-([a-z]{2})$`)

func musicAuthorized(r *http.Request) bool {
	data, ok := verifyTelegramInitData(r.Header.Get("X-Telegram-Init-Data"), config.Token)
	return ok && data != nil && data.User != nil && data.User.ID > 0
}

func musicDiscoveryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !musicAuthorized(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var request struct {
		Seeds []downloader.MusicSeed `json:"seeds"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16384)).Decode(&request); err != nil || len(request.Seeds) > 6 {
		http.Error(w, "invalid recent tracks", http.StatusBadRequest)
		return
	}
	for i := range request.Seeds {
		seed := &request.Seeds[i]
		seed.Title, seed.Artist = strings.TrimSpace(seed.Title), strings.TrimSpace(seed.Artist)
		if len([]rune(seed.Title)) > 120 || len([]rune(seed.Artist)) > 100 || len(seed.Thumbnail) > 1024 || len(seed.ID) > 64 {
			http.Error(w, "invalid recent tracks", http.StatusBadRequest)
			return
		}
	}
	region := ""
	language, _, _ := strings.Cut(r.Header.Get("Accept-Language"), ",")
	language, _, _ = strings.Cut(language, ";")
	if match := musicLocaleRegion.FindStringSubmatch(language); len(match) > 1 {
		region = strings.ToUpper(match[1])
	}
	shelves, err := downloader.MusicDiscovery(r.Context(), request.Seeds, region)
	if err != nil {
		http.Error(w, "YouTube Music is unavailable", http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "private, max-age=600")
	_ = json.NewEncoder(w).Encode(shelves)
}

func musicCollectionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !musicAuthorized(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	tracks, err := downloader.MusicCollectionTracks(r.Context(), downloader.MusicCollection{ID: r.URL.Query().Get("id")})
	if err != nil {
		http.Error(w, "YouTube Music collection is unavailable", http.StatusBadGateway)
		return
	}
	results := make([]*TrackData, 0, len(tracks))
	for _, track := range tracks {
		if !allowedTrackDuration(track.Duration) || track.Title == "" {
			continue
		}
		results = append(results, &TrackData{ID: track.Id, Title: track.Title, Artist: track.Channel, Duration: track.Duration, Thumbnail: track.Thumbnail, Platform: track.Platform, URL: track.Url})
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "private, max-age=600")
	_ = json.NewEncoder(w).Encode(map[string]any{"results": results})
}
