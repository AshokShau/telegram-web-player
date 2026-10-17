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
	complete    bool
	retryAfter  time.Time
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

func telegramThumbnailHandler(bot botPhotoSource, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	key := strings.TrimPrefix(r.URL.Path, "/thumbnail/telegram/")
	entry, ok := telegramThumbnails.Get(key)
	if !ok {
		http.NotFound(w, r)
		return
	}

	entry.mu.Lock()
	if entry.file != nil && !entry.complete && !time.Now().Before(entry.retryAfter) {
		select {
		case telegramThumbnailDownloads <- struct{}{}:
		case <-r.Context().Done():
			entry.mu.Unlock()
			return
		}
		file := entry.file
		var err error
		if (file.Local == nil || !file.Local.IsDownloadingCompleted) && bot != nil {
			file, err = bot.DownloadFile(file.Id, 1, &td.DownloadFileOpts{Synchronous: true})
		}
		if err == nil && file != nil && file.Local != nil && file.Local.IsDownloadingCompleted {
			if data, contentType := readPhotoFile(file.Local.Path); len(data) > 0 {
				entry.image, entry.contentType, entry.complete = data, contentType, true
			}
		}
		if !entry.complete {
			entry.retryAfter = time.Now().Add(time.Second)
		}
		<-telegramThumbnailDownloads
	}
	data, contentType, complete := entry.image, entry.contentType, entry.complete || entry.file == nil
	entry.mu.Unlock()
	if len(data) == 0 {
		w.Header().Set("Retry-After", "1")
		http.Error(w, "Thumbnail temporarily unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if complete {
		w.Header().Set("Cache-Control", "public, max-age=900")
	}
	http.ServeContent(w, r, "telegram-thumbnail", time.Time{}, bytes.NewReader(data))
}
