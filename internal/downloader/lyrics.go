package downloader

import (
	"ashokshau/tg-web/internal/config"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var ErrLyricsUnsupported = errors.New("lyrics source is not supported")
var ErrLyricsUnavailable = errors.New("lyrics are unavailable")

type Lyrics struct {
	Text string `json:"text"`
	LRC  string `json:"lrc"`
}

func LyricsURL(raw string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Hostname() == "" || u.User != nil || u.Port() != "" || (u.Scheme != "https" && u.Scheme != "http") {
		return "", ErrLyricsUnsupported
	}
	u.Fragment = ""
	return u.String(), nil
}

// FetchLyrics keeps the configured API key on the server and respects request cancellation.
func FetchLyrics(ctx context.Context, trackURL string) (*Lyrics, error) {
	canonical, err := LyricsURL(trackURL)
	if err != nil {
		return nil, err
	}
	if config.ApiUrl == "" || config.ApiKey == "" {
		return nil, ErrLyricsUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	endpoint := strings.TrimRight(config.ApiUrl, "/") + "/api/lyrics?" + url.Values{
		"url": {canonical},
	}.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, ErrLyricsUnavailable
	}
	req.Header.Set("X-API-Key", config.ApiKey)
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, ErrLyricsUnavailable
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusNoContent {
		return &Lyrics{}, nil
	}
	if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusUnprocessableEntity {
		return nil, ErrLyricsUnsupported
	}
	if resp.StatusCode != http.StatusOK {
		return nil, ErrLyricsUnavailable
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, (1<<20)+1))
	if err != nil || len(body) > 1<<20 {
		return nil, ErrLyricsUnavailable
	}
	var lyrics Lyrics
	if err := json.Unmarshal(body, &lyrics); err != nil {
		return nil, ErrLyricsUnavailable
	}
	return &lyrics, nil
}
