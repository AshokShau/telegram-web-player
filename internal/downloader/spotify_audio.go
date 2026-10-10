package downloader

import (
	"ashokshau/tg-web/internal/config"
	"crypto/sha256"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sync/singleflight"
)

var spotifyAudioDownloads singleflight.Group

func DlSpotifyAudio(trackID, cdnURL string) (string, error) {
	u, err := url.Parse(cdnURL)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return "", fmt.Errorf("invalid Spotify audio URL")
	}

	ext := strings.ToLower(filepath.Ext(u.Path))
	if format := u.Query().Get("format"); format != "" {
		ext = "." + strings.ToLower(format)
	}

	switch ext {
	case ".mp3", ".m4a", ".ogg", ".opus", ".flac", ".wav":
	default:
		ext = ".audio"
	}

	identity := trackID
	if identity == "" {
		identity = cdnURL
	}
	name := fmt.Sprintf("spotify-%x%s", sha256.Sum256([]byte(identity)), ext)
	path := filepath.Join(config.DownloadsDir, name)

	value, err, _ := spotifyAudioDownloads.Do(path, func() (any, error) {
		if info, err := os.Stat(path); err == nil && info.Size() > 0 {
			return path, nil
		}
		return downloadFile(cdnURL, path, true)
	})

	if err != nil {
		return "", err
	}
	return value.(string), nil
}
