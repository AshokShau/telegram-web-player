package downloader

import (
	"ashokshau/tg-web/internal/config"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var ErrLyricsUnsupported = errors.New("lyrics source is not supported")
var ErrLyricsUnavailable = errors.New("lyrics are unavailable")

// Add future OneGrab lyrics providers here, independently of audio download support.
var lyricsProviders = map[string]func(*url.URL) (string, error){
	"open.spotify.com":  spotifyLyricsURL,
	"youtube.com":       youtubeLyricsURL,
	"www.youtube.com":   youtubeLyricsURL,
	"m.youtube.com":     youtubeLyricsURL,
	"music.youtube.com": youtubeLyricsURL,
	"youtu.be":          youtubeLyricsURL,
}

var spotifyLyricsPath = regexp.MustCompile(`^/track/[A-Za-z0-9]{22}/?$`)
var youtubeLyricsID = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)

type LyricsLine struct {
	StartTimeMs json.RawMessage `json:"startTimeMs"`
	Words       string          `json:"words"`
}

type Lyrics struct {
	Text   string       `json:"text"`
	LRC    string       `json:"lrc"`
	Synced bool         `json:"synced"`
	Lines  []LyricsLine `json:"lines"`
}

func LyricsURL(raw string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.User != nil || u.Port() != "" || (u.Scheme != "https" && u.Scheme != "http") {
		return "", ErrLyricsUnsupported
	}
	host := strings.ToLower(u.Hostname())
	provider, supported := lyricsProviders[host]
	if !supported {
		return "", ErrLyricsUnsupported
	}
	return provider(u)
}

func spotifyLyricsURL(u *url.URL) (string, error) {
	if !spotifyLyricsPath.MatchString(u.Path) {
		return "", ErrLyricsUnsupported
	}
	return "https://open.spotify.com" + strings.TrimRight(u.Path, "/"), nil
}

func youtubeLyricsURL(u *url.URL) (string, error) {
	path := strings.TrimSuffix(u.Path, "/")
	var id string
	if strings.EqualFold(u.Hostname(), "youtu.be") {
		id = strings.TrimPrefix(path, "/")
	} else if path == "/watch" {
		query, err := url.ParseQuery(u.RawQuery)
		if err != nil || len(query["v"]) != 1 {
			return "", ErrLyricsUnsupported
		}
		id = query.Get("v")
	} else {
		parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
		if len(parts) == 2 && (parts[0] == "shorts" || parts[0] == "embed" || parts[0] == "live") {
			id = parts[1]
		}
	}
	if !youtubeLyricsID.MatchString(id) {
		return "", ErrLyricsUnsupported
	}
	return "https://www.youtube.com/watch?" + url.Values{"v": {id}}.Encode(), nil
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
	if resp.StatusCode != http.StatusOK {
		return nil, ErrLyricsUnavailable
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, (1<<20)+1))
	if err != nil || len(body) > 1<<20 {
		return nil, ErrLyricsUnavailable
	}
	var upstream struct {
		Lyrics
		Segments []struct {
			StartMs json.RawMessage `json:"startMs"`
			Text    string          `json:"text"`
		} `json:"segments"`
	}
	if err := json.Unmarshal(body, &upstream); err != nil {
		return nil, ErrLyricsUnavailable
	}
	// YouTube captions use startMs/text; expose the same lines as Spotify.
	if upstream.Synced && len(upstream.Lines) == 0 {
		for _, segment := range upstream.Segments {
			upstream.Lines = append(upstream.Lines, LyricsLine{StartTimeMs: segment.StartMs, Words: segment.Text})
		}
	}
	return &upstream.Lyrics, nil
}
