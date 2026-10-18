package webapp

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	td "github.com/AshokShau/gotdbot"
)

func publicBotProfile(user *td.User) map[string]any {
	name := strings.TrimSpace(user.FirstName + " " + user.LastName)
	if name == "" {
		name = "SyncTune"
	}
	profile := map[string]any{"name": name}
	if user.Usernames != nil {
		username := user.Usernames.EditableUsername
		if username == "" && len(user.Usernames.ActiveUsernames) > 0 {
			username = user.Usernames.ActiveUsernames[0]
		}
		if username != "" {
			profile["username"] = username
		}
	}
	if kind, ok := user.Type.(*td.UserTypeBot); ok && kind.ActiveUserCount > 0 {
		profile["monthlyActiveUsers"] = kind.ActiveUserCount
	}
	return profile
}

type botPhotoSource interface {
	DownloadFile(fileId int32, priority int32, opts *td.DownloadFileOpts) (*td.File, error)
}

type botPhotoCache struct {
	mu          sync.Mutex
	photoID     int64
	photo       []byte
	contentType string
}

var botAvatar botPhotoCache

func (c *botPhotoCache) load(photo *td.ProfilePhoto, source botPhotoSource) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if photo == nil || photo.Small == nil {
		c.photoID, c.photo, c.contentType = 0, nil, ""
		return ""
	}

	if c.photoID != photo.Id || len(c.photo) == 0 {
		c.photoID, c.photo, c.contentType = photo.Id, nil, ""
		file, err := source.DownloadFile(photo.Small.Id, 1, &td.DownloadFileOpts{Synchronous: true})
		if err == nil && file != nil && file.Local != nil && file.Local.IsDownloadingCompleted {
			c.photo, c.contentType = readPhotoFile(file.Local.Path)
		}
	}
	if len(c.photo) == 0 {
		return ""
	}
	return "/static/bot-avatar.jpg?v=" + strconv.FormatInt(c.photoID, 10)
}

func readPhotoFile(path string) ([]byte, string) {
	file, err := os.Open(path)
	if err != nil {
		return nil, ""
	}
	defer file.Close()
	const maxPhotoSize = 1 << 20
	data, err := io.ReadAll(io.LimitReader(file, maxPhotoSize+1))
	if err != nil || len(data) == 0 || len(data) > maxPhotoSize {
		return nil, ""
	}
	contentType := http.DetectContentType(data)
	if contentType != "image/jpeg" && contentType != "image/png" && contentType != "image/webp" {
		return nil, ""
	}
	return data, contentType
}

func (c *botPhotoCache) servePhoto(w http.ResponseWriter, r *http.Request) {
	c.mu.Lock()
	photo, contentType := c.photo, c.contentType
	c.mu.Unlock()
	if len(photo) == 0 {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "public, max-age=900")
	http.ServeContent(w, r, "bot-avatar.jpg", time.Time{}, bytes.NewReader(photo))
}

func sendBotProfile(bot *td.Client, client *Client) {
	if bot == nil || bot.Me == nil {
		return
	}
	user := bot.Me
	profile := publicBotProfile(user)
	if photo := botAvatar.load(user.ProfilePhoto, bot); photo != "" {
		profile["photoUrl"] = photo
	}
	payload, _ := json.Marshal(EventMessage{Event: "bot_profile", Data: profile})
	_ = client.SendMessage(string(payload))
}
