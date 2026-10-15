package webapp

import (
	"ashokshau/tg-web/internal/cache"
	"bytes"
	"crypto/sha256"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	td "github.com/AshokShau/gotdbot"
)

type telegramThumbnail struct {
	mu          sync.Mutex
	file        *td.File
	image       []byte
	contentType string
	attempted   bool
}

var (
	telegramThumbnails         = cache.NewCache[*telegramThumbnail](24 * time.Hour)
	telegramThumbnailsMu       sync.Mutex
	telegramThumbnailDownloads = make(chan struct{}, 1)
)

func TelegramThumbnail(thumbnail *td.Thumbnail, mini *td.Minithumbnail) string {
	entry := &telegramThumbnail{}
	if thumbnail != nil && thumbnail.File != nil && thumbnail.File.Id > 0 && thumbnail.File.Size <= 1<<20 {
		entry.file = thumbnail.File
		if thumbnail.Format != nil {
			switch thumbnail.Format.GetType() {
			case "thumbnailFormatMpeg4", "thumbnailFormatWebm", "thumbnailFormatTgs":
				entry.file = nil
			}
		}
	}
	if mini != nil && len(mini.Data) > 0 && len(mini.Data) <= 1<<20 {
		entry.image = bytes.Clone(mini.Data)
		entry.contentType = "image/jpeg"
	}
	if entry.file == nil && len(entry.image) == 0 {
		return ""
	}
	identity := "mini:" + string(entry.image)
	if entry.file != nil {
		identity = fmt.Sprintf("file:%d", entry.file.Id)
	}
	key := fmt.Sprintf("%x", sha256.Sum256([]byte(identity)))
	telegramThumbnailsMu.Lock()
	if _, exists := telegramThumbnails.Get(key); !exists {
		telegramThumbnails.Set(key, entry)
	}
	telegramThumbnailsMu.Unlock()
	return "/thumbnail/telegram/" + key
}

func telegramThumbnailHandler(bot *td.Client, w http.ResponseWriter, r *http.Request) {
	key := strings.TrimPrefix(r.URL.Path, "/thumbnail/telegram/")
	entry, ok := telegramThumbnails.Get(key)
	if !ok {
		http.NotFound(w, r)
		return
	}

	entry.mu.Lock()
	canDownload := false
	if entry.file != nil && !entry.attempted {
		select {
		case telegramThumbnailDownloads <- struct{}{}:
			canDownload = true
		default:
		}
	}
	if canDownload {
		file := entry.file
		var err error
		if (file.Local == nil || !file.Local.IsDownloadingCompleted) && bot != nil {
			file, err = bot.DownloadFile(file.Id, 1, &td.DownloadFileOpts{Synchronous: true})
		}
		entry.attempted = true
		entry.image = nil
		if err == nil && file != nil && file.Local != nil && file.Local.IsDownloadingCompleted {
			entry.image, entry.contentType = readPhotoFile(file.Local.Path)
		}
		<-telegramThumbnailDownloads
	}
	data, contentType := entry.image, entry.contentType
	entry.mu.Unlock()
	if len(data) == 0 {
		http.Redirect(w, r, "/static/assets/artwork.svg", http.StatusFound)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "public, max-age=900")
	http.ServeContent(w, r, "telegram-thumbnail", time.Time{}, bytes.NewReader(data))
}
